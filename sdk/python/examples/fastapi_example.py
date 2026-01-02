"""FastAPI example with TunGo tunnel."""

import asyncio

import uvicorn
from fastapi import FastAPI

from tungo import TunGoClient, TunGoEvents, TunGoOptions

app = FastAPI(title="TunGo FastAPI Example")


@app.get("/")
async def root():
    """Root endpoint."""
    return {
        "message": "Hello from TunGo!",
        "framework": "FastAPI",
    }


@app.get("/api/users")
async def users():
    """Get users."""
    return [
        {"id": 1, "name": "Alice", "email": "alice@example.com"},
        {"id": 2, "name": "Bob", "email": "bob@example.com"},
        {"id": 3, "name": "Charlie", "email": "charlie@example.com"},
    ]


@app.get("/api/status")
async def status():
    """Get server status."""
    return {
        "status": "ok",
        "tunnel": "active",
    }


async def start_tunnel():
    """Start TunGo tunnel."""

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
        local_port=8000,
        log_level="info",
    )

    client = TunGoClient(options, events)
    await client.start()

    # Keep tunnel running
    try:
        await asyncio.Event().wait()
    except asyncio.CancelledError:
        await client.stop()


if __name__ == "__main__":
    print("✅ FastAPI server starting on http://localhost:8000")
    print("📍 Routes:")
    print("   GET  /")
    print("   GET  /api/users")
    print("   GET  /api/status")
    print()

    # Create config for uvicorn with lifespan support
    config = uvicorn.Config(app, host="0.0.0.0", port=8000, log_level="info")
    server = uvicorn.Server(config)

    async def main():
        """Run tunnel and server together."""
        # Start tunnel and server concurrently
        await asyncio.gather(
            start_tunnel(),
            server.serve(),
        )

    # Run both tunnel and server
    asyncio.run(main())
