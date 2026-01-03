import 'package:test/test.dart';
import 'package:tungo_sdk/tungo.dart';

void main() {
  group('TunGoOptions', () {
    test('creates with default values', () {
      final options = TunGoOptions(localPort: 3000);

      expect(options.serverHost, 'localhost');
      expect(options.controlPort, 5555);
      expect(options.localHost, 'localhost');
      expect(options.localPort, 3000);
      expect(options.connectTimeout, 10000);
      expect(options.maxRetries, 5);
      expect(options.retryInterval, 5000);
      expect(options.logLevel, 'INFO');
    });

    test('creates with custom values', () {
      final options = TunGoOptions(
        serverHost: 'example.com',
        controlPort: 8080,
        localHost: '127.0.0.1',
        localPort: 8000,
        subdomain: 'my-app',
        secretKey: 'secret123',
        connectTimeout: 5000,
        maxRetries: 3,
        retryInterval: 2000,
        logLevel: 'DEBUG',
      );

      expect(options.serverHost, 'example.com');
      expect(options.controlPort, 8080);
      expect(options.localHost, '127.0.0.1');
      expect(options.localPort, 8000);
      expect(options.subdomain, 'my-app');
      expect(options.secretKey, 'secret123');
      expect(options.connectTimeout, 5000);
      expect(options.maxRetries, 3);
      expect(options.retryInterval, 2000);
      expect(options.logLevel, 'DEBUG');
    });

    test('copyWith creates modified copy', () {
      final original = TunGoOptions(localPort: 3000);
      final modified = original.copyWith(
        serverHost: 'newhost.com',
        localPort: 4000,
      );

      expect(modified.serverHost, 'newhost.com');
      expect(modified.localPort, 4000);
      expect(modified.controlPort, original.controlPort);
      expect(modified.localHost, original.localHost);
    });
  });
}
