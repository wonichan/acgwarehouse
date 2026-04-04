---
phase: 16-duplicate-detection-migration
plan: 01
subsystem: testing
tags: [python, fastapi, imagehash, duplicate-detection, sidecar]

requires:
  - phase: 15-compute-sidecar-infrastructure
    provides: Go↔Python sidecar lifecycle and runtime boundary
provides:
  - Python sidecar duplicate detection compute pipeline (hashing/grouping/scoring)
  - Async detect task lifecycle endpoints: submit/poll/fetch-result
  - Structured recommendation rationale output (factor/value/score/weight)
affects: [phase-16-plan-02, phase-16-plan-03, duplicate-detection]

tech-stack:
  added: [imagehash, scipy, PyWavelets]
  patterns: [in-memory task state, union-find grouping, weighted recommendation scoring]

key-files:
  created:
    - services/python-sidecar/main.py
    - services/python-sidecar/routers/duplicates.py
    - services/python-sidecar/compute/hashing.py
    - services/python-sidecar/compute/grouping.py
    - services/python-sidecar/compute/scoring.py
    - services/python-sidecar/models/duplicates.py
    - services/python-sidecar/tests/test_task_state.py
    - services/python-sidecar/tests/test_hashing.py
    - services/python-sidecar/tests/test_grouping.py
    - services/python-sidecar/tests/test_scoring.py
    - services/python-sidecar/tests/test_duplicates_router.py
  modified:
    - services/python-sidecar/compute/task_state.py
    - services/python-sidecar/routers/__init__.py
    - services/python-sidecar/tests/conftest.py
    - services/python-sidecar/requirements.txt

key-decisions:
  - "pHash 升级为 imagehash hash_size=16（256-bit），阈值默认按 40 执行。"
  - "路由采用 submit → poll → fetch-result 三端点异步模式，并限制单活检测任务。"
  - "推荐保留改为多维加权评分并返回结构化 reasons，供 Go/前端透传展示。"

patterns-established:
  - "计算职责仅在 Python 侧，输入输出保持纯计算契约。"
  - "坏图/缺图按 skipped_images 返回，不中断整批任务。"

requirements-completed: [COMP-03, COMP-04]

duration: 11 min
completed: 2026-04-04
---

# Phase 16 Plan 01: Python sidecar 重复检测计算管线总结

**Python 侧车已完整提供 SHA256 + 256-bit pHash 计算、Union-Find 分组、多维推荐评分，以及可轮询的异步检测任务 API。**

## Performance

- **Duration:** 11 min
- **Started:** 2026-04-04T13:54:30+08:00
- **Completed:** 2026-04-04T14:06:27+08:00
- **Tasks:** 3
- **Files modified:** 18

## Accomplishments
- 完成 `services/python-sidecar` 侧车工程骨架、依赖锁定、Pydantic 模型和线程安全任务状态管理。
- 完成三大计算模块：`hashing.py`（SHA256 + pHash）、`grouping.py`（Union-Find + hamming）、`scoring.py`（多维加权评分与 reasons）。
- 完成 FastAPI 路由与入口：`POST /compute/duplicates/detect`、`GET /compute/duplicates/tasks/{id}`、`GET /compute/duplicates/tasks/{id}/result`。
- 完成端到端测试覆盖：任务生命周期、坏图跳过、单活互斥（409）、推荐项与结构化推荐依据。

## Task Commits

每个任务均按原子提交：

1. **Task 1: 项目骨架、模型与任务状态** - `1241b5d` (feat)
2. **Task 2: hashing/grouping/scoring 计算模块** - `ced9254` (feat)
3. **Task 3: FastAPI 路由与异步任务端点** - `776266c` (feat)

**Plan metadata:** 待本次文档提交生成。

## Files Created/Modified
- `services/python-sidecar/requirements.txt` - 侧车依赖版本锁定。
- `services/python-sidecar/models/duplicates.py` - 请求/响应与分组成员模型。
- `services/python-sidecar/compute/task_state.py` - 任务状态机与线程安全读写。
- `services/python-sidecar/compute/hashing.py` - SHA256 + 256-bit pHash 批量计算。
- `services/python-sidecar/compute/grouping.py` - Union-Find 分组与汉明距离计算。
- `services/python-sidecar/compute/scoring.py` - 推荐评分与结构化理由。
- `services/python-sidecar/routers/duplicates.py` - 检测任务三端点与后台线程执行。
- `services/python-sidecar/main.py` - FastAPI 入口与路由注册。
- `services/python-sidecar/tests/*.py` - TaskState / Compute / Router 测试集。

## Decisions Made
- 采用 `imagehash.phash(..., hash_size=16)` 作为 256-bit pHash 标准实现，并把哈希值统一为 64 字符 hex。
- 分组算法按“先 exact(sha256) 后 similar(phash)”执行，并保留 Union-Find 传递性语义。
- 推荐策略采用 `resolution(0.5) + file_size(0.3) + format(0.2)`，并返回 `factor/value/score/weight` 可解释结构。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] 补充 pyright 局部配置以消除侧车目录误报诊断**
- **Found during:** Task 2 / Task 3 的 LSP 诊断
- **Issue:** 工作区根在仓库顶层时，`services/python-sidecar` 内的绝对导入路径触发误报，影响“改动文件零错误”验收
- **Fix:** 新增 `services/python-sidecar/pyrightconfig.json`，并在个别测试/入口文件添加最小化 pyright 指令
- **Files modified:** `services/python-sidecar/pyrightconfig.json`, `services/python-sidecar/main.py`, `services/python-sidecar/routers/duplicates.py`, `services/python-sidecar/tests/test_*.py`
- **Verification:** `lsp_diagnostics`（sidecar 目录）返回 0 errors
- **Committed in:** `ced9254` / `776266c`

---

**Total deviations:** 1 auto-fixed（1 missing critical）
**Impact on plan:** 仅用于静态诊断对齐，不改变运行时计算行为与接口契约。

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- 16-01 计算管线已可独立被编排器 spot-check。
- 已满足 Wave 2 集成前置之一，待 16-02 产物对齐后可进入 `16-03-PLAN.md` 进行 Go↔Python 联调接线。

---
*Phase: 16-duplicate-detection-migration*
*Completed: 2026-04-04*
