# TunGo SDK for Python

Native Python SDK to expose your local server to the internet using TunGo. Pure Python implementation with native async/await support.

## Features

✨ **Native Implementation** - Pure Python with asyncio, no external binary required  
🚀 **Async/Await** - Built on asyncio for high performance  
📦 **Lightweight** - Minimal dependencies (websockets, aiohttp)  
🔄 **Auto-Reconnect** - Automatic reconnection with configurable retry logic  
🎯 **Type Hints** - Full type annotations for better IDE support  
⚡ **Event-Driven** - Rich event system for monitoring tunnel status  
🐍 **Python 3.8+** - Compatible with Python 3.8 and above

## Installation

### Using uv (Recommended)

[uv](https://github.com/astral-sh/uv) is a fast Python package installer and resolver.

```bash
# Install uv if you haven't already
curl -LsSf https://astral.sh/uv/install.sh | sh

# Add to your project
uv add tungo-sdk

# Or install globally
uv pip install tungo-sdk
```

### Using pip

```bash
pip install tungo-sdk
```

### Using poetry

```bash
poetry add tungo-sdk
```

## Prerequisites

You need a running TunGo server. The SDK connects to the server via WebSocket.

**Start the TunGo server:**

```bash
# From the TunGo project root
./bin/server --config server.yaml
```

**Default server configuration:**

-   Host: `localhost`
-   Control Port: `5555`
-   WebSocket URL: `ws://localhost:5555/ws`

## Quick Start

### Option 1: Run directly with uv (fastest)

```bash
# Create a simple tunnel script
cat > tunnel.py << 'EOF'
import asyncio
from tungo import TunGoClient, TunGoOptions

async def main():
    options = TunGoOptions(local_port=8000)
    client = TunGoClient(options)
    tunnel = await client.start()
    print(f"Tunnel: {tunnel.url}")

    try:
        await asyncio.Event().wait()
    except KeyboardInterrupt:
        await client.stop()

asyncio.run(main())
EOF

# Run with uv (installs dependencies automatically)
uv run --with tungo-sdk tunnel.py
```

### Option 2: Install in your project

```python
import asyncio
from tungo import TunGoClient, TunGoOptions

async def main():
    # Create client
    options = TunGoOptions(local_port=8000)
    client = TunGoClient(options)

    # Start tunnel
    tunnel = await client.start()
    print(f"Tunnel URL: {tunnel.url}")
    # Output: http://abc123.localhost

    # Keep running
    try:
        await asyncio.Event().wait()
    except KeyboardInterrupt:
        await client.stop()

if __name__ == "__main__":
    asyncio.run(main())
```

**Install and run:**

```bash
# Using uv
uv add tungo-sdk
uv run python main.py

# Using pip
pip install tungo-sdk
python main.py
```

## How It Works

1. **SDK connects** to TunGo server via WebSocket (`ws://host:port/ws`)
2. **Handshake** with `ClientHello` containing optional subdomain and auth key
3. **Server responds** with `ServerHello` containing your public tunnel URL
4. **Requests arrive** - Server forwards incoming HTTP requests via WebSocket
5. **SDK proxies** requests to your local server using aiohttp
6. **Auto-reconnect** - Maintains connection with configurable retry logic

## API Reference

### TunGoClient

#### Constructor

```python
TunGoClient(options: TunGoOptions, events: Optional[TunGoEvents] = None)
```

#### Options

```python
from tungo import TunGoOptions

options = TunGoOptions(
    local_port=8000,          # Required: Local server port to tunnel

    # Server connection (use ONE of the following):
    server_url="ws://localhost:5555/ws",  # Full WebSocket URL (supports ws:// or wss://)
    # OR
    server_host="localhost",  # TunGo server host (default: localhost)
    control_port=5555,        # TunGo server port (default: 5555)

    local_host="localhost",   # Local server host (default: localhost)
    subdomain=None,           # Custom subdomain (optional, random if not set)
    secret_key=None,          # Authentication key (optional)
    connect_timeout=10.0,     # Connection timeout in seconds (default: 10.0)
    max_retries=5,            # Max reconnection attempts (default: 5)
    retry_interval=5.0,       # Retry interval in seconds (default: 5.0)
    log_level="info",         # Log level: debug, info, warn, error
)
```

**Note:** If `server_url` is provided, `server_host` and `control_port` are ignored. The `server_url` can be:

-   Full URL: `ws://tunnel.example.com:5555/ws` or `wss://tunnel.example.com/ws`
-   Host and port: `tunnel.example.com:5555` (automatically adds `ws://` and `/ws`)
-   Just host: `tunnel.example.com` (uses default port 5555)

#### Events

```python
from tungo import TunGoEvents

def on_connect(info):
    print(f"Connected: {info.url}")

def on_disconnect(reason):
    print(f"Disconnected: {reason}")

def on_error(error):
    print(f"Error: {error}")

def on_reconnect(attempt):
    print(f"Reconnecting (attempt {attempt})...")

def on_status(status):
    print(f"Status: {status}")

events = TunGoEvents(
    on_connect=on_connect,
    on_disconnect=on_disconnect,
    on_error=on_error,
    on_reconnect=on_reconnect,
    on_status=on_status,
)

client = TunGoClient(options, events)
```

#### Methods

-   `async start() -> TunnelInfo` - Start the tunnel
-   `async stop() -> None` - Stop the tunnel
-   `get_info() -> Optional[TunnelInfo]` - Get current tunnel info
-   `is_active() -> bool` - Check if tunnel is active

## Usage Examples

### FastAPI Application

```python
import asyncio
from fastapi import FastAPI
import uvicorn
from tungo import TunGoClient, TunGoOptions, TunGoEvents

app = FastAPI()

@app.get("/")
async def root():
    return {"message": "Hello from TunGo!"}

@app.get("/api/users")
async def users():
    return [
        {"id": 1, "name": "Alice"},
        {"id": 2, "name": "Bob"},
    ]

async def start_tunnel():
    """Start tunnel in background."""
    def on_connect(info):
        print(f"\n🌍 Public URL: {info.url}\n")

    events = TunGoEvents(on_connect=on_connect)
    options = TunGoOptions(local_port=8000)
    client = TunGoClient(options, events)
    await client.start()

    # Keep tunnel running
    try:
        await asyncio.Event().wait()
    except asyncio.CancelledError:
        await client.stop()

if __name__ == "__main__":
    # Create config for uvicorn
    config = uvicorn.Config(app, host="0.0.0.0", port=8000, log_level="info")
    server = uvicorn.Server(config)

    async def main():
        """Run tunnel and server together."""
        await asyncio.gather(
            start_tunnel(),
            server.serve(),
        )

    # Run both tunnel and server
    asyncio.run(main())
```

**Run with uv:**

```bash
# Add dependencies
uv add fastapi uvicorn tungo-sdk

# Run the app
uv run fastapi_example.py
```

### Flask Application

```python
import asyncio
from flask import Flask, jsonify
from threading import Thread
from tungo import TunGoClient, TunGoOptions

app = Flask(__name__)

@app.route("/")
def index():
    return jsonify({"message": "Hello from TunGo!"})

@app.route("/api/users")
def users():
    return jsonify([
        {"id": 1, "name": "Alice"},
        {"id": 2, "name": "Bob"},
    ])

def start_tunnel():
    """Start tunnel in background thread."""
    async def _start():
        options = TunGoOptions(local_port=5000)
        client = TunGoClient(options)
        tunnel = await client.start()
        print(f"\n🌍 Public URL: {tunnel.url}\n")

        # Keep running
        await asyncio.Event().wait()

    asyncio.run(_start())

if __name__ == "__main__":
    # Start tunnel in background
    tunnel_thread = Thread(target=start_tunnel, daemon=True)
    tunnel_thread.start()

    # Start Flask
    app.run(host="0.0.0.0", port=5000)
```

### Django Application

```python
# tunnel_manager.py
import asyncio
from tungo import TunGoClient, TunGoOptions

class TunnelManager:
    def __init__(self, port: int = 8000):
        self.port = port
        self.client = None

    async def start(self):
        options = TunGoOptions(local_port=self.port)
        self.client = TunGoClient(options)
        tunnel = await self.client.start()
        print(f"\n🌍 Public URL: {tunnel.url}\n")

    async def stop(self):
        if self.client:
            await self.client.stop()

# In your Django app
from django.apps import AppConfig
import asyncio
from threading import Thread

class MyAppConfig(AppConfig):
    name = 'myapp'

    def ready(self):
        if os.environ.get('RUN_MAIN') == 'true':
            tunnel = TunnelManager(8000)
            thread = Thread(target=lambda: asyncio.run(tunnel.start()), daemon=True)
            thread.start()
```

### With Event Handlers

```python
import asyncio
from tungo import TunGoClient, TunGoOptions, TunGoEvents

async def main():
    def on_connect(info):
        print(f"✅ Connected: {info.url}")

    def on_disconnect(reason):
        print(f"❌ Disconnected: {reason}")

    def on_error(error):
        print(f"❌ Error: {error}")

    def on_reconnect(attempt):
        print(f"🔄 Reconnecting (attempt {attempt})...")

    events = TunGoEvents(
        on_connect=on_connect,
        on_disconnect=on_disconnect,
        on_error=on_error,
        on_reconnect=on_reconnect,
    )

    options = TunGoOptions(local_port=8000, subdomain="my-app")
    client = TunGoClient(options, events)

    await client.start()

    # Keep running
    try:
        await asyncio.Event().wait()
    except KeyboardInterrupt:
        await client.stop()

asyncio.run(main())
```

### Custom Server Configuration

```python
import asyncio
from tungo import TunGoClient, TunGoOptions

async def main():
    # Option 1: Using server_url (recommended for production)
    options = TunGoOptions(
        server_url="wss://tunnel.mycompany.com/ws",  # Secure WebSocket
        local_port=8080,
        subdomain="my-api",
        secret_key="my-secret-key",
        max_retries=10,
        retry_interval=3.0,
        log_level="debug",
    )

    # Option 2: Using server_host and control_port (legacy)
    # options = TunGoOptions(
    #     server_host="tunnel.mycompany.com",
    #     control_port=5555,
    #     local_port=8080,
    #     subdomain="my-api",
    #     secret_key="my-secret-key",
    # )

    client = TunGoClient(options)
    await client.start()

    try:
        await asyncio.Event().wait()
    except KeyboardInterrupt:
        await client.stop()

asyncio.run(main())
```

## Best Practices

1. **Use context manager pattern** (when available):

    ```python
    async with TunGoClient(options) as client:
        tunnel = await client.start()
        # Do work
    ```

2. **Handle errors gracefully**:

    ```python
    try:
        await client.start()
    except TimeoutError:
        print("Connection timeout")
    except Exception as e:
        print(f"Failed: {e}")
    ```

3. **Custom subdomains for consistency**:

    ```python
    options = TunGoOptions(
        local_port=8000,
        subdomain="my-stable-subdomain",
    )
    ```

4. **Environment variables**:

    ```python
    import os

    options = TunGoOptions(
        local_port=int(os.getenv("PORT", "8000")),
        server_host=os.getenv("TUNGO_HOST", "localhost"),
        subdomain=os.getenv("TUNGO_SUBDOMAIN"),
    )
    ```

## Architecture

```
┌─────────────┐         WebSocket           ┌──────────────┐
│             │ ◄─────────────────────────► │              │
│  TunGo SDK  │   ws://host:5555/ws         │ TunGo Server │
│   (Python)  │                             │              │
└──────┬──────┘                             └──────┬───────┘
       │                                           │
       │ HTTP (aiohttp)                            │ HTTP
       │                                           │
       ▼                                           ▼
┌─────────────┐                             ┌──────────────┐
│   Local     │                             │   Internet   │
│   Server    │                             │    Users     │
│ :8000       │                             │              │
└─────────────┘                             └──────────────┘
```

**Components:**

-   **WebSocket Client** - Maintains persistent connection using `websockets`
-   **HTTP Proxy** - Forwards requests using `aiohttp`
-   **Async Event Loop** - Built on `asyncio` for high performance
-   **Stream Manager** - Handles multiple concurrent HTTP streams
-   **Reconnection Logic** - Auto-reconnect with exponential backoff

## Troubleshooting

### Connection timeout

**Problem:** Cannot connect to TunGo server

**Solution:**

```bash
# Check if server is running
lsof -i :5555

# Start the server
cd /path/to/tungo
./bin/server --config server.yaml
```

### ECONNREFUSED when accessing tunnel URL

**Problem:** Local server not running

**Solution:**

```bash
# Start your local server first
python app.py  # or uvicorn, gunicorn, etc.
```

### Import errors

**Problem:** Module not found

**Solution:**

# Using uv (Recommended)

```bash
uv sync # Install all dependencies
uv run fastapi_example.py
uv run flask_example.py
uv run webhook_example.py
```

# Using pip

```bash
pip install -r requirements.txt
python fastapi_example.py
```

## uv Commands Quick Reference

```bash
# Install tungo-sdk
uv add tungo-sdk

# Run script with inline dependency
uv run --with tungo-sdk script.py

# Run project script
uv run python app.py

# Install all dependencies (including dev)
uv sync --all-extras

# Run tests
uv run pytest

# Format code
uv run black .

# Type check
uv run mypy .
```

**Why uv?**

-   ⚡ **10-100x faster** than pip
-   📦 **Single binary** - no Python required to bootstrap
-   🔒 **Reproducible** - automatic lockfile generation
-   🎯 **Simple** - just use `uv run`

Learn more at [docs.astral.sh/uv](https://docs.astral.sh/uv/)

```bash
pip install --force-reinstall tungo-sdk
```

## Examples

See the [examples](./examples) directory for complete working examples:

```bash
cd examples
pip install -r requirements.txt
python fastapi_example.py
python flask_example.py
python webhook_example.py
```

## Comparison

| Feature       | Python SDK     | Node SDK           | CLI Binary    |
| ------------- | -------------- | ------------------ | ------------- |
| Installation  | `pip install`  | `npm install`      | Binary setup  |
| Async Support | Native asyncio | Native async/await | N/A           |
| Type Hints    | Full           | Full TypeScript    | None          |
| Integration   | Programmatic   | Programmatic       | Process spawn |
| Performance   | High           | High               | Medium        |
| Dependencies  | 2 packages     | 1 package          | None          |

## Development

### Using uv (Recommended)

```bash
# Clone repository
git clone https://github.com/sombochea/tungo
cd tungo/sdk/python

# Install with dev dependencies
uv sync --all-extras

# Run tests
uv run pytest

# Format code
uv run black tungo/
uv run ruff check tungo/

# Type check
uv run mypy tungo/

# Run examples
uv run python examples/fastapi_example.py
```

### Using pip

```bash
# Install in development mode
pip install -e ".[dev]"

# Run tests
pytest

# Format code
black tungo/
ruff tungo/

# Type check
mypy tungo/
```

## License

MIT

## Contributing

Contributions welcome! Please open an issue or PR.

## Support

-   [GitHub Issues](https://github.com/sombochea/tungo/issues)
-   [Documentation](https://github.com/sombochea/tungo)
