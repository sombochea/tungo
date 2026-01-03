/// Configuration options for TunGo client
class TunGoOptions {
  /// Server URL (e.g., ws://localhost:5555/ws)
  final String? serverUrl;

  /// Server host (default: localhost)
  final String serverHost;

  /// Control port (default: 5555)
  final int controlPort;

  /// Local host to forward requests to (default: localhost)
  final String localHost;

  /// Local port to forward requests to
  final int localPort;

  /// Subdomain to request (optional)
  final String? subdomain;

  /// Secret key for authentication (optional)
  final String? secretKey;

  /// Connection timeout in milliseconds (default: 10000)
  final int connectTimeout;

  /// Maximum reconnection attempts (default: 5)
  final int maxRetries;

  /// Retry interval in milliseconds (default: 5000)
  final int retryInterval;

  /// Log level (default: INFO)
  final String logLevel;

  TunGoOptions({
    this.serverUrl,
    this.serverHost = 'localhost',
    this.controlPort = 5555,
    this.localHost = 'localhost',
    required this.localPort,
    this.subdomain,
    this.secretKey,
    this.connectTimeout = 10000,
    this.maxRetries = 5,
    this.retryInterval = 5000,
    this.logLevel = 'INFO',
  });

  /// Create a copy with modified fields
  TunGoOptions copyWith({
    String? serverUrl,
    String? serverHost,
    int? controlPort,
    String? localHost,
    int? localPort,
    String? subdomain,
    String? secretKey,
    int? connectTimeout,
    int? maxRetries,
    int? retryInterval,
    String? logLevel,
  }) {
    return TunGoOptions(
      serverUrl: serverUrl ?? this.serverUrl,
      serverHost: serverHost ?? this.serverHost,
      controlPort: controlPort ?? this.controlPort,
      localHost: localHost ?? this.localHost,
      localPort: localPort ?? this.localPort,
      subdomain: subdomain ?? this.subdomain,
      secretKey: secretKey ?? this.secretKey,
      connectTimeout: connectTimeout ?? this.connectTimeout,
      maxRetries: maxRetries ?? this.maxRetries,
      retryInterval: retryInterval ?? this.retryInterval,
      logLevel: logLevel ?? this.logLevel,
    );
  }
}
