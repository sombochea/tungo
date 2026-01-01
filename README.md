# TunGo 🚀

Secure, high-performance HTTP tunnel service in Go. Expose your local server to the internet via a public URL.

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://hub.docker.com)

## ✨ Features

-   🚀 High performance Go architecture with Fiber v3
-   🎨 Modern TailwindCSS dashboard for request inspection
-   🔒 TLS support with authentication & rate limiting
-   🔄 Redis clustering for horizontal scaling
-   📊 Prometheus metrics
-   🐳 Docker ready

## 🎯 Quick Start

### Installation

```bash
# Build from source
git clone https://github.com/sombochea/tungo
cd tungo
make build

# Binaries will be in bin/
```

### Start Server

```bash
# Basic
./bin/server

# With config
./bin/server --config server.yaml

# Docker
docker-compose up -d
```

### Start Client

```bash
# Connect to local port 3000
./bin/client --local-port 3000

# With dashboard
./bin/client --local-port 3000 --enable-dashboard

# Custom subdomain
./bin/client --local-port 3000 --subdomain myapp
```

Your app is now live at: `http://[subdomain].localhost:8080`

## 🎨 Dashboard

Enable the request inspector to debug HTTP traffic:

```bash
./bin/client --local-port 3000 --enable-dashboard --dashboard-port 3001
```

Open `http://localhost:3001` to view:

-   All HTTP requests/responses in real-time
-   Headers, body, query params
-   Filter and search requests
-   Replay requests

## 🐳 Docker Quick Start

```yaml
# docker-compose.yml
version: '3.8'
services:
    redis:
        image: redis:7-alpine
        ports:
            - '6379:6379'

    server:
        build: .
        ports:
            - '8080:8080'
            - '5555:5555'
        environment:
            - TUNGO_SERVER_REDIS_URL=redis://redis:6379
        depends_on:
            - redis
```

```bash
docker-compose up -d
```

## ⚙️ Configuration

### Server (`server.yaml`)

```yaml
# Server settings
id: 'server-1'
host: '0.0.0.0'
port: 8080
control_port: 5555

# Connection settings
max_connections: 1000
read_timeout: '30s'
write_timeout: '30s'

# Authentication
require_auth: false
allow_anonymous: true

# Domain settings
subdomain_suffix: 'localhost'

# Redis (required)
redis_url: 'redis://localhost:6379'

# Logging
log_level: 'info'
log_format: 'json'
```

### Client (`client.yaml`)

```yaml
# Server connection
server_host: 'localhost'
control_port: 5555

# Local server to tunnel
local_host: 'localhost'
local_port: 3000

# Tunnel settings
subdomain: '' # Empty for random
secret_key: '' # Optional auth

# Dashboard
enable_dashboard: false
dashboard_port: 3000

# Logging
log_level: 'info'
log_format: 'console'
```

### Environment Variables

```bash
# Server
export TUNGO_SERVER_HOST=0.0.0.0
export TUNGO_SERVER_PORT=8080
export TUNGO_SERVER_REDIS_URL=redis://localhost:6379

# Client
export TUNGO_CLIENT_SERVER_HOST=localhost
export TUNGO_CLIENT_LOCAL_PORT=3000
export TUNGO_CLIENT_ENABLE_DASHBOARD=true
```

## 🚀 Use Cases

**Webhook Development** - Test webhooks locally

```bash
./bin/client --local-port 4000 --subdomain webhooks
```

**Demo Apps** - Share your local app

```bash
./bin/client --local-port 3000 --enable-dashboard
```

**Mobile Testing** - Test mobile apps with local backend

```bash
./bin/client --local-port 8000
```

## 🔧 Development

```bash
# Build
make build

# Run tests
make test

# Format code
make fmt

# Run server (dev)
make run-server

# Run client (dev)
make run-client
```

### Project Structure

```
tungo/
├── cmd/
│   ├── server/    # Server binary
│   └── client/    # Client binary
├── internal/
│   ├── server/    # Server implementation
│   ├── client/    # Client implementation
│   ├── proxy/     # Proxy logic
│   └── registry/  # Connection registry
└── pkg/
    └── config/    # Configuration
```

## 📈 Monitoring

Prometheus metrics available at `/metrics`:

-   Active tunnels
-   Request counts
-   Error rates
-   Latency

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch
3. Commit changes
4. Push and open a PR

## 📝 License

MIT License - see [LICENSE](LICENSE)

---

Made with ❤️ in Go
