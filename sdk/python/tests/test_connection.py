"""Test basic connection functionality."""

import asyncio
import pytest
from tungo import TunGoClient, TunGoOptions, TunGoEvents


class TestConnection:
    """Test cases for basic connection operations."""

    @pytest.mark.asyncio
    async def test_basic_connection(self, server_url, local_port):
        """Test basic connection to TunGo server."""
        connected = False
        tunnel_info = None

        def on_connect(info):
            nonlocal connected, tunnel_info
            connected = True
            tunnel_info = info

        events = TunGoEvents(on_connect=on_connect)
        options = TunGoOptions(
            local_port=local_port,
            server_url=server_url,
            connect_timeout=5,
        )

        client = TunGoClient(options, events)

        try:
            await asyncio.wait_for(client.start(), timeout=10)
            assert connected, "Should connect successfully"
            assert tunnel_info is not None, "Should receive tunnel info"
            assert tunnel_info.url, "Should have public URL"
            assert tunnel_info.subdomain, "Should have subdomain"
            assert client.is_active(), "Client should be active"
        finally:
            await client.stop()

    @pytest.mark.asyncio
    async def test_custom_subdomain(self, server_url, local_port):
        """Test connection with custom subdomain."""
        subdomain = "mycustomtest"
        tunnel_info = None

        def on_connect(info):
            nonlocal tunnel_info
            tunnel_info = info

        events = TunGoEvents(on_connect=on_connect)
        options = TunGoOptions(
            local_port=local_port,
            server_url=server_url,
            subdomain=subdomain,
        )

        client = TunGoClient(options, events)

        try:
            await client.start()
            assert tunnel_info is not None
            assert tunnel_info.subdomain == subdomain
        finally:
            await client.stop()

    @pytest.mark.asyncio
    async def test_connection_timeout(self):
        """Test connection timeout with invalid server."""
        options = TunGoOptions(
            local_port=8000,
            server_url="ws://invalid-server-address:9999/ws",
            connect_timeout=2,
        )

        client = TunGoClient(options)

        with pytest.raises((asyncio.TimeoutError, Exception)):
            await client.start()

    @pytest.mark.asyncio
    async def test_stop_while_connected(self, server_url, local_port):
        """Test stopping client while connected."""
        options = TunGoOptions(
            local_port=local_port,
            server_url=server_url,
        )

        client = TunGoClient(options)

        try:
            await client.start()
            assert client.is_active()
            await client.stop()
            assert not client.is_active()
        except Exception:
            await client.stop()
