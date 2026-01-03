# TunGo Java SDK

[![Maven Central](https://img.shields.io/maven-central/v/com.github.sombochea/tungo-sdk.svg)](https://search.maven.org/artifact/com.github.sombochea/tungo-sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Java SDK for TunGo - Expose your local server to the internet with ease.

## Features

- ✅ **Java 8+ Support** - Compatible with Java 8 and higher
- 🔄 **Auto-Reconnection** - Automatic reconnection with exponential backoff
- 🔒 **Secure Tunneling** - WebSocket-based secure tunnel connections
- 🎯 **Event-Driven** - Comprehensive event system for connection lifecycle
- 📦 **Zero Configuration** - Works out of the box with sensible defaults
- 🏗️ **Builder Pattern** - Fluent API for easy configuration

## Requirements

- Java 8 or higher
- Maven 3.6+ or Gradle 6+

## Installation

### Maven

```xml
<dependency>
    <groupId>com.github.sombochea</groupId>
    <artifactId>tungo-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Gradle

```gradle
dependencies {
    implementation 'com.github.sombochea:tungo-sdk:1.0.0'
}
```

### Manual Build

```bash
git clone https://github.com/sombochea/tungo.git
cd tungo/sdk/java
mvn clean install
```

## Quick Start

### Basic Usage

```java
import com.github.sombochea.tungo.*;

public class Main {
    public static void main(String[] args) throws Exception {
        // Configure options
        TunGoOptions options = TunGoOptions.builder(3000)
            .serverHost("localhost")
            .controlPort(5555)
            .build();

        // Create and start client
        TunGoClient client = new TunGoClient(options);
        TunnelInfo info = client.start();

        System.out.println("Tunnel URL: " + info.getUrl());

        // Keep running...
        Thread.currentThread().join();
    }
}
```

### With Event Listeners

```java
TunGoEventListener listener = new TunGoEventListener() {
    @Override
    public void onConnect(TunnelInfo tunnelInfo) {
        System.out.println("Connected! URL: " + tunnelInfo.getUrl());
    }

    @Override
    public void onDisconnect(String reason) {
        System.out.println("Disconnected: " + reason);
    }

    @Override
    public void onError(Throwable error) {
        System.err.println("Error: " + error.getMessage());
    }

    @Override
    public void onReconnect(int attempt, int maxRetries) {
        System.out.println("Reconnecting... (" + attempt + "/" + maxRetries + ")");
    }
};

TunGoOptions options = TunGoOptions.builder(3000).build();
TunGoClient client = new TunGoClient(options, listener);
client.start();
```

### Complete Example with HTTP Server

```java
import com.sun.net.httpserver.HttpServer;
import java.net.InetSocketAddress;
import java.io.OutputStream;

public class CompleteExample {
    public static void main(String[] args) throws Exception {
        // Start local HTTP server
        HttpServer server = HttpServer.create(new InetSocketAddress(8080), 0);
        server.createContext("/", exchange -> {
            String response = "Hello from TunGo!";
            exchange.sendResponseHeaders(200, response.length());
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(response.getBytes());
            }
        });
        server.start();

        // Create tunnel
        TunGoOptions options = TunGoOptions.builder(8080)
            .serverHost("localhost")
            .subdomain("my-app")
            .build();

        TunGoClient client = new TunGoClient(options, new TunGoEventListener() {
            @Override
            public void onConnect(TunnelInfo info) {
                System.out.println("🚀 Tunnel ready!");
                System.out.println("📍 " + info.getUrl());
            }
        });

        // Start tunnel
        client.start();

        // Graceful shutdown
        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            client.shutdown();
            server.stop(0);
        }));

        Thread.currentThread().join();
    }
}
```

## Configuration

### TunGoOptions Builder

| Method | Description | Default |
|--------|-------------|---------|
| `serverUrl(String)` | Full WebSocket server URL | `""` |
| `serverHost(String)` | Server hostname | `localhost` |
| `controlPort(int)` | Server control port | `5555` |
| `localHost(String)` | Local server hostname | `localhost` |
| `localPort(int)` | Local server port | **Required** |
| `subdomain(String)` | Custom subdomain | `""` (auto-generated) |
| `secretKey(String)` | Authentication key | `""` |
| `connectTimeout(int)` | Connection timeout (ms) | `10000` |
| `maxRetries(int)` | Max reconnection attempts | `5` |
| `retryInterval(int)` | Initial retry delay (ms) | `5000` |
| `logLevel(String)` | Log level (DEBUG/INFO/WARN/ERROR) | `INFO` |

### Example Configuration

```java
TunGoOptions options = TunGoOptions.builder(3000)
    .serverUrl("wss://tunnel.example.com/ws")
    .subdomain("my-custom-subdomain")
    .secretKey("your-secret-key")
    .connectTimeout(15000)
    .maxRetries(10)
    .retryInterval(3000)
    .logLevel("DEBUG")
    .build();
```

## API Reference

### TunGoClient

#### Constructor

```java
TunGoClient(TunGoOptions options)
TunGoClient(TunGoOptions options, TunGoEventListener listener)
```

#### Methods

- `TunnelInfo start()` - Start the tunnel and wait for connection
- `void stop()` - Stop the tunnel
- `void shutdown()` - Shutdown client and cleanup resources
- `TunnelInfo getTunnelInfo()` - Get current tunnel information
- `boolean isConnected()` - Check if tunnel is connected

### TunnelInfo

```java
public class TunnelInfo {
    String getUrl();        // Public tunnel URL
    String getSubdomain();  // Assigned subdomain
    String getClientId();   // Unique client identifier
}
```

### TunGoEventListener

```java
public interface TunGoEventListener {
    void onConnect(TunnelInfo tunnelInfo);
    void onDisconnect(String reason);
    void onError(Throwable error);
    void onReconnect(int attempt, int maxRetries);
    void onStatus(String message);
}
```

## Reconnection Strategy

The SDK implements automatic reconnection with exponential backoff:

1. **Initial retry**: After `retryInterval` ms
2. **Exponential backoff**: Doubles delay on each attempt
3. **Max delay**: Capped at 30 seconds
4. **Continuous retry**: After `maxRetries`, resets counter with extended delay
5. **Extended delay**: 6x normal interval (capped at 30s)

```java
// Customize reconnection behavior
TunGoOptions options = TunGoOptions.builder(3000)
    .maxRetries(10)           // Retry 10 times before reset
    .retryInterval(2000)      // Start with 2 second delay
    .build();
```

## Logging

The SDK uses SLF4J for logging. Add a logging implementation to your project:

```xml
<!-- Logback -->
<dependency>
    <groupId>ch.qos.logback</groupId>
    <artifactId>logback-classic</artifactId>
    <version>1.2.11</version>
</dependency>

<!-- Or Simple SLF4J -->
<dependency>
    <groupId>org.slf4j</groupId>
    <artifactId>slf4j-simple</artifactId>
    <version>1.7.36</version>
</dependency>
```

Control log level in your application:

```java
TunGoOptions options = TunGoOptions.builder(3000)
    .logLevel("DEBUG")  // DEBUG, INFO, WARN, ERROR
    .build();
```

## Examples

See the [examples](src/main/java/com/github/sombochea/tungo/examples) directory for more:

- **BasicExample.java** - Simple tunnel setup
- **CompleteExample.java** - With local HTTP server and custom HTML

Run examples:

```bash
# Compile
mvn clean package

# Run basic example
mvn exec:java -Dexec.mainClass="com.github.sombochea.tungo.examples.BasicExample"

# Run complete example
mvn exec:java -Dexec.mainClass="com.github.sombochea.tungo.examples.CompleteExample"
```

## Testing

```bash
# Run all tests
mvn test

# Run with coverage
mvn clean test jacoco:report
```

## Error Handling

```java
try {
    TunnelInfo info = client.start();
    System.out.println("Connected: " + info.getUrl());
} catch (TimeoutException e) {
    System.err.println("Connection timeout");
} catch (Exception e) {
    System.err.println("Failed to start: " + e.getMessage());
}
```

## Graceful Shutdown

Always shutdown the client properly:

```java
Runtime.getRuntime().addShutdownHook(new Thread(() -> {
    System.out.println("Shutting down...");
    client.shutdown();
}));
```

## Thread Safety

The `TunGoClient` is thread-safe for:
- Starting/stopping the tunnel
- Sending messages
- Event callbacks

## Common Issues

### Connection Refused

```
Failed to connect: Connection refused
```

**Solution**: Ensure the TunGo server is running and accessible.

### Timeout

```
Connection timeout waiting for server hello
```

**Solution**: Increase `connectTimeout` or check network connectivity.

### Subdomain In Use

```
Server hello failed: sub_domain_in_use
```

**Solution**: Choose a different subdomain or use auto-generated one.

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](../../CONTRIBUTING.md).

## License

MIT License - see [LICENSE](../../LICENSE) for details.

## Support

- 📫 Issues: [GitHub Issues](https://github.com/sombochea/tungo/issues)
- 📖 Docs: [Full Documentation](https://github.com/sombochea/tungo)
- 💬 Discussions: [GitHub Discussions](https://github.com/sombochea/tungo/discussions)

## Related SDKs

- [Python SDK](../python)
- [Node.js SDK](../node)
