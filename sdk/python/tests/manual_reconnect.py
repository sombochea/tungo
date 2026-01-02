#!/usr/bin/env python3
"""Manual test for reconnection functionality.

This script helps manually test reconnection behavior:
1. Starts the tunnel client
2. Waits for connection
3. Instructs you to kill the server
4. Observes reconnection attempts with preserved subdomain
5. Shows continuous retry behavior after max attempts

Usage:
    uv run tests/manual_reconnect.py
"""

import asyncio
from tungo import TunGoClient, TunGoOptions, TunGoEvents


async def main():
    """Run manual reconnection test."""
    print("=" * 60)
    print("TunGo Reconnection Test")
    print("=" * 60)
    print()

    reconnect_count = 0
    disconnect_count = 0
    connect_count = 0
    original_subdomain = None

    def on_connect(info):
        nonlocal connect_count, original_subdomain
        connect_count += 1
        
        if original_subdomain is None:
            original_subdomain = info.subdomain
            print(f"✅ Connected (#{connect_count})")
            print(f"   URL: {info.url}")
            print(f"   Subdomain: {info.subdomain}")
        else:
            preserved = "✓" if info.subdomain == original_subdomain else "✗"
            print(f"✅ Reconnected (#{connect_count}) - Subdomain {preserved}")
            print(f"   URL: {info.url}")
            print(f"   Subdomain: {info.subdomain} (original: {original_subdomain})")

    def on_disconnect(reason):
        nonlocal disconnect_count
        disconnect_count += 1
        print(f"❌ Disconnected (#{disconnect_count}): {reason}")

    def on_reconnect(attempt):
        nonlocal reconnect_count
        reconnect_count = attempt
        print(f"🔄 Reconnecting... (attempt {attempt}/5)")

    def on_error(error):
        print(f"⚠️  Error: {error}")

    def on_status(status):
        print(f"📊 Status: {status}")

    events = TunGoEvents(
        on_connect=on_connect,
        on_disconnect=on_disconnect,
        on_reconnect=on_reconnect,
        on_error=on_error,
        on_status=on_status,
    )

    options = TunGoOptions(
        local_port=8000,
        server_url="ws://localhost:5555/ws",
        max_retries=5,
        retry_interval=3.0,
        log_level="info",
    )

    client = TunGoClient(options, events)

    try:
        print("Starting tunnel...")
        print()
        await client.start()
        
        print()
        print("=" * 60)
        print("Test Instructions:")
        print("=" * 60)
        print("1. The tunnel is now running")
        print("2. Kill the TunGo server (Ctrl+C in server terminal)")
        print("3. Watch the reconnection attempts")
        print("4. After 5 attempts, it will use extended delay")
        print("5. Restart the server to see successful reconnection")
        print("6. Verify the subdomain is preserved")
        print("7. Press Ctrl+C here to stop the test")
        print("=" * 60)
        print()

        # Keep running for testing
        while True:
            await asyncio.sleep(1)

    except KeyboardInterrupt:
        print("\n")
        print("=" * 60)
        print("Test Summary:")
        print("=" * 60)
        print(f"Total connections: {connect_count}")
        print(f"Total disconnections: {disconnect_count}")
        print(f"Last reconnect attempt: {reconnect_count}")
        print("=" * 60)
        print("\nStopping...")
        await client.stop()
    except Exception as e:
        print(f"\n❌ Error: {e}")
        await client.stop()


if __name__ == "__main__":
    asyncio.run(main())
