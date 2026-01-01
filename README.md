# TunGo 🚀

Secure, high-performance HTTP tunnel service in Go. Expose your local server to the internet via a public URL.

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://hub.docker.com)

## ✨ Features

-   🚀 High performance Go architecture with Fiber v3
-   🎨 Modern TailwindCSS dashboard for request inspection
-   🔒 TLS support with authentication & rate limiting
- 💾 **Dual datastore modes**: In-memory (zero config) or Redis (distributed)
- 🔄 Redis clustering for horizontal scaling
- 📊 Prometheus metrics
- 🐳 Docker ready
- ⚡ Zero dependencies to get started - runs standalone!

## 🎯 Quick Start

### Installation

#### Option 1: Download Pre-built Binaries (Recommended)

**Linux & macOS:**
```bash
curl -sSL https://raw.githubusercontent.com/sombochea/tungo/main/scripts/install.sh | bash
```

**Windows:**
```powershell
powershell -ExecutionPolicy Bypass -Command "Invoke-WebRequest -Uri 'https://raw.githubusercontent.com/sombochea/tungo/main/scripts/install.ps1' -OutFile install.ps1; .\install.ps1"
```

**Supported Platforms:**
- ✅ macOS (Intel & Apple Silicon)
- ✅ Linux (x86_64, ARM64)
- ✅ Windows (x86_64, ARM64)

For detailed installation guide, see [scripts/INSTALL.md](scripts/INSTALL.md).

#### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/sombochea/tungo
cd tungo

# Build binaries
make build

# Binaries will be in bin/
./bin/server
./bin/client
```

### Start Server

```bash
# Basic (in-memory mode - no Redis needed!)
./bin/server

# With config
./bin/server --config server.yaml

# Docker
docker-compose up -d
```

> **Note**: Server runs in in-memory mode by default. For distributed/clustered setup, configure `redis_url` in your config file.

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
services:
    redis:
        image: redis:7-alpine

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

# Datastore (optional)
# Leave empty for in-memory mode (single server)
# Set Redis URL for distributed mode (clustering)
redis_url: ''  # Example: "redis://localhost:6379"

# Logging
log_level: 'info'
log_format: 'json'
```

**Datastore Modes:**
- **In-Memory** (default): Perfect for development and single-server deployments. Zero setup required!
- **Redis**: For production clusters with multiple servers. Enables load balancing and high availability.

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
export TUNGO_SERVER_REDIS_URL=redis://localhost:6379  # Optional - omit for in-memory mode
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
