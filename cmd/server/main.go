package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/sombochea/tungo/internal/proxy"
	"github.com/sombochea/tungo/internal/registry"
	"github.com/sombochea/tungo/internal/server"
	"github.com/sombochea/tungo/pkg/config"
)

func main() {
	// Load configuration
	cfg, err := config.LoadServerConfig("")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	if err := cfg.Validate(); err != nil {
		log.Fatal().Err(err).Msg("Invalid configuration")
	}

	// Setup logger
	setupLogger(cfg)

	log.Info().Msg("Starting tungo server")
	log.Info().
		Str("server_id", cfg.ID).
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Int("control_port", cfg.ControlPort).
		Str("subdomain_suffix", cfg.SubDomainSuffix).
		Str("redis_url", cfg.RedisURL).
		Msg("Server configuration")

	// Initialize distributed registry (Redis as datastore)
	slogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	distRegistry, err := registry.NewDistributedRegistry(cfg.RedisURL, cfg.ID, slogger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize distributed registry")
	}
	defer distRegistry.Close()

	// Register this server and start heartbeat
	serverInfo := &registry.ServerInfo{
		ServerID:    cfg.ID,
		Host:        cfg.Host,
		ProxyPort:   cfg.Port,
		ControlPort: cfg.ControlPort,
	}
	if err := distRegistry.RegisterServer(serverInfo); err != nil {
		log.Fatal().Err(err).Msg("Failed to register server")
	}
	distRegistry.StartHeartbeat(serverInfo)

	// Initialize server proxy for cross-server communication
	serverProxy := proxy.NewServerProxy(distRegistry, slogger)

	log.Info().Str("redis_url", cfg.RedisURL).Msg("Redis datastore initialized")

	// Create connection manager (Redis-backed, no SQLite)
	connMgr := server.NewConnectionManager(distRegistry, log.Logger, cfg.MaxConnections)

	// Create control server
	controlServer := server.NewControlServer(cfg, connMgr, log.Logger, distRegistry)

	// Create proxy handler
	proxyHandler := server.NewProxyHandler(connMgr, log.Logger)

	// Create Fiber app for control server
	controlApp := fiber.New(fiber.Config{
		AppName:      "TunGo Control Server",
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	})

	// WebSocket upgrader
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}

	// WebSocket handler
	controlApp.Get("/ws", adaptor.HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error().Err(err).Msg("Failed to upgrade WebSocket")
			return
		}
		defer conn.Close()

		// Wrap the gorilla/websocket connection for compatibility
		controlServer.HandleConnection(conn)
	})))

	// Health check endpoint
	controlApp.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":      "ok",
			"connections": connMgr.GetActiveConnections(),
			"subdomains":  connMgr.ListSubDomains(),
		})
	})

	// Start control server
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.ControlPort)
		log.Info().Str("addr", addr).Msg("Control server listening")
		if err := controlApp.Listen(addr); err != nil {
			log.Fatal().Err(err).Msg("Control server failed")
		}
	}()

	// Create Fiber app for HTTP proxy
	proxyApp := fiber.New(fiber.Config{
		AppName:      "TunGo Proxy Server",
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	})

	// Catch-all handler for subdomain routing
	proxyApp.All("/*", func(c fiber.Ctx) error {
		host := c.Hostname()

		// Extract subdomain
		subDomain := extractSubDomain(host, cfg.SubDomainSuffix)
		if subDomain == "" {
			return c.Status(fiber.StatusNotFound).SendString("Tunnel not found")
		}

		// Check if we need to proxy to another server (distributed mode)
		shouldProxy, tunnelInfo, err := serverProxy.ShouldProxy(subDomain)
		if err != nil {
			log.Debug().Err(err).Str("subdomain", subDomain).Msg("Tunnel not found in registry")
			// Fall through to local check
		} else if shouldProxy {
			// Proxy to the server that owns this tunnel
			log.Info().
				Str("subdomain", subDomain).
				Str("target_server", tunnelInfo.ServerID).
				Msg("Proxying request to remote server")

			// Convert Fiber context to standard http.Request
			w := &responseWriter{c: c, headers: make(http.Header)}
			r, _ := http.NewRequest(
				c.Method(),
				c.OriginalURL(),
				nil,
			)
			r.Host = host

			// Copy headers from Fiber context
			c.Request().Header.VisitAll(func(key, value []byte) {
				r.Header.Add(string(key), string(value))
			})

			if err := serverProxy.ProxyToServer(w, r, tunnelInfo); err != nil {
				log.Error().Err(err).Msg("Failed to proxy request")
				return c.Status(fiber.StatusBadGateway).SendString("Failed to proxy request")
			}

			// Copy response headers back to Fiber
			for k, vals := range w.headers {
				for _, v := range vals {
					c.Response().Header.Add(k, v)
				}
			}

			return nil
		}

		// Get client connection from local connection manager
		client, exists := connMgr.GetClientBySubDomain(subDomain)
		if !exists {
			return c.Status(fiber.StatusBadGateway).SendString("Tunnel not active")
		}

		// Handle the request through the tunnel
		return proxyHandler.HandleRequest(c, client)
	})

	// Start proxy server
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
		log.Info().Str("addr", addr).Msg("Proxy server listening")
		if err := proxyApp.Listen(addr); err != nil {
			log.Fatal().Err(err).Msg("Proxy server failed")
		}
	}()

	// Start metrics server
	go func() {
		metricsPort := 9090
		http.Handle("/metrics", promhttp.Handler())
		addr := fmt.Sprintf("%s:%d", cfg.Host, metricsPort)
		log.Info().Str("addr", addr).Msg("Metrics server listening")
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Error().Err(err).Msg("Metrics server failed")
		}
	}()

	// Start load update goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			activeConns := connMgr.GetActiveConnectionsCount()
			if err := distRegistry.UpdateServerLoad(activeConns); err != nil {
				log.Warn().Err(err).Msg("Failed to update server load")
			}
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Info().Msg("Shutting down server...")

	// Graceful shutdown

	if err := controlApp.Shutdown(); err != nil {
		log.Error().Err(err).Msg("Control server shutdown error")
	}

	if err := proxyApp.Shutdown(); err != nil {
		log.Error().Err(err).Msg("Proxy server shutdown error")
	}

	log.Info().Msg("Server stopped")
}

func setupLogger(cfg *config.ServerConfig) {
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

func extractSubDomain(host, suffix string) string {
	// Simple subdomain extraction
	// Example: "test.localhost" with suffix "localhost" -> "test"
	if len(host) <= len(suffix) {
		return ""
	}

	if host[len(host)-len(suffix):] != suffix {
		return ""
	}

	subDomain := host[:len(host)-len(suffix)-1]
	return subDomain
}

// responseWriter is a wrapper to adapt fiber context to http.ResponseWriter
type responseWriter struct {
	c       fiber.Ctx
	headers http.Header
	status  int
}

func (w *responseWriter) Header() http.Header {
	return w.headers
}

func (w *responseWriter) Write(b []byte) (int, error) {
	return w.c.Write(b)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.c.Status(statusCode)
}
