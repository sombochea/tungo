import 'dart:convert';
import 'package:uuid/uuid.dart';

/// Message types in the TunGo protocol
enum MessageType {
  hello('hello'),
  serverHello('server_hello'),
  init('init'),
  data('data'),
  end('end'),
  ping('ping'),
  pong('pong');

  final String value;
  const MessageType(this.value);

  static MessageType fromValue(String value) {
    return MessageType.values.firstWhere((e) => e.value == value);
  }
}

/// Server hello types
enum ServerHelloType {
  success('success'),
  subdomainInUse('sub_domain_in_use'),
  invalidSubdomain('invalid_sub_domain'),
  authFailed('auth_failed'),
  error('error');

  final String value;
  const ServerHelloType(this.value);

  static ServerHelloType fromValue(String value) {
    return ServerHelloType.values.firstWhere(
      (e) => e.value == value,
      orElse: () => ServerHelloType.error,
    );
  }
}

/// Protocol utilities
class Protocol {
  static const _uuid = Uuid();

  /// Create a client hello message
  static String createClientHello({String? subdomain, String? secretKey}) {
    final message = <String, dynamic>{
      'id': _uuid.v4(),
      'client_type': secretKey != null ? 'auth' : 'anonymous',
    };

    if (subdomain != null && subdomain.isNotEmpty) {
      message['sub_domain'] = subdomain;
    }

    if (secretKey != null && secretKey.isNotEmpty) {
      message['secret_key'] = {'key': secretKey};
    }

    return jsonEncode(message);
  }

  /// Create a protocol message
  static String createMessage(
    MessageType type, {
    String? streamId,
    dynamic data,
  }) {
    final message = <String, dynamic>{
      'type': type.value,
    };

    if (streamId != null) {
      message['stream_id'] = streamId;
    }

    if (data != null) {
      message['data'] = data;
    }

    return jsonEncode(message);
  }

  /// Decode a message from JSON
  static Map<String, dynamic> decodeMessage(String json) {
    return jsonDecode(json) as Map<String, dynamic>;
  }

  /// Decode a server hello message
  static ServerHello decodeServerHello(String json) {
    final data = jsonDecode(json) as Map<String, dynamic>;
    return ServerHello.fromJson(data);
  }

  /// Generate a random stream ID
  static String generateStreamId() {
    return _uuid.v4();
  }
}

/// Server hello message
class ServerHello {
  final String type;
  final String? subDomain;
  final String? hostname;
  final String? publicUrl;
  final String? clientId;
  final Map<String, String>? reconnectToken;
  final String? error;

  ServerHello({
    required this.type,
    this.subDomain,
    this.hostname,
    this.publicUrl,
    this.clientId,
    this.reconnectToken,
    this.error,
  });

  factory ServerHello.fromJson(Map<String, dynamic> json) {
    return ServerHello(
      type: json['type'] as String,
      subDomain: json['sub_domain'] as String?,
      hostname: json['hostname'] as String?,
      publicUrl: json['public_url'] as String?,
      clientId: json['client_id'] as String?,
      reconnectToken: json['reconnect_token'] != null
          ? Map<String, String>.from(json['reconnect_token'] as Map)
          : null,
      error: json['error'] as String?,
    );
  }

  ServerHelloType get serverHelloType => ServerHelloType.fromValue(type);
}
