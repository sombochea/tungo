# TunGo Documentation

Welcome to the TunGo documentation. This guide will help you understand, deploy, and use the tunnel service.

## 📚 Documentation Structure

### Getting Started
- **[Quickstart Guide](./guides/quickstart.md)** - Get up and running in 5 minutes
- **[Configuration Guide](./guides/configuration.md)** - Complete configuration reference

### User Guides
- **[Dashboard Guide](./guides/dashboard.md)** - Request introspection dashboard
- **[Configuration Guide](./guides/configuration.md)** - Complete configuration reference
- **[Docker Deployment](./guides/docker.md)** - Deploy with Docker and Docker Compose

### Architecture
- **[System Architecture](./architecture/overview.md)** - High-level system design
- **[Protocol Design](./architecture/protocol.md)** - WebSocket protocol specifica
### Operations
- **[Deployment Guide](./operations/deployment.md)** - Production deployment strategies
- **[Monitoring](./operations/monitoring.md)** - Prometheus metrics and observability
- **[Cocker Deployment](./guides/docker.md)** - Docker and Docker Compose deployment
- See [Architecture Overview](./architecture/overview.md) for clustering and performance
- **[Development Guide](./guides/development.md)** - Contributing and local development

## 🚀 Quick Links

**For Users:**
- New to TunGo? Start with the [Quickstart Guide](./guides/quickstart.md)
- Want to inspect traffic? Check the [Dashboard Guide](./guides/dashboard.md)
- Deploying to production? See [Deployment Guide](./operations/deployment.md)

**For Developers:**
- Setting up dev environment? Read [Development Guide](./guides/development.md)
- Understanding the codebase? See [Architecture Overview](./architecture/overview.md)
- Need to make changes? Check [Contributing Guidelines](./guides/development.md#contributing)

**For Operators:**
- Running in production? Follow [Deployment Guide](./operations/deployment.md)
- Need monitoring? Set up [Prometheus Metrics](./operations/monitoring.md)
- Scaling up? Deploy a [Cluster](./operations/clustering.md)

## 🎯 Key Features

- **High Performance** - Built with Go and GoFiber v3+, handles thousands of concurrent connections
- **Modern Dashboard** - Beautiful TailwindCSS-based request inspector
- **Distributed Architecture** - Redis-backed clustering with load balancing
- **Production Ready** - Prometheus metrics, health checks, graceful shutdown
- **Secure** - Authentication, TLS support, rate limiting
- **Developer Friendly** - Hot reload, comprehensive logging, easy configuration

## 📖 What is TunGo?

TunGo is a secure tunneling service that exposes your local web server to the internet via a public URL. Perfect for:

- **Development** - Test webhooks and integrations locally
- **Demos** - Share your local work with clients or team members
- **IoT** - Connect devices behind NAT/firewalls
- **Testing** - Test mobile apps against local backends

## 🏗️ System Overview

```
┌─────────────┐         ┌──────────────┐         ┌─────────────┐
│   Browser   │ ──────▶ │ TunGo     │ ◀────── │ Your Local  │
│   (Client)  │ HTTPS   │ Server       │ WSS     │ Web Server  │
└─────────────┘         └──────────────┘         └─────────────┘
                              │
                              ▼
                        ┌──────────┐
                        │  Redis   │ (Optional - for clustering)
                        └──────────┘
```

## 💡 Example Use Cases

### Webhook Testing
```bash
# Start local server
python -m http.server 8000

# Create tunnel
./tungo-client --local-port 8000 --server tungo.example.com

# Configure webhook to: https://your-subdomain.tungo.example.com
```

### Demo Application
```bash
# Run your app with dashboard
./tungo-client -d --local-port 3000

# Share: https://your-subdomain.tungo.example.com
# Inspect requests: http://localhost:3000
```

### IoT Device Access
```bash
# Expose device interface
./tungo-client --local-port 8080 --subdomain my-iot-device
```

## 🔧 Requirements

**Server:**
- Go 1.22+
- Redis (for clustering, optional)
- Domain with wildcard DNS (*.example.com)

**Client:**
- Go 1.22+ (for building)
- Or download pre-built binary

## 🌟 Why TunGo?

- **Fast** - Written in Go with optimized connection handling
- **Reliable** - Automatic reconnection, health checks, graceful shutdown
- **Observable** - Prometheus metrics, structured logging
- **Scalable** - Horizontal scaling with Redis clustering
- **Modern** - Beautiful dashboard, REST API, WebSocket protocol
- **Open Source** - MIT licensed, contributions welcome

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/yourusername/tungo-go/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/tungo-go/discussions)
- **Documentation**: You're reading it!

## 📜 License

MIT License - see [LICENSE](../LICENSE) file for details.

---

Ready to get started? Head to the [Quickstart Guide](./guides/quickstart.md)!
