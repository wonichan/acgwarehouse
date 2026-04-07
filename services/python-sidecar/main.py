# pyright: reportMissingImports=false

import argparse
import atexit
import signal
import os
import sys
import threading

from fastapi import FastAPI
import uvicorn

import routers.duplicates as duplicates

app = FastAPI(title="ACGWarehouse Compute Sidecar")
_fallback_streams: list[object] = []


@app.get("/health")
async def health() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/shutdown", status_code=202)
async def shutdown() -> dict[str, str]:
    schedule_shutdown(1.0)
    return {"status": "shutting_down"}


def schedule_shutdown(delay_seconds: float = 1.0) -> None:
    timer = threading.Timer(delay_seconds, terminate_current_process)
    timer.daemon = True
    timer.start()


def terminate_current_process() -> None:
    os.kill(os.getpid(), signal.SIGTERM)


app.include_router(duplicates.router)


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=8000)
    return parser.parse_args(argv)


def ensure_standard_streams() -> None:
    if sys.stdout is None:
        stream = open(os.devnull, "w")
        _fallback_streams.append(stream)
        sys.stdout = stream
    if sys.stderr is None:
        stream = open(os.devnull, "w")
        _fallback_streams.append(stream)
        sys.stderr = stream


def close_fallback_streams() -> None:
    for stream in _fallback_streams:
        close = getattr(stream, "close", None)
        if callable(close):
            close()
    _fallback_streams.clear()


atexit.register(close_fallback_streams)


def main(argv: list[str] | None = None) -> None:
    args = parse_args(argv)
    ensure_standard_streams()
    uvicorn.run(app, host=args.host, port=args.port)


if __name__ == "__main__":
    main()
