# pyright: reportMissingImports=false

import argparse
import os
import sys

from fastapi import FastAPI
import uvicorn

import routers.duplicates as duplicates

app = FastAPI(title="ACGWarehouse Compute Sidecar")


@app.get("/health")
async def health() -> dict[str, str]:
    return {"status": "ok"}


app.include_router(duplicates.router)


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=8000)
    return parser.parse_args(argv)


def ensure_standard_streams() -> None:
    if sys.stdout is None:
        sys.stdout = open(os.devnull, "w")
    if sys.stderr is None:
        sys.stderr = open(os.devnull, "w")


def main(argv: list[str] | None = None) -> None:
    args = parse_args(argv)
    ensure_standard_streams()
    uvicorn.run(app, host=args.host, port=args.port)


if __name__ == "__main__":
    main()
