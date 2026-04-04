# pyright: reportMissingImports=false

import threading
import time

from fastapi import APIRouter, HTTPException

from compute.grouping import group_duplicates, hamming_distance
from compute.hashing import batch_compute_hashes
from compute.scoring import compute_recommendation_score, select_recommended
from compute.task_state import (
    TaskStatus,
    complete_task,
    create_task,
    fail_task,
    get_task_state,
    start_task,
    update_progress,
)
from models.duplicates import (
    DetectRequest,
    DetectionResult,
    DuplicateGroup,
    GroupMember,
    TaskResponse,
)

router = APIRouter(prefix="/compute/duplicates")

_active_task_id: str | None = None
_active_lock = threading.Lock()


def _task_response(task_id: str) -> TaskResponse:
    state = get_task_state(task_id)
    if state is None:
        raise HTTPException(status_code=404, detail="Task not found")
    return TaskResponse(
        task_id=task_id,
        status=state.status.value,
        progress=state.progress,
        message=state.message or (state.error or ""),
    )


@router.post("/detect")
async def detect_duplicates(request: DetectRequest) -> TaskResponse:
    global _active_task_id

    with _active_lock:
        if _active_task_id:
            active_state = get_task_state(_active_task_id)
            if active_state and active_state.status in {
                TaskStatus.PENDING,
                TaskStatus.RUNNING,
            }:
                raise HTTPException(409, "Detection task already running")

        task_id = create_task()
        _active_task_id = task_id

    if not request.images:
        result = DetectionResult(
            groups=[],
            total_images=0,
            total_groups=0,
            skipped_images=[],
            computation_time_ms=0,
        )
        complete_task(task_id, result.model_dump())
        with _active_lock:
            if _active_task_id == task_id:
                _active_task_id = None
        return TaskResponse(
            task_id=task_id,
            status=TaskStatus.COMPLETED.value,
            progress=100.0,
            message="",
        )

    thread = threading.Thread(
        target=run_detection, args=(task_id, request), daemon=True
    )
    thread.start()
    return TaskResponse(
        task_id=task_id,
        status=TaskStatus.PENDING.value,
        progress=0.0,
        message="",
    )


@router.get("/tasks/{task_id}")
async def get_task(task_id: str) -> TaskResponse:
    return _task_response(task_id)


@router.get("/tasks/{task_id}/result")
async def get_task_result(task_id: str) -> DetectionResult:
    state = get_task_state(task_id)
    if state is None:
        raise HTTPException(status_code=404, detail="Task not found")
    if state.status != TaskStatus.COMPLETED:
        raise HTTPException(
            status_code=400, detail=f"Task not completed: {state.status.value}"
        )
    return DetectionResult.model_validate(state.result)


def run_detection(task_id: str, request: DetectRequest) -> None:
    global _active_task_id

    try:
        start_task(task_id, "hashing")
        started_at = time.time()

        path_to_input = {image.path: image for image in request.images}

        def progress_callback(percent: float) -> None:
            mapped = round((percent / 100.0) * 60.0, 1)
            update_progress(task_id, mapped, "hashing")

        hash_results = batch_compute_hashes(
            [image.path for image in request.images],
            progress_callback=progress_callback,
        )

        skipped_images: list[dict] = []
        hash_inputs: list[dict] = []
        for result in hash_results:
            if result.get("error"):
                skipped_images.append(
                    {"path": result["path"], "error": result["error"]}
                )
                continue

            image = path_to_input.get(result["path"])
            if image is None:
                continue

            hash_inputs.append(
                {
                    "image_id": image.id,
                    "path": image.path,
                    "width": image.width,
                    "height": image.height,
                    "file_size": image.file_size,
                    "format": image.format,
                    "sha256": result.get("sha256"),
                    "phash": result.get("phash"),
                }
            )

        update_progress(task_id, 70.0, "grouping")
        grouped = group_duplicates(hash_inputs, request.threshold)

        update_progress(task_id, 90.0, "scoring")
        output_groups: list[DuplicateGroup] = []
        for group in grouped:
            member_indices: list[int] = group["member_indices"]
            members_source = [hash_inputs[index] for index in member_indices]

            recommended_idx, _, _ = select_recommended(members_source)
            recommended_member = members_source[recommended_idx]

            members: list[GroupMember] = []
            for index, item in enumerate(members_source):
                score, reasons = compute_recommendation_score(
                    width=item["width"],
                    height=item["height"],
                    file_size=item["file_size"],
                    format=item["format"],
                )
                distance = 0
                if item.get("phash") and recommended_member.get("phash"):
                    distance = hamming_distance(
                        item["phash"], recommended_member["phash"]
                    )

                members.append(
                    GroupMember(
                        image_id=item["image_id"],
                        sha256=item.get("sha256"),
                        phash=item.get("phash"),
                        distance=distance,
                        is_recommended=index == recommended_idx,
                        recommendation_score=score,
                        recommendation_reasons=reasons,
                    )
                )

            output_groups.append(
                DuplicateGroup(
                    group_id=group["group_id"],
                    recommended_id=recommended_member["image_id"],
                    members=members,
                )
            )

        update_progress(task_id, 100.0, "completed")
        result = DetectionResult(
            groups=output_groups,
            total_images=len(request.images),
            total_groups=len(output_groups),
            skipped_images=skipped_images,
            computation_time_ms=int((time.time() - started_at) * 1000),
        )
        complete_task(task_id, result.model_dump())
    except Exception as error:
        fail_task(task_id, str(error))
    finally:
        with _active_lock:
            if _active_task_id == task_id:
                _active_task_id = None
