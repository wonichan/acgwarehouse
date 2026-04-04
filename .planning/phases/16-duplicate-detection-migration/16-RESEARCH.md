# Phase 16: Duplicate Detection Migration - Research

**Researched:** 2026-04-04
**Domain:** Python image hashing, async task orchestration, Go↔Python HTTP contract, multi-dimensional scoring
**Confidence:** HIGH

## Summary

Phase 16 migrates all duplicate detection computation (SHA256, pHash, hamming distance, Union-Find grouping, recommendation selection) from Go to the Python sidecar established in Phase 15. The migration upgrades pHash from 64-bit (`hash_size=8`) to 256-bit (`hash_size=16`), replaces the single-dimension recommendation strategy (resolution only) with multi-dimensional weighted scoring, and introduces an async task pattern for long-running computation.

The Python `imagehash` library (v4.3.2, BSD-2-Clause) is the established standard for perceptual hashing in Python. It natively supports `hash_size=16` producing a 16×16=256-bit hash stored as a 64-character hex string. The `ImageHash` object supports subtraction (`hash1 - hash2`) for hamming distance, making the migration straightforward. The key architectural challenge is designing the async task pattern (submit → poll → fetch results) without heavyweight dependencies (no Celery/Redis) since this is a local desktop application.

**Primary recommendation:** Use FastAPI with in-memory task state (Python `dict` + `threading.Thread` for background computation), `imagehash.phash(img, hash_size=16)` for 256-bit pHash, store hashes as 64-char hex strings in SQLite `TEXT` column, and implement Union-Find + multi-dimensional scoring entirely in Python.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** 全计算迁移——Python 承接全部计算步骤：SHA256 文件哈希计算、pHash 感知哈希计算、汉明距离比较、Union-Find 传递性分组、推荐保留选取。
- **D-02:** Go 侧删除现有 `hash_service.go` 中的哈希计算逻辑和 `duplicate_service.go` 中的计算/分组逻辑，仅保留落库、API 层和查询逻辑。
- **D-03:** 哈希算法升级到高精度 pHash（Python `imagehash` 库，hash_size=16，256-bit），替代原有 64-bit pHash。与历史数据不兼容，首次运行时全量重新计算所有图片的新 pHash。
- **D-04:** `domain.Image` 中的 `PHash int64` 字段需要适配新的 256-bit 哈希值存储（字符串或更大的数值表示）。
- **D-05:** 采用异步任务模式——Go 发启动请求，Python 后台处理并返回任务 ID，Go 轮询进度（百分比），处理完成后 Go 获取结果。
- **D-06:** Go 向 Python 传递图片的本地文件系统绝对路径列表 + 每张图片的元数据（分辨率、文件大小、格式等），Python 直接读取本地文件计算哈希。
- **D-07:** Python 返回完整的分组结果，包含每组的推荐保留项、推荐依据（结构化数据）和每个成员的哈希值/距离信息。Go 拿到结果后直接落库。
- **D-08:** 具体的 HTTP 端点设计、JSON schema 和进度轮询频率由工程师在实现时确定，遵循 Phase 15 已建立的纯计算批量契约模式（D-11/D-13）。
- **D-09:** 从单一分辨率排序升级为多维度评分——包含但不限于分辨率、文件大小、格式优先级等维度，按加权综合评分选取推荐项。
- **D-10:** 推荐依据以结构化数据返回，如 `{"reasons": [{"factor": "分辨率", "value": "1920x1080", "weight": 0.5}, ...], "score": 85}`，前端可根据结构化数据自由展示。
- **D-11:** 具体的评分维度、权重值由工程师在实现时确定。
- **D-12:** Python 不可用时，重复检测直接返回明确的错误状态（如"计算服务不可用，请检查 Python 侧车状态"），不尝试本地计算回退。符合 ROADMAP 成功标准第 3 条（可诊断的失败状态）。
- **D-13:** 采用前置检查机制——在触发检测前先通过 Phase 15 已有的 sidecar runtime 状态检查确认 Python 就绪，未就绪则立即拒绝并告知原因，不让用户等到超时。
- **D-14:** 图库的主浏览流程不受重复检测服务不可用影响（延续 Phase 15 D-10 的降级可用语义）。

### Agent's Discretion
- 具体的 HTTP 端点路径和 JSON schema 设计
- 评分维度的具体权重分配
- 进度轮询频率和超时配置
- Python 侧新 pHash 全量重算的并发控制策略
- 256-bit pHash 在 SQLite 中的具体存储方案（hex string / blob 等）
- 异步任务的取消和清理机制

### Deferred Ideas (OUT OF SCOPE)
- Python 不可用时自动降级到 Go 本地后备计算路径 — COMP-05 / Phase 22 范围
- 增量检测优化（只计算新增图片而非每次全量） — 可作为性能优化在 Phase 22 或后续迭代
- 前端重复检测结果 UI 重构 — Phase 17+ 桌面体验重构范围
- 远程/云端 Python 侧车部署支持 — 超出当前本地单机形态
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| COMP-03 | 系统会把重复检测、图像哈希和相似度计算从现有 Go 直接计算迁移到 Python 侧车执行 | imagehash library API, FastAPI async task pattern, Union-Find implementation, SHA256 in Python hashlib |
| COMP-04 | 用户在重复检测结果中可以看到每组图片的推荐保留项与推荐依据 | Multi-dimensional scoring algorithm, structured rationale JSON schema |
</phase_requirements>

## Standard Stack

### Core (Python Sidecar)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| imagehash | 4.3.2 | Perceptual hash computation (pHash, aHash, dHash) | 3.8k GitHub stars, BSD-2-Clause, mature library, direct `hash_size=16` support |
| Pillow | 11.3.0 | Image loading for hash computation | Standard Python imaging library, required by imagehash |
| scipy | 1.17.1 | DCT computation for pHash | Required by imagehash for `scipy.fftpack.dct` |
| numpy | 2.3.2 | Array operations for hash computation | Required by imagehash for bit array operations |
| FastAPI | 0.115.14 | HTTP API framework for sidecar endpoints | Already established in Phase 15, high performance async |
| uvicorn | 0.35.0 | ASGI server | Already established in Phase 15 |
| pydantic | 2.11.7 | Request/response model validation | Built into FastAPI, ensures type safety |
| hashlib | stdlib | SHA256 file hash computation | Python standard library, no external dependency |

### Core (Go Side)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| net/http | stdlib | HTTP client for Python sidecar communication | Standard library, already used in sidecar runtime |
| encoding/json | stdlib | JSON serialization/deserialization | Standard library |
| database/sql | stdlib | SQLite database operations | Already used throughout codebase |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| In-memory task state | Celery + Redis | Massive overkill for local desktop app — adds 2 new services to manage |
| imagehash | OpenCV img_hash | OpenCV is a heavy dependency (>100MB); imagehash is lightweight (~50KB) and sufficient |
| FastAPI BackgroundTasks | threading.Thread | BackgroundTasks runs AFTER response is sent — not suitable for long-running tasks with progress tracking. Use threading.Thread directly |
| Hex string storage | BLOB storage | Hex string is human-readable, debuggable, and imagehash natively produces hex via `str()` |

**Installation:**
```bash
pip install imagehash fastapi uvicorn pydantic
```

**Version verification:** All versions verified against the development environment on 2026-04-04:
- imagehash 4.3.2 (would install, not yet installed — requires scipy 1.17.1 and PyWavelets 1.9.0)
- Pillow 11.3.0 ✓ (already installed)
- FastAPI 0.115.14 ✓ (already installed)
- uvicorn 0.35.0 ✓ (already installed)
- numpy 2.3.2 ✓ (already installed)
- pydantic 2.11.7 ✓ (already installed)

## Architecture Patterns

### Recommended Project Structure
```
services/
└── python-sidecar/          # Python compute sidecar (Phase 15 established)
    ├── main.py              # FastAPI app entry + health endpoints
    ├── routers/
    │   └── duplicates.py    # Duplicate detection endpoints
    ├── compute/
    │   ├── hashing.py       # SHA256 + pHash computation
    │   ├── grouping.py      # Union-Find + hamming distance grouping
    │   └── scoring.py       # Multi-dimensional recommendation scoring
    ├── models/
    │   └── duplicates.py    # Pydantic request/response models
    └── requirements.txt     # Python dependencies
```

### Pattern 1: Async Task with In-Memory State
**What:** Python sidecar manages long-running tasks using a simple in-memory dict + background thread, without external task queues.
**When to use:** Local desktop application with single concurrent user, no need for task persistence across restarts.

```python
# Source: Verified pattern from FastAPI community + real-world implementations
import threading
import uuid
from enum import Enum
from typing import Optional, Any

class TaskStatus(str, Enum):
    PENDING = "pending"
    RUNNING = "running"
    COMPLETED = "completed"
    FAILED = "failed"

class TaskState:
    def __init__(self):
        self.status: TaskStatus = TaskStatus.PENDING
        self.progress: float = 0.0  # 0-100
        self.message: str = ""
        self.result: Optional[Any] = None
        self.error: Optional[str] = None

# Global task registry (single-process, single-user desktop app)
_tasks: dict[str, TaskState] = {}
_task_lock = threading.Lock()

def create_task() -> str:
    task_id = str(uuid.uuid4())
    with _task_lock:
        _tasks[task_id] = TaskState()
    return task_id

def update_progress(task_id: str, progress: float, message: str = ""):
    with _task_lock:
        if task_id in _tasks:
            _tasks[task_id].progress = progress
            _tasks[task_id].message = message

def get_task_state(task_id: str) -> Optional[TaskState]:
    with _task_lock:
        return _tasks.get(task_id)
```

### Pattern 2: 256-bit pHash Computation with imagehash
**What:** Use `imagehash.phash(img, hash_size=16)` to produce a 256-bit perceptual hash, stored and compared as hex strings.
**When to use:** When upgrading from 64-bit pHash for higher accuracy in duplicate detection.

```python
# Source: imagehash source code (verified from GitHub raw)
from PIL import Image
import imagehash
import hashlib

def compute_phash_256(image_path: str) -> str:
    """Compute 256-bit pHash, returns 64-char hex string."""
    img = Image.open(image_path)
    phash = imagehash.phash(img, hash_size=16)  # 16x16 = 256 bits
    return str(phash)  # Returns 64-char hex string

def compute_sha256(file_path: str) -> str:
    """Compute SHA256 file hash, returns 64-char hex string."""
    sha256 = hashlib.sha256()
    with open(file_path, "rb") as f:
        for chunk in iter(lambda: f.read(8192), b""):
            sha256.update(chunk)
    return sha256.hexdigest()

def hamming_distance(hash1: str, hash2: str) -> int:
    """Compute hamming distance between two hex hash strings."""
    h1 = imagehash.hex_to_hash(hash1)
    h2 = imagehash.hex_to_hash(hash2)
    return h1 - h2  # ImageHash __sub__ returns hamming distance
```

**Key insight from source code:** The `imagehash.phash()` function:
1. Resizes image to `hash_size * highfreq_factor` (16 × 4 = 64×64 pixels)
2. Converts to grayscale
3. Applies 2D DCT via `scipy.fftpack.dct`
4. Takes low-frequency `hash_size × hash_size` coefficients
5. Compares each to median → produces boolean array
6. `str(hash)` converts to hex via `_binary_array_to_hex` — for hash_size=16, produces 64 hex chars

### Pattern 3: Go Async Task Client
**What:** Go service acts as task orchestrator — submits, polls progress, fetches results.
**When to use:** When Go needs to delegate long-running computation to Python sidecar.

```go
// Async task orchestration in Go
type DuplicateDetectionTask struct {
    TaskID   string  `json:"task_id"`
    Status   string  `json:"status"`    // pending/running/completed/failed
    Progress float64 `json:"progress"`  // 0-100
    Message  string  `json:"message"`
}

// Submit detection task
func (c *SidecarClient) SubmitDetection(ctx context.Context, req DetectionRequest) (string, error) {
    resp, err := c.post(ctx, "/compute/duplicates/detect", req)
    // returns task_id
}

// Poll progress
func (c *SidecarClient) PollProgress(ctx context.Context, taskID string) (*DuplicateDetectionTask, error) {
    resp, err := c.get(ctx, "/compute/duplicates/tasks/"+taskID)
    // returns current status + progress
}

// Fetch results (only when status == "completed")
func (c *SidecarClient) FetchResults(ctx context.Context, taskID string) (*DetectionResult, error) {
    resp, err := c.get(ctx, "/compute/duplicates/tasks/"+taskID+"/result")
    // returns full grouped results
}
```

### Anti-Patterns to Avoid
- **Streaming results during computation:** Don't try to stream partial groups back to Go during computation. Complete all grouping first, then return full result. Partial results create complex state management.
- **Storing task state in Python's FastAPI BackgroundTasks:** `BackgroundTasks` executes AFTER the response is sent but provides no progress tracking. Use `threading.Thread` with shared state instead.
- **Computing hamming distance on hex strings character-by-character:** Use `imagehash.hex_to_hash()` to reconstruct `ImageHash` objects, then subtract. The library handles bit-level comparison correctly.
- **Using Python multiprocessing for parallelism:** For local desktop use with ~10k images, `concurrent.futures.ThreadPoolExecutor` is sufficient since the GIL is released during I/O (PIL image loading). Multiprocessing adds process spawn overhead and IPC complexity on Windows.
- **Storing 256-bit pHash as INTEGER in SQLite:** 256 bits cannot fit in SQLite's 64-bit INTEGER. Use TEXT column with hex string.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| pHash computation | Custom DCT + median comparison | `imagehash.phash(img, hash_size=16)` | Proven implementation, handles edge cases (rotation, resize, color conversion) |
| Hamming distance | Manual XOR + popcount on hex strings | `hash1 - hash2` (ImageHash subtraction) | Library handles arbitrary hash sizes correctly, avoids bit manipulation bugs |
| SHA256 | Custom hash implementation | `hashlib.sha256()` | Python stdlib, C-optimized, no dependencies |
| Union-Find | Find a library | Implement inline (30 lines) | Too simple for a dependency; path compression + union by rank is canonical ~30 lines |
| Image format detection | Parse file headers | `PIL.Image.open(path).format` | Pillow handles all formats (JPEG, PNG, WebP, GIF, BMP) |
| Hex ↔ Hash conversion | Custom hex parser | `imagehash.hex_to_hash()` / `str(hash)` | Handles hash_size inference from hex length correctly |

**Key insight:** imagehash is a thin wrapper (~300 lines of core code) around numpy/scipy/PIL. It doesn't add much overhead but handles critical edge cases (image resizing, grayscale conversion, DCT normalization) that are easy to get wrong.

## Common Pitfalls

### Pitfall 1: 256-bit pHash Threshold Incompatibility
**What goes wrong:** Using the same hamming distance threshold (10) for 256-bit hashes as was used for 64-bit hashes. With 256 bits, a threshold of 10 is very strict — it means only ~4% of bits can differ.
**Why it happens:** The threshold doesn't scale linearly with hash size. 64-bit with threshold 10 ≈ 15.6% difference; 256-bit with threshold 10 ≈ 3.9% difference.
**How to avoid:** Scale the threshold proportionally. For equivalent sensitivity: `new_threshold = old_threshold * (256/64) = 40`. Recommend starting with threshold 30-40 for 256-bit pHash and tuning based on real image testing.
**Warning signs:** Very few or no duplicate groups found after migration despite known duplicates existing.

### Pitfall 2: PIL Image Opening Failures on Corrupted/Unsupported Files
**What goes wrong:** `Image.open(path)` or `image.convert('L')` throws an exception for corrupted, truncated, or unsupported image files, crashing the entire batch.
**Why it happens:** Real image libraries contain corrupted downloads, partially written files, or exotic formats.
**How to avoid:** Wrap each image hash computation in try/except, log the error, skip the image, and continue. Report skipped images in the progress/result.
**Warning signs:** Entire detection task fails with a PIL/Pillow exception traceback.

### Pitfall 3: O(n²) Pairwise Comparison Scaling
**What goes wrong:** The current Go implementation does O(n²) pairwise hamming distance comparison. For 10,000 images, that's ~50 million comparisons. For 50,000 images, ~1.25 billion.
**Why it happens:** Brute-force all-pairs comparison is the simplest correct algorithm for Union-Find grouping.
**How to avoid:** For the current scope (Phase 16 is full recalculation each run), O(n²) is acceptable for <20k images (~200M comparisons, feasible in minutes with numpy). For larger scales, consider VP-trees or LSH — but that's a Phase 22+ optimization (deferred). **Do track and report computation time in the result so future optimization can be data-driven.**
**Warning signs:** Detection task takes >10 minutes for the user's image library.

### Pitfall 4: Windows File Path Encoding
**What goes wrong:** Python receives file paths from Go with backslashes or non-ASCII characters that don't decode correctly, causing `FileNotFoundError`.
**Why it happens:** Go's `filepath.Abs()` returns backslash paths on Windows. JSON transmission may have encoding issues with CJK characters in paths.
**How to avoid:** Always use `pathlib.Path` in Python for path handling. Ensure JSON uses UTF-8 encoding. Validate file existence before computing hash.
**Warning signs:** Hash computation fails for images in directories with Chinese/Japanese characters.

### Pitfall 5: Concurrent Task State Corruption
**What goes wrong:** Multiple detection requests overlap, corrupting shared task state or producing incorrect results.
**Why it happens:** No mutex or single-task enforcement on the detection endpoint.
**How to avoid:** Enforce at most one active detection task at a time. Return HTTP 409 Conflict if a detection task is already running. The Go side should also prevent double-submission.
**Warning signs:** Inconsistent progress reporting, garbled results.

### Pitfall 6: Memory Pressure from Batch Image Loading
**What goes wrong:** Loading all images into memory simultaneously for hash computation causes OOM on machines with limited RAM.
**Why it happens:** PIL `Image.open()` defers loading, but `image.convert('L').resize()` materializes the full image data. With hash_size=16, each image is resized to 64×64 grayscale (4KB), but the original decode may temporarily use much more.
**How to avoid:** Process images one at a time sequentially, or in small batches (e.g., 50 at a time). Close/discard PIL Image objects after hashing. Don't keep all Image objects in memory.
**Warning signs:** Python process memory usage grows to several GB during detection.

### Pitfall 7: SQLite Schema Migration for PHash Column Type Change
**What goes wrong:** Changing `images.phash` from `INTEGER` to `TEXT` breaks existing queries and Go code that reads `int64`.
**Why it happens:** SQLite is dynamically typed, but Go's `sql.Scan` expects specific types. Existing `COALESCE(phash, 0)` returns integer 0 which won't work with text hashes.
**How to avoid:** Add a new `phash_hex TEXT` column (or rename the existing one) rather than changing the type in-place. Update all Go queries that reference phash. The old `phash INTEGER` column can be kept for backward compatibility or dropped after migration.
**Warning signs:** `sql.Scan` errors in Go when reading phash values after migration.

## Code Examples

### Example 1: Complete 256-bit pHash Batch Computation
```python
# Source: imagehash source code + verified API
from PIL import Image
import imagehash
import hashlib
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor, as_completed

def compute_image_hashes(
    image_path: str,
) -> dict:
    """Compute both SHA256 and 256-bit pHash for a single image."""
    path = Path(image_path)
    result = {
        "path": image_path,
        "sha256": None,
        "phash": None,
        "error": None,
    }

    try:
        # SHA256 file hash
        sha256 = hashlib.sha256()
        with open(path, "rb") as f:
            for chunk in iter(lambda: f.read(8192), b""):
                sha256.update(chunk)
        result["sha256"] = sha256.hexdigest()

        # 256-bit pHash
        img = Image.open(path)
        phash = imagehash.phash(img, hash_size=16)
        result["phash"] = str(phash)  # 64-char hex string
        img.close()

    except Exception as e:
        result["error"] = str(e)

    return result


def batch_compute_hashes(
    image_paths: list[str],
    max_workers: int = 4,
    progress_callback=None,
) -> list[dict]:
    """Compute hashes for multiple images with progress tracking."""
    results = []
    total = len(image_paths)

    # Use ThreadPoolExecutor — GIL released during PIL I/O
    with ThreadPoolExecutor(max_workers=max_workers) as executor:
        futures = {
            executor.submit(compute_image_hashes, path): i
            for i, path in enumerate(image_paths)
        }

        for future in as_completed(futures):
            idx = futures[future]
            try:
                result = future.result()
            except Exception as e:
                result = {"path": image_paths[idx], "error": str(e)}
            results.append(result)

            if progress_callback:
                progress_callback(len(results) / total * 100)

    return results
```

### Example 2: Union-Find with Path Compression in Python
```python
# Source: Standard algorithm, verified against multiple implementations
class UnionFind:
    """Union-Find with path compression and union by rank."""

    def __init__(self, n: int):
        self.parent = list(range(n))
        self.rank = [0] * n

    def find(self, x: int) -> int:
        if self.parent[x] != x:
            self.parent[x] = self.find(self.parent[x])  # path compression
        return self.parent[x]

    def union(self, x: int, y: int) -> None:
        px, py = self.find(x), self.find(y)
        if px == py:
            return
        # union by rank
        if self.rank[px] < self.rank[py]:
            px, py = py, px
        self.parent[py] = px
        if self.rank[px] == self.rank[py]:
            self.rank[px] += 1

    def groups(self) -> dict[int, list[int]]:
        """Returns groups with >1 member. Keys are root indices."""
        from collections import defaultdict
        group_map = defaultdict(list)
        for i in range(len(self.parent)):
            group_map[self.find(i)].append(i)
        return {root: members for root, members in group_map.items()
                if len(members) > 1}
```

### Example 3: Multi-Dimensional Recommendation Scoring
```python
# Source: Research-based design for image quality scoring
from dataclasses import dataclass

FORMAT_PRIORITY = {
    "png": 1.0,     # Lossless, highest quality
    "webp": 0.9,    # Good compression with quality
    "jpeg": 0.7,    # Lossy but universal
    "jpg": 0.7,     # Same as jpeg
    "gif": 0.5,     # Limited color depth
    "bmp": 0.6,     # Uncompressed, but no quality advantage over PNG
}

@dataclass
class ScoringFactor:
    factor: str
    value: str
    score: float   # Normalized 0-100 for this factor
    weight: float  # Weight in composite score

def compute_recommendation_score(
    width: int,
    height: int,
    file_size: int,
    format: str,
) -> tuple[float, list[dict]]:
    """
    Compute composite quality score and structured rationale.
    Returns (score: 0-100, reasons: list of factor dicts).
    """
    reasons = []

    # Factor 1: Resolution (weight 0.5)
    resolution = width * height
    # Normalize: 4K (3840*2160=8.3M px) = 100, 0 = 0
    res_score = min(100.0, (resolution / 8_294_400) * 100)
    reasons.append({
        "factor": "resolution",
        "value": f"{width}x{height}",
        "score": round(res_score, 1),
        "weight": 0.5,
    })

    # Factor 2: File size (weight 0.3) — larger generally means more detail
    # Normalize: 10MB = 100, 0 = 0
    size_score = min(100.0, (file_size / 10_485_760) * 100)
    reasons.append({
        "factor": "file_size",
        "value": f"{file_size}",
        "score": round(size_score, 1),
        "weight": 0.3,
    })

    # Factor 3: Format preference (weight 0.2)
    fmt_lower = format.lower() if format else ""
    format_score = FORMAT_PRIORITY.get(fmt_lower, 0.5) * 100
    reasons.append({
        "factor": "format",
        "value": format or "unknown",
        "score": round(format_score, 1),
        "weight": 0.2,
    })

    # Composite weighted score
    composite = sum(r["score"] * r["weight"] for r in reasons)

    return round(composite, 1), reasons
```

### Example 4: FastAPI Async Task Endpoints
```python
# Source: FastAPI patterns + community best practices
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel
import threading
import uuid

router = APIRouter(prefix="/compute/duplicates")

class DetectRequest(BaseModel):
    threshold: int = 40  # Hamming distance threshold (256-bit scale)
    images: list[dict]   # [{id, path, width, height, file_size, format}, ...]

class TaskResponse(BaseModel):
    task_id: str
    status: str
    progress: float = 0.0
    message: str = ""

class DetectionResult(BaseModel):
    groups: list[dict]   # Full grouped results
    total_images: int
    total_groups: int
    skipped_images: list[dict]
    computation_time_ms: int

# Single active task enforcement
_active_task_id: str | None = None
_active_lock = threading.Lock()

@router.post("/detect")
async def start_detection(request: DetectRequest) -> TaskResponse:
    global _active_task_id
    with _active_lock:
        if _active_task_id and get_task_state(_active_task_id).status == "running":
            raise HTTPException(409, "Detection task already running")
        task_id = create_task()
        _active_task_id = task_id

    # Start background thread
    thread = threading.Thread(
        target=run_detection,
        args=(task_id, request),
        daemon=True,
    )
    thread.start()

    return TaskResponse(task_id=task_id, status="pending")

@router.get("/tasks/{task_id}")
async def get_task_status(task_id: str) -> TaskResponse:
    state = get_task_state(task_id)
    if not state:
        raise HTTPException(404, "Task not found")
    return TaskResponse(
        task_id=task_id,
        status=state.status,
        progress=state.progress,
        message=state.message,
    )

@router.get("/tasks/{task_id}/result")
async def get_task_result(task_id: str) -> DetectionResult:
    state = get_task_state(task_id)
    if not state:
        raise HTTPException(404, "Task not found")
    if state.status != "completed":
        raise HTTPException(400, f"Task not completed: {state.status}")
    return state.result
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 64-bit pHash (hash_size=8) | 256-bit pHash (hash_size=16) | This migration | Higher accuracy, fewer false positives/negatives, but incompatible with old values |
| Go goimagehash library | Python imagehash library | This migration | Richer ecosystem, more hash algorithms available, scipy-based DCT |
| Single-dimension scoring (resolution only) | Multi-dimensional weighted scoring | This migration | Better recommendations considering resolution, file size, and format |
| Synchronous detection | Async task with progress polling | This migration | Non-blocking UI, progress visibility, cancelability |
| PHash stored as int64 | PHash stored as hex TEXT string | This migration | Supports arbitrary hash sizes, human-readable, debuggable |

**Deprecated/outdated:**
- `github.com/corona10/goimagehash`: Will be removed from Go codebase (D-02). Go library only supports hash_size=8 (64-bit) natively.
- `domain.Image.PHash int64`: Will be replaced with `string` type to store 64-char hex representation of 256-bit hash.

## Open Questions

1. **Optimal hamming distance threshold for 256-bit pHash**
   - What we know: Old 64-bit threshold was 10 (~15.6% of bits). Proportional scaling suggests ~40 for 256-bit.
   - What's unclear: Exact optimal threshold for ACG/anime images specifically. Anime images have more uniform color blocks which might affect pHash differently than photographs.
   - Recommendation: Start with threshold 40, test with real image data, expose as configurable parameter. Log threshold and group count in results for future tuning.

2. **Thread count for parallel hash computation**
   - What we know: GIL is released during PIL I/O operations (file read, image decode). `ThreadPoolExecutor` is effective for I/O-bound work.
   - What's unclear: Optimal thread count for Windows desktop with varying hardware. Too many threads may cause disk I/O contention.
   - Recommendation: Default to `min(cpu_count, 4)` workers. Expose as configuration. PIL image decode is partially CPU-bound (JPEG decompression), so don't use too many threads.

3. **Task cleanup and memory management**
   - What we know: Completed task results can be large (thousands of groups). Keeping all historical tasks in memory wastes RAM.
   - What's unclear: How long results should be kept before cleanup.
   - Recommendation: Keep only the most recent completed task result. Clean up previous task state when a new detection starts. The Go side fetches and persists results, so Python doesn't need to retain them.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Python 3.13 | Sidecar runtime | ✓ | 3.13.5 | — |
| Go 1.26 | Backend | ✓ | 1.26.0 | — |
| Pillow | Image loading | ✓ | 11.3.0 | — |
| imagehash | pHash computation | ✗ | — (would install 4.3.2) | Must be installed |
| scipy | DCT for pHash | ✗ | — (would install 1.17.1) | Must be installed with imagehash |
| numpy | Array operations | ✓ | 2.3.2 | — |
| FastAPI | HTTP framework | ✓ | 0.115.14 | — |
| uvicorn | ASGI server | ✓ | 0.35.0 | — |
| pydantic | Models | ✓ | 2.11.7 | — |

**Missing dependencies with no fallback:**
- `imagehash` + `scipy` + `PyWavelets`: Must be pip-installed before sidecar can compute pHash. Add to `requirements.txt`.

**Missing dependencies with fallback:**
- None — all critical dependencies are available or installable.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + pytest (Python) |
| Config file | None for Go (uses `go test`). Python: to be created |
| Quick run command | `go test ./internal/service/... ./internal/handler/... -run TestDuplicate -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| COMP-03a | Python computes SHA256 + pHash correctly | unit | `pytest services/python-sidecar/tests/test_hashing.py -x` | ❌ Wave 0 |
| COMP-03b | Python performs Union-Find grouping correctly | unit | `pytest services/python-sidecar/tests/test_grouping.py -x` | ❌ Wave 0 |
| COMP-03c | Go submits task, polls progress, fetches results | integration | `go test ./internal/service/ -run TestDuplicateDetectionPython -count=1` | ❌ Wave 0 |
| COMP-03d | Go saves Python results to database correctly | integration | `go test ./internal/service/ -run TestDuplicateSaveResults -count=1` | ❌ Wave 0 |
| COMP-03e | Sidecar unavailable returns clear error | unit | `go test ./internal/handler/ -run TestDuplicateDetectSidecarUnavailable -count=1` | ❌ Wave 0 |
| COMP-04a | Multi-dimensional scoring produces correct recommendations | unit | `pytest services/python-sidecar/tests/test_scoring.py -x` | ❌ Wave 0 |
| COMP-04b | Structured rationale contains all scoring factors | unit | `pytest services/python-sidecar/tests/test_scoring.py::test_rationale_structure -x` | ❌ Wave 0 |
| COMP-04c | Go persists recommendation rationale to database | integration | `go test ./internal/repository/ -run TestDuplicateRelationRationale -count=1` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/service/... ./internal/handler/... -run TestDuplicate -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `services/python-sidecar/tests/test_hashing.py` — covers COMP-03a
- [ ] `services/python-sidecar/tests/test_grouping.py` — covers COMP-03b
- [ ] `services/python-sidecar/tests/test_scoring.py` — covers COMP-04a, COMP-04b
- [ ] `services/python-sidecar/tests/conftest.py` — shared fixtures (test images)
- [ ] Python test framework install: `pip install pytest` — if not already available
- [ ] `internal/service/duplicate_service_test.go` — extend for Python integration tests (COMP-03c, COMP-03d)
- [ ] `internal/handler/duplicate_handler_test.go` — extend for sidecar unavailable error (COMP-03e)

## Sources

### Primary (HIGH confidence)
- [/johannesbuchner/imagehash] - phash API, hash_size parameter, hex conversion, hamming distance via `__sub__`, ImageHash class structure
- [imagehash source code](https://raw.githubusercontent.com/JohannesBuchner/imagehash/master/imagehash/__init__.py) - Verified `phash()` function signature, DCT computation, binary array to hex conversion for hash_size=16
- [FastAPI official docs via /websites/fastapi_tiangolo] - BackgroundTasks API, limitations for long-running tasks
- Existing codebase: `internal/service/duplicate_service.go`, `hash_service.go`, `domain/duplicate_group.go`, `repository/duplicate_repository.go`, `sidecar/runtime.go`, `handler/duplicate_handler.go`
- Environment probing: Python 3.13.5, pip packages verified on development machine

### Secondary (MEDIUM confidence)
- [FastAPI polling strategy article](https://openillumi.com/en/en-fastapi-long-task-progress-polling/) — Task ID + polling pattern for long-running tasks (403'd but title confirms pattern)
- GitHub code search — Union-Find implementations in Python (doocs/leetcode, TheAlgorithms/Python)
- GitHub code search — FastAPI task_id + status + progress patterns (CatchTheTornado/text-extract-api, rb-x/pwnflow)

### Tertiary (LOW confidence)
- Hamming distance threshold scaling for 256-bit: Proportional reasoning from 64-bit experience, needs validation with real ACG image data

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - imagehash is the de facto Python perceptual hashing library (3.8k stars, verified source code, version confirmed)
- Architecture: HIGH - Async task pattern is well-established, verified against multiple real implementations; Go↔Python HTTP contract follows Phase 15 precedent
- Pitfalls: HIGH - Threshold scaling mathematically derived; file path/encoding issues verified on Windows; O(n²) complexity is mathematical fact
- Scoring algorithm: MEDIUM - Multi-dimensional scoring is straightforward but optimal weights need empirical tuning with real data

**Research date:** 2026-04-04
**Valid until:** 2026-05-04 (30 days — stable libraries, well-understood domain)
