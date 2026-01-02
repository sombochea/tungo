import express, { Request, Response } from 'express';
import { TunGoClient } from '@tungo/sdk';

const app = express();
app.use(express.json());

interface WebhookPayload {
  event: string;
  data: any;
  timestamp: string;
}

// Webhook endpoint
app.post('/webhook', (req: Request, res: Response) => {
  const payload = req.body as WebhookPayload;
  
  console.log('\n📨 Webhook received:');
  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
  console.log('Event:', payload.event || 'unknown');
  console.log('Timestamp:', payload.timestamp || new Date().toISOString());
  console.log('Headers:', JSON.stringify(req.headers, null, 2));
  console.log('Body:', JSON.stringify(payload, null, 2));
  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n');
  
  res.json({
    success: true,
    received: new Date().toISOString(),
    event: payload.event || 'unknown'
  });
});

// Test endpoint
app.get('/', (req: Request, res: Response) => {
  res.send(`
    <!DOCTYPE html>
    <html>
      <head>
        <title>Webhook Server</title>
        <style>
          body { font-family: system-ui; max-width: 600px; margin: 50px auto; padding: 20px; }
          code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
          pre { background: #f4f4f4; padding: 15px; border-radius: 5px; overflow-x: auto; }
        </style>
      </head>
      <body>
        <h1>🎣 Webhook Server</h1>
        <p>Your webhook receiver is running!</p>
        
        <h2>Test with curl:</h2>
        <pre>curl -X POST ${req.protocol}://${req.get('host')}/webhook \\
  -H "Content-Type: application/json" \\
  -d '{"event":"test","data":{"message":"Hello"},"timestamp":"${new Date().toISOString()}"}'</pre>
        
        <h2>Endpoints:</h2>
        <ul>
          <li><code>POST /webhook</code> - Receive webhooks</li>
          <li><code>GET /</code> - This page</li>
        </ul>
      </body>
    </html>
  `);
});

const PORT = 4000;

app.listen(PORT, async () => {
  console.log(`✅ Webhook server running on http://localhost:${PORT}`);

  try {
    const client = new TunGoClient(
      {
        localPort: PORT,
        subdomain: 'my-webhooks',
      },
      {
        onConnect: (info) => {
          console.log(`\n🌍 Webhook URL: ${info.url}/webhook`);
          console.log(`🔗 Web UI: ${info.url}`);
          console.log('\n📝 Configure this URL in your webhook provider!');
          console.log('\n💡 Test command:');
          console.log(`   curl -X POST ${info.url}/webhook -H "Content-Type: application/json" -d '{"event":"test","data":"hello"}'`);
          console.log('');
        }
      }
    );

    await client.start();
  } catch (error) {
    console.error('Failed to start tunnel:', (error as Error).message);
  }
});
