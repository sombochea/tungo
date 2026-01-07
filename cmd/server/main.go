package main

import (
	"crypto/sha256"
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

		// Check password authentication if client has set one
		if client.Password != "" {
			authenticated := false
			providedPassword := ""

			// Check x-tungo-password header first (for API access)
			providedPassword = c.Get("x-tungo-password")

			if providedPassword == "" {
				// Check auth cookie
				authCookie := c.Cookies("tungo-auth-" + subDomain)
				if authCookie != "" {
					// Verify cookie matches expected password hash
					expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(client.Password)))
					if authCookie == expectedHash {
						authenticated = true
					}
				}
			}

			// If header provided, verify it
			if providedPassword != "" {
				if providedPassword == client.Password {
					authenticated = true
					// Set cookie for browser sessions (valid for 24 hours)
					c.Cookie(&fiber.Cookie{
						Name:     "tungo-auth-" + subDomain,
						Value:    fmt.Sprintf("%x", sha256.Sum256([]byte(client.Password))),
						Path:     "/",
						MaxAge:   86400, // 24 hours
						HTTPOnly: true,
						Secure:   false, // Set to true if using HTTPS
						SameSite: "Lax",
					})
					// Return success response for auth check (don't proxy yet)
					// The client will reload to get the actual content
					return c.JSON(fiber.Map{"authenticated": true})
				} else {
					// Wrong password
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"authenticated": false, "error": "invalid password"})
				}
			}

			if !authenticated {
				// Return 401 with password prompt for browsers (no WWW-Authenticate to avoid browser dialog)
				c.Set("Content-Type", "text/html; charset=utf-8")
				return c.Status(fiber.StatusUnauthorized).SendString(getPasswordPromptHTML())
			}
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

// getPasswordPromptHTML returns HTML for password authentication
func getPasswordPromptHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Authentication Required - TunGo</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
            padding: 20px;
        }
        .auth-container {
            background: white;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            padding: 48px 40px;
            max-width: 420px;
            width: 100%;
        }
        .lock-icon {
            font-size: 72px;
            margin-bottom: 24px;
            text-align: center;
            animation: pulse 2s ease-in-out infinite;
        }
        @keyframes pulse {
            0%, 100% { transform: scale(1); }
            50% { transform: scale(1.05); }
        }
        h1 {
            font-size: 28px;
            color: #2d3748;
            margin-bottom: 12px;
            text-align: center;
            font-weight: 700;
        }
        .subtitle {
            color: #718096;
            margin-bottom: 32px;
            font-size: 15px;
            text-align: center;
            line-height: 1.5;
        }
        .form-group {
            margin-bottom: 24px;
        }
        label {
            display: block;
            color: #4a5568;
            font-size: 14px;
            font-weight: 600;
            margin-bottom: 8px;
            text-align: left;
        }
        .password-input-wrapper {
            position: relative;
        }
        input[type="password"], input[type="text"] {
            width: 100%;
            padding: 14px 44px 14px 16px;
            border: 2px solid #e2e8f0;
            border-radius: 10px;
            font-size: 15px;
            transition: all 0.3s ease;
            background: #f7fafc;
        }
        input[type="password"]:focus, input[type="text"]:focus {
            outline: none;
            border-color: #667eea;
            background: white;
            box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
        }
        .toggle-password {
            position: absolute;
            right: 14px;
            top: 50%;
            transform: translateY(-50%);
            background: none;
            border: none;
            cursor: pointer;
            font-size: 20px;
            color: #a0aec0;
            padding: 4px;
            transition: color 0.2s;
        }
        .toggle-password:hover {
            color: #667eea;
        }
        .submit-btn {
            width: 100%;
            padding: 14px 24px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 10px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s ease;
            box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
        }
        .submit-btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(102, 126, 234, 0.6);
        }
        .submit-btn:active {
            transform: translateY(0);
        }
        .error-message {
            background: #fed7d7;
            color: #c53030;
            padding: 12px;
            border-radius: 8px;
            margin-bottom: 20px;
            font-size: 14px;
            display: none;
            text-align: center;
        }
        .error-message.show {
            display: block;
        }
        .api-hint {
            margin-top: 24px;
            padding: 16px;
            background: #f0f4ff;
            border-radius: 10px;
            border-left: 4px solid #667eea;
        }
        .api-hint-title {
            color: #4c51bf;
            font-size: 13px;
            font-weight: 600;
            margin-bottom: 6px;
        }
        .api-hint-content {
            color: #5a67d8;
            font-size: 12px;
            font-family: 'Courier New', monospace;
            word-break: break-all;
        }
        .footer {
            margin-top: 32px;
            text-align: center;
            color: #a0aec0;
            font-size: 13px;
        }
        .footer a {
            color: #667eea;
            text-decoration: none;
            font-weight: 600;
            transition: color 0.2s;
        }
        .footer a:hover {
            color: #764ba2;
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="auth-container">
        <div class="lock-icon">🔒</div>
        <h1>Authentication Required</h1>
        <p class="subtitle">This tunnel is password protected. Please enter the password to continue.</p>
        
        <div id="errorMessage" class="error-message">
            Invalid password. Please try again.
        </div>

        <form id="authForm" onsubmit="return handleSubmit(event)">
            <div class="form-group">
                <label for="password">Password</label>
                <div class="password-input-wrapper">
                    <input 
                        type="password" 
                        id="password" 
                        name="password" 
                        placeholder="Enter tunnel password"
                        required 
                        autocomplete="current-password"
                        autofocus
                    />
                    <button type="button" class="toggle-password" onclick="togglePassword()" title="Show/Hide password">
                        <span id="toggleIcon">👁️</span>
                    </button>
                </div>
            </div>
            <button type="submit" class="submit-btn">Access Tunnel</button>
        </form>

        <div class="api-hint">
            <div class="api-hint-title">💡 API Access</div>
            <div class="api-hint-content">x-tungo-password: your_password</div>
        </div>

        <div class="footer">
            Powered by <a href="https://github.com/sombochea/tungo" target="_blank">TunGo</a>
        </div>
    </div>

    <script>
        function togglePassword() {
            const passwordInput = document.getElementById('password');
            const toggleIcon = document.getElementById('toggleIcon');
            
            if (passwordInput.type === 'password') {
                passwordInput.type = 'text';
                toggleIcon.textContent = '🙈';
            } else {
                passwordInput.type = 'password';
                toggleIcon.textContent = '👁️';
            }
        }

        function handleSubmit(event) {
            event.preventDefault();
            
            const password = document.getElementById('password').value;
            const errorMessage = document.getElementById('errorMessage');
            
            // Send request with password in header
            fetch(window.location.href, {
                method: 'GET',
                headers: {
                    'x-tungo-password': password
                }
            })
            .then(response => {
                if (response.ok) {
                    // Password correct, reload page to show content
                    window.location.reload();
                } else {
                    // Show error message
                    errorMessage.classList.add('show');
                    document.getElementById('password').value = '';
                    document.getElementById('password').focus();
                    
                    // Hide error after 3 seconds
                    setTimeout(() => {
                        errorMessage.classList.remove('show');
                    }, 3000);
                }
            })
            .catch(error => {
                errorMessage.textContent = 'Connection error. Please try again.';
                errorMessage.classList.add('show');
            });
            
            return false;
        }

        // Check for stored auth and auto-submit
        document.addEventListener('DOMContentLoaded', function() {
            // Focus on password input
            document.getElementById('password').focus();
        });
    </script>
</body>
</html>`
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
