"""TunGo protocol implementation."""

import json
import uuid
from dataclasses import dataclass, asdict
from enum import Enum
from typing import Any, Dict, Optional


class MessageType(str, Enum):
    """Message types in the TunGo protocol."""

    HELLO = "hello"
    SERVER_HELLO = "server_hello"
    INIT = "init"
    DATA = "data"
    END = "end"
    PING = "ping"
    PONG = "pong"


class ServerHelloType(str, Enum):
    """Server hello response types."""

    SUCCESS = "success"
    SUBDOMAIN_IN_USE = "sub_domain_in_use"
    INVALID_SUBDOMAIN = "invalid_sub_domain"
    AUTH_FAILED = "auth_failed"
    ERROR = "error"


class ClientType(str, Enum):
    """Client type."""

    AUTH = "auth"
    ANONYMOUS = "anonymous"


@dataclass
class SecretKey:
    """Secret key for authentication."""

    key: str


@dataclass
class ClientHello:
    """Client hello message."""

    id: str
    client_type: str
    sub_domain: Optional[str] = None
    secret_key: Optional[Dict[str, str]] = None
    reconnect_token: Optional[Dict[str, str]] = None


@dataclass
class ServerHello:
    """Server hello message."""

    type: str
    sub_domain: Optional[str] = None
    hostname: Optional[str] = None
    client_id: Optional[str] = None
    reconnect_token: Optional[Dict[str, str]] = None
    error: Optional[str] = None


@dataclass
class Message:
    """Protocol message."""

    type: str
    stream_id: Optional[str] = None
    data: Optional[Any] = None


@dataclass
class InitStreamMessage:
    """Init stream message."""

    stream_id: str
    protocol: str


def create_client_hello(
    subdomain: Optional[str] = None, secret_key: Optional[str] = None
) -> Dict[str, Any]:
    """Create a new client hello message."""
    client_id = str(uuid.uuid4())
    client_type = ClientType.AUTH if secret_key else ClientType.ANONYMOUS

    hello: Dict[str, Any] = {
        "id": client_id,
        "client_type": client_type.value,
    }

    if subdomain:
        hello["sub_domain"] = subdomain

    if secret_key:
        hello["secret_key"] = {"key": secret_key}

    return hello


def create_message(
    msg_type: MessageType,
    stream_id: Optional[str] = None,
    data: Optional[Any] = None,
) -> Dict[str, Any]:
    """Create a protocol message."""
    message: Dict[str, Any] = {"type": msg_type.value}

    if stream_id:
        message["stream_id"] = stream_id

    if data is not None:
        message["data"] = data

    return message


def encode_message(message: Dict[str, Any]) -> str:
    """Encode message to JSON."""
    return json.dumps(message)


def decode_message(data: str) -> Dict[str, Any]:
    """Decode message from JSON."""
    return json.loads(data)


def generate_stream_id() -> str:
    """Generate a random stream ID."""
    return str(uuid.uuid4())
