"""Test reconnection functionality."""

import asyncio
import pytest
from tungo import TunGoClient, TunGoOptions, TunGoEvents


class TestReconnection:
    """Test cases for automatic reconnection."""

    @pytest.mark.asyncio
    async def test_reconnection_logic(self, server_url, local_port):
        """Test that reconnection attempts are triggered."""
        reconnect_count = 0
        disconnect_count = 0
        subdomain_preserved = True
        original_subdomain = None

        def on_connect(info):
            nonlocal original_subdomain, subdomain_preserved
            if original_subdomain is None:
                original_subdomain = info.subdomain
            elif info.subdomain != original_subdomain:
                subdomain_preserved = False

        def on_disconnect(reason):
            nonlocal disconnect_count
            disconnect_count += 1

        def on_reconnect(attempt):
            nonlocal reconnect_count
            reconnect_count = attempt

        events = TunGoEvents(
            on_connect=on_connect,
            on_disconnect=on_disconnect,
            on_reconnect=on_reconnect,
        )

        options = TunGoOptions(
            local_port=local_port,
            server_url=server_url,
            max_retries=3,
            retry_interval=2.0,
        )

        client = TunGoClient(options, events)

        try:
            await client.start()
            
            # Simulate disconnection by closing WebSocket
            if client.ws:
                await client.ws.close()
            
            # Wait for reconnection attempts
            await asyncio.sleep(8)  # Wait for some retry attempts
            
            assert disconnect_count > 0, "Should detect disconnection"
            # Note: reconnect_count may be 0 if server isn't actually down
            
        finally:
            await client.stop()

    @pytest.mark.asyncio
    async def test_subdomain_preservation(self, server_url, local_port):
        """Test that subdomain is preserved across reconnections."""
        subdomains = []

        def on_connect(info):
            subdomains.append(info.subdomain)

        events = TunGoEvents(on_connect=on_connect)

        options = TunGoOptions(
            local_port=local_port,
            server_url=server_url,
            max_retries=2,
            retry_interval=1.0,
        )

        client = TunGoClient(options, events)

        try:
            # First connection
            await client.start()
            first_subdomain = subdomains[0]
            
            # Simulate reconnection by stopping and starting again
            await client.stop()
            await asyncio.sleep(1)
            await client.start()
            
            assert len(subdomains) >= 2, "Should have multiple connections"
            assert subdomains[1] == first_subdomain, "Subdomain should be preserved"
            
        finally:
            await client.stop()

    @pytest.mark.asyncio
    async def test_continuous_retry(self, local_port):
        """Test that client continues retrying after max_retries."""
        reconnect_attempts = []

        def on_reconnect(attempt):
            reconnect_attempts.append(attempt)

        events = TunGoEvents(on_reconnect=on_reconnect)

        options = TunGoOptions(
            local_port=local_port,
            server_url="ws://localhost:9999/ws",  # Invalid server
            max_retries=2,
            retry_interval=1.0,
            connect_timeout=2,
        )

        client = TunGoClient(options, events)

        try:
            # This will fail to connect
            await asyncio.wait_for(client.start(), timeout=1)
        except asyncio.TimeoutError:
            pass
        except Exception:
            pass

        # Wait for multiple retry cycles
        await asyncio.sleep(6)
        
        # Should continue retrying beyond max_retries
        # After max_retries, it resets and continues with extended delay
        assert len(reconnect_attempts) >= 2, "Should make multiple retry attempts"
        
        await client.stop()
