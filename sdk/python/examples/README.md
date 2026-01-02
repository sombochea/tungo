# TunGo SDK Examples

Python examples demonstrating how to use the TunGo SDK.

## Setup

### Using uv (Recommended)

```bash
# Install uv if you haven't already
curl -LsSf https://astral.sh/uv/install.sh | sh

# Install all dependencies (this will use the local tungo-sdk package)
uv sync

# That's it! Now you can run any example with: uv run <example>.py
```

### Using pip

```bash
pip install -r requirements.txt
```

## Running Examples

### FastAPI Example
Simple FastAPI server with TunGo tunnel:

```bash
# With uv (Recommended)
uv run fastapi_example.py

# Or with python directly
uv run python fastapi_example.py

# Without uv
python fastapi_example.py
```

Features:
- REST API endpoints
- JSON responses
- Automatic tunnel setup
- Async/await support

### Flask Example
Flask ser (Recommended)
uv run flask_example.py

# Or with python directly
uv run python flask_example.py

# Without uv
uv run python flask_example.py

# With python
python flask_example.py
```

Features:
- Traditional Flask routes
- Background tunnel thread
- Simple integration

### Webhook Receiver
Webhook e (Recommended)
uv run webhook_example.py

# Or with python directly
uv run python webhook_example.py

# Without uv
uv run python webhook_example.py

# With python
python webhook_example.py
```

Features:
- POST endpoint for webhooks
- Request logging
- Custom subdomain

## Environment Variables

```bash
# Server configuration
export PORT=8000
export TUNGO_HOST=localhost
export TUNGO_SUBDOMAIN=my-app

# Run with custom config
python fastapi_example.py
```

## Common Issues

### "Connection refused"
Ensure the TunGo server is running:
```bash
cd /path/to/tungo
./bin/server --config server.yaml
```

### Port already in use
Change the port in the example or set PORT environment variable:
```bash
PORT=9000 python fastapi_example.py
```

## Learn More

- [SDK Documentation](../README.md)
- [TunGo Documentation](../../README.md)
