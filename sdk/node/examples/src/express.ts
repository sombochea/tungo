import express, { Request, Response } from 'express';
import { TunGoClient } from '@tungo/sdk';

const app = express();
const PORT = 3000;

// Middleware
app.use(express.json());

// Routes
app.get('/', (req: Request, res: Response) => {
  res.json({
    message: 'Hello from TunGo!',
    timestamp: new Date().toISOString(),
    environment: process.env.NODE_ENV || 'development'
  });
});

app.get('/api/users', (req: Request, res: Response) => {
  res.json([
    { id: 1, name: 'Alice', email: 'alice@example.com' },
    { id: 2, name: 'Bob', email: 'bob@example.com' },
    { id: 3, name: 'Charlie', email: 'charlie@example.com' }
  ]);
});

app.get('/api/status', (req: Request, res: Response) => {
  res.json({
    status: 'ok',
    uptime: process.uptime(),
    memory: process.memoryUsage()
  });
});

// Start server
const server = app.listen(PORT, async () => {
  console.log(`✅ Express server running on http://localhost:${PORT}`);
  console.log(`📍 Routes:`);
  console.log(`   GET  /`);
  console.log(`   GET  /api/users`);
  console.log(`   GET  /api/status`);

  try {
    const client = new TunGoClient(
      {
        localPort: PORT,
        logLevel: 'info'
      },
      {
        onConnect: (info) => {
          console.log(`\n🌍 Public URL: ${info.url}`);
          console.log(`📊 Subdomain: ${info.subdomain}`);
          console.log(`\n✨ Your local server is now accessible from anywhere!\n`);
        },
        onError: (error) => {
          console.error('❌ Tunnel error:', error.message);
        },
        onDisconnect: (reason) => {
          console.log('⚠️  Tunnel disconnected:', reason);
        },
        onReconnect: (attempt) => {
          console.log(`🔄 Reconnecting... (attempt ${attempt})`);
        }
      }
    );

    await client.start();
  } catch (error) {
    console.error('Failed to start tunnel:', (error as Error).message);
  }
});

// Graceful shutdown
process.on('SIGINT', () => {
  console.log('\n👋 Shutting down gracefully...');
  server.close(() => {
    console.log('✅ Server closed');
    process.exit(0);
  });
});

process.on('SIGTERM', () => {
  console.log('\n👋 Received SIGTERM, shutting down...');
  server.close(() => {
    console.log('✅ Server closed');
    process.exit(0);
  });
});
