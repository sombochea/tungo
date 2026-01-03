package com.cubetiqs.tungo.examples;

import com.sun.net.httpserver.HttpServer;
import com.cubetiqs.tungo.TunGoClient;
import com.cubetiqs.tungo.TunGoEventListener;
import com.cubetiqs.tungo.TunGoOptions;
import com.cubetiqs.tungo.TunnelInfo;

import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;

/**
 * Complete example with local HTTP server and TunGo tunnel
 */
public class CompleteExample {
    private static final int PORT = 8080;

    public static void main(String[] args) throws Exception {
        // Start local HTTP server
        HttpServer server = HttpServer.create(new InetSocketAddress(PORT), 0);
        
        server.createContext("/", exchange -> {
            String response = buildHtmlResponse();
            exchange.getResponseHeaders().add("Content-Type", "text/html");
            exchange.sendResponseHeaders(200, response.getBytes(StandardCharsets.UTF_8).length);
            
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(response.getBytes(StandardCharsets.UTF_8));
            }
        });

        server.createContext("/api/hello", exchange -> {
            String response = "{\"message\": \"Hello from TunGo!\", \"timestamp\": " + 
                System.currentTimeMillis() + "}";
            exchange.getResponseHeaders().add("Content-Type", "application/json");
            exchange.sendResponseHeaders(200, response.getBytes().length);
            
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(response.getBytes());
            }
        });

        server.setExecutor(null);
        server.start();

        System.out.println("✓ Local HTTP server started on port " + PORT);

        // Configure TunGo
        TunGoOptions options = TunGoOptions.builder(PORT)
            .serverHost("localhost")
            .controlPort(5555)
            .build();

        TunGoEventListener listener = new TunGoEventListener() {
            @Override
            public void onConnect(TunnelInfo tunnelInfo) {
                System.out.println("\n✓ Tunnel established!");
                System.out.println("┌─────────────────────────────────────────────┐");
                System.out.println("│  Public URL: " + tunnelInfo.getUrl());
                System.out.println("│  Subdomain:  " + tunnelInfo.getSubdomain());
                System.out.println("└─────────────────────────────────────────────┘\n");
                System.out.println("Your local server is now accessible via:");
                System.out.println("  → " + tunnelInfo.getUrl());
                System.out.println("  → " + tunnelInfo.getUrl() + "/api/hello");
            }

            @Override
            public void onDisconnect(String reason) {
                System.out.println("✗ Tunnel disconnected: " + reason);
            }

            @Override
            public void onError(Throwable error) {
                System.err.println("✗ Error: " + error.getMessage());
            }

            @Override
            public void onReconnect(int attempt, int maxRetries) {
                System.out.println("↻ Reconnecting... (attempt " + attempt + "/" + maxRetries + ")");
            }
        };

        TunGoClient client = new TunGoClient(options, listener);

        // Shutdown hook
        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            System.out.println("\n\nShutting down...");
            client.shutdown();
            server.stop(0);
            System.out.println("Goodbye!");
        }));

        try {
            System.out.println("Starting TunGo tunnel...\n");
            client.start();
            
            System.out.println("\nPress Ctrl+C to stop.");
            Thread.currentThread().join();

        } catch (Exception e) {
            System.err.println("Failed to start tunnel: " + e.getMessage());
            e.printStackTrace();
            server.stop(0);
            System.exit(1);
        }
    }

    private static String buildHtmlResponse() {
        return "<!DOCTYPE html>\n" +
            "<html>\n" +
            "<head>\n" +
            "    <title>TunGo Java SDK Demo</title>\n" +
            "    <style>\n" +
            "        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }\n" +
            "        h1 { color: #2563eb; }\n" +
            "        .card { background: #f3f4f6; padding: 20px; border-radius: 8px; margin: 20px 0; }\n" +
            "        button { background: #2563eb; color: white; border: none; padding: 10px 20px; border-radius: 4px; cursor: pointer; }\n" +
            "        button:hover { background: #1d4ed8; }\n" +
            "        #response { margin-top: 10px; padding: 10px; background: white; border-radius: 4px; }\n" +
            "    </style>\n" +
            "</head>\n" +
            "<body>\n" +
            "    <h1>🚀 TunGo Java SDK Demo</h1>\n" +
            "    <div class='card'>\n" +
            "        <h2>Welcome!</h2>\n" +
            "        <p>This is a local HTTP server exposed through TunGo tunnel.</p>\n" +
            "        <p>Your server is running on <strong>localhost:" + PORT + "</strong> but accessible from anywhere!</p>\n" +
            "    </div>\n" +
            "    <div class='card'>\n" +
            "        <h2>Test API</h2>\n" +
            "        <button onclick='testApi()'>Call /api/hello</button>\n" +
            "        <div id='response'></div>\n" +
            "    </div>\n" +
            "    <script>\n" +
            "        function testApi() {\n" +
            "            fetch('/api/hello')\n" +
            "                .then(r => r.json())\n" +
            "                .then(data => {\n" +
            "                    document.getElementById('response').innerHTML = \n" +
            "                        '<strong>Response:</strong><br>' + JSON.stringify(data, null, 2);\n" +
            "                });\n" +
            "        }\n" +
            "    </script>\n" +
            "</body>\n" +
            "</html>";
    }
}
