"""Type definitions for TunGo SDK."""

from dataclasses import dataclass
from typing import Callable, Literal, Optional

LogLevel = Literal["debug", "info", "warn", "error"]


@dataclass
class TunGoOptions:
    """TunGo client configuration options."""

    local_port: int
    """Local server port to tunnel (required)."""

    server_host: str = "localhost"
    """TunGo server host (default: localhost)."""

    control_port: int = 5555
    """Control port of the TunGo server (default: 5555)."""

    local_host: str = "localhost"
    """Local server host (default: localhost)."""

    subdomain: Optional[str] = None
    """Custom subdomain (optional, random if not provided)."""

    secret_key: Optional[str] = None
    """Authentication secret key (optional)."""

    connect_timeout: float = 10.0
    """Connection timeout in seconds (default: 10.0)."""

    max_retries: int = 5
    """Maximum reconnection attempts (default: 5)."""

    retry_interval: float = 5.0
    """Retry interval in seconds (default: 5.0)."""

    log_level: LogLevel = "info"
    """Log level: debug, info, warn, error (default: info)."""


@dataclass
class TunnelInfo:
    """Tunnel information returned after successful connection."""

    url: str
    """The public URL to access your local server."""

    subdomain: str
    """The assigned subdomain."""

    server_host: str
    """Server host."""

    server_port: int
    """Server port."""


@dataclass
class TunGoEvents:
    """Event handlers for TunGo client."""

    on_connect: Optional[Callable[[TunnelInfo], None]] = None
    """Called when tunnel is successfully established."""

    on_disconnect: Optional[Callable[[Optional[str]], None]] = None
    """Called when connection is lost."""

    on_error: Optional[Callable[[Exception], None]] = None
    """Called when an error occurs."""

    on_reconnect: Optional[Callable[[int], None]] = None
    """Called on reconnection attempt."""

    on_status: Optional[Callable[[str], None]] = None
    """Called on status updates."""
