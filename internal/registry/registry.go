package registry

import (
	"log/slog"
)

// Registry is the interface that all registry implementations must satisfy
type Registry interface {
	// Tunnel operations
	RegisterTunnel(info *TunnelInfo) error
	GetTunnel(subdomain string) (*TunnelInfo, error)
	UnregisterTunnel(subdomain string) error
	RefreshTunnel(subdomain string) error
	GetAllTunnels() ([]*TunnelInfo, error)
	IsLocalTunnel(subdomain string) (bool, error)

	// Server operations
	RegisterServer(info *ServerInfo) error
	GetServer(serverID string) (*ServerInfo, error)
	GetAllServers() ([]*ServerInfo, error)
	StartHeartbeat(serverInfo *ServerInfo)
	GetLeastLoadedServer() (*ServerInfo, error)
	UpdateServerLoad(activeConnections int) error

	// Cache operations
	GetCacheStats() (hits, misses int, hitRate float64)

	// Lifecycle
	Close() error
}

// TunnelInfo stores information about a tunnel in the registry
// (already defined in distributed.go but kept here for reference)

// ServerInfo stores information about a server in the cluster
// (already defined in distributed.go but kept here for reference)

// NewRegistry creates a registry based on the provided Redis URL
// If redisURL is empty, returns an in-memory registry
// Otherwise, returns a distributed Redis-backed registry
func NewRegistry(redisURL, serverID string, logger interface{}) (Registry, error) {
	slogger, ok := logger.(*slog.Logger)
	if !ok {
		slogger = slog.Default()
	}

	if redisURL == "" {
		return NewInMemoryRegistry(serverID, slogger)
	}
	return NewDistributedRegistry(redisURL, serverID, slogger)
}
