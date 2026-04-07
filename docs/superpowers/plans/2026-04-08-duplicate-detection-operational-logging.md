# Duplicate Detection Operational Logging Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add minimal stdout operational logs so duplicate detection progress and task state are visible from Go and Python logs without changing APIs.

**Architecture:** Keep the existing duplicate-detection flow intact and add narrow logging hooks in the Go service/client path plus the Python sidecar task lifecycle. Throttle noisy progress logs so operators can see status changes and stage transitions without one line per poll or one line per image.

**Tech Stack:** Go (`log.Printf`, httptest, go test), Python (`logging`, pytest, FastAPI TestClient).

---

## Chunk 1: Go duplicate-detection lifecycle logs

### Task 1: Add failing tests for service/client logging

**Files:**
- Modify: `internal/service/duplicate_service_test.go`
- Modify: `internal/sidecar/client_test.go`
- Modify: `internal/service/duplicate_service.go`
- Modify: `internal/sidecar/client.go`

- [ ] **Step 1: Write failing Go tests**

Cover these behaviors:
- `DetectDuplicates` logs submission context, accepted `task_id`, status/progress changes, and final completion/failure.
- Result handling logs enough metadata for operators to understand completion (`task_id`, groups, skipped images, duration).

- [ ] **Step 2: Run targeted Go tests to verify failures**

Run: `go test ./internal/service ./internal/sidecar -run "DuplicateService_DetectDuplicates_Logs|SidecarClient_" -count=1`
Expected: FAIL because the new log lines do not exist yet.

- [ ] **Step 3: Implement minimal Go logs**

Requirements:
- Use existing `log.Printf` style.
- Log only on meaningful events or state changes; do not log every 2-second poll.
- Include `task_id`, status, progress, stage/message, and final totals where available.

- [ ] **Step 4: Re-run targeted Go tests**

Run: `go test ./internal/service ./internal/sidecar -run "DuplicateService_DetectDuplicates_Logs|SidecarClient_" -count=1`
Expected: PASS.

## Chunk 2: Python sidecar task lifecycle logs

### Task 2: Add failing router tests for lifecycle/progress logging

**Files:**
- Modify: `services/python-sidecar/tests/test_duplicates_router.py`
- Modify: `services/python-sidecar/routers/duplicates.py`

- [ ] **Step 1: Write failing Python tests**

Cover these behaviors:
- submit/start logs include task id and image count
- stage transitions log hashing/grouping/scoring/completed
- failures log task id plus error
- progress logs are throttled to meaningful milestones

- [ ] **Step 2: Run targeted pytest to verify failures**

Run: `pytest tests/test_duplicates_router.py -k logging -q`
Expected: FAIL because lifecycle logs are not emitted yet.

- [ ] **Step 3: Implement minimal Python logs**

Requirements:
- Use stdlib `logging`.
- Keep logs in `routers/duplicates.py`; no new logging subsystem.
- Avoid per-image logs and avoid logging every poll request.
- Log skipped-image count and total duration on completion.

- [ ] **Step 4: Re-run targeted pytest**

Run: `pytest tests/test_duplicates_router.py -k logging -q`
Expected: PASS.

## Chunk 3: Verification and manual QA

### Task 3: Verify changed behavior end to end

**Files:**
- Verify: `internal/service/duplicate_service.go`
- Verify: `internal/sidecar/client.go`
- Verify: `services/python-sidecar/routers/duplicates.py`
- Verify: new/updated tests only

- [ ] **Step 1: Run focused automated tests**

Run: `go test ./internal/service ./internal/sidecar -count=1`
Run: `pytest tests/test_duplicates_router.py -q`
Expected: PASS.

- [ ] **Step 2: Run diagnostics on changed files**

Run diagnostics for changed Go and Python files.
Expected: zero new issues.

- [ ] **Step 3: Manual QA**

Run a real duplicate-detection flow and capture stdout/stderr showing:
- Go logs for submit, task acceptance, status change, completion/failure.
- Python logs for task start, stage transitions, and completion/failure.

Expected: operators can tell what stage the task is in without relying on repeated GET access logs.
