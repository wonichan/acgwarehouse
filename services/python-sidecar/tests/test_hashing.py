"""Hashing module tests."""

# pyright: reportMissingImports=false

from pathlib import Path

from PIL import Image

from compute.hashing import batch_compute_hashes, compute_image_hashes, default_max_workers


def test_compute_image_hashes_returns_sha256_and_phash_for_valid_image(
    test_images_dir: Path,
):
    image_path = sorted(test_images_dir.glob("*.png"))[0]

    result = compute_image_hashes(str(image_path))

    assert result["path"] == str(image_path)
    assert result["error"] is None
    assert isinstance(result["sha256"], str)
    assert len(result["sha256"]) == 64
    assert isinstance(result["phash"], str)
    assert len(result["phash"]) == 16


def test_compute_image_hashes_returns_error_for_nonexistent_file():
    result = compute_image_hashes("D:/missing/not-exist.png")

    assert result["sha256"] is None
    assert result["phash"] is None
    assert result["error"] is not None


def test_compute_image_hashes_returns_error_for_corrupted_file(tmp_path: Path):
    corrupted = tmp_path / "broken.png"
    corrupted.write_bytes(b"not-an-image")

    result = compute_image_hashes(str(corrupted))

    assert result["sha256"] is None or isinstance(result["sha256"], str)
    assert result["phash"] is None
    assert result["error"] is not None


def test_batch_compute_hashes_processes_list_and_reports_progress(
    test_images_dir: Path,
):
    image_paths = [str(path) for path in sorted(test_images_dir.glob("*.png"))]
    progress_values: list[float] = []

    results = batch_compute_hashes(
        image_paths, progress_callback=progress_values.append
    )

    assert len(results) == 3
    assert all("path" in item for item in results)
    assert progress_values
    assert progress_values[-1] == 100.0


def test_batch_compute_hashes_skips_bad_files_and_continues(
    test_images_dir: Path, tmp_path: Path
):
    broken = tmp_path / "broken.dat"
    broken.write_bytes(b"broken")
    image_paths = [str(sorted(test_images_dir.glob("*.png"))[0]), str(broken)]

    results = batch_compute_hashes(image_paths)

    assert len(results) == 2
    by_path = {item["path"]: item for item in results}
    assert by_path[str(broken)]["error"] is not None
    assert by_path[str(sorted(test_images_dir.glob("*.png"))[0])]["error"] is None


def test_phash_is_exactly_16_hex_characters(test_images_dir: Path):
    image_path = sorted(test_images_dir.glob("*.png"))[0]

    result = compute_image_hashes(str(image_path))

    assert result["phash"] is not None
    assert len(result["phash"]) == 16


def test_hashing_module_disables_pillow_pixel_limit_for_trusted_images():
    assert Image.MAX_IMAGE_PIXELS is None


def test_compute_image_hashes_uses_full_cache_without_recompute(test_images_dir: Path):
    image_path = sorted(test_images_dir.glob("*.png"))[0]

    result = compute_image_hashes(
        str(image_path),
        cached_sha256="cached-sha",
        cached_phash="cached-phash",
    )

    assert result["error"] is None
    assert result["sha256"] == "cached-sha"
    assert result["phash"] == "cached-phash"


def test_compute_image_hashes_recomputes_only_missing_value(test_images_dir: Path):
    image_path = sorted(test_images_dir.glob("*.png"))[0]

    result = compute_image_hashes(str(image_path), cached_sha256="cached-sha")

    assert result["error"] is None
    assert result["sha256"] == "cached-sha"
    assert isinstance(result["phash"], str)
    assert len(result["phash"]) == 16


def test_batch_compute_hashes_accepts_input_dict_with_cache(test_images_dir: Path):
    image_path = sorted(test_images_dir.glob("*.png"))[0]

    results = batch_compute_hashes(
        [
            {
                "path": str(image_path),
                "sha256": "cached-sha",
                "phash": "cached-phash",
            }
        ]
    )

    assert len(results) == 1
    assert results[0]["sha256"] == "cached-sha"
    assert results[0]["phash"] == "cached-phash"


def test_default_max_workers_cpu_times_two(monkeypatch):
    monkeypatch.setattr("compute.hashing.os.cpu_count", lambda: 6)
    assert default_max_workers() == 12
