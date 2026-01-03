# TunGo SDKs

Official SDKs for integrating TunGo tunnel client into your applications.

## Available SDKs

### [Python SDK](./python)

Expose your local Python servers to the internet.

```bash
pip install tungo-sdk
```

**Quick Start:**

```python
from tungo import TunGoClient, TunGoOptions

options = TunGoOptions(local_port=3000)
client = TunGoClient(options)
tunnel = await client.start()
print(f'Public URL: {tunnel.url}')
```

[View Python SDK Documentation](./python/README.md)

### [Node.js/TypeScript SDK](./node)

Expose your local Node.js servers to the internet.

```bash
npm install @tungo/sdk
```

**Quick Start:**

```typescript
import { TunGoClient } from '@tungo/sdk';

const client = new TunGoClient({ localPort: 3000 });
const tunnel = await client.start();
console.log('Public URL:', tunnel.url);
```

[View Node.js SDK Documentation](./node/README.md)

### [Java SDK](./java)

Expose your local Java servers to the internet (Java 8+).

```xml
<dependency>
    <groupId>com.cubetiqs</groupId>
    <artifactId>tungo-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```

**Quick Start:**

```java
import com.cubetiqs.tungo.*;

TunGoOptions options = TunGoOptions.builder(3000).build();
TunGoClient client = new TunGoClient(options);
TunnelInfo info = client.start();
System.out.println("Public URL: " + info.getUrl());
```

[View Java SDK Documentation](./java/README.md)

## Coming Soon

-   **Go SDK** - `sdk/go`
-   **Dart SDK** - `sdk/dart`
-   **Rust SDK** - `sdk/rust`
-   **PHP SDK** - `sdk/php`
-   **Ruby SDK** - `sdk/ruby`

## Contributing

We welcome SDK contributions for other languages! Please see our [contributing guidelines](../README.md).

## License

MIT
