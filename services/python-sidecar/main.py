# pyright: reportMissingImports=false

from fastapi import FastAPI

import routers.duplicates as duplicates

app = FastAPI(title="ACGWarehouse Compute Sidecar")


@app.get("/health")
async def health() -> dict[str, str]:
    return {"status": "ok"}


app.include_router(duplicates.router)


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="127.0.0.1", port=8000)
