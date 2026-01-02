# TunGo SDK Examples

TypeScript examples demonstrating how to use the TunGo SDK.

## Setup

```bash
# Install dependencies
npm install

# Build the parent SDK first (if not already built)
cd ..
npm install
npm run build

# Back to examples
cd examples
```

## Running Examples

### Express Server
Simple Express.js server with TunGo tunnel:

```bash
npm run express
```

Features:
- Basic REST API endpoints
- JSON responses
- Automatic tunnel setup
- Graceful shutdown

### Webhook Receiver
Webhook endpoint for testing webhooks:

```bash
npm run webhook
```

Features:
- POST endpoint for webhooks
- Request logging
- Web UI for testing
- Custom subdomain

### Advanced Example
Full-featured example with all options:

```bash
npm run advanced
```

Features:
- Custom middleware
- Request logging
- Tunnel info endpoint
- Environment variables
- Error handling
- Status monitoring

## Environment Variables

```bash
# Server configuration
export PORT=3000
export TUNGO_HOST=localhost
export TUNGO_SUBDOMAIN=my-app

# Run with custom config
npm run advanced
```

## Custom Configuration

Modify the examples to use your own TunGo server:

```typescript
const client = new TunGoClient({
  serverHost: 'tunnel.mycompany.com',
  controlPort: 5555,
  localPort: 3000,
  subdomain: 'my-custom-app',
  secretKey: 'my-secret',
  tls: true
});
```

## Building

To compile TypeScript to JavaScript:

```bash
npm run build
```

Output will be in the `dist/` directory.

## Common Issues

### "tungo-client command not found"
Make sure the TunGo client binary is in your PATH:
```bash
export PATH=$PATH:/path/to/tungo/bin
```

### "Connection refused"
Ensure the TunGo server is running:
```bash
# In the main project directory
./bin/server --config server.yaml
```

### Port already in use
Change the port in the example:
```bash
PORT=4000 npm run express
```

## Learn More

- [SDK Documentation](../README.md)
- [TunGo Documentation](../../README.md)
