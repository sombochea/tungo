import 'dart:convert';
import 'package:test/test.dart';
import 'package:tungo_sdk/src/protocol.dart';

void main() {
  group('Protocol', () {
    test('creates client hello with anonymous client type', () {
      final hello = Protocol.createClientHello();
      final data = jsonDecode(hello) as Map<String, dynamic>;

      expect(data['client_type'], 'anonymous');
      expect(data['id'], isNotNull);
      expect(data.containsKey('sub_domain'), false);
      expect(data.containsKey('secret_key'), false);
    });

    test('creates client hello with subdomain', () {
      final hello = Protocol.createClientHello(subdomain: 'my-app');
      final data = jsonDecode(hello) as Map<String, dynamic>;

      expect(data['client_type'], 'anonymous');
      expect(data['sub_domain'], 'my-app');
    });

    test('creates client hello with auth', () {
      final hello = Protocol.createClientHello(
        subdomain: 'my-app',
        secretKey: 'secret123',
      );
      final data = jsonDecode(hello) as Map<String, dynamic>;

      expect(data['client_type'], 'auth');
      expect(data['sub_domain'], 'my-app');
      expect(data['secret_key'], {'key': 'secret123'});
    });

    test('creates message with type only', () {
      final message = Protocol.createMessage(MessageType.ping);
      final data = jsonDecode(message) as Map<String, dynamic>;

      expect(data['type'], 'ping');
      expect(data.containsKey('stream_id'), false);
      expect(data.containsKey('data'), false);
    });

    test('creates message with stream ID and data', () {
      final message = Protocol.createMessage(
        MessageType.data,
        streamId: 'stream-123',
        data: {'test': 'value'},
      );
      final data = jsonDecode(message) as Map<String, dynamic>;

      expect(data['type'], 'data');
      expect(data['stream_id'], 'stream-123');
      expect(data['data'], {'test': 'value'});
    });

    test('decodes server hello successfully', () {
      final json = jsonEncode({
        'type': 'success',
        'sub_domain': 'my-app',
        'hostname': 'my-app.localhost',
        'public_url': 'http://my-app.localhost:8080',
        'client_id': 'client-123',
      });

      final hello = Protocol.decodeServerHello(json);

      expect(hello.type, 'success');
      expect(hello.serverHelloType, ServerHelloType.success);
      expect(hello.subDomain, 'my-app');
      expect(hello.hostname, 'my-app.localhost');
      expect(hello.publicUrl, 'http://my-app.localhost:8080');
      expect(hello.clientId, 'client-123');
    });

    test('generates unique stream IDs', () {
      final id1 = Protocol.generateStreamId();
      final id2 = Protocol.generateStreamId();

      expect(id1, isNot(equals(id2)));
      expect(id1.length, greaterThan(0));
    });
  });

  group('MessageType', () {
    test('converts from value', () {
      expect(MessageType.fromValue('hello'), MessageType.hello);
      expect(MessageType.fromValue('data'), MessageType.data);
      expect(MessageType.fromValue('ping'), MessageType.ping);
    });
  });

  group('ServerHelloType', () {
    test('converts from value', () {
      expect(ServerHelloType.fromValue('success'), ServerHelloType.success);
      expect(ServerHelloType.fromValue('sub_domain_in_use'),
          ServerHelloType.subdomainInUse);
      expect(ServerHelloType.fromValue('invalid_sub_domain'),
          ServerHelloType.invalidSubdomain);
    });

    test('returns error for unknown value', () {
      expect(ServerHelloType.fromValue('unknown'), ServerHelloType.error);
    });
  });
}
