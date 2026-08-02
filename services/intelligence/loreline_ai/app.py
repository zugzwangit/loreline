from __future__ import annotations

import asyncio
import json
import time
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

import uvicorn
from fastapi import FastAPI, Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse, PlainTextResponse, StreamingResponse
from prometheus_client import CONTENT_TYPE_LATEST, Counter, Histogram, generate_latest

from .config import settings
from .models import RetrieveRequest
from .providers import build_provider

REQUESTS = Counter("loreline_ai_requests_total", "Intelligence requests", ["route", "status"])
LATENCY = Histogram("loreline_ai_request_seconds", "Intelligence latency", ["route"])


@asynccontextmanager
async def lifespan(app: FastAPI):
    app.state.provider = build_provider(settings())
    yield


app = FastAPI(title="Loreline Intelligence", version="1.1.0", docs_url=None, redoc_url=None, lifespan=lifespan)


@app.exception_handler(RequestValidationError)
async def validation_error(request: Request, exc: RequestValidationError) -> JSONResponse:
    REQUESTS.labels(request.url.path, "422").inc()
    return JSONResponse(
        status_code=422,
        content={
            "type": "https://docs.loreline.dev/problems/validation_error",
            "title": "Validation failed",
            "status": 422,
            "code": "validation_error",
            "detail": exc.errors(),
        },
    )


@app.get("/livez")
async def livez() -> dict[str, str]:
    return {"status": "ok", "service": "loreline-intelligence", "version": "1.1.0"}


@app.get("/readyz")
async def readyz(request: Request) -> dict[str, str]:
    if not getattr(request.app.state, "provider", None):
        return JSONResponse(status_code=503, content={"status": "not_ready"})
    return {"status": "ready"}


@app.get("/metrics")
async def metrics() -> PlainTextResponse:
    return PlainTextResponse(generate_latest().decode(), media_type=CONTENT_TYPE_LATEST)


@app.post("/v1/retrieve")
async def retrieve(payload: RetrieveRequest, request: Request):
    start = time.monotonic()
    route = "retrieve"
    try:
        answer = await request.app.state.provider.answer(payload.question, payload.documents)
        REQUESTS.labels(route, "200").inc()
        return answer
    except Exception:
        REQUESTS.labels(route, "500").inc()
        raise
    finally:
        LATENCY.labels(route).observe(time.monotonic() - start)


@app.post("/v1/stream")
async def stream(payload: RetrieveRequest, request: Request) -> StreamingResponse:
    answer = await request.app.state.provider.answer(payload.question, payload.documents)

    async def events() -> AsyncIterator[str]:
        yield (
            "event: meta\ndata: "
            + json.dumps({"grounded": answer.grounded, "confidence": answer.confidence, "provider": answer.provider})
            + "\n\n"
        )
        words = answer.answer.split()
        for index in range(0, len(words), 5):
            if await request.is_disconnected():
                return
            yield (
                "event: token\ndata: "
                + json.dumps({"text": " ".join(words[index : index + 5]) + (" " if index + 5 < len(words) else "")})
                + "\n\n"
            )
            await asyncio.sleep(0)
        yield "event: citations\ndata: " + json.dumps([c.model_dump() for c in answer.citations]) + "\n\n"
        yield "event: done\ndata: {}\n\n"

    REQUESTS.labels("stream", "200").inc()
    return StreamingResponse(
        events(), media_type="text/event-stream", headers={"Cache-Control": "no-cache", "X-Accel-Buffering": "no"}
    )


def main() -> None:
    host, port = settings().host_port
    uvicorn.run("loreline_ai.app:app", host=host, port=port, workers=1, proxy_headers=False, server_header=False)
