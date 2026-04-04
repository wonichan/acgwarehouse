from models.duplicates import (
    DetectRequest,
    DetectionResult,
    DuplicateGroup,
    GroupMember,
    ImageInput,
    TaskResponse,
)
from compute.task_state import (
    TaskStatus,
    complete_task,
    create_task,
    fail_task,
    get_task_state,
    update_progress,
)


def test_image_input_model_accepts_required_fields(tmp_path):
    image_path = tmp_path / "a.png"
    image_path.write_bytes(b"abc")

    payload = ImageInput(
        id=1,
        path=str(image_path),
        width=100,
        height=80,
        file_size=3,
        format="png",
    )

    assert payload.id == 1
    assert payload.path == str(image_path)
    assert payload.width == 100
    assert payload.height == 80
    assert payload.file_size == 3
    assert payload.format == "png"


def test_detect_request_defaults_threshold_and_images(tmp_path):
    image_path = tmp_path / "b.png"
    image_path.write_bytes(b"abc")
    image = ImageInput(
        id=1,
        path=str(image_path),
        width=200,
        height=100,
        file_size=3,
        format="png",
    )

    request = DetectRequest(images=[image])

    assert request.threshold == 40
    assert isinstance(request.images, list)
    assert isinstance(request.images[0], ImageInput)


def test_task_response_model_serialization():
    response = TaskResponse(
        task_id="task-1", status="pending", progress=12.5, message="ok"
    )

    serialized = response.model_dump()
    assert serialized["task_id"] == "task-1"
    assert serialized["status"] == "pending"
    assert serialized["progress"] == 12.5
    assert serialized["message"] == "ok"


def test_detection_result_model_shape():
    member = GroupMember(
        image_id=1,
        sha256="a" * 64,
        phash="b" * 64,
        distance=0,
        is_recommended=True,
        recommendation_score=88.8,
        recommendation_reasons=[
            {"factor": "resolution", "value": "10x10", "score": 1.0, "weight": 0.5}
        ],
    )
    group = DuplicateGroup(group_id=1, recommended_id=1, members=[member])
    result = DetectionResult(
        groups=[group],
        total_images=1,
        total_groups=1,
        skipped_images=[],
        computation_time_ms=23,
    )

    assert result.total_images == 1
    assert result.total_groups == 1
    assert len(result.groups) == 1
    assert result.groups[0].members[0].recommendation_reasons is not None


def test_create_task_returns_uuid_and_initial_state():
    task_id = create_task()
    state = get_task_state(task_id)

    assert isinstance(task_id, str)
    assert state is not None
    assert state.status == TaskStatus.PENDING
    assert state.progress == 0.0


def test_update_progress_changes_progress_and_message_atomically():
    task_id = create_task()

    update_progress(task_id, 42.0, "hashing")
    state = get_task_state(task_id)

    assert state is not None
    assert state.progress == 42.0
    assert state.message == "hashing"


def test_complete_task_sets_completed_status_and_result():
    task_id = create_task()

    complete_task(task_id, {"ok": True})
    state = get_task_state(task_id)

    assert state is not None
    assert state.status == TaskStatus.COMPLETED
    assert state.result == {"ok": True}


def test_fail_task_sets_failed_status_and_error():
    task_id = create_task()

    fail_task(task_id, "boom")
    state = get_task_state(task_id)

    assert state is not None
    assert state.status == TaskStatus.FAILED
    assert state.error == "boom"


def test_get_task_state_returns_none_for_unknown_task_id():
    assert get_task_state("missing-task") is None
