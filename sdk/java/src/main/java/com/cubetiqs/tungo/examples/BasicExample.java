package com.cubetiqs.tungo.examples;

import com.cubetiqs.tungo.TunGoClient;
import com.cubetiqs.tungo.TunGoEventListener;
import com.cubetiqs.tungo.TunGoOptions;
import com.cubetiqs.tungo.TunnelInfo;

/**
 * Basic example of using TunGo SDK
 */
public class BasicExample {
    public static void main(String[] args) {
        // Configure options
        TunGoOptions options = TunGoOptions.builder(3000)
            .serverHost("localhost")
            .controlPort(5555)
            .subdomain("my-custom-subdomain")
            .logLevel("INFO")
            .build();

        // Create event listener
        TunGoEventListener listener = new TunGoEventListener() {
            @Override
            public void onConnect(TunnelInfo tunnelInfo) {
                System.out.println("✓ Tunnel established!");
                System.out.println("  Public URL: " + tunnelInfo.getUrl());
                System.out.println("  Subdomain: " + tunnelInfo.getSubdomain());
                System.out.println("  Client ID: " + tunnelInfo.getClientId());
            }

            @Override
            public void onDisconnect(String reason) {
                System.out.println("✗ Disconnected: " + reason);
            }

            @Override
            public void onError(Throwable error) {
                System.err.println("✗ Error: " + error.getMessage());
            }

            @Override
            public void onReconnect(int attempt, int maxRetries) {
                System.out.println("↻ Reconnecting (attempt " + attempt + "/" + maxRetries + ")...");
            }

            @Override
            public void onStatus(String message) {
                System.out.println("ℹ Status: " + message);
            }
        };

        // Create client
        TunGoClient client = new TunGoClient(options, listener);

        // Add shutdown hook
        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            System.out.println("\nShutting down...");
            client.shutdown();
        }));

        try {
            // Start tunnel
            System.out.println("Starting TunGo tunnel...");
            TunnelInfo info = client.start();
            
            System.out.println("\nTunnel is running. Press Ctrl+C to stop.");
            System.out.println("Try accessing: " + info.getUrl());

            // Keep running
            Thread.currentThread().join();

        } catch (Exception e) {
            System.err.println("Failed to start tunnel: " + e.getMessage());
            e.printStackTrace();
            System.exit(1);
        }
    }
}
