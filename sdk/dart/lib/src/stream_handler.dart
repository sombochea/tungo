import 'dart:convert';
import 'dart:typed_data';
import 'package:http/http.dart' as http;
import 'package:logging/logging.dart';
import 'protocol.dart';

/// Handles HTTP stream forwarding to local server
class StreamHandler {
  final String streamId;
  final String localHost;
  final int localPort;
  final void Function(MessageType type, String streamId, dynamic data)
      sendMessage;
  final Logger _logger = Logger('StreamHandler');
  bool _closed = false;

  StreamHandler({
    required this.streamId,
    required this.localHost,
    required this.localPort,
    required this.sendMessage,
  });

  /// Start the stream handler
  void start() {
    _logger.fine('Stream handler ready for: $streamId');
  }

  /// Handle incoming stream data
  Future<void> handleData(Map<String, dynamic> data) async {
    if (_closed) return;

    try {
      // Get base64 encoded HTTP request data
      final base64Data = data['data'] as String?;
      if (base64Data == null || base64Data.isEmpty) {
        _logger.fine('No data in message for stream $streamId');
        return;
      }

      // Decode base64 data to get raw HTTP request
      final httpRequestBytes = base64.decode(base64Data);
      final httpRequest = utf8.decode(httpRequestBytes);

      // Parse HTTP request
      final lines = httpRequest.split('\r\n');
      if (lines.isEmpty) {
        throw Exception('Empty HTTP request');
      }

      // Parse request line: METHOD /path HTTP/1.1
      final requestLine = lines[0].split(' ');
      if (requestLine.length < 2) {
        throw Exception('Invalid HTTP request line: ${lines[0]}');
      }

      final method = requestLine[0];
      final path = requestLine[1];

      // Parse headers
      final headers = <String, String>{};
      var bodyStart = 0;
      for (var i = 1; i < lines.length; i++) {
        if (lines[i].isEmpty) {
          // Empty line marks end of headers
          bodyStart = httpRequest.indexOf('\r\n\r\n');
          if (bodyStart != -1) {
            bodyStart += 4;
          }
          break;
        }

        final colonIndex = lines[i].indexOf(':');
        if (colonIndex > 0) {
          final key = lines[i].substring(0, colonIndex).trim();
          final value = lines[i].substring(colonIndex + 1).trim();
          headers[key] = value;
        }
      }

      // Extract body
      Uint8List? body;
      if (bodyStart > 0 && bodyStart < httpRequestBytes.length) {
        body = httpRequestBytes.sublist(bodyStart);
      }

      // Forward request to local server
      final url = Uri.parse('http://$localHost:$localPort$path');
      _logger.fine('Forwarding $method $path to local server');

      final request = http.Request(method, url);

      // Set headers (skip some that http package manages)
      headers.forEach((key, value) {
        final lowerKey = key.toLowerCase();
        if (lowerKey != 'host' &&
            lowerKey != 'connection' &&
            lowerKey != 'content-length' &&
            lowerKey != 'transfer-encoding') {
          request.headers[key] = value;
        }
      });

      // Set body if present
      if (body != null && body.isNotEmpty) {
        request.bodyBytes = body;
      }

      // Send request and get response
      final streamedResponse = await request.send();
      final responseBody = await streamedResponse.stream.toBytes();

      // Build HTTP response
      final responseLines = <String>[
        'HTTP/1.1 ${streamedResponse.statusCode} ${streamedResponse.reasonPhrase ?? "OK"}',
      ];

      // Add headers, excluding transfer-encoding and setting correct content-length
      streamedResponse.headers.forEach((key, value) {
        final lowerKey = key.toLowerCase();
        // Skip transfer-encoding since we have the full body
        if (lowerKey != 'transfer-encoding') {
          responseLines.add('$key: $value');
        }
      });

      // Add content-length header with actual body size
      responseLines.add('Content-Length: ${responseBody.length}');
      responseLines.add('');
      responseLines.add('');

      final responseHeader = utf8.encode(responseLines.join('\r\n'));
      final fullResponse =
          Uint8List.fromList([...responseHeader, ...responseBody]);

      // Send response back through tunnel
      sendMessage(
        MessageType.data,
        streamId,
        {'data': base64.encode(fullResponse)},
      );
      sendMessage(MessageType.end, streamId, null);
    } catch (e) {
      _logger.severe('Error handling stream data: $e');
      close();
    }
  }

  /// Close the stream handler
  void close() {
    if (_closed) return;
    _closed = true;
    _logger.fine('Stream closed: $streamId');
  }
}
