#!/usr/bin/env node
/**
 * Manual test for reconnection functionality
 *
 * This script helps manually test reconnection behavior:
 * 1. Starts the tunnel client
 * 2. Waits for connection
 * 3. Instructs you to kill the server
 * 4. Observes reconnection attempts with preserved subdomain
 * 5. Shows continuous retry behavior after max attempts
 *
 * Usage:
 *     npm run test:manual
 *     or
 *     node --loader ts-node/esm tests/manual_reconnect.ts
 */

import { TunGoClient } from '../src/client.js';
import type { TunGoOptions, TunGoEvents, TunnelInfo } from '../src/types.js';

async function main() {
  console.log('='.repeat(60));
  console.log('TunGo Reconnection Test');
  console.log('='.repeat(60));
  console.log();

  let reconnectCount = 0;
  let disconnectCount = 0;
  let connectCount = 0;
  let originalSubdomain: string | null = null;

  const events: TunGoEvents = {
    onConnect: (info: TunnelInfo) => {
      connectCount++;

      if (originalSubdomain === null) {
        originalSubdomain = info.subdomain;
        console.log(`✅ Connected (#${connectCount})`);
        console.log(`   URL: ${info.url}`);
        console.log(`   Subdomain: ${info.subdomain}`);
      } else {
        const preserved = info.subdomain === originalSubdomain ? '✓' : '✗';
        console.log(`✅ Reconnected (#${connectCount}) - Subdomain ${preserved}`);
        console.log(`   URL: ${info.url}`);
        console.log(`   Subdomain: ${info.subdomain} (original: ${originalSubdomain})`);
      }
    },

    onDisconnect: (reason: string) => {
      disconnectCount++;
      console.log(`❌ Disconnected (#${disconnectCount}): ${reason}`);
    },

    onReconnect: (attempt: number) => {
      reconnectCount = attempt;
      console.log(`🔄 Reconnecting... (attempt ${attempt}/5)`);
    },

    onError: (error: Error) => {
      console.log(`⚠️  Error: ${error.message}`);
    },

    onStatus: (status: string) => {
      console.log(`📊 Status: ${status}`);
    },
  };

  const options: TunGoOptions = {
    localPort: 8000,
    serverUrl: 'ws://localhost:5555/ws',
    maxRetries: 5,
    retryInterval: 3000,
    logLevel: 'info',
  };

  const client = new TunGoClient(options, events);

  try {
    console.log('Starting tunnel...');
    console.log();
    await client.start();

    console.log();
    console.log('='.repeat(60));
    console.log('Test Instructions:');
    console.log('='.repeat(60));
    console.log('1. The tunnel is now running');
    console.log('2. Kill the TunGo server (Ctrl+C in server terminal)');
    console.log('3. Watch the reconnection attempts');
    console.log('4. After 5 attempts, it will use extended delay');
    console.log('5. Restart the server to see successful reconnection');
    console.log('6. Verify the subdomain is preserved');
    console.log('7. Press Ctrl+C here to stop the test');
    console.log('='.repeat(60));
    console.log();

    // Keep running for testing
    await new Promise(() => {});
  } catch (error) {
    if (error instanceof Error && error.message !== 'SIGINT') {
      console.log(`\n❌ Error: ${error.message}`);
    }
  } finally {
    console.log('\n');
    console.log('='.repeat(60));
    console.log('Test Summary:');
    console.log('='.repeat(60));
    console.log(`Total connections: ${connectCount}`);
    console.log(`Total disconnections: ${disconnectCount}`);
    console.log(`Last reconnect attempt: ${reconnectCount}`);
    console.log('='.repeat(60));
    console.log('\nStopping...');
    client.stop();
  }
}

// Handle Ctrl+C gracefully
process.on('SIGINT', () => {
  console.log('\n\nReceived SIGINT, stopping...');
  process.exit(0);
});

main();
