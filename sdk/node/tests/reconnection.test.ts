/**
 * Test reconnection functionality
 */

import { TunGoClient } from '../src/client.js';
import type { TunGoOptions } from '../src/types.js';

describe('Reconnection Tests', () => {
    const serverUrl = 'ws://localhost:5555/ws';
    const localPort = 8000;

    test('should attempt reconnection on disconnect', async () => {
        let disconnectCount = 0;
        let reconnectCount = 0;

        const options: TunGoOptions = {
            localPort,
            serverUrl,
            maxRetries: 3,
            retryInterval: 2000,
        };

        const client = new TunGoClient(options, {
            onDisconnect: () => {
                disconnectCount++;
            },
            onReconnect: (attempt) => {
                reconnectCount = attempt;
            },
        });

        try {
            await client.start();

            // Simulate disconnection
            if (client['ws']) {
                client['ws'].close();
            }

            // Wait for reconnection attempts
            await new Promise((resolve) => setTimeout(resolve, 8000));

            expect(disconnectCount).toBeGreaterThan(0);
        } finally {
            client.stop();
        }
    }, 15000);

    test('should preserve subdomain across reconnections', async () => {
        const subdomains: string[] = [];

        const options: TunGoOptions = {
            localPort,
            serverUrl,
            maxRetries: 2,
            retryInterval: 1000,
        };

        const client = new TunGoClient(options, {
            onConnect: (info) => {
                subdomains.push(info.subdomain);
            },
        });

        try {
            // First connection
            await client.start();
            const firstSubdomain = subdomains[0];

            // Simulate reconnection
            client.stop();
            await new Promise((resolve) => setTimeout(resolve, 1000));
            await client.start();

            expect(subdomains.length).toBeGreaterThanOrEqual(2);
            expect(subdomains[1]).toBe(firstSubdomain);
        } finally {
            client.stop();
        }
    }, 15000);

    test.skip('should continue retrying after max attempts', async () => {
        // This test is skipped because it's difficult to reliably test reconnection
        // without an actual server being started and stopped during the test.
        // Use the manual test script (npm run test:manual) to verify this behavior.
        const reconnectAttempts: number[] = [];

        const options: TunGoOptions = {
            localPort,
            serverUrl: 'ws://localhost:9999/ws', // Invalid server
            maxRetries: 2,
            retryInterval: 1000,
            connectTimeout: 1000,
        };

        const client = new TunGoClient(options, {
            onReconnect: (attempt) => {
                reconnectAttempts.push(attempt);
            },
        });

        try {
            await Promise.race([
                client.start(),
                new Promise((_, reject) => setTimeout(() => reject(new Error('timeout')), 500)),
            ]);
        } catch (error) {
            // Expected to fail
        }

        await new Promise((resolve) => setTimeout(resolve, 5000));
        expect(reconnectAttempts.length).toBeGreaterThanOrEqual(1);
        client.stop();
    }, 10000);
});
