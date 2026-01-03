import 'tunnel_info.dart';

/// Event listener for TunGo client events
class TunGoEventListener {
  /// Called when tunnel is successfully established
  final void Function(TunnelInfo info)? onConnect;

  /// Called when connection is lost
  final void Function(String? reason)? onDisconnect;

  /// Called when an error occurs
  final void Function(Object error)? onError;

  /// Called on reconnection attempt
  final void Function(int attempt, int maxRetries)? onReconnect;

  /// Called on status updates
  final void Function(String message)? onStatus;

  TunGoEventListener({
    this.onConnect,
    this.onDisconnect,
    this.onError,
    this.onReconnect,
    this.onStatus,
  });
}
