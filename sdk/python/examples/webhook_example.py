"""Webhook receiver example with TunGo tunnel."""

import asyncio
from datetime import datetime

import uvicorn
from fastapi import FastAPI, Request

from tungo import TunGoClient, TunGoEvents, TunGoOptions

app = FastAPI(title="Webhook Receiver")


@app.post("/webhook")
async def webhook(request: Request):
    """Receive webhook."""
    body = await request.json()

    print("\n📨 Webhook received:")
    print("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    print(f"Event: {body.get('event', 'unknown')}")
    print(f"Timestamp: {body.get('timestamp', datetime.now().isoformat())}")
    print(f"Headers: {dict(request.headers)}")
    print(f"Body: {body}")
    print("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

    return {
        "success": True,
        "received": datetime.now().isoformat(),
        "event": body.get("event", "unknown"),
    }


@app.get("/")
async def root():
    """Root endpoint with instructions."""
    return {
        "title": "🎣 Webhook Server",
        "description": "Your webhook receiver is running!",
        "endpoints": [
            "POST /webhook - Receive webhooks",
            "GET / - This page",
        ],
    }


async def start_tunnel():
    """Start TunGo tunnel."""

    def on_connect(info):
        print(f"\n🌍 Webhook URL: {info.url}/webhook")
        print(f"🔗 Web UI: {info.url}")
        print("\n📝 Configure this URL in your webhook provider!")
        print("\n💡 Test command:")
        print(
            f"   curl -X POST {info.url}/webhook -H 'Content-Type: application/json' "
            f'-d \'{{"event":"test","data":"hello"}}\''
        )
        print()

    events = TunGoEvents(on_connect=on_connect)

    options = TunGoOptions(
        local_port=4000,
        subdomain="my-webhooks",
    )

    client = TunGoClient(options, events)
    await client.start()

    # Keep running
    try:
        await asyncio.Event().wait()
    except asyncio.CancelledError:
        await client.stop()


if __name__ == "__main__":
    print("✅ Webhook server starting on http://localhost:4000")
    print()

    # Start tunnel in background
    asyncio.create_task(start_tunnel())

    # Start server
    uvicorn.run(app, host="0.0.0.0", port=4000, log_level="info")
