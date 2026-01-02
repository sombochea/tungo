# Node.js SDK Tests

## Running Tests

```bash
# Install dependencies first
npm install

# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run manual reconnection test
npm run test:manual
```

## Test Suites

### ✅ connection.test.ts
Tests basic connection functionality:
- Connect to TunGo server
- Connect with custom subdomain
- Stop while connected
- Get tunnel info

### ✅ events.test.ts
Tests event callbacks:
- Connect event
- Disconnect event
- Status events
- Error event (skipped - timing issues)

### ✅ reconnection.test.ts
Tests reconnection logic:
- Automatic reconnection on disconnect
- Subdomain preservation across reconnections
- Continuous retry after max attempts (skipped - needs server control)

## Skipped Tests

Some tests are skipped due to timing and environment constraints:
- **Connection error handling**: Difficult to test reliably due to WebSocket timing
- **Continuous retry verification**: Requires starting/stopping server during test

Use `npm run test:manual` for interactive testing of these features.

## Test Results

```
Test Suites: 3 passed, 3 total
Tests:       2 skipped, 10 passed, 12 total
```

The skipped tests verify behavior that requires manual server control or has timing dependencies. These features are validated through the manual test script and real-world usage.
