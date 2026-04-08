"""Duplicate router integration tests."""

# pyright: reportMissingImports=false, reportAttributeAccessIssue=false

import time
import logging

import pytest
from fastapi.testclient import TestClient

from main import app
from models.duplicates import DetectRequest


@pytest.fixture(autouse=True)
def wait_for_router_idle():
    from routers import duplicates as duplicates_router

    def wait_until_idle() -> None:
        for _ in range(500):
            task_id = duplicates_router._active_task_id
            if not task_id:
                break
            state = duplicates_router.get_task_state(task_id)
            if not state or state.status not in {
                duplicates_router.TaskStatus.PENDING,
                duplicates_router.TaskStatus.RUNNING,
            }:
                break
            time.sleep(0.05)

    wait_until_idle()
    yield
    wait_until_idle()


def test_post_detect_returns_task_id_and_pending(sample_image_inputs):
    client = TestClient(app)

    response = client.post(
        "/compute/duplicates/detect",
        json={
            "threshold": 40,
            "images": [item.model_dump() for item in sample_image_inputs],
        },
    )

    assert response.status_code == 200
    payload = response.json()
    assert payload["task_id"]
    assert payload["status"] == "pending"


def test_post_detect_while_running_returns_409(monkeypatch, sample_image_inputs):
    from routers import duplicates as duplicates_router

    original_run_detection = duplicates_router.run_detection

    def blocked_run_detection(task_id, request):
        duplicates_router.start_task(task_id, "blocked")
        duplicates_router.update_progress(task_id, 1.0, "blocked")
        time.sleep(0.2)
        duplicates_router.fail_task(task_id, "blocked")
        with duplicates_router._active_lock:
            if duplicates_router._active_task_id == task_id:
                duplicates_router._active_task_id = None

    monkeypatch.setattr(duplicates_router, "run_detection", blocked_run_detection)
    client = TestClient(app)

    first = client.post(
        "/compute/duplicates/detect",
        json={
            "threshold": 40,
            "images": [item.model_dump() for item in sample_image_inputs],
        },
    )
    second = client.post(
        "/compute/duplicates/detect",
        json={
            "threshold": 40,
            "images": [item.model_dump() for item in sample_image_inputs],
        },
    )

    monkeypatch.setattr(duplicates_router, "run_detection", original_run_detection)

    assert first.status_code == 200
    assert second.status_code == 409


def test_post_detect_with_empty_images_returns_immediate_completed_task():
    client = TestClient(app)

    response = client.post(
        "/compute/duplicates/detect", json={"threshold": 40, "images": []}
    )

    assert response.status_code == 200
    task_id = response.json()["task_id"]

    result_response = client.get(f"/compute/duplicates/tasks/{task_id}/result")
    assert result_response.status_code == 200
    result_payload = result_response.json()
    assert result_payload["total_images"] == 0
    assert result_payload["total_groups"] == 0


def test_get_task_status_returns_current_status_and_progress(sample_image_inputs):
    client = TestClient(app)
    create = client.post(
        "/compute/duplicates/detect",
        json={
            "threshold": 40,
            "images": [item.model_dump() for item in sample_image_inputs],
        },
    )
    task_id = create.json()["task_id"]

    status_response = client.get(f"/compute/duplicates/tasks/{task_id}")

    assert status_response.status_code == 200
    payload = status_response.json()
    assert payload["task_id"] == task_id
    assert "status" in payload
    assert "progress" in payload


def test_get_unknown_task_status_returns_404():
    client = TestClient(app)

    response = client.get("/compute/duplicates/tasks/not-found")

    assert response.status_code == 404


def test_get_result_for_non_completed_task_returns_400(
    monkeypatch, sample_image_inputs
):
    from routers import duplicates as duplicates_router

    original_run_detection = duplicates_router.run_detection

    def blocked_run_detection(task_id, request):
        duplicates_router.start_task(task_id, "waiting")
        duplicates_router.update_progress(task_id, 10.0, "waiting")
        time.sleep(0.2)
        duplicates_router.fail_task(task_id, "waiting")
        with duplicates_router._active_lock:
            if duplicates_router._active_task_id == task_id:
                duplicates_router._active_task_id = None

    monkeypatch.setattr(duplicates_router, "run_detection", blocked_run_detection)
    client = TestClient(app)

    create = client.post(
        "/compute/duplicates/detect",
        json={
            "threshold": 40,
            "images": [item.model_dump() for item in sample_image_inputs],
        },
    )
    task_id = create.json()["task_id"]

    response = client.get(f"/compute/duplicates/tasks/{task_id}/result")
    monkeypatch.setattr(duplicates_router, "run_detection", original_run_detection)

    assert response.status_code == 400


def test_full_flow_submit_poll_fetch_result(sample_image_inputs):
    client = TestClient(app)

    create = client.post(
        "/compute/duplicates/detect",
        json={
            "threshold": 40,
            "images": [item.model_dump() for item in sample_image_inputs],
        },
    )
    assert create.status_code == 200
    task_id = create.json()["task_id"]

    for _ in range(500):
        state = client.get(f"/compute/duplicates/tasks/{task_id}")
        assert state.status_code == 200
        if state.json()["status"] == "completed":
            break
        time.sleep(0.05)

    result = client.get(f"/compute/duplicates/tasks/{task_id}/result")
    assert result.status_code == 200

    payload = result.json()
    assert "groups" in payload
    if payload["groups"]:
        first_member = payload["groups"][0]["members"][0]
        assert "recommendation_reasons" in first_member


def test_detection_with_test_images_returns_group_structure(test_images_dir):
    first = sorted(test_images_dir.glob("*.png"))[0]
    duplicate = test_images_dir / "red-copy.png"
    duplicate.write_bytes(first.read_bytes())

    images = [
        {
            "id": 1,
            "path": str(first),
            "width": 10,
            "height": 10,
            "file_size": first.stat().st_size,
            "format": "png",
        },
        {
            "id": 2,
            "path": str(duplicate),
            "width": 10,
            "height": 10,
            "file_size": duplicate.stat().st_size,
            "format": "png",
        },
    ]

    client = TestClient(app)
    create = client.post(
        "/compute/duplicates/detect", json={"threshold": 40, "images": images}
    )
    task_id = create.json()["task_id"]

    for _ in range(500):
        state = client.get(f"/compute/duplicates/tasks/{task_id}")
        if state.json()["status"] == "completed":
            break
        time.sleep(0.05)

    result = client.get(f"/compute/duplicates/tasks/{task_id}/result")
    assert result.status_code == 200
    payload = result.json()
    assert payload["total_groups"] >= 1
    assert payload["groups"][0]["recommended_id"] in [1, 2]


def test_run_detection_logs_lifecycle_and_progress(
    monkeypatch, sample_image_inputs, caplog
):
    from routers import duplicates as duplicates_router

    task_id = duplicates_router.create_task()
    with duplicates_router._active_lock:
        duplicates_router._active_task_id = task_id

    def fake_batch_compute_hashes(image_paths, progress_callback=None, max_workers=4):
        assert progress_callback is not None
        progress_callback(20.0)
        progress_callback(50.0)
        progress_callback(100.0)
        return [
            {
                "path": sample_image_inputs[0].path,
                "sha256": "sha-a",
                "phash": "0123456789abcdef",
                "error": None,
            },
            {
                "path": sample_image_inputs[1].path,
                "sha256": None,
                "phash": None,
                "error": "broken image",
            },
        ]

    monkeypatch.setattr(
        duplicates_router, "batch_compute_hashes", fake_batch_compute_hashes
    )
    monkeypatch.setattr(
        duplicates_router,
        "group_duplicates",
        lambda hash_inputs, threshold: [{"group_id": 1, "member_indices": [0]}],
    )
    monkeypatch.setattr(
        duplicates_router, "select_recommended", lambda members_source: (0, None, None)
    )
    monkeypatch.setattr(
        duplicates_router,
        "compute_recommendation_score",
        lambda **kwargs: (88.0, []),
    )

    request = DetectRequest(threshold=40, images=sample_image_inputs[:2])
    with caplog.at_level(logging.INFO, logger="uvicorn.error"):
        duplicates_router.run_detection(task_id, request)

    text = caplog.text
    for want in [
        f"duplicate detection task started: task_id={task_id} image_count=2 threshold=40",
        f"duplicate detection hashing progress: task_id={task_id}",
        f"duplicate detection stage: task_id={task_id} stage=grouping progress=70.0",
        f"duplicate detection stage: task_id={task_id} stage=scoring progress=90.0",
        f"duplicate detection completed: task_id={task_id} total_images=2 total_groups=1 skipped_images=1",
    ]:
        assert want in text


def test_run_detection_logs_failures(monkeypatch, sample_image_inputs, caplog):
    from routers import duplicates as duplicates_router

    task_id = duplicates_router.create_task()
    with duplicates_router._active_lock:
        duplicates_router._active_task_id = task_id

    def fake_batch_compute_hashes(image_paths, progress_callback=None, max_workers=4):
        raise RuntimeError("hashing exploded")

    monkeypatch.setattr(
        duplicates_router, "batch_compute_hashes", fake_batch_compute_hashes
    )

    request = DetectRequest(threshold=40, images=sample_image_inputs[:1])
    with caplog.at_level(logging.INFO, logger="uvicorn.error"):
        duplicates_router.run_detection(task_id, request)

    text = caplog.text
    assert (
        f"duplicate detection task started: task_id={task_id} image_count=1 threshold=40"
        in text
    )
    assert (
        f"duplicate detection failed: task_id={task_id} error=hashing exploded" in text
    )


def test_run_detection_passes_cached_hash_fields_to_batch_compute(
    monkeypatch, sample_image_inputs
):
    from routers import duplicates as duplicates_router

    task_id = duplicates_router.create_task()
    with duplicates_router._active_lock:
        duplicates_router._active_task_id = task_id

    sample_image_inputs[0].sha256 = "cached-sha"
    sample_image_inputs[0].phash = "cached-phash"

    captured_inputs = []

    def fake_batch_compute_hashes(image_inputs, progress_callback=None, max_workers=None):
        nonlocal captured_inputs
        captured_inputs = image_inputs
        if progress_callback is not None:
            progress_callback(100.0)
        return [
            {
                "path": sample_image_inputs[0].path,
                "sha256": "computed-or-cached-sha",
                "phash": "computed-or-cached-phash",
                "error": None,
            }
        ]

    monkeypatch.setattr(
        duplicates_router, "batch_compute_hashes", fake_batch_compute_hashes
    )
    monkeypatch.setattr(
        duplicates_router,
        "group_duplicates",
        lambda hash_inputs, threshold: [{"group_id": 1, "member_indices": [0]}],
    )
    monkeypatch.setattr(
        duplicates_router, "select_recommended", lambda members_source: (0, None, None)
    )
    monkeypatch.setattr(
        duplicates_router,
        "compute_recommendation_score",
        lambda **kwargs: (88.0, []),
    )

    request = DetectRequest(threshold=40, images=[sample_image_inputs[0]])
    duplicates_router.run_detection(task_id, request)

    assert len(captured_inputs) == 1
    assert captured_inputs[0]["path"] == sample_image_inputs[0].path
    assert captured_inputs[0]["sha256"] == "cached-sha"
    assert captured_inputs[0]["phash"] == "cached-phash"
