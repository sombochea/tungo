/**
 * Test event callbacks
 */

import { TunGoClient } from '../src/client.js';
import type { TunGoOptions, TunnelInfo } from '../src/types.js';

describe('Event Tests', () => {
  const serverUrl = 'ws://localhost:5555/ws';
  const localPort = 8000;

  test('should trigger connect event', async () => {
    let connectCalled = false;
    let receivedInfo: TunnelInfo | null = null;

    const options: TunGoOptions = {
      localPort,
      serverUrl,
    };

    const client = new TunGoClient(options, {
      onConnect: (info) => {
        connectCalled = true;
        receivedInfo = info;
      },
    });

    try {
      await client.start();
      expect(connectCalled).toBe(true);
      expect(receivedInfo).not.toBeNull();
      expect(receivedInfo!.url).toBeTruthy();
      expect(receivedInfo!.subdomain).toBeTruthy();
    } finally {
      client.stop();
    }
  }, 10000);

  test('should trigger disconnect event', async () => {
    let disconnectCalled = false;

    const options: TunGoOptions = {
      localPort,
      serverUrl,
    };

    const client = new TunGoClient(options, {
      onDisconnect: () => {
        disconnectCalled = true;
      },
    });

    try {
      await client.start();
      if (client['ws']) {
        client['ws'].close();
      }
      await new Promise((resolve) => setTimeout(resolve, 1000));
      expect(disconnectCalled).toBe(true);
    } finally {
      client.stop();
    }
  }, 10000);

  test('should trigger status events', async () => {
    const statuses: string[] = [];

    const options: TunGoOptions = {
      localPort,
      serverUrl,
    };

    const client = new TunGoClient(options, {
      onStatus: (status) => {
        statuses.push(status);
      },
    });

    try {
      await client.start();
      expect(statuses).toContain('connected');
    } finally {
      client.stop();
    }
  }, 10000);

  test('should trigger error event', async () => {
    let errorCalled = false;

    const options: TunGoOptions = {
      localPort,
      serverUrl: 'ws://invalid:9999/ws',
      connectTimeout: 2000,
    };

    const client = new TunGoClient(options, {
      onError: () => {
        errorCalled = true;
      },
    });

    try {
      await client.start();
    } catch (error) {
      // Expected to fail
    }

    await new Promise((resolve) => setTimeout(resolve, 500));
  }, 5000);
});
