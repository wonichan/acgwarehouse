import copy
import threading
import uuid
from dataclasses import dataclass
from enum import Enum
from typing import Any


class TaskStatus(str, Enum):
    PENDING = "pending"
    RUNNING = "running"
    COMPLETED = "completed"
    FAILED = "failed"


@dataclass
class TaskState:
    status: TaskStatus = TaskStatus.PENDING
    progress: float = 0.0
    message: str = ""
    result: Any = None
    error: str | None = None


_tasks: dict[str, TaskState] = {}
_task_lock = threading.Lock()
_MAX_TERMINAL_TASKS = 1


def _cleanup_old_tasks_unlocked() -> None:
    historical = [
        task_id
        for task_id, state in _tasks.items()
        if state.status in {TaskStatus.COMPLETED, TaskStatus.FAILED}
    ]
    while len(historical) > _MAX_TERMINAL_TASKS:
        stale_task_id = historical.pop(0)
        _tasks.pop(stale_task_id, None)


def create_task() -> str:
    task_id = str(uuid.uuid4())
    with _task_lock:
        _cleanup_old_tasks_unlocked()
        _tasks[task_id] = TaskState()
    return task_id


def update_progress(task_id: str, progress: float, message: str = "") -> None:
    with _task_lock:
        state = _tasks.get(task_id)
        if state is None:
            return
        state.progress = max(0.0, min(100.0, progress))
        state.message = message


def start_task(task_id: str, message: str = "") -> None:
    with _task_lock:
        state = _tasks.get(task_id)
        if state is None:
            return
        state.status = TaskStatus.RUNNING
        state.message = message


def get_task_state(task_id: str) -> TaskState | None:
    with _task_lock:
        state = _tasks.get(task_id)
        if state is None:
            return None
        return copy.deepcopy(state)


def complete_task(task_id: str, result: Any) -> None:
    with _task_lock:
        state = _tasks.get(task_id)
        if state is None:
            return
        state.status = TaskStatus.COMPLETED
        state.progress = 100.0
        state.result = result
        state.error = None
        _cleanup_old_tasks_unlocked()


def fail_task(task_id: str, error: str) -> None:
    with _task_lock:
        state = _tasks.get(task_id)
        if state is None:
            return
        state.status = TaskStatus.FAILED
        state.message = ""
        state.error = error
        _cleanup_old_tasks_unlocked()
