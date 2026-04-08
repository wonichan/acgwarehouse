# Duplicate Detection SSE Incremental Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert duplicate detection into a Go-owned async task with SSE progress, image-level hash caching, and paged/batched sidecar execution so Flutter shows live progress and later runs become incremental.

**Architecture:** Keep Flutter talking only to Go. Add a dedicated Go duplicate-task state layer plus SSE/status endpoints, persist reusable `sha256`/`phash` on `images`, teach Go to validate cached hashes before sending work to Python, and keep pairwise duplicate data such as `distance` on `duplicate_relations`. Replace all-at-once duplicate submission with paged image reads and bounded Python batches under one logical task.

**Tech Stack:** Go (`gin`, repository tests, service tests, `go test`), Python sidecar (`FastAPI`, `pytest`), Flutter desktop (`provider`, `http`, SSE client package, `flutter test`).

**Reference Spec:** `docs/superpowers/specs/2026-04-08-duplicate-detection-sse-incremental-design.md`

**Execution Constraint:** Do not create git commits unless the user explicitly requests them during execution.

---

## Chunk 1: Image hash cache schema and transport contract

### Task 1: Persist reusable duplicate hash metadata on `images`

**Files:**
- Modify: `internal/domain/image.go`
- Modify: `internal/repository/schema.go`
- Modify: `internal/repository/image_repository.go`
- Modify: `internal/repository/image_repository_test.go`

- [ ] **Step 1: Write failing repository tests for new image hash-cache fields**

Cover these behaviors:
- `images` can persist and read back `sha256`
- `images` can persist and read back a freshness field for source-file validation
- existing `phash` / `phash_hex` reads keep working
- older rows without the new columns still load safely after schema migration

- [ ] **Step 2: Run focused Go repository tests to verify failure**

Run: `go test ./internal/repository -run "ImageRepository|EnsureScanSchema" -count=1`
Expected: FAIL because the new cache fields and repository mappings do not exist yet.

- [ ] **Step 3: Implement the schema/domain/repository changes**

Requirements:
- add `sha256` to `images`
- add one persisted freshness field for source-file validation; use a filesystem-derived value rather than database `updated_at`
- extend `domain.Image` and all image scans/inserts/updates accordingly
- add a narrow repository update method for duplicate-hash cache persistence instead of scattering raw SQL across services

- [ ] **Step 4: Re-run focused Go repository tests**

Run: `go test ./internal/repository -run "ImageRepository|EnsureScanSchema" -count=1`
Expected: PASS.

### Task 2: Expand the Go↔Python duplicate request/response contract for cache reuse

**Files:**
- Modify: `internal/sidecar/client.go`
- Modify: `internal/sidecar/client_test.go`
- Modify: `services/python-sidecar/models/duplicates.py`

- [ ] **Step 1: Write failing transport tests for optional cached hash fields**

Cover these behaviors:
- Go serializes optional `sha256` and `phash` cache fields when present
- Go still handles sidecar responses when those fields are absent
- Python request models accept optional cached hash fields without breaking existing request parsing

- [ ] **Step 2: Run focused transport tests to verify failure**

Run: `go test ./internal/sidecar -run "Detection|SidecarClient" -count=1`
Run from `services/python-sidecar`: `pytest tests/test_duplicates_router.py -k detect -q`
Expected: FAIL because the duplicate request/response models still use the old shape.

- [ ] **Step 3: Implement the transport-model changes**

Requirements:
- add optional cached hash fields to Go `DetectionImageInput`
- add matching optional fields to Python `ImageInput`
- keep the existing result-member shape stable for `sha256`, `phash`, and `distance`
- do not move `distance` into image-level cache fields

- [ ] **Step 4: Re-run focused transport tests**

Run: `go test ./internal/sidecar -run "Detection|SidecarClient" -count=1`
Run from `services/python-sidecar`: `pytest tests/test_duplicates_router.py -k detect -q`
Expected: PASS.

## Chunk 2: Go-owned duplicate task lifecycle and SSE endpoints

### Task 3: Introduce duplicate-task state and structured progress events in Go

**Files:**
- Create: `internal/service/duplicate_task_service.go`
- Create: `internal/service/duplicate_task_service_test.go`
- Modify: `internal/service/duplicate_service.go`
- Modify: `internal/service/duplicate_service_test.go`

- [ ] **Step 1: Write failing Go tests for duplicate-task lifecycle and progress aggregation**

Cover these behaviors:
- duplicate detection creates a Go-owned task ID immediately
- task status transitions are `queued -> preparing -> hashing -> grouping -> persisting -> completed/failed`
- progress is monotonic and carries processed/total counts
- failure in sidecar submission, polling, or persistence leaves a terminal failed task state

- [ ] **Step 2: Run focused Go service tests to verify failure**

Run: `go test ./internal/service -run "DuplicateService_|DuplicateTaskService" -count=1`
Expected: FAIL because the task-state service and status model do not exist yet.

- [ ] **Step 3: Implement the duplicate-task service and integrate it into duplicate execution**

Requirements:
- create an in-memory duplicate-task registry for this phase
- store task metadata needed by both status polling and SSE
- make Go the owner of user-visible phase names and progress
- keep Python task IDs internal to Go
- do not promise restart durability in this phase

- [ ] **Step 4: Re-run focused Go service tests**

Run: `go test ./internal/service -run "DuplicateService_|DuplicateTaskService" -count=1`
Expected: PASS.

### Task 4: Add duplicate task creation, status, and SSE endpoints

**Files:**
- Modify: `internal/handler/duplicate_handler.go`
- Modify: `internal/handler/duplicate_handler_test.go`
- Modify: `internal/handler/routes.go`

- [ ] **Step 1: Write failing handler tests for async duplicate endpoints**

Cover these behaviors:
- `POST /api/v1/duplicates/detect` returns immediately with `task_id`
- `GET /api/v1/duplicates/tasks/:task_id` returns current status payload
- `GET /api/v1/duplicates/tasks/:task_id/events` streams `text/event-stream`
- invalid task IDs return `404`
- existing duplicate list/detail/delete routes remain intact

- [ ] **Step 2: Run focused Go handler tests to verify failure**

Run: `go test ./internal/handler -run "DuplicateHandler_" -count=1`
Expected: FAIL because the handler contract is still synchronous and SSE does not exist.

- [ ] **Step 3: Implement the async duplicate endpoints**

Requirements:
- keep the detection create route path unchanged
- add explicit task-status and SSE routes under `/api/v1/duplicates/tasks/...`
- use structured SSE events instead of log-text forwarding
- send keepalive/heartbeat output so long desktop runs do not look dead

- [ ] **Step 4: Re-run focused Go handler tests**

Run: `go test ./internal/handler -run "DuplicateHandler_" -count=1`
Expected: PASS.

## Chunk 3: Incremental hashing and bounded sidecar execution

### Task 5: Teach Go duplicate detection to validate cache, page reads, and batch sidecar work

**Files:**
- Modify: `internal/service/duplicate_service.go`
- Modify: `internal/service/duplicate_service_test.go`
- Modify: `internal/repository/image_repository.go`
- Modify: `test/e2e/duplicate_test.go`

- [ ] **Step 1: Write failing Go tests for incremental duplicate execution**

Cover these behaviors:
- unchanged images with valid `sha256` / `phash` are passed to Python as cache hits
- stale or missing cache values trigger recomputation
- duplicate execution reads images in pages instead of relying on one giant effective payload
- Go can aggregate multiple internal sidecar batches under one logical duplicate task
- relation rows still retain `phash_distance` and recommendation metadata after persistence

- [ ] **Step 2: Run focused Go tests to verify failure**

Run: `go test ./internal/service ./test/e2e -run "Duplicate" -count=1`
Expected: FAIL because duplicate detection still does all-at-once full-library execution.

- [ ] **Step 3: Implement paged/batched incremental orchestration in Go**

Requirements:
- validate cache freshness before sending cached values to Python
- use the persisted freshness field plus current file stat for reuse decisions
- read images in bounded pages
- submit bounded sidecar batches under one logical Go task
- persist refreshed image-level cache values before marking the task complete
- keep `duplicate_relations.file_hash` as a derived read-model field if existing API responses still need it

- [ ] **Step 4: Re-run focused Go tests**

Run: `go test ./internal/service ./test/e2e -run "Duplicate" -count=1`
Expected: PASS.

### Task 6: Add Python-side cache reuse, configurable concurrency, and finer progress updates

**Files:**
- Modify: `services/python-sidecar/compute/hashing.py`
- Modify: `services/python-sidecar/routers/duplicates.py`
- Modify: `services/python-sidecar/tests/test_hashing.py`
- Modify: `services/python-sidecar/tests/test_duplicates_router.py`

- [ ] **Step 1: Write failing Python tests for cache reuse and worker configuration**

Cover these behaviors:
- cached `sha256` + `phash` skip recomputation
- partial cache recomputes only the missing value
- stale/missing cache still computes both values
- worker count defaults to a configurable `cpu_count * 2` strategy instead of fixed `4`
- router progress updates expose meaningful stage and processed-count advancement

- [ ] **Step 2: Run focused pytest to verify failure**

Run from `services/python-sidecar`: `pytest tests/test_hashing.py tests/test_duplicates_router.py -q`
Expected: FAIL because hashing still always recomputes both values and worker count is still capped by the old default.

- [ ] **Step 3: Implement Python cache reuse and progress changes**

Requirements:
- keep the hashing logic in `compute/hashing.py`
- make worker count configurable; default to `cpu_count * 2`, but keep an override seam
- preserve per-image error reporting
- expose progress that Go can map to user-visible phase/count updates

- [ ] **Step 4: Re-run focused pytest**

Run from `services/python-sidecar`: `pytest tests/test_hashing.py tests/test_duplicates_router.py -q`
Expected: PASS.

## Chunk 4: Flutter duplicate task API, provider state, and progress UI

### Task 7: Refactor Flutter duplicate service to create tasks, query status, and consume SSE

**Files:**
- Modify: `flutter_app/lib/services/duplicate_service.dart`
- Modify: `flutter_app/lib/config/api_config.dart`
- Modify: `flutter_app/test/services/duplicate_service_test.dart`
- Create: `flutter_app/test/providers/duplicate_provider_test.dart`

- [ ] **Step 1: Write failing Flutter tests for async duplicate task APIs**

Cover these behaviors:
- duplicate detection create call parses `task_id` and initial status
- service can query duplicate task status
- service can subscribe to or parse SSE event payloads
- old duplicate group list/detail parsing keeps working

- [ ] **Step 2: Run focused Flutter tests to verify failure**

Run from `flutter_app`: `flutter test test/services/duplicate_service_test.dart test/providers/duplicate_provider_test.dart`
Expected: FAIL because the service still only supports one blocking detect call.

- [ ] **Step 3: Implement the Flutter duplicate task client**

Requirements:
- add endpoint constants for task status and SSE stream
- model the async duplicate task response and progress event payload
- keep group list/detail/delete APIs backward compatible
- encapsulate SSE lifecycle inside the service/provider boundary rather than the screen widget

- [ ] **Step 4: Re-run focused Flutter tests**

Run from `flutter_app`: `flutter test test/services/duplicate_service_test.dart test/providers/duplicate_provider_test.dart`
Expected: PASS.

### Task 8: Replace spinner-only duplicate UI with live task progress in the provider and screen

**Files:**
- Modify: `flutter_app/lib/providers/duplicate_provider.dart`
- Modify: `flutter_app/lib/screens/duplicate_screen.dart`
- Create: `flutter_app/test/screens/duplicate_screen_test.dart`

- [ ] **Step 1: Write failing Flutter tests for progress rendering and reconnect-safe state**

Cover these behaviors:
- provider exposes task ID, phase, progress, processed/total, and terminal error state
- screen renders progress details instead of only a spinner
- screen shows completion/failure feedback and refreshes duplicate groups after completion
- provider can recover by reading task status when the SSE stream drops

- [ ] **Step 2: Run focused Flutter screen/provider tests to verify failure**

Run from `flutter_app`: `flutter test test/providers/duplicate_provider_test.dart test/screens/duplicate_screen_test.dart`
Expected: FAIL because provider state still only tracks `isDetecting`.

- [ ] **Step 3: Implement the provider and screen state changes**

Requirements:
- replace the single boolean detect state with a task-progress model
- render phase label, percentage, and processed/total counts
- stop SSE subscription on terminal task states
- keep duplicate-group pagination and delete behavior unchanged

- [ ] **Step 4: Re-run focused Flutter screen/provider tests**

Run from `flutter_app`: `flutter test test/providers/duplicate_provider_test.dart test/screens/duplicate_screen_test.dart`
Expected: PASS.

## Chunk 5: End-to-end verification and manual QA

### Task 9: Verify the full duplicate-detection redesign across Go, Python, and Flutter contracts

**Files:**
- Verify: `internal/handler/duplicate_handler.go`
- Verify: `internal/service/duplicate_service.go`
- Verify: `internal/service/duplicate_task_service.go`
- Verify: `internal/sidecar/client.go`
- Verify: `internal/repository/image_repository.go`
- Verify: `services/python-sidecar/routers/duplicates.py`
- Verify: `services/python-sidecar/compute/hashing.py`
- Verify: `flutter_app/lib/services/duplicate_service.dart`
- Verify: `flutter_app/lib/providers/duplicate_provider.dart`
- Verify: `flutter_app/lib/screens/duplicate_screen.dart`

- [ ] **Step 1: Run focused automated tests**

Run: `go test ./internal/handler ./internal/repository ./internal/service ./internal/sidecar ./test/e2e -count=1`
Run from `services/python-sidecar`: `pytest tests/test_hashing.py tests/test_duplicates_router.py -q`
Run from `flutter_app`: `flutter test test/services/duplicate_service_test.dart test/providers/duplicate_provider_test.dart test/screens/duplicate_screen_test.dart`
Expected: PASS.

- [ ] **Step 2: Run diagnostics on changed files**

Run diagnostics for changed Go, Python, and Dart files.
Expected: zero new issues.

- [ ] **Step 3: Manual QA cold run**

Execute a real duplicate-detection run against a library with uncached images.
Verify:
- task creation returns immediately
- Flutter receives live SSE progress
- `images.sha256` and `images.phash_hex` are populated after completion
- duplicate groups still render correctly

- [ ] **Step 4: Manual QA warm run**

Execute duplicate detection again without changing files.
Verify:
- unchanged images reuse cached hashes
- progress reflects cache-hit-heavy execution
- total runtime is materially lower than the cold run

- [ ] **Step 5: Manual QA stale-cache run**

Modify or replace at least one source image, then run detection again.
Verify:
- the changed image invalidates cache and recomputes hashes
- unchanged images still reuse cache
- relation-level `distance` data remains correct in duplicate detail responses

Plan complete and saved to `docs/superpowers/plans/2026-04-08-duplicate-detection-sse-incremental-plan.md`. Ready to execute?
