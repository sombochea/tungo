import express, { Request, Response, NextFunction } from 'express';
import { TunGoClient, TunnelInfo } from '@tungo/sdk';

const app = express();
app.use(express.json());

// Custom middleware for request logging
app.use((req: Request, res: Response, next: NextFunction) => {
  const start = Date.now();
  res.on('finish', () => {
    const duration = Date.now() - start;
    console.log(`${req.method} ${req.path} - ${res.statusCode} (${duration}ms)`);
  });
  next();
});

// API routes
app.get('/', (req: Request, res: Response) => {
  res.json({
    name: 'TunGo Advanced Example',
    version: '1.0.0',
    endpoints: [
      'GET /',
      'GET /api/health',
      'POST /api/echo',
      'GET /api/tunnel-info'
    ]
  });
});

app.get('/api/health', (req: Request, res: Response) => {
  res.json({
    status: 'healthy',
    uptime: process.uptime(),
    memory: process.memoryUsage(),
    timestamp: new Date().toISOString()
  });
});

app.post('/api/echo', (req: Request, res: Response) => {
  res.json({
    received: req.body,
    timestamp: new Date().toISOString(),
    headers: req.headers
  });
});

// Tunnel info endpoint
let tunnelInfo: TunnelInfo | null = null;

app.get('/api/tunnel-info', (req: Request, res: Response) => {
  if (tunnelInfo) {
    res.json({
      ...tunnelInfo,
      active: client?.isActive() || false
    });
  } else {
    res.status(503).json({
      error: 'Tunnel not established yet',
      active: false
    });
  }
});

const PORT = parseInt(process.env.PORT || '3000');
const SUBDOMAIN = process.env.TUNGO_SUBDOMAIN || 'my-advanced-app';
const SERVER_HOST = process.env.TUNGO_HOST || 'localhost';

// Create tunnel client
const client = new TunGoClient(
  {
    serverHost: SERVER_HOST,
    localPort: PORT,
    subdomain: SUBDOMAIN,
    maxRetries: 10,
    retryInterval: 3000,
    logLevel: 'debug',
  },
  {
    onConnect: (info) => {
      tunnelInfo = info;
      console.log('\n✅ Tunnel established!');
      console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
      console.log(`🌍 Public URL: ${info.url}`);
      console.log(`📊 Subdomain: ${info.subdomain}`);
      console.log(`🖥️  Server: ${info.serverHost}:${info.serverPort}`);
      console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n');
      console.log('📍 API Endpoints:');
      console.log(`   ${info.url}/`);
      console.log(`   ${info.url}/api/health`);
      console.log(`   ${info.url}/api/echo`);
      console.log(`   ${info.url}/api/tunnel-info`);
      console.log('');
    },
    onDisconnect: (reason) => {
      console.log(`\n⚠️  Tunnel disconnected: ${reason}`);
      tunnelInfo = null;
    },
    onError: (error) => {
      console.error(`\n❌ Tunnel error: ${error.message}`);
    },
    onReconnect: (attempt) => {
      console.log(`\n🔄 Reconnecting... (attempt ${attempt}/10)`);
    },
    onStatus: (status) => {
      console.log(`📊 Status: ${status}`);
    }
  }
);

// Start server
const server = app.listen(PORT, async () => {
  console.log('\n🚀 Starting Advanced TunGo Example...');
  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
  console.log(`✅ Local server: http://localhost:${PORT}`);
  console.log(`🔧 Environment:`);
  console.log(`   PORT: ${PORT}`);
  console.log(`   SUBDOMAIN: ${SUBDOMAIN}`);
  console.log(`   SERVER_HOST: ${SERVER_HOST}`);
  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n');
  console.log('🔌 Connecting to TunGo server...\n');

  try {
    await client.start();
  } catch (error) {
    console.error('\n❌ Failed to start tunnel:', (error as Error).message);
    console.error('💡 Make sure the TunGo server is running!\n');
  }
});

// Graceful shutdown
const shutdown = () => {
  console.log('\n👋 Shutting down gracefully...');
  client.stop();
  server.close(() => {
    console.log('✅ Server and tunnel closed');
    process.exit(0);
  });
};

process.on('SIGINT', shutdown);
process.on('SIGTERM', shutdown);

// Error handling
process.on('uncaughtException', (error) => {
  console.error('❌ Uncaught exception:', error);
  shutdown();
});

process.on('unhandledRejection', (reason) => {
  console.error('❌ Unhandled rejection:', reason);
  shutdown();
});
