package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/sombochea/tungo/internal/client"
	"github.com/sombochea/tungo/internal/client/introspect"
	"github.com/sombochea/tungo/pkg/config"
	"github.com/sombochea/tungo/pkg/version"
)

var (
	cfgFile         string
	serverURL       string
	serverHost      string
	serverPort      int
	localHost       string
	localPort       int
	subDomain       string
	secretKey       string
	password        string
	enableDashboard bool
	dashboardPort   int
	insecureTLS     bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "tungo",
		Short:   "TunGo client - expose your local server to the internet",
		Long:    `TunGo client creates a secure tunnel from a public URL to your local development server.`,
		Version: version.GetShortVersion(),
		Run:     runClient,
	}

	// Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.GetFullVersion())
		},
	}

	// Upgrade command
	upgradeCmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade to the latest version",
		Long:  `Downloads and installs the latest version of TunGo client from GitHub releases.`,
		Run:   runUpgrade,
	}

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(upgradeCmd)

	// Flags for the root command (tunnel)
	rootCmd.Flags().StringVarP(&cfgFile, "config", "c", "", "config file path")
	rootCmd.Flags().StringVar(&serverURL, "server-url", "", "full server URL with control port (e.g., http://tungo.example.com:5555 or ws://tungo.example.com:5555)")
	rootCmd.Flags().StringVar(&serverHost, "server", "localhost", "tungo server host")
	rootCmd.Flags().IntVar(&serverPort, "port", 5555, "tungo server control port")
	rootCmd.Flags().StringVar(&localHost, "local-host", "localhost", "local server host")
	rootCmd.Flags().IntVar(&localPort, "local-port", 8000, "local server port")
	rootCmd.Flags().StringVarP(&subDomain, "subdomain", "s", "", "requested subdomain")
	rootCmd.Flags().StringVarP(&secretKey, "key", "k", "", "secret key for authentication")
	rootCmd.Flags().StringVarP(&password, "password", "p", "", "password to protect tunnel access")
	rootCmd.Flags().BoolVarP(&enableDashboard, "dashboard", "d", false, "enable introspection dashboard")
	rootCmd.Flags().IntVar(&dashboardPort, "dashboard-port", 3000, "introspection dashboard port")
	rootCmd.Flags().BoolVar(&insecureTLS, "insecure", false, "skip TLS certificate verification (for testing only)")

	// Set version template
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runClient(cmd *cobra.Command, args []string) {
	// Load configuration
	cfg, err := config.LoadClientConfig(cfgFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Override with command-line flags
	// ServerURL takes precedence over individual server/port flags
	if serverURL != "" && cmd.Flags().Changed("server-url") {
		cfg.ServerURL = serverURL
		// Clear individual host/port to ensure ServerURL is used
		cfg.ServerHost = ""
		cfg.ControlPort = 0
	} else {
		if serverURL == "" && version.GetShortVersion() != "dev" {
			// For production releases, use default server URL if none provided
			cfg.ServerURL = "wss://singal-tg01.ctdn.dev"
			cfg.ServerHost = ""
			cfg.ControlPort = 0
		} else {
			if serverHost != "" && cmd.Flags().Changed("server") {
				cfg.ServerHost = serverHost
			}
			if cmd.Flags().Changed("port") {
				cfg.ControlPort = serverPort
			}
		}
	}
	if localHost != "" && cmd.Flags().Changed("local-host") {
		cfg.LocalHost = localHost
	}
	if cmd.Flags().Changed("local-port") {
		cfg.LocalPort = localPort
	}
	if subDomain != "" && cmd.Flags().Changed("subdomain") {
		cfg.SubDomain = subDomain
	}
	if secretKey != "" && cmd.Flags().Changed("key") {
		cfg.SecretKey = secretKey
	}
	if password != "" && cmd.Flags().Changed("password") {
		cfg.Password = password
	}
	if cmd.Flags().Changed("dashboard") {
		cfg.EnableDashboard = enableDashboard
	}
	if cmd.Flags().Changed("dashboard-port") {
		cfg.DashboardPort = dashboardPort
	}
	if cmd.Flags().Changed("insecure") {
		cfg.InsecureTLS = insecureTLS
	}

	if err := cfg.Validate(); err != nil {
		log.Fatal().Err(err).Msg("Invalid configuration")
	}

	// Setup logger
	setupLogger(cfg)

	// Start dashboard if enabled
	var dashboard *introspect.Dashboard
	if cfg.EnableDashboard {
		var err error
		dashboard, err = introspect.NewDashboard(cfg.DashboardPort)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create dashboard")
		}
		go func() {
			if err := dashboard.Start(); err != nil {
				log.Error().Err(err).Msg("Dashboard server error")
			}
		}()
		defer dashboard.Stop()
	}

	log.Info().Msg("Starting tungo client")
	log.Info().
		Str("server", fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ControlPort)).
		Str("local", fmt.Sprintf("%s:%d", cfg.LocalHost, cfg.LocalPort)).
		Str("subdomain", cfg.SubDomain).
		Msg("Client configuration")

	// Create tunnel client
	tunnelClient := client.NewTunnelClient(cfg, log.Logger)

	// Setup signal handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Continuous connection loop with auto-reconnect
	firstConnection := true
	serverRotation := 0 // Track server rotation attempts

	for {
		// Connect to server with retry logic
		connected := false
		for retry := 0; retry <= cfg.MaxRetries; retry++ {
			// Check if we should exit
			select {
			case <-quit:
				log.Info().Msg("Shutting down client...")
				tunnelClient.Close()
				return
			default:
			}

			if retry > 0 {
				currentServer := tunnelClient.GetCurrentServer()
				log.Info().
					Int("retry", retry).
					Int("max_retries", cfg.MaxRetries).
					Dur("retry_interval", cfg.RetryInterval).
					Str("server", fmt.Sprintf("%s:%d", currentServer.Host, currentServer.Port)).
					Msg("Retrying connection")
				time.Sleep(cfg.RetryInterval)
			} else if !firstConnection {
				// Not first connection and first retry - wait before reconnecting
				log.Info().Msg("Attempting to reconnect...")
				time.Sleep(cfg.RetryInterval)
			}

			if err := tunnelClient.Connect(); err != nil {
				currentServer := tunnelClient.GetCurrentServer()
				log.Error().
					Err(err).
					Str("server", fmt.Sprintf("%s:%d", currentServer.Host, currentServer.Port)).
					Msg("Failed to connect to server")

				if retry == cfg.MaxRetries {
					// Max retries for current server reached
					if tunnelClient.GetServerCount() > 1 {
						// Rotate to next server in cluster
						tunnelClient.RotateToNextServer()
						serverRotation++

						// If we've tried all servers, wait before retrying
						if serverRotation >= tunnelClient.GetServerCount() {
							log.Warn().Msg("Tried all servers in cluster, will retry cycle again")
							time.Sleep(cfg.RetryInterval)
							serverRotation = 0 // Reset rotation counter
						}
					} else {
						log.Warn().Msg("Max retries reached, will retry cycle again")
						time.Sleep(cfg.RetryInterval)
					}
					break // Break inner loop to restart retry cycle
				}
				continue
			}

			// Successfully connected - reset rotation counter
			connected = true
			serverRotation = 0
			break
		}

		if !connected {
			log.Warn().Msg("Connection cycle failed, retrying...")
			continue // Restart retry cycle
		}

		// Display connection info
		serverInfo := tunnelClient.GetServerInfo()
		currentServer := tunnelClient.GetCurrentServer()

		if firstConnection {
			// Use PublicURL if available, otherwise fall back to Hostname
			publicURL := serverInfo.PublicURL
			if publicURL == "" {
				publicURL = fmt.Sprintf("http://%s", serverInfo.Hostname)
			}

			log.Info().
				Str("url", publicURL).
				Str("subdomain", serverInfo.SubDomain).
				Str("server", fmt.Sprintf("%s:%d", currentServer.Host, currentServer.Port)).
				Int("cluster_size", tunnelClient.GetServerCount()).
				Msg("✓ Tunnel established successfully!")

			fmt.Println()
			fmt.Println("┌────────────────────────────────────────────────────────────┐")
			fmt.Printf("│  🌐 Your tunnel is ready!                                  │\n")
			fmt.Println("├────────────────────────────────────────────────────────────┤")
			fmt.Printf("│  Public URL:  %-44s │\n", publicURL)
			fmt.Printf("│  Local:       http://%-36s │\n", fmt.Sprintf("%s:%d", cfg.LocalHost, cfg.LocalPort))
			if tunnelClient.GetServerCount() > 1 {
				fmt.Printf("│  Cluster:     %d servers (auto-failover enabled)%-9s│\n", tunnelClient.GetServerCount(), "")
			}
			fmt.Println("└────────────────────────────────────────────────────────────┘")
			fmt.Println()
			firstConnection = false
		} else {
			// Use PublicURL if available, otherwise fall back to Hostname
			publicURL := serverInfo.PublicURL
			if publicURL == "" {
				publicURL = fmt.Sprintf("http://%s", serverInfo.Hostname)
			}

			log.Info().
				Str("url", publicURL).
				Str("subdomain", serverInfo.SubDomain).
				Str("server", fmt.Sprintf("%s:%d", currentServer.Host, currentServer.Port)).
				Msg("✓ Reconnected successfully!")
		}

		// Start periodic stats logging
		statsQuit := make(chan struct{})
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					activeStreams := tunnelClient.GetActiveStreams()
					if activeStreams > 0 {
						log.Debug().Int("active_streams", activeStreams).Msg("Client stats")
					}
				case <-statsQuit:
					return
				}
			}
		}()

		// Run the client event loop (blocks until connection drops)
		log.Info().Msg("Starting tunnel...")
		err := tunnelClient.Run()

		// Connection dropped or error
		close(statsQuit)

		select {
		case <-quit:
			// User interrupt during Run()
			log.Info().Msg("Shutting down client...")
			tunnelClient.Close()
			return
		default:
			// Connection dropped, will reconnect
			if err != nil {
				log.Warn().Err(err).Msg("Connection error, will reconnect")
			} else {
				log.Warn().Msg("Connection lost, will reconnect")
			}
			// Continue outer loop to reconnect
		}
	}
}

func setupLogger(cfg *config.ClientConfig) {
	// Set log level
	var level zerolog.Level
	switch cfg.LogLevel {
	case "debug":
		level = zerolog.DebugLevel
	case "info":
		level = zerolog.InfoLevel
	case "warn":
		level = zerolog.WarnLevel
	case "error":
		level = zerolog.ErrorLevel
	case "fatal":
		level = zerolog.FatalLevel
	default:
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Set log format
	if cfg.LogFormat == "console" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	}
}

func runUpgrade(cmd *cobra.Command, args []string) {
	fmt.Println("🔄 Checking for updates...")
	fmt.Printf("Current version: %s\n", version.GetShortVersion())

	// Check for updates
	hasUpdate, latestVersion, err := version.CheckForUpdates()
	if err != nil {
		log.Error().Err(err).Msg("Failed to check for updates")
		fmt.Printf("❌ Failed to check for updates: %v\n", err)
		os.Exit(1)
	}

	if !hasUpdate {
		fmt.Println("✅ You are already running the latest version!")
		return
	}

	fmt.Printf("📦 New version available: %s\n", latestVersion)
	fmt.Println("⬇️  Downloading and installing...")

	// Download and install
	if err := version.DownloadAndInstall(); err != nil {
		log.Error().Err(err).Msg("Failed to upgrade")
		fmt.Printf("❌ Upgrade failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Upgrade completed successfully!")
	fmt.Println("Please run 'tungo' again to use the new version.")
}
