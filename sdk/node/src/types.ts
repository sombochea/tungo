/**
 * TunGo client configuration options
 */
export interface TunGoOptions {
  /**
   * Full server WebSocket URL (e.g., ws://localhost:5555/ws or wss://tunnel.example.com/ws)
   * If provided, serverHost and controlPort are ignored
   */
  serverUrl?: string;

  /**
   * TunGo server host (default: localhost)
   * Ignored if serverUrl is set
   */
  serverHost?: string;

  /**
   * Control port of the TunGo server (default: 5555)
   * Ignored if serverUrl is set
   */
  controlPort?: number;

  /**
   * Local server host to tunnel (default: localhost)
   */
  localHost?: string;

  /**
   * Local server port to tunnel (required)
   */
  localPort: number;

  /**
   * Custom subdomain (optional, random if not provided)
   */
  subdomain?: string;

  /**
   * Authentication secret key (optional)
   */
  secretKey?: string;

  /**
   * Connection timeout in milliseconds (default: 10000)
   */
  connectTimeout?: number;

  /**
   * Maximum reconnection attempts (default: 5)
   */
  maxRetries?: number;

  /**
   * Retry interval in milliseconds (default: 5000)
   */
  retryInterval?: number;

  /**
   * Log level: 'debug' | 'info' | 'warn' | 'error' (default: 'info')
   */
  logLevel?: 'debug' | 'info' | 'warn' | 'error';
}

/**
 * Tunnel information returned after successful connection
 */
export interface TunnelInfo {
  /**
   * The public URL to access your local server
   */
  url: string;

  /**
   * The assigned subdomain
   */
  subdomain: string;
}

/**
 * Event handlers for TunGo client
 */
export interface TunGoEvents {
  /**
   * Called when tunnel is successfully established
   */
  onConnect?: (info: TunnelInfo) => void;

  /**
   * Called when connection is lost
   */
  onDisconnect?: (reason?: string) => void;

  /**
   * Called when an error occurs
   */
  onError?: (error: Error) => void;

  /**
   * Called on reconnection attempt
   */
  onReconnect?: (attempt: number) => void;

  /**
   * Called on status updates
   */
  onStatus?: (status: string) => void;
}
