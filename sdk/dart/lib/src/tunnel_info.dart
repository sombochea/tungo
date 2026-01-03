/// Tunnel information returned after successful connection
class TunnelInfo {
  /// Public URL to access your local server
  final String url;

  /// Assigned subdomain
  final String subdomain;

  /// Client ID
  final String clientId;

  TunnelInfo({
    required this.url,
    required this.subdomain,
    required this.clientId,
  });

  @override
  String toString() {
    return 'TunnelInfo(url: $url, subdomain: $subdomain, clientId: $clientId)';
  }
}
