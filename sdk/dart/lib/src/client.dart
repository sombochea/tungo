import 'dart:async';
import 'package:logging/logging.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import 'options.dart';
import 'tunnel_info.dart';
import 'event_listener.dart';
import 'protocol.dart';
import 'stream_handler.dart';

/// TunGo Client - Expose your local server to the internet
class TunGoClient {
  final TunGoOptions options;
  final TunGoEventListener? eventListener;
  final Logger _logger = Logger('TunGoClient');

  WebSocketChannel? _channel;
  TunnelInfo? _tunnelInfo;
  bool _running = false;
  int _reconnectAttempts = 0;
  Timer? _pingTimer;
  final Map<String, StreamHandler> _streams = {};

  TunGoClient(this.options, [this.eventListener]) {
    _setupLogging();
  }

  void _setupLogging() {
    Logger.root.level = Level.INFO;
    Logger.root.onRecord.listen((record) {
      if (options.logLevel == 'DEBUG' ||
          (options.logLevel == 'INFO' && record.level >= Level.INFO) ||
          (options.logLevel == 'WARNING' && record.level >= Level.WARNING) ||
          (options.logLevel == 'ERROR' && record.level >= Level.SEVERE)) {
        print('[${record.level.name}] ${record.loggerName}: ${record.message}');
      }
    });
  }

  /// Start the tunnel
  Future<TunnelInfo> start() async {
    if (_channel != null) {
      throw StateError('Tunnel is already running');
    }

    // Build WebSocket URL
    final wsUrl = _buildWebSocketUrl();

    _logger.info('Connecting to TunGo server: $wsUrl');

    try {
      // Connect with timeout
      final channel = WebSocketChannel.connect(Uri.parse(wsUrl));
      _channel = channel;

      // Send client hello
      await _sendClientHello();

      // Wait for server hello
      final completer = Completer<TunnelInfo>();
      StreamSubscription? subscription;

      final timeoutTimer = Timer(
        Duration(milliseconds: options.connectTimeout),
        () {
          if (!completer.isCompleted) {
            subscription?.cancel();
            _channel?.sink.close();
            _channel = null;
            completer.completeError(TimeoutException('Connection timeout'));
          }
        },
      );

      subscription = channel.stream.listen(
        (message) {
          if (!completer.isCompleted) {
            try {
              final serverHello = Protocol.decodeServerHello(message as String);
              _handleServerHello(serverHello);

              if (_tunnelInfo != null) {
                timeoutTimer.cancel();
                completer.complete(_tunnelInfo!);
              } else {
                completer
                    .completeError(Exception('Failed to establish tunnel'));
              }
            } catch (e) {
              completer.completeError(e);
            }
          } else {
            // Handle subsequent messages
            _handleMessage(message as String);
          }
        },
        onError: (error) {
          if (!completer.isCompleted) {
            timeoutTimer.cancel();
            completer.completeError(error);
          } else {
            _handleError(error);
          }
        },
        onDone: () {
          if (!completer.isCompleted) {
            timeoutTimer.cancel();
            completer.completeError(Exception('Connection closed'));
          } else {
            _handleDisconnect('Connection closed');
          }
        },
      );

      final info = await completer.future;

      // Start message handling and ping
      _running = true;
      _startPing();

      return info;
    } catch (e) {
      _logger.severe('Failed to connect: $e');
      _channel = null;
      rethrow;
    }
  }

  /// Stop the tunnel
  Future<void> stop() async {
    _running = false;
    _pingTimer?.cancel();
    _pingTimer = null;

    for (final handler in _streams.values) {
      handler.close();
    }
    _streams.clear();

    await _channel?.sink.close();
    _channel = null;
    _tunnelInfo = null;

    _logger.info('Tunnel stopped');
  }

  /// Check if tunnel is active
  bool get isActive => _channel != null && _running;

  /// Get tunnel information
  TunnelInfo? get tunnelInfo => _tunnelInfo;

  String _buildWebSocketUrl() {
    if (options.serverUrl != null) {
      var url = options.serverUrl!;
      if (!url.startsWith('ws://') && !url.startsWith('wss://')) {
        url = 'ws://$url';
      }
      if (!url.endsWith('/ws')) {
        url = url.endsWith('/') ? '${url}ws' : '$url/ws';
      }
      return url;
    }
    return 'ws://${options.serverHost}:${options.controlPort}/ws';
  }

  Future<void> _sendClientHello() async {
    final hello = Protocol.createClientHello(
      subdomain: options.subdomain,
      secretKey: options.secretKey,
    );
    _channel?.sink.add(hello);
    _logger.fine('Sent client hello');
  }

  void _handleServerHello(ServerHello hello) {
    if (hello.serverHelloType != ServerHelloType.success) {
      final error = hello.error ?? 'Server hello failed: ${hello.type}';
      throw Exception(error);
    }

    final publicUrl = hello.publicUrl ?? 'http://${hello.hostname}';

    _tunnelInfo = TunnelInfo(
      url: publicUrl,
      subdomain: hello.subDomain ?? '',
      clientId: hello.clientId ?? '',
    );

    _logger.info('Tunnel established: $publicUrl');
    eventListener?.onConnect?.call(_tunnelInfo!);
  }

  void _handleMessage(String message) {
    try {
      final data = Protocol.decodeMessage(message);
      final type = MessageType.fromValue(data['type'] as String);
      final streamId = data['stream_id'] as String?;

      switch (type) {
        case MessageType.init:
          if (streamId != null) {
            _handleInitStream(streamId, data['data']);
          }
          break;

        case MessageType.data:
          if (streamId != null) {
            _handleStreamData(streamId, data['data'] as Map<String, dynamic>);
          }
          break;

        case MessageType.end:
          if (streamId != null) {
            _handleStreamEnd(streamId);
          }
          break;

        case MessageType.ping:
          _handlePing();
          break;

        default:
          _logger.fine('Unknown message type: $type');
      }
    } catch (e) {
      _logger.warning('Error handling message: $e');
    }
  }

  void _handleInitStream(String streamId, dynamic data) {
    _logger.fine('Initiating stream: $streamId');

    final handler = StreamHandler(
      streamId: streamId,
      localHost: options.localHost,
      localPort: options.localPort,
      sendMessage: _sendMessage,
    );

    _streams[streamId] = handler;
    handler.start();
  }

  void _handleStreamData(String streamId, Map<String, dynamic> data) {
    final handler = _streams[streamId];
    if (handler != null) {
      handler.handleData(data);
    }
  }

  void _handleStreamEnd(String streamId) {
    _logger.fine('Ending stream: $streamId');
    final handler = _streams.remove(streamId);
    handler?.close();
  }

  void _handlePing() {
    _sendMessage(MessageType.pong, null, null);
  }

  void _sendMessage(MessageType type, String? streamId, dynamic data) {
    if (_channel != null && _running) {
      try {
        final message =
            Protocol.createMessage(type, streamId: streamId, data: data);
        _channel?.sink.add(message);
      } catch (e) {
        _logger.fine('Failed to send message: $e');
      }
    }
  }

  void _startPing() {
    _pingTimer = Timer.periodic(const Duration(seconds: 30), (_) {
      if (_running && _channel != null) {
        _sendMessage(MessageType.ping, null, null);
      }
    });
  }

  void _handleError(Object error) {
    _logger.severe('WebSocket error: $error');
    eventListener?.onError?.call(error);
    _handleReconnect();
  }

  void _handleDisconnect(String reason) {
    _logger.info('Disconnected: $reason');
    eventListener?.onDisconnect?.call(reason);

    if (_running) {
      _handleReconnect();
    }
  }

  void _handleReconnect() {
    if (_reconnectAttempts >= options.maxRetries) {
      _logger.warning(
        'Max reconnection attempts reached (${options.maxRetries}), resetting counter',
      );
      _reconnectAttempts = 0;

      // Extended delay
      final extendedDelay = (options.retryInterval * 6).clamp(0, 30000);
      _scheduleReconnect(Duration(milliseconds: extendedDelay));
      return;
    }

    _reconnectAttempts++;

    // Exponential backoff
    final delay = (options.retryInterval * (1 << (_reconnectAttempts - 1)))
        .clamp(0, 30000);

    _logger.info(
      'Reconnecting in ${delay}ms (attempt $_reconnectAttempts/${options.maxRetries})',
    );

    eventListener?.onReconnect?.call(_reconnectAttempts, options.maxRetries);
    _scheduleReconnect(Duration(milliseconds: delay));
  }

  void _scheduleReconnect(Duration delay) {
    Timer(delay, () async {
      if (_running) {
        try {
          await stop();
          await start();
          _reconnectAttempts = 0;
        } catch (e) {
          _logger.severe('Reconnection failed: $e');
          _handleReconnect();
        }
      }
    });
  }
}
