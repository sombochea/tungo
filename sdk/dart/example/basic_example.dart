import 'dart:async';
import 'package:tungo_sdk/tungo.dart';

void main() async {
  // Create client with options
  final options = TunGoOptions(
    localPort: 3000,
    subdomain: 'my-dart-app',
    logLevel: 'INFO',
  );

  // Set up event listeners
  final eventListener = TunGoEventListener(
    onConnect: (info) {
      print('✓ Tunnel established!');
      print('  Public URL: ${info.url}');
      print('  Subdomain: ${info.subdomain}');
      print('  Client ID: ${info.clientId}');
      print('\nTunnel is running. Press Ctrl+C to stop.');
      print('Try accessing: ${info.url}');
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
  );

  final client = TunGoClient(options, eventListener);

  // Start tunnel
  try {
    await client.start();

    // Keep running
    await Completer<void>().future;
  } catch (e) {
    print('Failed to start tunnel: $e');
  }
}
