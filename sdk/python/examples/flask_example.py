"""Flask example with TunGo tunnel."""

import asyncio
from threading import Thread

from flask import Flask, jsonify

from tungo import TunGoClient, TunGoEvents, TunGoOptions

app = Flask(__name__)


@app.route("/")
def root():
    """Root endpoint."""
    return jsonify(
        {
            "message": "Hello from TunGo!",
            "framework": "Flask",
        }
    )


@app.route("/api/users")
def users():
    """Get users."""
    return jsonify(
        [
            {"id": 1, "name": "Alice", "email": "alice@example.com"},
            {"id": 2, "name": "Bob", "email": "bob@example.com"},
            {"id": 3, "name": "Charlie", "email": "charlie@example.com"},
        ]
    )


@app.route("/api/status")
def status():
    """Get server status."""
    return jsonify(
        {
            "status": "ok",
            "tunnel": "active",
        }
    )


def start_tunnel():
    """Start TunGo tunnel in background thread."""

    async def _start():
        def on_connect(info):
            print(f"\n🌍 Public URL: {info.url}")
            print(f"📊 Subdomain: {info.subdomain}")
            print("\n✨ Your local server is now accessible from anywhere!\n")

        def on_error(error):
            print(f"❌ Tunnel error: {error}")

        events = TunGoEvents(
            on_connect=on_connect,
            on_error=on_error,
        )

        options = TunGoOptions(
            local_port=5000,
            log_level="info",
        )

        client = TunGoClient(options, events)
        await client.start()

        # Keep running
        try:
            await asyncio.Event().wait()
        except (KeyboardInterrupt, asyncio.CancelledError):
            await client.stop()

    asyncio.run(_start())


if __name__ == "__main__":
    print("✅ Flask server starting on http://localhost:5000")
    print("📍 Routes:")
    print("   GET  /")
    print("   GET  /api/users")
    print("   GET  /api/status")
    print()

    # Start tunnel in background thread
    tunnel_thread = Thread(target=start_tunnel, daemon=True)
    tunnel_thread.start()

    # Start Flask server
    app.run(host="0.0.0.0", port=5000, debug=False)
