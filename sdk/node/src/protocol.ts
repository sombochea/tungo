import { v4 as uuidv4 } from './uuid.js';

/**
 * Message types in the TunGo protocol
 */
export enum MessageType {
  Hello = 'hello',
  ServerHello = 'server_hello',
  Init = 'init',
  Data = 'data',
  End = 'end',
  Ping = 'ping',
  Pong = 'pong',
}

/**
 * Server hello response types
 */
export enum ServerHelloType {
  Success = 'success',
  SubDomainInUse = 'sub_domain_in_use',
  InvalidSubDomain = 'invalid_sub_domain',
  AuthFailed = 'auth_failed',
  Error = 'error',
}

/**
 * Client type
 */
export enum ClientType {
  Auth = 'auth',
  Anonymous = 'anonymous',
}

/**
 * Client hello message
 */
export interface ClientHello {
  id: string;
  sub_domain?: string;
  client_type: ClientType;
  secret_key?: { key: string };
  reconnect_token?: { token: string };
}

/**
 * Server hello message
 */
export interface ServerHello {
  type: ServerHelloType;
  sub_domain?: string;
  hostname?: string;
  client_id?: string;
  reconnect_token?: { token: string };
  error?: string;
}

/**
 * Protocol message
 */
export interface Message {
  type: MessageType;
  stream_id?: string;
  data?: any;
}

/**
 * Init stream message
 */
export interface InitStreamMessage {
  stream_id: string;
  protocol: string;
}

/**
 * Create a new client hello message
 */
export function createClientHello(
  subdomain?: string,
  secretKey?: string
): ClientHello {
  const hello: ClientHello = {
    id: uuidv4(),
    client_type: secretKey ? ClientType.Auth : ClientType.Anonymous,
  };

  if (subdomain) {
    hello.sub_domain = subdomain;
  }

  if (secretKey) {
    hello.secret_key = { key: secretKey };
  }

  return hello;
}

/**
 * Create a protocol message
 */
export function createMessage(
  type: MessageType,
  streamId?: string,
  data?: any
): Message {
  const message: Message = { type };

  if (streamId) {
    message.stream_id = streamId;
  }

  if (data !== undefined) {
    message.data = data;
  }

  return message;
}

/**
 * Encode message to JSON
 */
export function encodeMessage(message: Message): string {
  return JSON.stringify(message);
}

/**
 * Decode message from JSON
 */
export function decodeMessage(data: string): Message {
  return JSON.parse(data);
}

/**
 * Generate a random stream ID
 */
export function generateStreamId(): string {
  return uuidv4();
}
