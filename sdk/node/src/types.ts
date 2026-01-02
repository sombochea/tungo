/**
 * TunGo client configuration options
 */
export interface TunGoOptions {
  /**
   * TunGo server host (default: localhost)
   */
  serverHost?: string;

  /**
   * Control port of the TunGo server (default: 5555)
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

  /**
   * Server host
   */
  serverHost: string;

  /**
   * Server port
   */
  serverPort: number;
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
