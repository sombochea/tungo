# Test Directory

This directory contains comprehensive tests for the TunGo SDK.

## Test Structure

### Python Tests (`sdk/python/tests/`)

```
tests/
├── __init__.py              # Test package initialization
├── conftest.py              # Pytest fixtures and configuration
├── test_connection.py       # Basic connection tests
├── test_reconnection.py     # Reconnection and retry tests
├── test_events.py           # Event callback tests
└── manual_reconnect.py      # Manual reconnection testing script
```

### Node.js Tests (`sdk/node/tests/`)

```
tests/
├── connection.test.ts       # Basic connection tests
├── reconnection.test.ts     # Reconnection and retry tests
├── events.test.ts           # Event callback tests
├── manual_reconnect.ts      # Manual reconnection testing script
└── jest.config.js           # Jest configuration
```

## Running Tests

### Python

```bash
# Run all tests
cd sdk/python
uv run pytest tests/

# Run specific test file
uv run pytest tests/test_connection.py

# Run with verbose output
uv run pytest tests/ -v

# Run manual reconnection test
uv run tests/manual_reconnect.py
```

### Node.js

```bash
# Install test dependencies first
cd sdk/node
npm install

# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run manual reconnection test
npm run test:manual
```

## Test Categories

### Connection Tests
- Basic connection establishment
- Custom subdomain handling
- Connection timeout behavior
- Stop while connected
- Tunnel info retrieval

### Reconnection Tests
- Automatic reconnection on disconnect
- Subdomain preservation across reconnections
- Continuous retry after max attempts
- Extended delay after max retries

### Event Tests
- `onConnect` event triggering
- `onDisconnect` event triggering
- `onReconnect` event triggering
- `onError` event triggering
- `onStatus` event triggering

## Manual Testing

The manual reconnection scripts help you visually test reconnection behavior:

1. Start the TunGo server
2. Run the manual test script
3. Kill the server (Ctrl+C)
4. Watch reconnection attempts
5. Restart the server
6. Verify subdomain preservation

**Python:**
```bash
uv run tests/manual_reconnect.py
```

**Node.js:**
```bash
npm run test:manual
```

## Test Requirements

### Python
- pytest >= 7.4.0
- pytest-asyncio >= 0.21.0
- Running TunGo server for integration tests

### Node.js
- jest >= 29.7.0
- ts-jest >= 29.1.1
- Running TunGo server for integration tests

## Writing New Tests

### Python Example

```python
import pytest
from tungo import TunGoClient, TunGoOptions

@pytest.mark.asyncio
async def test_something(server_url, local_port):
    options = TunGoOptions(
        local_port=local_port,
        server_url=server_url,
    )
    client = TunGoClient(options)
    
    try:
        await client.start()
        # Your test assertions
    finally:
        await client.stop()
```

### Node.js Example

```typescript
import { TunGoClient } from '../src/client.js';

test('should do something', async () => {
  const client = new TunGoClient({
    localPort: 8000,
    serverUrl: 'ws://localhost:5555/ws',
  });
  
  try {
    await client.start();
    // Your test assertions
  } finally {
    client.stop();
  }
});
```

## Notes

- Most tests require a running TunGo server at `ws://localhost:5555/ws`
- Some tests intentionally use invalid servers to test error handling
- Manual tests are for visual verification and debugging
- Automated tests use timeouts to prevent hanging
