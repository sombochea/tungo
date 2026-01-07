package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
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
		Str("domain", cfg.Domain).
		Str("redis_url", cfg.RedisURL).
		Msg("Server configuration")

	// Initialize registry (auto-detect: Redis if URL provided, otherwise in-memory)
	slogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	datastore, err := registry.NewRegistry(cfg.RedisURL, cfg.ID, slogger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize registry")
	}
	defer datastore.Close()

	// Log the datastore mode
	if cfg.RedisURL == "" {
		log.Info().Msg("Using in-memory datastore (non-distributed mode)")
	} else {
		log.Info().Str("redis_url", cfg.RedisURL).Msg("Using Redis datastore (distributed mode)")
	}

	// Register this server and start heartbeat
	serverInfo := &registry.ServerInfo{
		ServerID:    cfg.ID,
		Host:        cfg.Host,
		ProxyPort:   cfg.Port,
		ControlPort: cfg.ControlPort,
	}
	if err := datastore.RegisterServer(serverInfo); err != nil {
		log.Fatal().Err(err).Msg("Failed to register server")
	}
	datastore.StartHeartbeat(serverInfo)

	// Initialize server proxy for cross-server communication
	serverProxy := proxy.NewServerProxy(datastore, slogger)

	// Create connection manager
	connMgr := server.NewConnectionManager(datastore, log.Logger, cfg.MaxConnections)

	// Create control server
	controlServer := server.NewControlServer(cfg, connMgr, log.Logger, datastore)

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
		subDomain := extractSubDomain(host, cfg.Domain)
		if subDomain == "" {
			return sendPrettyError(c, fiber.StatusNotFound,
				"Tunnel Not Found",
				"No tunnel is configured for this subdomain. Please check your tunnel URL and ensure your client is connected.")
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
				return sendPrettyError(c, fiber.StatusBadGateway,
					"Proxy Error",
					"Unable to forward your request to the target server. The remote tunnel server may be unavailable.")
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
			return sendPrettyError(c, fiber.StatusServiceUnavailable,
				"Tunnel Not Active",
				"This tunnel is currently not connected. Please start your tunnel client and try again.")
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
			if err := datastore.UpdateServerLoad(activeConns); err != nil {
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

func extractSubDomain(host, domainTemplate string) string {
	// Extract subdomain from domain template
	// Examples:
	// - "{{ .subdomain }}.localhost" with host "test.localhost" -> "test"
	// - "{{ .subdomain }}-tungo.example.com" with host "test-tungo.example.com" -> "test"

	// Find the subdomain placeholder position
	placeholder := "{{ .subdomain }}"
	idx := strings.Index(domainTemplate, placeholder)
	if idx == -1 {
		return ""
	}

	// Extract prefix and suffix around the placeholder
	prefix := domainTemplate[:idx]
	suffix := domainTemplate[idx+len(placeholder):]

	// Check if host matches the pattern
	if !strings.HasPrefix(host, prefix) || !strings.HasSuffix(host, suffix) {
		return ""
	}

	// Extract the subdomain
	subDomain := host[len(prefix) : len(host)-len(suffix)]
	return subDomain
}

// sendPrettyError sends a user-friendly HTML error response
func sendPrettyError(c fiber.Ctx, status int, title, message string) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            padding: 20px;
        }
        .error-container {
            background: white;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            padding: 60px 40px;
            max-width: 600px;
            text-align: center;
        }
        .error-icon {
            font-size: 72px;
            margin-bottom: 20px;
        }
        h1 {
            color: #333;
            font-size: 32px;
            margin-bottom: 16px;
            font-weight: 700;
        }
        p {
            color: #666;
            font-size: 18px;
            line-height: 1.6;
            margin-bottom: 32px;
        }
        .status-code {
            display: inline-block;
            background: #f0f0f0;
            color: #888;
            padding: 8px 16px;
            border-radius: 20px;
            font-size: 14px;
            font-weight: 600;
            margin-top: 20px;
        }
        .footer {
            margin-top: 40px;
            color: #999;
            font-size: 14px;
        }
        a {
            color: #667eea;
            text-decoration: none;
            font-weight: 600;
        }
        a:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="error-container">
        <div class="error-icon">🔌</div>
        <h1>%s</h1>
        <p>%s</p>
        <div class="status-code">Status Code: %d</div>
        <div class="footer">
            Powered by <a href="https://github.com/sombochea/tungo">TunGo</a>
        </div>
    </div>
</body>
</html>`, title, title, message, status)
	return c.Status(status).SendString(html)
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
