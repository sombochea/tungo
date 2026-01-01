package proxy

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sombochea/tungo/internal/registry"
)

var (
	proxyRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tungo_proxy_requests_total",
			Help: "Total number of proxied requests",
		},
		[]string{"status"},
	)
	proxyLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "tungo_proxy_latency_seconds",
			Help:    "Proxy request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)
)

// ServerProxy handles proxying requests to other servers in the cluster
type ServerProxy struct {
	registry registry.Registry
	logger   *slog.Logger
	client   *http.Client
}

// NewServerProxy creates a new server-to-server proxy with connection pooling
func NewServerProxy(reg registry.Registry, logger *slog.Logger) *ServerProxy {
	return &ServerProxy{
		registry: reg,
		logger:   logger,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,              // Total idle connections
				MaxIdleConnsPerHost: 20,               // Idle connections per host (increased)
				MaxConnsPerHost:     50,               // Max connections per host
				IdleConnTimeout:     90 * time.Second, // Keep connections alive
				DisableKeepAlives:   false,            // Enable connection reuse
				DisableCompression:  false,
			},
		},
	}
}

// ProxyToServer proxies an HTTP request to another server that owns the tunnel
func (p *ServerProxy) ProxyToServer(w http.ResponseWriter, r *http.Request, tunnelInfo *registry.TunnelInfo) error {
	// Build the target URL
	targetURL := fmt.Sprintf("http://%s:%d%s",
		tunnelInfo.ServerHost,
		tunnelInfo.ProxyPort,
		r.URL.Path)

	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	p.logger.Info("Proxying request to remote server",
		"subdomain", tunnelInfo.Subdomain,
		"target_server", tunnelInfo.ServerID,
		"target_url", targetURL,
		"method", r.Method,
		"path", r.URL.Path)

	// Create the proxy request
	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		proxyRequests.WithLabelValues("error").Inc()
		return fmt.Errorf("failed to create proxy request: %w", err)
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Add proxy headers
	proxyReq.Header.Set("X-Forwarded-For", r.RemoteAddr)
	proxyReq.Header.Set("X-Forwarded-Proto", "http")
	proxyReq.Header.Set("X-TunGo-Proxy", "true")
	proxyReq.Header.Set("X-Original-Host", r.Host)

	// Execute the request with latency tracking
	start := time.Now()
	resp, err := p.client.Do(proxyReq)
	proxyLatency.Observe(time.Since(start).Seconds())

	if err != nil {
		proxyRequests.WithLabelValues("error").Inc()
		return fmt.Errorf("failed to proxy request: %w", err)
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Add proxy response headers
	w.Header().Set("X-TunGo-Proxied-By", tunnelInfo.ServerID)

	// Copy status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		proxyRequests.WithLabelValues("error").Inc()
		p.logger.Error("Failed to copy proxy response", "error", err)
		return fmt.Errorf("failed to copy proxy response: %w", err)
	}

	proxyRequests.WithLabelValues("success").Inc()

	p.logger.Debug("Successfully proxied request",
		"subdomain", tunnelInfo.Subdomain,
		"status", resp.StatusCode,
		"target_server", tunnelInfo.ServerID)

	return nil
}

// ShouldProxy determines if a request should be proxied to another server
func (p *ServerProxy) ShouldProxy(subdomain string) (bool, *registry.TunnelInfo, error) {
	// Check if tunnel exists in registry
	tunnelInfo, err := p.registry.GetTunnel(subdomain)
	if err != nil {
		return false, nil, fmt.Errorf("tunnel not found: %w", err)
	}

	// Check if tunnel belongs to this server
	isLocal, err := p.registry.IsLocalTunnel(subdomain)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check tunnel ownership: %w", err)
	}

	// If tunnel is local, don't proxy
	if isLocal {
		return false, nil, nil
	}

	// Tunnel belongs to another server, should proxy
	return true, tunnelInfo, nil
}
