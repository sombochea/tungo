# TunGo SDK for Node.js

Native Node.js/TypeScript SDK to expose your local server to the internet using TunGo. Pure JavaScript implementation with no external binary dependencies.

## Features

✨ **Native Implementation** - Pure Node.js/TypeScript, no CLI binary required  
🚀 **Zero External Dependencies** - Only uses `ws` for WebSocket support  
📦 **Lightweight** - Minimal footprint, fast startup  
🔄 **Auto-Reconnect** - Automatic reconnection with configurable retry logic  
🎯 **TypeScript First** - Full type safety and IntelliSense support  
🌐 **Deno Compatible** - Works seamlessly with Deno via npm: specifier  
⚡ **Event-Driven** - Rich event system for monitoring tunnel status  

## Installation

```bash
npm install @tungo/sdk
# or
yarn add @tungo/sdk
# or
pnpm add @tungo/sdk
```

## Prerequisites

You need a running TunGo server. The SDK connects to the server via WebSocket.

**Start the TunGo server:**

```bash
# From the TunGo project root
./bin/server --config server.yaml

# Or using Go
go run cmd/server/main.go
```

**Default server configuration:**
- Host: `localhost`
- Control Port: `5555`
- WebSocket URL: `ws://localhost:5555/ws`

## Quick Start

```typescript
import { TunGoClient } from '@tungo/sdk';

// Create client
const client = new TunGoClient({
  localPort: 3000,  // Your local server port
});

// Start tunnel
const tunnel = await client.start();
console.log('Tunnel URL:', tunnel.url);
// Output: http://abc123.localhost

// Your local server is now accessible via the public URL!

// Stop when done
process.on('SIGINT', () => {
  client.stop();
  process.exit();
});
```

## How It Works

1. **SDK connects** to TunGo server via WebSocket (`ws://host:port/ws`)
2. **Handshake** with `ClientHello` containing optional subdomain and auth key
3. **Server responds** with `ServerHello` containing your public tunnel URL
4. **Requests arrive** - Server forwards incoming HTTP requests via WebSocket
5. **SDK proxies** requests to your local server and streams responses back
6. **Auto-reconnect** - Maintains connection with configurable retry logic

## API Reference

### TunGoClient

#### Constructor

```typescript
new TunGoClient(options: TunGoOptions, events?: TunGoEvents)
```

**Options:**

```typescript
interface TunGoOptions {
  localPort: number;           // Required: Local server port to tunnel
  serverHost?: string;         // Default: 'localhost' - TunGo server host
  controlPort?: number;        // Default: 5555 - TunGo server WebSocket port
  localHost?: string;          // Default: 'localhost' - Local server host
  subdomain?: string;          // Optional: Custom subdomain (random if not set)
  secretKey?: string;          // Optional: Authentication key
  connectTimeout?: number;     // Default: 10000ms - Connection timeout
  maxRetries?: number;         // Default: 5 - Max reconnection attempts
  retryInterval?: number;      // Default: 5000ms - Time between retries
  logLevel?: 'debug' | 'info' | 'warn' | 'error';  // Default: 'info'
}
```

**Events:**

```typescript
interface TunGoEvents {
  onConnect?: (info: TunnelInfo) => void;
  onDisconnect?: (reason?: string) => void;
  onError?: (error: Error) => void;
  onReconnect?: (attempt: number) => void;
  onStatus?: (status: string) => void;
}
```

#### Methods

- `start(): Promise<TunnelInfo>` - Start the tunnel
- `stop(): void` - Stop the tunnel
- `getInfo(): TunnelInfo | null` - Get current tunnel info
- `isActive(): boolean` - Check if tunnel is active

## Usage Examples

### Basic Express App

```typescript
import express from 'express';
import { TunGoClient } from '@tungo/sdk';

const app = express();
const PORT = 3000;

app.get('/', (req, res) => {
  res.json({ message: 'Hello from TunGo!' });
});

const server = app.listen(PORT, async () => {
  console.log(`Server running on http://localhost:${PORT}`);

  // Start tunnel
  const client = new TunGoClient({ localPort: PORT });
  const tunnel = await client.start();
  
  console.log(`🌍 Public URL: ${tunnel.url}`);
});
```

### With Event Handlers

```typescript
import { TunGoClient } from '@tungo/sdk';

const client = new TunGoClient(
  {
    localPort: 3000,
    subdomain: 'my-app',
  },
  {
    onConnect: (info) => {
      console.log('✅ Connected:', info.url);
    },
    onDisconnect: (reason) => {
      console.log('❌ Disconnected:', reason);
    },
    onError: (error) => {
      console.error('❌ Error:', error.message);
    },
    onReconnect: (attempt) => {
      console.log(`🔄 Reconnecting... (attempt ${attempt})`);
    },
    onStatus: (status) => {
      console.log(`📊 Status: ${status}`);
    },
  }
);

await client.start();
```

### Custom Server Configuration

```typescript
import { TunGoClient } from '@tungo/sdk';

// Option 1: Using serverUrl (recommended for production)
const client = new TunGoClient({
  serverUrl: 'wss://tunnel.mycompany.com/ws',  // Secure WebSocket
  localPort: 8080,
  subdomain: 'my-api',
  secretKey: 'my-secret-key',
  maxRetries: 10,
  retryInterval: 3000,
  logLevel: 'debug',
});

// Option 2: Using serverHost and controlPort (legacy)
// const client = new TunGoClient({
//   serverHost: 'tunnel.mycompany.com',
//   controlPort: 5555,
//   localPort: 8080,
//   subdomain: 'my-api',
//   secretKey: 'my-secret-key',
// });

await client.start();
```

### Webhook Development

```typescript
import { TunGoClient } from '@tungo/sdk';
import express from 'express';

const app = express();
app.use(express.json());

app.post('/webhook', (req, res) => {
  console.log('Webhook received:', req.body);
  res.json({ received: true });
});

const server = app.listen(4000, async () => {
  const client = new TunGoClient({
    localPort: 4000,
    subdomain: 'my-webhooks',
  });

  const tunnel = await client.start();
  console.log(`Webhook URL: ${tunnel.url}/webhook`);
  console.log('Configure this URL in your webhook provider!');
});
```

### Next.js Development

```typescript
// scripts/dev-with-tunnel.ts
import { TunGoClient } from '@tungo/sdk';
import { spawn } from 'child_process';

// Start Next.js dev server
const next = spawn('npm', ['run', 'dev'], { stdio: 'inherit' });

// Wait a bit for Next.js to start
setTimeout(async () => {
  const client = new TunGoClient({ localPort: 3000 });
  const tunnel = await client.start();
  console.log(`\n🌍 Share your app: ${tunnel.url}\n`);
}, 3000);

process.on('SIGINT', () => {
  next.kill();
  process.exit();
});
```

### Deno Support

```typescript
// deno.ts
import { TunGoClient } from 'npm:@tungo/sdk';

const client = new TunGoClient({
  localPort: 8000,
});

const tunnel = await client.start();
console.log('Tunnel URL:', tunnel.url);
```

## TypeScript Support

Full TypeScript support with type definitions included:

```typescript
import { TunGoClient, TunGoOptions, TunnelInfo } from '@tungo/sdk';

const options: TunGoOptions = {
  localPort: 3000,
  subdomain: 'my-app',
};

const client = new TunGoClient(options);
const info: TunnelInfo = await client.start();
```

## Error Handling

```typescript
import { TunGoClient } from '@tungo/sdk';

const client = new TunGoClient(
  { localPort: 3000 },
  {
    onError: (error) => {
      if (error.message.includes('ECONNREFUSED')) {
        console.error('Cannot connect to TunGo server');
      } else if (error.message.includes('timeout')) {
        console.error('Connection timeout');
      } else {
        console.error('Tunnel error:', error.message);
      }
    },
  }
);

try {
  await client.start();
} catch (error) {
  console.error('Failed to start tunnel:', error);
  process.exit(1);
}
```

## Best Practices

1. **Graceful Shutdown**: Always stop the tunnel on process exit
   ```typescript
   process.on('SIGINT', () => {
     client.stop();
     process.exit();
   });
   ```

2. **Error Handling**: Handle connection errors appropriately
   ```typescript
   client.on('error', (error) => {
     // Log and handle errors
   });
   ```

3. **Custom Subdomains**: Use custom subdomains for consistency
   ```typescript
   const client = new TunGoClient({
     localPort: 3000,
     subdomain: 'my-stable-subdomain',
   });
   ```

4. **Environment Variables**: Configure via env vars
   ```typescript
   const client = new TunGoClient({
     localPort: parseInt(process.env.PORT || '3000'),
     serverHost: process.env.TUNGO_HOST,
     subdomain: process.env.TUNGO_SUBDOMAIN,
   });
   ```

## Architecture

The SDK uses native Node.js APIs for optimal performance:

```
┌─────────────┐           WebSocket            ┌──────────────┐
│             │ ◄────────────────────────────► │              │
│  TunGo SDK  │   ws://host:5555/ws            │ TunGo Server │
│   (Node)    │                                │              │
└──────┬──────┘                                └──────┬───────┘
       │                                              │
       │ HTTP Proxy                                   │ HTTP
       │                                              │
       ▼                                              ▼
┌─────────────┐                                ┌──────────────┐
│   Local     │                                │   Internet   │
│   Server    │                                │    Users     │
│ :3000       │                                │              │
└─────────────┘                                └──────────────┘
```

**Components:**
- **WebSocket Client** - Maintains persistent connection to TunGo server
- **HTTP Proxy** - Forwards requests from tunnel to local server
- **Stream Manager** - Handles multiple concurrent HTTP streams
- **Reconnection Logic** - Auto-reconnect with exponential backoff

## Troubleshooting

### "Connection timeout" or "WebSocket closed"

**Problem:** Cannot connect to TunGo server

**Solution:**
```bash
# 1. Check if TunGo server is running
lsof -i :5555

# 2. Start the server if not running
cd /path/to/tungo
./bin/server --config server.yaml

# 3. Verify server host and port in SDK config
const client = new TunGoClient({
  serverHost: 'localhost',  // Check this matches your server
  controlPort: 5555,        // Check this matches your server
  localPort: 3000,
});
```

### "ECONNREFUSED" when accessing tunnel URL

**Problem:** Local server not running

**Solution:**
```bash
# Ensure your local server is running on the specified port
# Example:
node server.js  # or npm start, etc.
```

### Subdomain already in use

**Problem:** Another client is using the same subdomain

**Solution:**
```typescript
// Don't specify subdomain to get a random one
const client = new TunGoClient({
  localPort: 3000,
  // subdomain: 'my-app',  // Remove this line
});

// Or use a unique subdomain
const client = new TunGoClient({
  localPort: 3000,
  subdomain: `my-app-${Date.now()}`,
});
```

## Advanced Usage

### Programmatic Control

```typescript
import { TunGoClient } from '@tungo/sdk';

class MyApp {
  private tunnel: TunGoClient;

  async start() {
    this.tunnel = new TunGoClient({ localPort: 3000 });
    const info = await this.tunnel.start();
    console.log('Tunnel ready:', info.url);
  }

  async stop() {
    this.tunnel?.stop();
  }

  isRunning(): boolean {
    return this.tunnel?.isActive() ?? false;
  }
}
```

### Multiple Tunnels

```typescript
import { TunGoClient } from '@tungo/sdk';

// Run multiple tunnels simultaneously
const api = new TunGoClient({ localPort: 3000, subdomain: 'api' });
const web = new TunGoClient({ localPort: 8080, subdomain: 'web' });

const [apiTunnel, webTunnel] = await Promise.all([
  api.start(),
  web.start(),
]);

console.log('API:', apiTunnel.url);
console.log('Web:', webTunnel.url);
```

### Custom Event Handling

```typescript
import { TunGoClient } from '@tungo/sdk';

const client = new TunGoClient({ localPort: 3000 });

// Monitor all events
client.on('connect', (info) => {
  console.log('✅ Tunnel established:', info.url);
  // Send notification, update UI, etc.
});

client.on('disconnect', (reason) => {
  console.log('⚠️  Disconnected:', reason);
  // Log to monitoring service
});

client.on('error', (error) => {
  console.error('❌ Error:', error.message);
  // Alert administrators
});

client.on('reconnect', (attempt) => {
  console.log(`🔄 Reconnecting (attempt ${attempt})...`);
  // Show loading indicator
});

client.on('status', (status) => {
  console.log('📊 Status:', status);
  // Update status dashboard
});

await client.start();
```

## Performance Tips

1. **Keep connections alive** - Don't stop/start frequently, reuse connections
2. **Use subdomain** - Specify a subdomain to ensure URL consistency
3. **Adjust timeouts** - Increase `connectTimeout` for slow networks
4. **Monitor events** - Use event handlers to track connection health
5. **Handle errors** - Implement proper error handling for production use

## Comparison with CLI

| Feature | Native SDK | CLI Binary |
|---------|-----------|------------|
| Installation | `npm install` | Binary + PATH setup |
| Startup Time | ~50ms | ~200ms |
| Memory Usage | ~20MB | ~30MB |
| Dependencies | Only `ws` | None |
| Integration | Programmatic | Process spawn |
| Type Safety | Full TypeScript | None |
| Events | Rich event system | Parse stdout |
| Control | Full API access | Limited |

## Examples

See the [examples](./examples) directory for complete working examples:

- [express.ts](./examples/src/express.ts) - Basic Express.js server
- [webhook.ts](./examples/src/webhook.ts) - Webhook receiver with UI
- [advanced.ts](./examples/src/advanced.ts) - Full-featured example with all options

```bash
cd examples
npm install
npm run express   # Run Express example
npm run webhook   # Run webhook example
npm run advanced  # Run advanced example
```

## License

MIT

## Contributing

Contributions welcome! Please open an issue or PR.

## Support

- [GitHub Issues](https://github.com/sombochea/tungo/issues)
- [Documentation](https://github.com/sombochea/tungo)
