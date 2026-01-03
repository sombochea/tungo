import 'dart:async';
import 'dart:io';
import 'package:tungo_sdk/tungo.dart';

void main() async {
  // Start a local HTTP server
  final server = await startLocalServer(3000);
  print('Local server running on http://localhost:3000');

  // Create TunGo client
  final options = TunGoOptions(
    localPort: 3000,
    subdomain: 'my-dart-app',
    serverHost: 'localhost',
    controlPort: 5555,
    logLevel: 'INFO',
  );

  final eventListener = TunGoEventListener(
    onConnect: (info) {
      print('\n✓ Tunnel established!');
      print('  Public URL: ${info.url}');
      print('  Subdomain: ${info.subdomain}');
      print('  Client ID: ${info.clientId}');
      print('\nLocal server: http://localhost:3000');
      print('Public URL: ${info.url}');
      print('\nPress Ctrl+C to stop.');
    },
    onDisconnect: (reason) {
      print('✗ Disconnected: $reason');
    },
    onError: (error) {
      print('✗ Error: $error');
    },
    onReconnect: (attempt, maxRetries) {
      print('↻ Reconnecting (attempt $attempt/$maxRetries)...');
    },
    onStatus: (message) {
      print('ℹ Status: $message');
    },
  );

  final client = TunGoClient(options, eventListener);

  // Handle shutdown
  ProcessSignal.sigint.watch().listen((_) async {
    print('\nShutting down...');
    await client.stop();
    await server.close();
    exit(0);
  });

  // Start tunnel
  try {
    await client.start();
    await Completer<void>().future;
  } catch (e) {
    print('Failed to start tunnel: $e');
    await server.close();
    exit(1);
  }
}

/// Start a simple HTTP server for testing
Future<HttpServer> startLocalServer(int port) async {
  final server = await HttpServer.bind('localhost', port);

  server.listen((request) async {
    final response = request.response;

    if (request.uri.path == '/api/hello') {
      // API endpoint
      response.headers.contentType = ContentType.json;
      response.write(
          '{"message": "Hello from Dart!", "timestamp": "${DateTime.now().toIso8601String()}"}');
    } else {
      // HTML page
      response.headers.contentType = ContentType.html;
      response.write('''
<!DOCTYPE html>
<html>
<head>
    <title>TunGo Dart Example</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 50px auto;
            padding: 20px;
        }
        .card {
            border: 1px solid #ddd;
            padding: 20px;
            border-radius: 8px;
            margin: 20px 0;
        }
        button {
            background: #0066cc;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 4px;
            cursor: pointer;
        }
        button:hover {
            background: #0052a3;
        }
        #result {
            margin-top: 10px;
            padding: 10px;
            background: #f0f0f0;
            border-radius: 4px;
        }
    </style>
</head>
<body>
    <h1>🚀 TunGo Dart SDK Example</h1>
    
    <div class="card">
        <h2>Welcome!</h2>
        <p>This page is served from a local Dart HTTP server and exposed through TunGo tunnel.</p>
        <p><strong>Local:</strong> http://localhost:$port</p>
    </div>

    <div class="card">
        <h2>Test API</h2>
        <button onclick="testAPI()">Call API Endpoint</button>
        <div id="result"></div>
    </div>

    <script>
        async function testAPI() {
            const result = document.getElementById('result');
            result.textContent = 'Loading...';
            
            try {
                const response = await fetch('/api/hello');
                const data = await response.json();
                result.textContent = JSON.stringify(data, null, 2);
            } catch (error) {
                result.textContent = 'Error: ' + error.message;
            }
        }
    </script>
</body>
</html>
      ''');
    }

    await response.close();
  });

  return server;
}
