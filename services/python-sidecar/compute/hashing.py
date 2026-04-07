# pyright: reportMissingImports=false

import hashlib
import os
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path
from typing import Callable

from imagededup.methods import PHash


_PHASHER = PHash(verbose=False)
HashResult = dict[str, str | None]


def compute_image_hashes(image_path: str) -> HashResult:
    path = Path(image_path)
    result: HashResult = {
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
    image_paths: list[str],
    max_workers: int = 4,
    progress_callback: Callable[[float], None] | None = None,
) -> list[HashResult]:
    if not image_paths:
        return []

    cpu_count = os.cpu_count() or 1
    workers = max(1, min(cpu_count, max_workers))
    results: list[HashResult] = []
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
