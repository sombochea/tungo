# TunGo SDKs

Official SDKs for integrating TunGo tunnel client into your applications.

## Available SDKs

### [Node.js/TypeScript SDK](./node)
Expose your local Node.js servers to the internet.

```bash
cd node
npm install
npm run build
```

**Quick Start:**
```typescript
import { TunGoClient } from '@tungo/sdk';

const client = new TunGoClient({ localPort: 3000 });
const tunnel = await client.start();
console.log('Public URL:', tunnel.url);
```

[View Node.js SDK Documentation](./node/README.md)

## Coming Soon

- **Python SDK** - `sdk/python`
- **Go SDK** - `sdk/go`
- **Java SDK** - `sdk/java`
- **Ruby SDK** - `sdk/ruby`

## Contributing

We welcome SDK contributions for other languages! Please see our [contributing guidelines](../README.md).

## License

MIT
