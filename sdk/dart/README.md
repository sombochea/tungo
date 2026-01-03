# TunGo SDK for Dart

Expose your local Dart/Flutter servers to the internet via secure tunnels.

[![Pub Version](https://img.shields.io/pub/v/tungo_sdk)](https://pub.dev/packages/tungo_sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Features

- 🚀 **Simple API** - Expose your local server with just a few lines of code
- 🔐 **Secure Tunnels** - All traffic goes through authenticated WebSocket connections
- 🔄 **Auto-Reconnection** - Maintains connection with configurable retry logic
- 🎯 **Custom Subdomains** - Request your preferred subdomain
- 📡 **Event-Driven** - React to connection, disconnection, and error events
- 🧪 **Well-Tested** - Comprehensive test coverage
- 📝 **Type-Safe** - Full Dart null safety support

## Installation

Add this to your `pubspec.yaml`:

```yaml
dependencies:
  tungo_sdk: ^1.0.0
```

Then run:

```bash
dart pub get
```

## Prerequisites

You need a running TunGo server. The SDK connects to the server via WebSocket.

**Start the TunGo server:**

```bash
# From the TunGo project root
./bin/server --config server.yaml
```

**Default server configuration:**

- Host: `localhost`
- Control Port: `5555`
- WebSocket URL: `ws://localhost:5555/ws`

## Quick Start

### Basic Example

```dart
import 'package:tungo_sdk/tungo.dart';

void main() async {
  // Create client with options
  final options = TunGoOptions(
    localPort: 3000,
    subdomain: 'my-dart-app',
  );

  // Set up event listeners
  final eventListener = TunGoEventListener(
    onConnect: (info) {
      print('Tunnel established: ${info.url}');
    },
    onDisconnect: (reason) {
      print('Disconnected: $reason');
    },
    onError: (error) {
      print('Error: $error');
    },
  );

  final client = TunGoClient(options, eventListener);

  // Start tunnel
  try {
    final info = await client.start();
    print('Public URL: ${info.url}');
    
    // Keep running...
  } catch (e) {
    print('Failed to start: $e');
  }
}
```

### With HTTP Server

```dart
import 'dart:io';
import 'package:tungo_sdk/tungo.dart';

void main() async {
  // Start local HTTP server
  final server = await HttpServer.bind('localhost', 3000);
  server.listen((request) {
    request.response
      ..write('Hello from Dart!')
      ..close();
  });

  // Create tunnel
  final client = TunGoClient(
    TunGoOptions(localPort: 3000),
  );

  final info = await client.start();
  print('Public URL: ${info.url}');
}
```

## Configuration

### TunGoOptions

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `serverUrl` | `String?` | `null` | Complete WebSocket URL (e.g., `ws://localhost:5555/ws`) |
| `serverHost` | `String` | `'localhost'` | Server hostname |
| `controlPort` | `int` | `5555` | Server control port |
| `localHost` | `String` | `'localhost'` | Local server hostname |
| `localPort` | `int` | **required** | Local server port to forward requests to |
| `subdomain` | `String?` | `null` | Requested subdomain (optional, server assigns if not provided) |
| `secretKey` | `String?` | `null` | Authentication secret key |
| `connectTimeout` | `int` | `10000` | Connection timeout in milliseconds |
| `maxRetries` | `int` | `5` | Maximum reconnection attempts before extended delay |
| `retryInterval` | `int` | `5000` | Base retry interval in milliseconds |
| `logLevel` | `String` | `'INFO'` | Log level: `'DEBUG'`, `'INFO'`, `'WARNING'`, `'ERROR'` |

### Example with Custom Configuration

```dart
final options = TunGoOptions(
  serverHost: 'tunnel.example.com',
  controlPort: 8080,
  localHost: '127.0.0.1',
  localPort: 8000,
  subdomain: 'my-custom-app',
  secretKey: 'your-secret-key',
  connectTimeout: 15000,
  maxRetries: 10,
  retryInterval: 3000,
  logLevel: 'DEBUG',
);
```

## API Reference

### TunGoClient

#### Constructor

```dart
TunGoClient(TunGoOptions options, [TunGoEventListener? eventListener])
```

#### Methods

- **`Future<TunnelInfo> start()`** - Start the tunnel and return tunnel information
- **`Future<void> stop()`** - Stop the tunnel and clean up resources
- **`bool get isActive`** - Check if tunnel is currently active
- **`TunnelInfo? get tunnelInfo`** - Get current tunnel information

### TunGoEventListener

Event callbacks for tunnel lifecycle:

```dart
final eventListener = TunGoEventListener(
  onConnect: (TunnelInfo info) {
    // Called when tunnel is established
  },
  onDisconnect: (String? reason) {
    // Called when connection is lost
  },
  onError: (Object error) {
    // Called when an error occurs
  },
  onReconnect: (int attempt, int maxRetries) {
    // Called on reconnection attempts
  },
  onStatus: (String message) {
    // Called on status updates
  },
);
```

### TunnelInfo

Information about the established tunnel:

```dart
class TunnelInfo {
  final String url;        // Public URL (e.g., http://abc123.localhost:8080)
  final String subdomain;  // Assigned subdomain
  final String clientId;   // Unique client identifier
}
```

## Reconnection Strategy

The SDK implements intelligent reconnection with exponential backoff:

1. **Initial Attempts** (1-5): Exponential backoff starting from `retryInterval`
2. **Extended Delay**: After `maxRetries`, applies 6x delay (up to 30s)
3. **Continuous Retry**: Keeps attempting with extended delays indefinitely

Example timing with defaults:
- Attempt 1: 5s delay
- Attempt 2: 10s delay
- Attempt 3: 20s delay
- Attempt 4: 30s delay (capped)
- Attempt 5+: 30s delay (extended, continuous)

## Error Handling

```dart
try {
  final info = await client.start();
  print('Connected: ${info.url}');
} on TimeoutException {
  print('Connection timeout');
} on SocketException catch (e) {
  print('Network error: $e');
} catch (e) {
  print('Failed to start tunnel: $e');
}
```

## Examples

### Minimal Example

```dart
import 'package:tungo_sdk/tungo.dart';

void main() async {
  final client = TunGoClient(TunGoOptions(localPort: 3000));
  final info = await client.start();
  print('Public URL: ${info.url}');
}
```

### With Event Handling

```dart
final eventListener = TunGoEventListener(
  onConnect: (info) {
    print('✓ Connected: ${info.url}');
  },
  onDisconnect: (reason) {
    print('✗ Disconnected: $reason');
  },
  onError: (error) {
    print('✗ Error: $error');
  },
  onReconnect: (attempt, maxRetries) {
    print('↻ Reconnecting ($attempt/$maxRetries)...');
  },
);

final client = TunGoClient(
  TunGoOptions(localPort: 3000),
  eventListener,
);

await client.start();
```

## Testing

Run tests:

```bash
dart test
```

Run with coverage:

```bash
dart test --coverage=coverage
dart pub global activate coverage
format_coverage --lcov --in=coverage --out=coverage/lcov.info --report-on=lib
```

## How It Works

1. **SDK connects** to TunGo server via WebSocket (`ws://host:port/ws`)
2. **Handshake** with `ClientHello` containing optional subdomain and auth key
3. **Server responds** with `ServerHello` containing your public tunnel URL
4. **Requests arrive** - Server forwards incoming HTTP requests via WebSocket
5. **SDK proxies** requests to your local server using `http` package
6. **Auto-reconnect** - Maintains connection with configurable retry logic

## Thread Safety

The TunGoClient is designed for single-threaded use. For concurrent operations, ensure proper synchronization or use separate client instances.

## Common Issues

### Connection Refused

```dart
// Ensure TunGo server is running
SocketException: Connection refused
```

**Solution**: Start the TunGo server first

### Port Already in Use

```dart
// Local server port conflict
SocketException: Address already in use
```

**Solution**: Choose a different `localPort` or stop the conflicting process

### Subdomain Already Taken

```dart
// Requested subdomain is in use
Exception: Server hello failed: sub_domain_in_use
```

**Solution**: Choose a different subdomain or omit to get auto-assigned

## Platform Support

- ✅ Dart VM (Windows, macOS, Linux)
- ✅ Flutter (iOS, Android, Desktop, Web)

## Contributing

Contributions are welcome! Please see the main [contributing guidelines](../../README.md).

## License

MIT License - see [LICENSE](../../LICENSE) for details

## Links

- [GitHub Repository](https://github.com/sombochea/tungo)
- [Report Issues](https://github.com/sombochea/tungo/issues)
- [Main Documentation](../../README.md)

## Related SDKs

- [Python SDK](../python)
- [Node.js SDK](../node)
- [Java SDK](../java)
