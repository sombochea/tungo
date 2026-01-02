"""TunGo client implementation."""

import asyncio
import base64
import logging
from typing import Dict, Optional

import aiohttp
import websockets
from websockets.client import WebSocketClientProtocol

from tungo.protocol import (
    MessageType,
    ServerHelloType,
    create_client_hello,
    create_message,
    decode_message,
    encode_message,
)
from tungo.types import TunGoEvents, TunGoOptions, TunnelInfo

logger = logging.getLogger(__name__)


class TunGoClient:
    """TunGo Client - Expose your local server to the internet."""

    def __init__(self, options: TunGoOptions, events: Optional[TunGoEvents] = None):
        """Initialize TunGo client.

        Args:
            options: Client configuration options
            events: Optional event handlers
        """
        self.options = options
        self.events = events or TunGoEvents()
        self.ws: Optional[WebSocketClientProtocol] = None
        self.tunnel_info: Optional[TunnelInfo] = None
        self.reconnect_attempts = 0
        self.streams: Dict[str, asyncio.Task] = {}
        self.running = False
        self.ping_task: Optional[asyncio.Task] = None

        # Setup logging
        log_level = getattr(logging, options.log_level.upper())
        logging.basicConfig(
            level=log_level,
            format="[TunGo] %(message)s",
        )

    async def start(self) -> TunnelInfo:
        """Start the tunnel.

        Returns:
            TunnelInfo with the public URL and connection details

        Raises:
            Exception: If connection fails or times out
        """
        if self.ws and self.ws.close_code is None:
            raise RuntimeError("Tunnel is already running")

        # Build WebSocket URL
        if self.options.server_url:
            ws_url = self.options.server_url
            # Ensure URL starts with ws:// or wss://
            if not ws_url.startswith(("ws://", "wss://")):
                ws_url = f"ws://{ws_url}"
            # Ensure URL ends with /ws path
            if not ws_url.endswith("/ws"):
                ws_url = f"{ws_url}/ws" if ws_url.endswith("/") else f"{ws_url}/ws"
        else:
            ws_url = f"ws://{self.options.server_host}:{self.options.control_port}/ws"

        logger.info(f"Connecting to TunGo server: {ws_url}")

        try:
            # Connect with timeout
            self.ws = await asyncio.wait_for(
                websockets.connect(ws_url),
                timeout=self.options.connect_timeout,
            )
        except asyncio.TimeoutError:
            raise TimeoutError("Connection timeout")
        except Exception as e:
            raise Exception(f"Failed to connect: {e}")

        # Send client hello
        await self._send_client_hello()

        # Receive server hello
        try:
            message_str = await asyncio.wait_for(
                self.ws.recv(),
                timeout=self.options.connect_timeout,
            )
            server_hello = decode_message(message_str)
            self._handle_server_hello(server_hello)
        except asyncio.TimeoutError:
            await self.stop()
            raise TimeoutError("Server hello timeout")

        # Start message handler and ping task
        self.running = True
        asyncio.create_task(self._message_loop())
        self.ping_task = asyncio.create_task(self._ping_loop())

        logger.info(f"Tunnel established: {self.tunnel_info.url}")

        # Call connect callback
        if self.events.on_connect and self.tunnel_info:
            self.events.on_connect(self.tunnel_info)

        if self.events.on_status:
            self.events.on_status("connected")

        return self.tunnel_info

    async def stop(self) -> None:
        """Stop the tunnel."""
        self.running = False

        if self.ping_task:
            self.ping_task.cancel()
            try:
                await self.ping_task
            except asyncio.CancelledError:
                pass
            self.ping_task = None

        # Cancel all active streams
        for key, value in list(self.streams.items()):
            try:
                if isinstance(value, asyncio.Task):
                    value.cancel()
            except Exception:
                pass
        self.streams.clear()

        if self.ws:
            logger.info("Stopping tunnel...")
            await self.ws.close()
            self.ws = None
            self.tunnel_info = None
            self.reconnect_attempts = 0

            if self.events.on_status:
                self.events.on_status("stopped")

    def get_info(self) -> Optional[TunnelInfo]:
        """Get current tunnel information.

        Returns:
            TunnelInfo or None if not connected
        """
        return self.tunnel_info

    def is_active(self) -> bool:
        """Check if tunnel is active.

        Returns:
            True if tunnel is active, False otherwise
        """
        return self.ws is not None and self.ws.close_code is None

    async def _send_client_hello(self) -> None:
        """Send client hello message."""
        hello = create_client_hello(self.options.subdomain, self.options.secret_key)
        await self.ws.send(encode_message(hello))
        logger.debug("Sent client hello")

    def _handle_server_hello(self, hello: Dict) -> None:
        """Handle server hello message."""
        if hello.get("type") != ServerHelloType.SUCCESS.value:
            error = hello.get("error", f"Server hello failed: {hello.get('type')}")
            raise Exception(error)

        # Use public_url from server if available, otherwise construct from hostname
        public_url = hello.get("public_url")
        if not public_url:
            # Fallback for older servers
            public_url = f"http://{hello['hostname']}"

        self.tunnel_info = TunnelInfo(
            url=public_url,
            subdomain=hello["sub_domain"],
        )

        # Preserve subdomain for reconnection
        if hello.get("sub_domain"):
            self.options.subdomain = hello["sub_domain"]

    async def _message_loop(self) -> None:
        """Main message handling loop."""
        try:
            async for message_str in self.ws:
                try:
                    message = decode_message(message_str)
                    await self._handle_message(message)
                except Exception as e:
                    logger.error(f"Failed to handle message: {e}")
        except websockets.exceptions.ConnectionClosed:
            logger.info("WebSocket closed")
        except Exception as e:
            logger.error(f"Message loop error: {e}")
            if self.events.on_error:
                self.events.on_error(e)
        finally:
            if self.running and self.tunnel_info:
                if self.events.on_disconnect:
                    self.events.on_disconnect("Connection closed")
                await self._handle_reconnect()

    async def _handle_message(self, message: Dict) -> None:
        """Handle incoming protocol messages."""
        msg_type = message.get("type")

        if msg_type == MessageType.INIT.value:
            stream_id = message.get("stream_id")
            task = asyncio.create_task(self._handle_init_stream(stream_id, message.get("data")))
            self.streams[stream_id] = task

        elif msg_type == MessageType.DATA.value:
            await self._handle_stream_data(message.get("stream_id"), message.get("data"))

        elif msg_type == MessageType.END.value:
            await self._handle_stream_end(message.get("stream_id"))

        elif msg_type == MessageType.PING.value:
            await self._handle_ping()

        else:
            logger.debug(f"Unknown message type: {msg_type}")

    async def _handle_init_stream(self, stream_id: str, init_data: Dict) -> None:
        """Handle new stream initialization."""
        logger.debug(f"New stream: {stream_id}")

        try:
            # Create HTTP session for this stream
            async with aiohttp.ClientSession() as session:
                # Store session for receiving data
                self.streams[f"{stream_id}_session"] = session

        except Exception as e:
            logger.error(f"Stream {stream_id} error: {e}")
            await self._send_stream_end(stream_id)
            self.streams.pop(stream_id, None)

    async def _handle_stream_data(self, stream_id: str, data: Dict) -> None:
        """Handle stream data."""
        try:
            # Decode base64 data
            raw_data = base64.b64decode(data.get("data", ""))

            # Parse HTTP request
            http_str = raw_data.decode("utf-8", errors="ignore")
            lines = http_str.split("\r\n")

            if not lines:
                return

            request_line = lines[0].split(" ")
            if len(request_line) < 2:
                return

            method = request_line[0]
            path = request_line[1]

            # Parse headers
            headers = {}
            body_start = 0
            for i, line in enumerate(lines[1:], 1):
                if line == "":
                    body_start = http_str.find("\r\n\r\n") + 4
                    break
                if ":" in line:
                    key, value = line.split(":", 1)
                    headers[key.strip()] = value.strip()

            # Extract body
            body = raw_data[body_start:] if body_start < len(raw_data) else b""

            # Forward to local server
            url = f"http://{self.options.local_host}:{self.options.local_port}{path}"

            async with aiohttp.ClientSession() as session:
                async with session.request(
                    method=method,
                    url=url,
                    headers=headers,
                    data=body,
                    timeout=aiohttp.ClientTimeout(total=30),
                ) as response:
                    # Build HTTP response
                    response_lines = [f"HTTP/1.1 {response.status} {response.reason}"]

                    for key, value in response.headers.items():
                        response_lines.append(f"{key}: {value}")

                    response_lines.append("")
                    response_lines.append("")

                    response_header = "\r\n".join(response_lines).encode()
                    response_body = await response.read()

                    response_data = response_header + response_body

                # Send response back through tunnel (after response context is closed)
                await self._send_stream_data(stream_id, response_data)
                await self._send_stream_end(stream_id)

        except Exception as e:
            logger.error(f"Stream data error: {e}")
            await self._send_stream_end(stream_id)

    async def _handle_stream_end(self, stream_id: str) -> None:
        """Handle stream end."""
        self.streams.pop(stream_id, None)
        logger.debug(f"Stream ended: {stream_id}")

    async def _handle_ping(self) -> None:
        """Handle ping message."""
        await self._send_pong()

    async def _send_message(self, message: Dict) -> None:
        """Send a message through the WebSocket."""
        if self.ws and self.ws.close_code is None:
            try:
                await self.ws.send(encode_message(message))
            except Exception as e:
                logger.debug(f"Failed to send message: {e}")

    async def _send_stream_data(self, stream_id: str, data: bytes) -> None:
        """Send stream data."""
        message = create_message(
            MessageType.DATA,
            stream_id,
            {"data": base64.b64encode(data).decode()},
        )
        await self._send_message(message)

    async def _send_stream_end(self, stream_id: str) -> None:
        """Send stream end."""
        message = create_message(MessageType.END, stream_id)
        await self._send_message(message)

    async def _send_ping(self) -> None:
        """Send ping message."""
        message = create_message(MessageType.PING)
        await self._send_message(message)

    async def _send_pong(self) -> None:
        """Send pong message."""
        message = create_message(MessageType.PONG)
        await self._send_message(message)

    async def _ping_loop(self) -> None:
        """Periodic ping loop."""
        try:
            while self.running:
                await asyncio.sleep(30)
                if self.is_active():
                    await self._send_ping()
        except asyncio.CancelledError:
            pass

    async def _handle_reconnect(self) -> None:
        """Handle reconnection logic."""
        # Reset counter if max retries reached, but continue with longer delay
        if self.reconnect_attempts >= self.options.max_retries:
            logger.warning(
                f"Max retry attempts ({self.options.max_retries}) reached, continuing with extended delay..."
            )
            self.reconnect_attempts = 0
            # Use longer delay after max retries (e.g., 30 seconds)
            delay = min(self.options.retry_interval * 6, 30)
            await asyncio.sleep(delay)

        self.reconnect_attempts += 1

        if self.events.on_reconnect:
            self.events.on_reconnect(self.reconnect_attempts)

        if self.events.on_status:
            self.events.on_status("reconnecting")

        logger.info(
            f"Reconnecting... (attempt {self.reconnect_attempts}/{self.options.max_retries})"
        )

        await asyncio.sleep(self.options.retry_interval)

        # Clean up existing connection state before reconnecting
        if self.ws:
            try:
                await self.ws.close()
            except Exception:
                pass
            self.ws = None

        # Cancel ping task if exists
        if self.ping_task:
            self.ping_task.cancel()
            try:
                await self.ping_task
            except asyncio.CancelledError:
                pass
            self.ping_task = None

        # Clear streams
        for key, value in list(self.streams.items()):
            try:
                if isinstance(value, asyncio.Task):
                    value.cancel()
                # ClientSession objects are automatically closed by context manager
            except Exception:
                pass
        self.streams.clear()

        try:
            await self.start()
        except Exception as e:
            logger.error(f"Reconnection failed: {e}")
            await self._handle_reconnect()
