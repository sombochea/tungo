import { EventEmitter } from 'events';
import * as http from 'http';
import WebSocket from 'ws';
import type { TunGoOptions, TunnelInfo, TunGoEvents } from './types.js';
import {
  MessageType,
  ServerHelloType,
  createClientHello,
  createMessage,
  encodeMessage,
  decodeMessage,
  generateStreamId,
  type Message,
  type ServerHello,
  type InitStreamMessage,
} from './protocol.js';

/**
 * TunGo Client - Expose your local server to the internet
 */
export class TunGoClient extends EventEmitter {
  private options: Required<TunGoOptions>;
  private ws: WebSocket | null = null;
  private tunnelInfo: TunnelInfo | null = null;
  private reconnectAttempts = 0;
  private reconnectTimer: NodeJS.Timeout | null = null;
  private streams: Map<string, http.ClientRequest> = new Map();
  private pingInterval: NodeJS.Timeout | null = null;

  constructor(options: TunGoOptions, events?: TunGoEvents) {
    super();

    // Set defaults
    this.options = {
      serverUrl: options.serverUrl,
      serverHost: options.serverHost || 'localhost',
      controlPort: options.controlPort || 5555,
      localHost: options.localHost || 'localhost',
      localPort: options.localPort,
      subdomain: options.subdomain || '',
      secretKey: options.secretKey || '',
      connectTimeout: options.connectTimeout || 10000,
      maxRetries: options.maxRetries || 5,
      retryInterval: options.retryInterval || 5000,
      logLevel: options.logLevel || 'info',
    };

    // Register event handlers
    if (events) {
      if (events.onConnect) this.on('connect', events.onConnect);
      if (events.onDisconnect) this.on('disconnect', events.onDisconnect);
      if (events.onError) this.on('error', events.onError);
      if (events.onReconnect) this.on('reconnect', events.onReconnect);
      if (events.onStatus) this.on('status', events.onStatus);
    }
  }

  /**
   * Start the tunnel
   */
  async start(): Promise<TunnelInfo> {
    if (this.ws) {
      throw new Error('Tunnel is already running');
    }

    return new Promise((resolve, reject) => {
      // Build WebSocket URL
      let wsUrl: string;
      if (this.options.serverUrl) {
        wsUrl = this.options.serverUrl;
        // Ensure URL starts with ws:// or wss://
        if (!wsUrl.startsWith('ws://') && !wsUrl.startsWith('wss://')) {
          wsUrl = `ws://${wsUrl}`;
        }
        // Ensure URL ends with /ws path
        if (!wsUrl.endsWith('/ws')) {
          wsUrl = wsUrl.endsWith('/') ? `${wsUrl}ws` : `${wsUrl}/ws`;
        }
      } else {
        wsUrl = `ws://${this.options.serverHost}:${this.options.controlPort}/ws`;
      }

      this.log('info', `Connecting to TunGo server: ${wsUrl}`);

      const connectTimeout = setTimeout(() => {
        if (!this.tunnelInfo) {
          this.stop();
          reject(new Error('Connection timeout'));
        }
      }, this.options.connectTimeout);

      // Create WebSocket connection
      this.ws = new WebSocket(wsUrl);

      // Handle connection open
      this.ws.on('open', () => {
        this.log('debug', 'WebSocket connected');
        this.sendClientHello();
      });

      // Handle messages
      this.ws!.on('message', (data: Buffer) => {
        try {
          const json = JSON.parse(data.toString());

          // First message is ServerHello (sent directly)
          if (!this.tunnelInfo && json.type) {
            this.handleServerHello(json as ServerHello, resolve, reject, connectTimeout);
          } else {
            // Subsequent messages are wrapped in Message protocol
            const message = json as Message;
            this.handleMessage(message, resolve, reject, connectTimeout);
          }
        } catch (error) {
          this.log('error', `Failed to decode message: ${(error as Error).message}`);
        }
      });

      // Handle errors
      this.ws!.on('error', (error: Error) => {
        this.log('error', `WebSocket error: ${error.message}`);
        clearTimeout(connectTimeout);
        if (!this.tunnelInfo) {
          reject(error);
        }
        this.emit('error', error);
      });

      // Handle close
      this.ws!.on('close', () => {
        this.log('info', 'WebSocket closed');
        clearTimeout(connectTimeout);

        if (this.pingInterval) {
          clearInterval(this.pingInterval);
          this.pingInterval = null;
        }

        this.ws = null;

        if (this.tunnelInfo) {
          this.emit('disconnect', 'Connection closed');
          this.handleReconnect();
        } else if (!this.reconnectTimer) {
          reject(new Error('Connection closed before tunnel established'));
        }
      });
    });
  }

  /**
   * Stop the tunnel
   */
  stop(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }

    if (this.pingInterval) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }

    // Close all active streams
    for (const [streamId, req] of this.streams.entries()) {
      req.destroy();
      this.streams.delete(streamId);
    }

    if (this.ws) {
      this.log('info', 'Stopping tunnel...');
      this.ws.close();
      this.ws = null;
      this.tunnelInfo = null;
      this.reconnectAttempts = 0;
      this.emit('status', 'stopped');
    }
  }

  /**
   * Send client hello message
   */
  private sendClientHello(): void {
    const hello = createClientHello(
      this.options.subdomain || undefined,
      this.options.secretKey || undefined
    );

    // Send ClientHello directly as JSON (not wrapped in Message)
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(hello));
      this.log('debug', 'Sent client hello');
    }
  }

  /**
   * Handle incoming messages
   */
  private handleMessage(
    message: Message,
    resolve: (info: TunnelInfo) => void,
    reject: (error: Error) => void,
    connectTimeout: NodeJS.Timeout
  ): void {
    switch (message.type) {
      case MessageType.Init:
        this.handleInitStream(message.stream_id!, message.data as InitStreamMessage);
        break;

      case MessageType.Data:
        this.handleStreamData(message.stream_id!, message.data);
        break;

      case MessageType.End:
        this.handleStreamEnd(message.stream_id!);
        break;

      case MessageType.Ping:
        this.handlePing();
        break;

      default:
        this.log('debug', `Unknown message type: ${message.type}`);
    }
  }

  /**
   * Handle server hello
   */
  private handleServerHello(
    hello: ServerHello,
    resolve: (info: TunnelInfo) => void,
    reject: (error: Error) => void,
    connectTimeout: NodeJS.Timeout
  ): void {
    clearTimeout(connectTimeout);

    if (hello.type !== ServerHelloType.Success) {
      const error = new Error(hello.error || `Server hello failed: ${hello.type}`);
      reject(error);
      this.stop();
      return;
    }

    // Use public_url from server if available, otherwise construct from hostname
    const publicUrl = hello.public_url || `http://${hello.hostname}`;

    this.tunnelInfo = {
      url: publicUrl,
      subdomain: hello.sub_domain!,
    };

    this.log('info', `Tunnel established: ${this.tunnelInfo.url}`);

    // Start ping interval
    this.pingInterval = setInterval(() => {
      this.sendPing();
    }, 30000); // Ping every 30 seconds

    this.emit('connect', this.tunnelInfo);
    this.emit('status', 'connected');
    resolve(this.tunnelInfo);
  }

  /**
   * Handle init stream
   */
  private handleInitStream(streamId: string, initData: InitStreamMessage): void {
    this.log('debug', `New stream: ${streamId}`);

    // Forward to local server
    const req = http.request(
      {
        host: this.options.localHost,
        port: this.options.localPort,
        method: 'GET', // Will be updated when we receive the actual request
        path: '/',
      },
      (res) => {
        // Collect response
        const chunks: Buffer[] = [];

        res.on('data', (chunk: Buffer) => {
          chunks.push(chunk);
        });

        res.on('end', () => {
          const responseData = Buffer.concat(chunks);

          // Build HTTP response
          let httpResponse = `HTTP/1.1 ${res.statusCode} ${res.statusMessage}\r\n`;
          for (const [key, value] of Object.entries(res.headers)) {
            if (Array.isArray(value)) {
              value.forEach(v => {
                httpResponse += `${key}: ${v}\r\n`;
              });
            } else if (value) {
              httpResponse += `${key}: ${value}\r\n`;
            }
          }
          httpResponse += '\r\n';

          const responseBuffer = Buffer.concat([
            Buffer.from(httpResponse),
            responseData
          ]);

          // Send response back through tunnel
          this.sendStreamData(streamId, responseBuffer);
          this.sendStreamEnd(streamId);
          this.streams.delete(streamId);
        });
      }
    );

    req.on('error', (error) => {
      this.log('error', `Stream ${streamId} error: ${error.message}`);
      this.sendStreamEnd(streamId);
      this.streams.delete(streamId);
    });

    this.streams.set(streamId, req);
  }

  /**
   * Handle stream data
   */
  private handleStreamData(streamId: string, data: any): void {
    const req = this.streams.get(streamId);
    if (!req) {
      this.log('warn', `Received data for unknown stream: ${streamId}`);
      return;
    }

    // Data is base64 encoded
    const buffer = Buffer.from(data.data || data, 'base64');

    // Parse HTTP request if this is the first data
    if (!req.headersSent) {
      const httpString = buffer.toString();
      const lines = httpString.split('\r\n');
      const requestLine = lines[0].split(' ');
      const method = requestLine[0];
      const path = requestLine[1];

      // Find headers
      const headers: Record<string, string> = {};
      let bodyStart = 0;
      for (let i = 1; i < lines.length; i++) {
        if (lines[i] === '') {
          bodyStart = lines.slice(0, i + 1).join('\r\n').length + 2;
          break;
        }
        const [key, ...valueParts] = lines[i].split(':');
        if (key && valueParts.length > 0) {
          headers[key.toLowerCase()] = valueParts.join(':').trim();
        }
      }

      // Update request
      req.method = method;
      req.path = path;
      for (const [key, value] of Object.entries(headers)) {
        req.setHeader(key, value);
      }

      // Send request
      if (bodyStart < buffer.length) {
        req.write(buffer.slice(bodyStart));
      }
      req.end();
    } else {
      req.write(buffer);
    }
  }

  /**
   * Handle stream end
   */
  private handleStreamEnd(streamId: string): void {
    const req = this.streams.get(streamId);
    if (req) {
      req.end();
      this.streams.delete(streamId);
    }
    this.log('debug', `Stream ended: ${streamId}`);
  }

  /**
   * Handle ping
   */
  private handlePing(): void {
    this.sendPong();
  }

  /**
   * Send message
   */
  private sendMessage(message: Message): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(encodeMessage(message));
    }
  }

  /**
   * Send stream data
   */
  private sendStreamData(streamId: string, data: Buffer): void {
    const message = createMessage(MessageType.Data, streamId, {
      data: data.toString('base64'),
    });
    this.sendMessage(message);
  }

  /**
   * Send stream end
   */
  private sendStreamEnd(streamId: string): void {
    const message = createMessage(MessageType.End, streamId);
    this.sendMessage(message);
  }

  /**
   * Send ping
   */
  private sendPing(): void {
    const message = createMessage(MessageType.Ping);
    this.sendMessage(message);
  }

  /**
   * Send pong
   */
  private sendPong(): void {
    const message = createMessage(MessageType.Pong);
    this.sendMessage(message);
  }

  /**
   * Get current tunnel information
   */
  getInfo(): TunnelInfo | null {
    return this.tunnelInfo;
  }

  /**
   * Check if tunnel is active
   */
  isActive(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }

  /**
   * Handle reconnection logic
   */
  private handleReconnect(): void {
    if (this.reconnectAttempts >= this.options.maxRetries) {
      this.log('error', 'Max reconnection attempts reached');
      this.emit('error', new Error('Max reconnection attempts reached'));
      return;
    }

    this.reconnectAttempts++;
    this.emit('reconnect', this.reconnectAttempts);
    this.emit('status', 'reconnecting');

    this.log('info', `Reconnecting... (attempt ${this.reconnectAttempts}/${this.options.maxRetries})`);

    this.reconnectTimer = setTimeout(() => {
      this.start().catch((err) => {
        this.log('error', `Reconnection failed: ${err.message}`);
        this.handleReconnect();
      });
    }, this.options.retryInterval);
  }

  /**
   * Internal logging
   */
  private log(level: string, message: string): void {
    const levels = ['debug', 'info', 'warn', 'error'];
    const currentLevel = levels.indexOf(this.options.logLevel);
    const messageLevel = levels.indexOf(level);

    if (messageLevel >= currentLevel) {
      console[level as 'debug' | 'info' | 'warn' | 'error']?.(`[TunGo] ${message}`);
    }
  }
}
