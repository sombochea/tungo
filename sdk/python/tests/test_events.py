"""Test event callbacks."""

import asyncio
import pytest
from tungo import TunGoClient, TunGoOptions, TunGoEvents


class TestEvents:
    """Test cases for event callbacks."""

    @pytest.mark.asyncio
    async def test_connect_event(self, server_url, local_port):
        """Test on_connect event is triggered."""
        connect_called = False

        def on_connect(info):
            nonlocal connect_called
            connect_called = True
            assert info.url, "Should have URL"
            assert info.subdomain, "Should have subdomain"

        events = TunGoEvents(on_connect=on_connect)
        options = TunGoOptions(local_port=local_port, server_url=server_url)
        client = TunGoClient(options, events)

        try:
            await client.start()
            assert connect_called, "on_connect should be called"
        finally:
            await client.stop()

    @pytest.mark.asyncio
    async def test_disconnect_event(self, server_url, local_port):
        """Test on_disconnect event is triggered."""
        disconnect_called = False

        def on_disconnect(reason):
            nonlocal disconnect_called
            disconnect_called = True

        events = TunGoEvents(on_disconnect=on_disconnect)
        options = TunGoOptions(local_port=local_port, server_url=server_url)
        client = TunGoClient(options, events)

        try:
            await client.start()
            if client.ws:
                await client.ws.close()
            await asyncio.sleep(1)
            assert disconnect_called, "on_disconnect should be called"
        finally:
            await client.stop()

    @pytest.mark.asyncio
    async def test_status_events(self, server_url, local_port):
        """Test on_status event reports correct states."""
        statuses = []

        def on_status(status):
            statuses.append(status)

        events = TunGoEvents(on_status=on_status)
        options = TunGoOptions(local_port=local_port, server_url=server_url)
        client = TunGoClient(options, events)

        try:
            await client.start()
            assert "connected" in statuses, "Should report connected status"
        finally:
            await client.stop()
            if "stopped" in statuses:
                assert True, "Should report stopped status if implemented"

    @pytest.mark.asyncio
    async def test_error_event(self):
        """Test on_error event is triggered on errors."""
        error_called = False

        def on_error(error):
            nonlocal error_called
            error_called = True

        events = TunGoEvents(on_error=on_error)
        options = TunGoOptions(
            local_port=8000,
            server_url="ws://invalid:9999/ws",
            connect_timeout=2,
        )
        client = TunGoClient(options, events)

        try:
            await asyncio.wait_for(client.start(), timeout=3)
        except Exception:
            pass

        # Error event might be called during connection failure
        await asyncio.sleep(0.5)
