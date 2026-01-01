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
	SubDomainSuffix   string        `mapstructure:"subdomain_suffix"`
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
	v.SetDefault("subdomain_suffix", "localhost")
	v.SetDefault("log_level", "info")
	v.SetDefault("log_format", "json")
	v.SetDefault("read_timeout", "30s")
	v.SetDefault("write_timeout", "30s")
	v.SetDefault("idle_timeout", "120s")
	v.SetDefault("ping_interval", "30s")
	v.SetDefault("connection_timeout", "10s")
	v.SetDefault("redis_url", "redis://localhost:6379")

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

	if c.RedisURL == "" {
		return fmt.Errorf("redis URL cannot be empty")
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

// ClientConfig represents the client configuration
type ClientConfig struct {
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
}

// ServerNode represents a single server in the cluster
type ServerNode struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// LoadClientConfig loads the client configuration
func LoadClientConfig(configPath string) (*ClientConfig, error) {
	v := viper.New()

	// Set defaults
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
	// Check if either single server or cluster is configured
	if c.ServerHost == "" && len(c.ServerCluster) == 0 {
		return fmt.Errorf("either server_host or server_cluster must be configured")
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
	if len(c.ServerCluster) > 0 {
		return c.ServerCluster
	}
	// Fallback to single server (backward compatibility)
	return []ServerNode{{Host: c.ServerHost, Port: c.ControlPort}}
}
