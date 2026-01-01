package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
)

// TunnelInfo stores information about a tunnel in the distributed registry
type TunnelInfo struct {
	Subdomain   string    `json:"subdomain"`
	ServerID    string    `json:"server_id"`
	ServerHost  string    `json:"server_host"` // Internal server address for proxying
	ClientID    string    `json:"client_id"`
	CreatedAt   time.Time `json:"created_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
	ProxyPort   int       `json:"proxy_port"`
	ControlPort int       `json:"control_port"`
}

// ServerInfo stores information about a server in the cluster
type ServerInfo struct {
	ServerID          string    `json:"server_id"`
	Host              string    `json:"host"`
	ProxyPort         int       `json:"proxy_port"`
	ControlPort       int       `json:"control_port"`
	LastHeartbeat     time.Time `json:"last_heartbeat"`
	ActiveTunnels     int       `json:"active_tunnels"`
	ActiveConnections int       `json:"active_connections"` // For load-aware routing
}

// cacheEntry represents a cached tunnel lookup
type cacheEntry struct {
	tunnel    *TunnelInfo
	expiresAt time.Time
}

// DistributedRegistry manages tunnel state across multiple servers using Redis
type DistributedRegistry struct {
	client   *redis.Client
	serverID string
	logger   *slog.Logger
	ctx      context.Context

	// Local cache for tunnel lookups
	cache      map[string]*cacheEntry
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration

	// Pub/Sub for cache invalidation
	pubsub *redis.PubSub

	// Metrics
	metrics *registryMetrics
}

// registryMetrics holds Prometheus metrics
type registryMetrics struct {
	redisOps       *prometheus.CounterVec
	redisLatency   prometheus.Histogram
	cacheHits      prometheus.Counter
	cacheMisses    prometheus.Counter
	tunnelCount    prometheus.Gauge
	serverCount    prometheus.Gauge
	pubsubMessages prometheus.Counter
}

const (
	// Redis key prefixes
	tunnelPrefix = "tunnel:"
	serverPrefix = "server:"

	// Redis Pub/Sub channels
	tunnelUpdateChannel = "tunnel:updates"

	// Expiration times
	tunnelTTL         = 30 * time.Second // Tunnels expire if not refreshed
	serverTTL         = 10 * time.Second // Servers expire if heartbeat stops
	heartbeatInterval = 5 * time.Second

	// Cache settings
	defaultCacheTTL = 2 * time.Second // Local cache TTL
)

// initMetrics initializes Prometheus metrics
func initMetrics() *registryMetrics {
	return &registryMetrics{
		redisOps: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tungo_redis_operations_total",
				Help: "Total number of Redis operations",
			},
			[]string{"operation", "status"},
		),
		redisLatency: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "tungo_redis_latency_seconds",
				Help:    "Redis operation latency in seconds",
				Buckets: prometheus.DefBuckets,
			},
		),
		cacheHits: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "tungo_cache_hits_total",
				Help: "Total number of cache hits",
			},
		),
		cacheMisses: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "tungo_cache_misses_total",
				Help: "Total number of cache misses",
			},
		),
		tunnelCount: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "tungo_tunnels_active",
				Help: "Number of active tunnels",
			},
		),
		serverCount: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "tungo_servers_active",
				Help: "Number of active servers in cluster",
			},
		),
		pubsubMessages: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "tungo_pubsub_messages_total",
				Help: "Total number of Pub/Sub messages received",
			},
		),
	}
}

// NewDistributedRegistry creates a new distributed registry
func NewDistributedRegistry(redisURL, serverID string, logger *slog.Logger) (*DistributedRegistry, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)
	ctx := context.Background()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Connected to Redis", "url", redisURL, "server_id", serverID)

	// Initialize pub/sub for cache invalidation
	pubsub := client.Subscribe(ctx, tunnelUpdateChannel)

	registry := &DistributedRegistry{
		client:   client,
		serverID: serverID,
		logger:   logger,
		ctx:      ctx,
		cache:    make(map[string]*cacheEntry),
		cacheTTL: defaultCacheTTL,
		pubsub:   pubsub,
		metrics:  initMetrics(),
	}

	// Start pub/sub listener for cache invalidation
	go registry.listenForUpdates()

	// Start cache cleanup goroutine
	go registry.cleanupCache()

	return registry, nil
}

// RegisterTunnel registers a tunnel in the distributed registry
func (r *DistributedRegistry) RegisterTunnel(info *TunnelInfo) error {
	info.ServerID = r.serverID
	info.LastSeenAt = time.Now()

	if info.CreatedAt.IsZero() {
		info.CreatedAt = time.Now()
	}

	data, err := json.Marshal(info)
	if err != nil {
		r.metrics.redisOps.WithLabelValues("register_tunnel", "error").Inc()
		return fmt.Errorf("failed to marshal tunnel info: %w", err)
	}

	key := tunnelPrefix + info.Subdomain

	start := time.Now()
	if err := r.client.Set(r.ctx, key, data, tunnelTTL).Err(); err != nil {
		r.metrics.redisOps.WithLabelValues("register_tunnel", "error").Inc()
		return fmt.Errorf("failed to register tunnel: %w", err)
	}
	r.metrics.redisLatency.Observe(time.Since(start).Seconds())
	r.metrics.redisOps.WithLabelValues("register_tunnel", "success").Inc()

	// Invalidate cache and notify other servers
	r.invalidateCache(info.Subdomain)
	r.publishUpdate(info.Subdomain, "register")

	r.logger.Info("Registered tunnel",
		"subdomain", info.Subdomain,
		"server_id", info.ServerID,
		"client_id", info.ClientID)

	return nil
}

// GetTunnel retrieves tunnel information from the registry (with local caching)
func (r *DistributedRegistry) GetTunnel(subdomain string) (*TunnelInfo, error) {
	// Check local cache first
	if cached := r.getCached(subdomain); cached != nil {
		r.metrics.cacheHits.Inc()
		return cached, nil
	}
	r.metrics.cacheMisses.Inc()

	// Cache miss, fetch from Redis
	key := tunnelPrefix + subdomain

	start := time.Now()
	data, err := r.client.Get(r.ctx, key).Result()
	r.metrics.redisLatency.Observe(time.Since(start).Seconds())

	if err == redis.Nil {
		r.metrics.redisOps.WithLabelValues("get_tunnel", "not_found").Inc()
		return nil, fmt.Errorf("tunnel not found: %s", subdomain)
	}
	if err != nil {
		r.metrics.redisOps.WithLabelValues("get_tunnel", "error").Inc()
		return nil, fmt.Errorf("failed to get tunnel: %w", err)
	}
	r.metrics.redisOps.WithLabelValues("get_tunnel", "success").Inc()

	var info TunnelInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tunnel info: %w", err)
	}

	// Store in local cache
	r.setCached(subdomain, &info)

	return &info, nil
}

// UnregisterTunnel removes a tunnel from the registry
func (r *DistributedRegistry) UnregisterTunnel(subdomain string) error {
	key := tunnelPrefix + subdomain

	start := time.Now()
	if err := r.client.Del(r.ctx, key).Err(); err != nil {
		r.metrics.redisOps.WithLabelValues("unregister_tunnel", "error").Inc()
		return fmt.Errorf("failed to unregister tunnel: %w", err)
	}
	r.metrics.redisLatency.Observe(time.Since(start).Seconds())
	r.metrics.redisOps.WithLabelValues("unregister_tunnel", "success").Inc()

	// Invalidate cache and notify other servers
	r.invalidateCache(subdomain)
	r.publishUpdate(subdomain, "unregister")

	r.logger.Info("Unregistered tunnel", "subdomain", subdomain, "server_id", r.serverID)
	return nil
}

// RefreshTunnel updates the last seen time for a tunnel
func (r *DistributedRegistry) RefreshTunnel(subdomain string) error {
	info, err := r.GetTunnel(subdomain)
	if err != nil {
		return err
	}

	info.LastSeenAt = time.Now()
	return r.RegisterTunnel(info)
}

// RegisterServer registers this server in the cluster
func (r *DistributedRegistry) RegisterServer(info *ServerInfo) error {
	info.ServerID = r.serverID
	info.LastHeartbeat = time.Now()

	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal server info: %w", err)
	}

	key := serverPrefix + r.serverID
	if err := r.client.Set(r.ctx, key, data, serverTTL).Err(); err != nil {
		return fmt.Errorf("failed to register server: %w", err)
	}

	return nil
}

// GetServer retrieves information about a specific server
func (r *DistributedRegistry) GetServer(serverID string) (*ServerInfo, error) {
	key := serverPrefix + serverID
	data, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("server not found: %s", serverID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	var info ServerInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal server info: %w", err)
	}

	return &info, nil
}

// GetAllServers returns all active servers in the cluster
func (r *DistributedRegistry) GetAllServers() ([]*ServerInfo, error) {
	pattern := serverPrefix + "*"
	keys, err := r.client.Keys(r.ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get server keys: %w", err)
	}

	servers := make([]*ServerInfo, 0, len(keys))
	for _, key := range keys {
		data, err := r.client.Get(r.ctx, key).Result()
		if err != nil {
			continue // Skip expired or deleted keys
		}

		var info ServerInfo
		if err := json.Unmarshal([]byte(data), &info); err != nil {
			r.logger.Warn("Failed to unmarshal server info", "key", key, "error", err)
			continue
		}

		servers = append(servers, &info)
	}

	return servers, nil
}

// GetAllTunnels returns all active tunnels across all servers
func (r *DistributedRegistry) GetAllTunnels() ([]*TunnelInfo, error) {
	pattern := tunnelPrefix + "*"
	keys, err := r.client.Keys(r.ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get tunnel keys: %w", err)
	}

	tunnels := make([]*TunnelInfo, 0, len(keys))
	for _, key := range keys {
		data, err := r.client.Get(r.ctx, key).Result()
		if err != nil {
			continue // Skip expired or deleted keys
		}

		var info TunnelInfo
		if err := json.Unmarshal([]byte(data), &info); err != nil {
			r.logger.Warn("Failed to unmarshal tunnel info", "key", key, "error", err)
			continue
		}

		tunnels = append(tunnels, &info)
	}

	return tunnels, nil
}

// StartHeartbeat starts sending periodic heartbeats for this server
func (r *DistributedRegistry) StartHeartbeat(serverInfo *ServerInfo) {
	go func() {
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-r.ctx.Done():
				return
			case <-ticker.C:
				if err := r.RegisterServer(serverInfo); err != nil {
					r.logger.Error("Failed to send heartbeat", "error", err)
				}
			}
		}
	}()

	r.logger.Info("Started heartbeat", "interval", heartbeatInterval)
}

// Close closes the Redis connection
func (r *DistributedRegistry) Close() error {
	// Unregister this server
	key := serverPrefix + r.serverID
	if err := r.client.Del(r.ctx, key).Err(); err != nil {
		r.logger.Warn("Failed to unregister server", "error", err)
	}

	// Close pub/sub
	if r.pubsub != nil {
		r.pubsub.Close()
	}

	return r.client.Close()
}

// IsLocalTunnel checks if a tunnel belongs to this server
func (r *DistributedRegistry) IsLocalTunnel(subdomain string) (bool, error) {
	info, err := r.GetTunnel(subdomain)
	if err != nil {
		return false, err
	}
	return info.ServerID == r.serverID, nil
}

// getCached retrieves a tunnel from local cache
func (r *DistributedRegistry) getCached(subdomain string) *TunnelInfo {
	r.cacheMutex.RLock()
	defer r.cacheMutex.RUnlock()

	entry, exists := r.cache[subdomain]
	if !exists {
		return nil
	}

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		return nil
	}

	return entry.tunnel
}

// setCached stores a tunnel in local cache
func (r *DistributedRegistry) setCached(subdomain string, tunnel *TunnelInfo) {
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()

	r.cache[subdomain] = &cacheEntry{
		tunnel:    tunnel,
		expiresAt: time.Now().Add(r.cacheTTL),
	}
}

// invalidateCache removes a tunnel from local cache
func (r *DistributedRegistry) invalidateCache(subdomain string) {
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()

	delete(r.cache, subdomain)
}

// publishUpdate publishes a tunnel update to other servers via Pub/Sub
func (r *DistributedRegistry) publishUpdate(subdomain, action string) {
	message := fmt.Sprintf("%s:%s", action, subdomain)
	if err := r.client.Publish(r.ctx, tunnelUpdateChannel, message).Err(); err != nil {
		r.logger.Warn("Failed to publish update", "error", err, "subdomain", subdomain)
	}
}

// listenForUpdates listens for tunnel updates via Pub/Sub and invalidates cache
func (r *DistributedRegistry) listenForUpdates() {
	ch := r.pubsub.Channel()

	r.logger.Info("Started Pub/Sub listener for cache invalidation")

	for msg := range ch {
		r.metrics.pubsubMessages.Inc()

		// Parse message: \"action:subdomain\"
		parts := make([]byte, 0, len(msg.Payload))
		copy(parts, []byte(msg.Payload))

		var action, subdomain string
		for i, b := range msg.Payload {
			if b == ':' {
				action = msg.Payload[:i]
				subdomain = msg.Payload[i+1:]
				break
			}
		}

		if subdomain != "" {
			r.invalidateCache(subdomain)
			r.logger.Debug("Cache invalidated via Pub/Sub",
				"subdomain", subdomain,
				"action", action)
		}
	}
}

// cleanupCache periodically removes expired cache entries
func (r *DistributedRegistry) cleanupCache() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		r.cacheMutex.Lock()
		now := time.Now()
		for subdomain, entry := range r.cache {
			if now.After(entry.expiresAt) {
				delete(r.cache, subdomain)
			}
		}
		r.cacheMutex.Unlock()
	}
}

// GetLeastLoadedServer returns the server with the lowest active connections
func (r *DistributedRegistry) GetLeastLoadedServer() (*ServerInfo, error) {
	servers, err := r.GetAllServers()
	if err != nil {
		return nil, err
	}

	if len(servers) == 0 {
		return nil, fmt.Errorf("no servers available")
	}

	var leastLoaded *ServerInfo
	minConnections := int(^uint(0) >> 1) // Max int

	for i := range servers {
		if servers[i].ActiveConnections < minConnections {
			minConnections = servers[i].ActiveConnections
			leastLoaded = servers[i] // servers[i] is already a pointer
		}
	}

	return leastLoaded, nil
}

// UpdateServerLoad updates the active connections count for this server
func (r *DistributedRegistry) UpdateServerLoad(activeConnections int) error {
	key := serverPrefix + r.serverID

	data, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to get server info: %w", err)
	}

	var info ServerInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return fmt.Errorf("failed to unmarshal server info: %w", err)
	}

	info.ActiveConnections = activeConnections
	info.LastHeartbeat = time.Now()

	newData, err := json.Marshal(&info)
	if err != nil {
		return fmt.Errorf("failed to marshal server info: %w", err)
	}

	if err := r.client.Set(r.ctx, key, newData, serverTTL).Err(); err != nil {
		return fmt.Errorf("failed to update server load: %w", err)
	}

	return nil
}

// GetCacheStats returns cache hit/miss statistics
func (r *DistributedRegistry) GetCacheStats() (hits, misses int, hitRate float64) {
	// Note: In production, you'd want to track these properly
	// For now, this is a placeholder - metrics are exported via Prometheus
	return 0, 0, 0.0
}
