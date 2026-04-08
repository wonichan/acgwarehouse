# pyright: reportMissingImports=false

import hashlib
import os
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path
from typing import Callable, Sequence

from imagededup.methods import PHash
from PIL import Image


Image.MAX_IMAGE_PIXELS = None


_PHASHER = PHash(verbose=False)
HashResult = dict[str, str | None]


def default_max_workers() -> int:
    cpu_count = os.cpu_count() or 1
    return max(1, cpu_count * 2)


def compute_image_hashes(
    image_path: str,
    cached_sha256: str | None = None,
    cached_phash: str | None = None,
) -> HashResult:
    path = Path(image_path)
    result: HashResult = {
        "path": str(path),
        "sha256": None,
        "phash": None,
        "error": None,
    }

    try:
        if cached_sha256:
            result["sha256"] = cached_sha256
        else:
            sha256 = hashlib.sha256()
            with path.open("rb") as file:
                for chunk in iter(lambda: file.read(8192), b""):
                    sha256.update(chunk)
            result["sha256"] = sha256.hexdigest()

        if cached_phash:
            result["phash"] = cached_phash
        else:
            phash = _PHASHER.encode_image(image_file=str(path))
            if not isinstance(phash, str) or not phash:
                raise ValueError("failed to compute perceptual hash")
            result["phash"] = phash
    except Exception as error:
        result["sha256"] = None
        result["phash"] = None
        result["error"] = str(error)

    return result


def batch_compute_hashes(
    image_inputs: Sequence[str | dict[str, str | None]],
    max_workers: int | None = None,
    progress_callback: Callable[[float], None] | None = None,
) -> list[HashResult]:
    if not image_inputs:
        return []

    effective_max_workers = default_max_workers() if max_workers is None else max_workers
    cpu_count = os.cpu_count() or 1
    workers = max(1, min(cpu_count * 2, effective_max_workers))
    results: list[HashResult] = []
    total = len(image_inputs)

    normalized_inputs: list[dict[str, str | None]] = []
    for item in image_inputs:
        if isinstance(item, str):
            normalized_inputs.append({"path": item, "sha256": None, "phash": None})
            continue
        normalized_inputs.append(
            {
                "path": item.get("path") or "",
                "sha256": item.get("sha256"),
                "phash": item.get("phash"),
            }
        )

    with ThreadPoolExecutor(max_workers=workers) as executor:
        futures = {
            executor.submit(
                compute_image_hashes,
                item["path"] or "",
                item.get("sha256"),
                item.get("phash"),
            ): item
            for item in normalized_inputs
        }
        completed = 0
        for future in as_completed(futures):
            item = futures[future]
            try:
                result = future.result()
            except Exception as error:
                result = {
                    "path": str(Path(item.get("path") or "")),
                    "sha256": None,
                    "phash": None,
                    "error": str(error),
                }

            results.append(result)
            completed += 1

            if progress_callback is not None:
                progress_callback(round(completed / total * 100, 1))

    return results
