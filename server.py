"""
Dumb mock LLM server.

What this DOES:
- Pretends to be an LLM inference server (like vLLM/SGLang would run).
- Simulates latency proportional to prompt length (i.e. pretends every request
  is a full "cold" prefill - no cache awareness yet).
- Adds random jitter so latency isn't a fixed number (real systems aren't
  deterministic, and you'll want this later for p99 measurements).
- Tracks a simple concurrent-request counter so the router can eventually
  ask "how loaded are you".

What this DOES NOT do (on purpose - this is your job to add):
- No concept of a "cache" - every request is treated as a cold miss.
- No eviction policy.
- No task specialization (code/chat/math).
- No simulated failures/timeouts.

Run one instance per "server". Example:
    python server.py --port 8001 --name server-A
    python server.py --port 8002 --name server-B
    python server.py --port 8003 --name server-C
"""

import argparse
import asyncio
import random
import time

from fastapi import FastAPI
from pydantic import BaseModel
import uvicorn

app = FastAPI()

# --- simple in-memory state for this one server instance ---
STATE = {
    "name": "unnamed-server",
    "active_requests": 0,
    "total_requests": 0,
}


class GenerateRequest(BaseModel):
    prompt: str
    request_id: str | None = None


class GenerateResponse(BaseModel):
    server_name: str
    request_id: str | None
    prompt_tokens: int
    latency_ms: float
    response_text: str


def fake_prefill_latency_ms(num_tokens: int) -> float:
    """
    Very rough stand-in for prefill cost: roughly linear in prompt length,
    plus lognormal jitter so it's not a perfectly straight line.

    This is deliberately crude. Once you add cache-awareness yourself,
    replace this with something that distinguishes "tokens that must be
    recomputed" vs "tokens already cached" (see fake_decode_latency idea
    in your own cache-state implementation).
    """
    base_ms_per_token = 4.0  # tune this - arbitrary for now
    base = num_tokens * base_ms_per_token
    jitter = random.lognormvariate(0, 0.25)# multiplicative noise
    return base * jitter


@app.post("/generate", response_model=GenerateResponse)
async def generate(req: GenerateRequest):
    STATE["active_requests"] += 1
    STATE["total_requests"] += 1
    try:
        num_tokens = len(req.prompt.split())  # crude word-count as token proxy
        latency_ms = fake_prefill_latency_ms(num_tokens)

        # extra queueing delay if this server is already busy -
        # crude stand-in for load-dependent slowdown
        load_penalty_ms = STATE["active_requests"] * 15
        total_latency_ms = latency_ms + load_penalty_ms

        await asyncio.sleep(total_latency_ms / 1000.0)

        return GenerateResponse(
            server_name=STATE["name"],
            request_id=req.request_id,
            prompt_tokens=num_tokens,
            latency_ms=total_latency_ms,
            response_text=f"[mock response from {STATE['name']}]",
        )
    finally:
        STATE["active_requests"] -= 1


@app.get("/health")
async def health():
    return {
        "status": "ok",
        "server_name": STATE["name"],
        "active_requests": STATE["active_requests"],
        "total_requests": STATE["total_requests"],
    }


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--port", type=int, required=True)
    parser.add_argument("--name", type=str, required=True)
    args = parser.parse_args()

    STATE["name"] = args.name
    uvicorn.run(app, host="0.0.0.0", port=args.port)