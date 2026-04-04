import hashlib
import os
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

import imagehash
from PIL import Image


def compute_image_hashes(image_path: str) -> dict:
    path = Path(image_path)
    result = {
        "path": str(path),
        "sha256": None,
        "phash": None,
        "error": None,
    }

    try:
        sha256 = hashlib.sha256()
        with path.open("rb") as file:
            for chunk in iter(lambda: file.read(8192), b""):
                sha256.update(chunk)
        result["sha256"] = sha256.hexdigest()

        with Image.open(path) as img:
            phash = imagehash.phash(img, hash_size=16)
        result["phash"] = str(phash)
    except Exception as error:
        result["sha256"] = None
        result["phash"] = None
        result["error"] = str(error)

    return result


def batch_compute_hashes(
    image_paths: list[str],
    max_workers: int = 4,
    progress_callback=None,
) -> list[dict]:
    if not image_paths:
        return []

    cpu_count = os.cpu_count() or 1
    workers = max(1, min(cpu_count, max_workers))
    results: list[dict] = []
    total = len(image_paths)

    with ThreadPoolExecutor(max_workers=workers) as executor:
        futures = {
            executor.submit(compute_image_hashes, path): path for path in image_paths
        }
        completed = 0
        for future in as_completed(futures):
            path = futures[future]
            try:
                result = future.result()
            except Exception as error:
                result = {
                    "path": str(Path(path)),
                    "sha256": None,
                    "phash": None,
                    "error": str(error),
                }

            results.append(result)
            completed += 1

            if progress_callback is not None:
                progress_callback(round(completed / total * 100, 1))

    return results
