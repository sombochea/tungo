package registry

import (
    "fmt"
    "log/slog"
    "sync"
    "time"
)

// InMemoryRegistry is a simple in-memory implementation of the Registry interface
type InMemoryRegistry struct {
    serverID      string
    logger        *slog.Logger
    tunnels       map[string]*TunnelInfo
    tunnelsMutex  sync.RWMutex
    servers       map[string]*ServerInfo
    serversMutex  sync.RWMutex
    lookups       int
    hits          int
    heartbeatStop chan struct{}
}

// NewInMemoryRegistry creates a new in-memory registry
func NewInMemoryRegistry(serverID string, logger interface{}) (*InMemoryRegistry, error) {
    slogger, ok := logger.(*slog.Logger)
    if !ok {
        slogger = slog.Default()
    }

    slogger.Info("Initializing in-memory registry (non-distributed mode)", "server_id", serverID)

    registry := &InMemoryRegistry{
        serverID:      serverID,
        logger:        slogger,
        tunnels:       make(map[string]*TunnelInfo),
        servers:       make(map[string]*ServerInfo),
        heartbeatStop: make(chan struct{}),
    }

    go registry.cleanupExpiredTunnels()
    return registry, nil
}

// RegisterTunnel registers a tunnel
func (r *InMemoryRegistry) RegisterTunnel(info *TunnelInfo) error {
    r.tunnelsMutex.Lock()
    defer r.tunnelsMutex.Unlock()

    info.ServerID = r.serverID
    info.LastSeenAt = time.Now()

    if info.CreatedAt.IsZero() {
        info.CreatedAt = time.Now()
    }

    r.tunnels[info.Subdomain] = info
    r.logger.Info("Tunnel registered", "subdomain", info.Subdomain, "client_id", info.ClientID)
    return nil
}

// GetTunnel retrieves tunnel information
func (r *InMemoryRegistry) GetTunnel(subdomain string) (*TunnelInfo, error) {
    r.tunnelsMutex.RLock()
    defer r.tunnelsMutex.RUnlock()

    r.lookups++

    tunnel, exists := r.tunnels[subdomain]
    if !exists {
        return nil, fmt.Errorf("tunnel not found: %s", subdomain)
    }

    r.hits++

    if time.Since(tunnel.LastSeenAt) > tunnelTTL {
        return nil, fmt.Errorf("tunnel expired: %s", subdomain)
    }

    return tunnel, nil
}

// UnregisterTunnel removes a tunnel
func (r *InMemoryRegistry) UnregisterTunnel(subdomain string) error {
    r.tunnelsMutex.Lock()
    defer r.tunnelsMutex.Unlock()

    delete(r.tunnels, subdomain)
    r.logger.Info("Tunnel unregistered", "subdomain", subdomain)
    return nil
}

// RefreshTunnel updates the last seen timestamp
func (r *InMemoryRegistry) RefreshTunnel(subdomain string) error {
    r.tunnelsMutex.Lock()
    defer r.tunnelsMutex.Unlock()

    tunnel, exists := r.tunnels[subdomain]
    if !exists {
        return fmt.Errorf("tunnel not found: %s", subdomain)
    }

    tunnel.LastSeenAt = time.Now()
    return nil
}

// GetAllTunnels returns all registered tunnels
func (r *InMemoryRegistry) GetAllTunnels() ([]*TunnelInfo, error) {
    r.tunnelsMutex.RLock()
    defer r.tunnelsMutex.RUnlock()

    tunnels := make([]*TunnelInfo, 0, len(r.tunnels))
    now := time.Now()

    for _, tunnel := range r.tunnels {
        if now.Sub(tunnel.LastSeenAt) <= tunnelTTL {
            tunnels = append(tunnels, tunnel)
        }
    }

    return tunnels, nil
}

// IsLocalTunnel checks if tunnel is managed by this server
func (r *InMemoryRegistry) IsLocalTunnel(subdomain string) (bool, error) {
    r.tunnelsMutex.RLock()
    defer r.tunnelsMutex.RUnlock()

    _, exists := r.tunnels[subdomain]
    return exists, nil
}

// RegisterServer registers this server
func (r *InMemoryRegistry) RegisterServer(info *ServerInfo) error {
    r.serversMutex.Lock()
    defer r.serversMutex.Unlock()

    info.LastHeartbeat = time.Now()
    r.servers[info.ServerID] = info
    r.logger.Info("Server registered", "server_id", info.ServerID)
    return nil
}

// GetServer retrieves server information
func (r *InMemoryRegistry) GetServer(serverID string) (*ServerInfo, error) {
    r.serversMutex.RLock()
    defer r.serversMutex.RUnlock()

    server, exists := r.servers[serverID]
    if !exists {
        return nil, fmt.Errorf("server not found: %s", serverID)
    }

    return server, nil
}

// GetAllServers returns all registered servers
func (r *InMemoryRegistry) GetAllServers() ([]*ServerInfo, error) {
    r.serversMutex.RLock()
    defer r.serversMutex.RUnlock()

    servers := make([]*ServerInfo, 0, len(r.servers))
    for _, server := range r.servers {
        servers = append(servers, server)
    }

    return servers, nil
}

// StartHeartbeat starts periodic heartbeat updates
func (r *InMemoryRegistry) StartHeartbeat(serverInfo *ServerInfo) {
    go func() {
        ticker := time.NewTicker(heartbeatInterval)
        defer ticker.Stop()

        for {
            select {
            case <-ticker.C:
                r.serversMutex.Lock()
                if server, exists := r.servers[serverInfo.ServerID]; exists {
                    server.LastHeartbeat = time.Now()
                    server.ActiveTunnels = len(r.tunnels)
                }
                r.serversMutex.Unlock()

            case <-r.heartbeatStop:
                return
            }
        }
    }()
}

// GetLeastLoadedServer returns this server
func (r *InMemoryRegistry) GetLeastLoadedServer() (*ServerInfo, error) {
    r.serversMutex.RLock()
    defer r.serversMutex.RUnlock()

    if len(r.servers) == 0 {
        return nil, fmt.Errorf("no servers available")
    }

    for _, server := range r.servers {
        return server, nil
    }

    return nil, fmt.Errorf("no servers available")
}

// UpdateServerLoad updates the active connections count
func (r *InMemoryRegistry) UpdateServerLoad(activeConnections int) error {
    r.serversMutex.Lock()
    defer r.serversMutex.Unlock()

    if server, exists := r.servers[r.serverID]; exists {
        server.ActiveConnections = activeConnections
    }

    return nil
}

// GetCacheStats returns cache statistics
func (r *InMemoryRegistry) GetCacheStats() (hits, misses int, hitRate float64) {
    if r.lookups == 0 {
        return 0, 0, 0.0
    }

    misses = r.lookups - r.hits
    hitRate = float64(r.hits) / float64(r.lookups) * 100

    return r.hits, misses, hitRate
}

// Close cleans up resources
func (r *InMemoryRegistry) Close() error {
    close(r.heartbeatStop)
    r.logger.Info("In-memory registry closed")
    return nil
}

// cleanupExpiredTunnels periodically removes expired tunnels
func (r *InMemoryRegistry) cleanupExpiredTunnels() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            r.tunnelsMutex.Lock()
            now := time.Now()
            for subdomain, tunnel := range r.tunnels {
                if now.Sub(tunnel.LastSeenAt) > tunnelTTL {
                    delete(r.tunnels, subdomain)
                    r.logger.Info("Tunnel expired and removed", "subdomain", subdomain)
                }
            }
            r.tunnelsMutex.Unlock()

        case <-r.heartbeatStop:
            return
        }
    }
}
