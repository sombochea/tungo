package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// ServerConfig represents the server configuration
type ServerConfig struct {
	ID                string        `mapstructure:"id"`
	Host              string        `mapstructure:"host"`
	Port              int           `mapstructure:"port"`
	ControlPort       int           `mapstructure:"control_port"`
	ProxyStartPort    int           `mapstructure:"proxy_start_port"`
	ProxyEndPort      int           `mapstructure:"proxy_end_port"`
	MaxConnections    int           `mapstructure:"max_connections"`
	RequireAuth       bool          `mapstructure:"require_auth"`
	AllowAnonymous    bool          `mapstructure:"allow_anonymous"`
	Domain            string        `mapstructure:"domain"`
	PublicURL         string        `mapstructure:"public_url"`
	LogLevel          string        `mapstructure:"log_level"`
	LogFormat         string        `mapstructure:"log_format"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`
	PingInterval      time.Duration `mapstructure:"ping_interval"`
	ConnectionTimeout time.Duration `mapstructure:"connection_timeout"`
	// Redis datastore (required)
	RedisURL string `mapstructure:"redis_url"`
}

// LoadServerConfig loads the server configuration
func LoadServerConfig(configPath string) (*ServerConfig, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("id", "server-1")
	v.SetDefault("host", "0.0.0.0")
	v.SetDefault("port", 8080)
	v.SetDefault("control_port", 5000)
	v.SetDefault("proxy_start_port", 10000)
	v.SetDefault("proxy_end_port", 20000)
	v.SetDefault("max_connections", 1000)
	v.SetDefault("require_auth", false)
	v.SetDefault("allow_anonymous", true)
	v.SetDefault("domain", "{{ .subdomain }}.localhost")
	v.SetDefault("public_url", "http://{{ .domain }}:{{ .port }}")
	v.SetDefault("log_level", "info")
	v.SetDefault("log_format", "json")
	v.SetDefault("read_timeout", "30s")
	v.SetDefault("write_timeout", "30s")
	v.SetDefault("idle_timeout", "120s")
	v.SetDefault("ping_interval", "30s")
	v.SetDefault("connection_timeout", "10s")
	v.SetDefault("redis_url", "") // Empty by default - will use in-memory mode

	// Set configuration file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("server")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("/etc/tungo")
	}

	// Enable environment variables
	v.SetEnvPrefix("TUNGO_SERVER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Explicitly bind Redis URL environment variable
	v.BindEnv("redis_url")

	// Read configuration
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var config ServerConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// Validate validates the server configuration
func (c *ServerConfig) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	if c.ControlPort <= 0 || c.ControlPort > 65535 {
		return fmt.Errorf("invalid control port: %d", c.ControlPort)
	}

	if c.ProxyStartPort <= 0 || c.ProxyStartPort > 65535 {
		return fmt.Errorf("invalid proxy start port: %d", c.ProxyStartPort)
	}

	if c.ProxyEndPort <= 0 || c.ProxyEndPort > 65535 {
		return fmt.Errorf("invalid proxy end port: %d", c.ProxyEndPort)
	}

	if c.ProxyStartPort >= c.ProxyEndPort {
		return fmt.Errorf("proxy start port must be less than end port")
	}

	if c.MaxConnections <= 0 {
		return fmt.Errorf("max connections must be positive")
	}

	// Redis URL is now optional - if not provided, server will use in-memory mode
	// No validation needed for empty redis_url

	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s", c.LogLevel)
	}

	validLogFormats := map[string]bool{
		"json": true, "console": true,
	}
	if !validLogFormats[c.LogFormat] {
		return fmt.Errorf("invalid log format: %s", c.LogFormat)
	}

	return nil
}

// ClientConfig represents the client configuration
type ClientConfig struct {
	ServerURL       string        `mapstructure:"server_url"`     // Full server URL (e.g., https://tungo.example.com or wss://tungo.example.com)
	ServerHost      string        `mapstructure:"server_host"`    // Primary server (backward compatibility)
	ControlPort     int           `mapstructure:"control_port"`   // Primary port (backward compatibility)
	ServerCluster   []ServerNode  `mapstructure:"server_cluster"` // Multiple servers for failover
	LocalHost       string        `mapstructure:"local_host"`
	LocalPort       int           `mapstructure:"local_port"`
	SubDomain       string        `mapstructure:"subdomain"`
	SecretKey       string        `mapstructure:"secret_key"`
	ReconnectToken  string        `mapstructure:"reconnect_token"`
	LogLevel        string        `mapstructure:"log_level"`
	LogFormat       string        `mapstructure:"log_format"`
	ConnectTimeout  time.Duration `mapstructure:"connect_timeout"`
	RetryInterval   time.Duration `mapstructure:"retry_interval"`
	MaxRetries      int           `mapstructure:"max_retries"`
	DashboardPort   int           `mapstructure:"dashboard_port"`
	EnableDashboard bool          `mapstructure:"enable_dashboard"`
	InsecureTLS     bool          `mapstructure:"insecure_tls"` // Skip TLS certificate verification (for testing only)
}

// ServerNode represents a single server in the cluster
type ServerNode struct {
	Host   string `mapstructure:"host"`
	Port   int    `mapstructure:"port"`
	Secure bool   `mapstructure:"secure"` // Use wss:// instead of ws://
}

// LoadClientConfig loads the client configuration
func LoadClientConfig(configPath string) (*ClientConfig, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server_url", "")
	v.SetDefault("server_host", "localhost")
	v.SetDefault("control_port", 5000)
	v.SetDefault("local_host", "localhost")
	v.SetDefault("local_port", 8000)
	v.SetDefault("subdomain", "")
	v.SetDefault("secret_key", "")
	v.SetDefault("reconnect_token", "")
	v.SetDefault("log_level", "info")
	v.SetDefault("log_format", "console")
	v.SetDefault("connect_timeout", "10s")
	v.SetDefault("retry_interval", "5s")
	v.SetDefault("max_retries", 5)
	v.SetDefault("dashboard_port", 3000)
	v.SetDefault("enable_dashboard", false)
	v.SetDefault("insecure_tls", false)

	// Set configuration file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("client")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("$HOME/.tungo")
	}

	// Enable environment variables
	v.SetEnvPrefix("TUNGO_CLIENT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read configuration
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var config ClientConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// Validate validates the client configuration
func (c *ClientConfig) Validate() error {
	// Check if either ServerURL, single server, or cluster is configured
	if c.ServerURL == "" && c.ServerHost == "" && len(c.ServerCluster) == 0 {
		return fmt.Errorf("either server_url, server_host, or server_cluster must be configured")
	}

	// Validate single server config (if provided)
	if c.ServerHost != "" {
		if c.ControlPort <= 0 || c.ControlPort > 65535 {
			return fmt.Errorf("invalid control port: %d", c.ControlPort)
		}
	}

	// Validate cluster config (if provided)
	for i, node := range c.ServerCluster {
		if node.Host == "" {
			return fmt.Errorf("server_cluster[%d]: host cannot be empty", i)
		}
		if node.Port <= 0 || node.Port > 65535 {
			return fmt.Errorf("server_cluster[%d]: invalid port: %d", i, node.Port)
		}
	}

	if c.LocalHost == "" {
		return fmt.Errorf("local host cannot be empty")
	}

	if c.LocalPort <= 0 || c.LocalPort > 65535 {
		return fmt.Errorf("invalid local port: %d", c.LocalPort)
	}

	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s", c.LogLevel)
	}

	validLogFormats := map[string]bool{
		"json": true, "console": true,
	}
	if !validLogFormats[c.LogFormat] {
		return fmt.Errorf("invalid log format: %s", c.LogFormat)
	}

	return nil
}

// GetServerList returns the list of servers to try (cluster if available, otherwise single server)
func (c *ClientConfig) GetServerList() []ServerNode {
	// If ServerURL is provided, parse it first
	if c.ServerURL != "" {
		if host, port, secure, err := ParseServerURL(c.ServerURL); err == nil {
			return []ServerNode{{Host: host, Port: port, Secure: secure}}
		}
	}

	if len(c.ServerCluster) > 0 {
		return c.ServerCluster
	}
	// Fallback to single server (backward compatibility)
	return []ServerNode{{Host: c.ServerHost, Port: c.ControlPort, Secure: false}}
}

// ParseServerURL parses a full server URL and extracts host, port, and secure flag
// Supports formats: https://example.com, wss://example.com:5000, http://example.com:8080
func ParseServerURL(serverURL string) (host string, port int, secure bool, err error) {
	// Add scheme if not present
	if !strings.HasPrefix(serverURL, "http://") &&
		!strings.HasPrefix(serverURL, "https://") &&
		!strings.HasPrefix(serverURL, "ws://") &&
		!strings.HasPrefix(serverURL, "wss://") {
		serverURL = "https://" + serverURL
	}

	// Parse URL
	parsedURL, err := parseURL(serverURL)
	if err != nil {
		return "", 0, false, fmt.Errorf("invalid server URL: %w", err)
	}

	host = parsedURL.Hostname()

	// Determine if secure based on scheme
	secure = (parsedURL.Scheme == "https" || parsedURL.Scheme == "wss")

	// Determine port
	if parsedURL.Port() != "" {
		// Explicit port in URL
		_, err := fmt.Sscanf(parsedURL.Port(), "%d", &port)
		if err != nil {
			return "", 0, false, fmt.Errorf("invalid port in URL: %w", err)
		}
	} else {
		// Default ports based on scheme
		switch parsedURL.Scheme {
		case "https", "wss":
			port = 443 // Standard HTTPS/WSS port
		case "http", "ws":
			port = 80 // Standard HTTP/WS port
		default:
			port = 5000 // Default tungo control port for other cases
		}
	}

	return host, port, secure, nil
}

// Helper function to avoid import cycle
func parseURL(rawURL string) (*urlParts, error) {
	// Simple URL parser for our needs
	var parts urlParts

	// Extract scheme
	schemeIdx := strings.Index(rawURL, "://")
	if schemeIdx == -1 {
		return nil, fmt.Errorf("missing scheme")
	}
	parts.Scheme = rawURL[:schemeIdx]
	rest := rawURL[schemeIdx+3:]

	// Extract host and port
	pathIdx := strings.Index(rest, "/")
	var hostPort string
	if pathIdx == -1 {
		hostPort = rest
	} else {
		hostPort = rest[:pathIdx]
	}

	// Split host and port
	portIdx := strings.LastIndex(hostPort, ":")
	if portIdx != -1 && strings.Count(hostPort, ":") == 1 {
		// IPv4 with port or hostname with port
		parts.Host = hostPort[:portIdx]
		parts.PortStr = hostPort[portIdx+1:]
	} else {
		// No port or IPv6
		parts.Host = hostPort
	}

	return &parts, nil
}

type urlParts struct {
	Scheme  string
	Host    string
	PortStr string
}

func (u *urlParts) Hostname() string {
	return u.Host
}

func (u *urlParts) Port() string {
	return u.PortStr
}
