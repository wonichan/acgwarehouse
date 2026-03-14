---
phase: 01-foundation-scan-tag-base
plan: 03
subsystem: scanner
tags: [go, scanner, fsnotify, sqlite, imagemeta, async-jobs]
requires:
  - 01-01
  - 01-02
provides:
  - Scan CLI that imports supported image metadata into SQLite
  - Recursive watcher foundation with debounce-based image import
  - SQLite-backed async job repository and sequential job manager
affects: [scanner, async-jobs, api, ai-tagging]
tech-stack:
  added: [fsnotify, golang.org/x/image]
  patterns: [scan-and-queue pipeline, recursive watcher wrapper, sequential sqlite job manager]
key-files:
  created:
    - cmd/scan/main.go
    - internal/repository/image_repository.go
    - internal/repository/job_repository.go
    - internal/repository/schema.go
    - internal/service/metadata_service.go
    - internal/service/scanner_service.go
    - internal/service/watcher_service.go
    - internal/service/watcher_service_os.go
    - internal/worker/job_manager.go
    - internal/service/metadata_service_test.go
    - internal/service/scanner_service_test.go
    - internal/service/watcher_service_test.go
    - internal/worker/job_manager_test.go
  modified:
    - go.mod
    - go.sum
    - cmd/server/main.go
    - .planning/STATE.md
    - .planning/ROADMAP.md
    - .planning/REQUIREMENTS.md
key-decisions:
  - "Use imagemeta as a tolerant metadata probe, then rely on DecodeConfig for width and height extraction across supported fixtures."
  - "Implement recursive watching explicitly because fsnotify does not watch subdirectories automatically."
  - "Keep async processing sequential and SQLite-safe, with recorded import jobs but no multi-writer complexity yet."
patterns-established:
  - "Import Pipeline Pattern: scan root -> metadata extraction -> image repository save -> async job enqueue."
  - "Watcher Debounce Pattern: create/write events are coalesced before a single import attempt."
requirements-completed: [IMPT-01, IMPT-02, IMPT-03, IMPT-04]
duration: interrupted session (~45 min)
completed: 2026-03-14
---

# Phase 01 Plan 03: 图片扫描、监控与异步任务 Summary

**Delivered a working scan/import pipeline with metadata extraction, recursive watcher behavior, and persisted `image_imported` jobs for future AI processing.**

## Performance

- **Duration:** interrupted session (~45 min)
- **Started:** 2026-03-14T13:05:00Z
- **Completed:** 2026-03-14T14:10:00Z
- **Tasks:** 3
- **Files modified:** 19

## Accomplishments
- Added `cmd/scan` so users can scan a configured root or explicit `-path` and get import statistics.
- Implemented metadata extraction for JPG/JPEG/PNG/WebP/GIF recognition, with verified width/height extraction on real image fixtures.
- Persisted imported image metadata to SQLite and queued `image_imported` rows in `async_jobs` for downstream AI work.
- Added a recursive watcher foundation with debounce so newly created images under watched roots are imported automatically.
- Added a sequential job manager and regression tests covering scan import, watcher import, and job processing order.

## Task Commits

1. **task 1+3: 图片扫描 CLI、元数据提取与异步任务基础设施** - `a96fad6` (feat)
2. **task 2: 文件夹监控服务** - `646f3bb` (feat)
3. **stability fix: 模块路径修正** - `5166a7a` (fix)

**Plan metadata:** committed separately after summary finalization.

## Files Created/Modified
- `cmd/scan/main.go` - Scan CLI entrypoint with config loading, SQLite bootstrap, and result output.
- `internal/repository/image_repository.go` - SQLite repository for imported image metadata.
- `internal/repository/job_repository.go` - SQLite repository for async job persistence and updates.
- `internal/repository/schema.go` - Minimal schema bootstrap for scan-specific tables in fresh SQLite files.
- `internal/service/metadata_service.go` - Supported-format detection and metadata extraction.
- `internal/service/scanner_service.go` - Scan orchestration, import pipeline, and async job enqueueing.
- `internal/service/watcher_service.go` - Recursive fsnotify wrapper with debounce-based imports.
- `internal/service/watcher_service_os.go` - OS stat helper for watcher directory detection.
- `internal/worker/job_manager.go` - Sequential job manager with handler registration.
- `internal/service/metadata_service_test.go` - Metadata regression coverage.
- `internal/service/scanner_service_test.go` - Scan/import regression coverage.
- `internal/service/watcher_service_test.go` - Watcher regression coverage.
- `internal/worker/job_manager_test.go` - Sequential job processing regression coverage.
- `go.mod` - Added `fsnotify` and `golang.org/x/image` dependencies.
- `go.sum` - Recorded dependency checksums for scan and watcher stack.
- `cmd/server/main.go` - Updated imports to match the real module path.
- `.planning/STATE.md` - Updated on disk for phase progress.
- `.planning/ROADMAP.md` - Updated on disk for plan completion progress.
- `.planning/REQUIREMENTS.md` - Updated on disk to mark Phase 1 image-import requirements complete.

## Decisions Made
- Used the real repository module path `github.com/wonichan/acgwarehouse-backend` instead of keeping placeholder imports.
- Kept watcher behavior inside `internal/service/watcher_service.go` instead of inventing a separate watcher binary prematurely.
- Used a sequential job manager because SQLite is single-writer sensitive and Phase 1 only needs durable event recording.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Recovered interrupted 01-03 execution from partial test-first state**
- **Found during:** task orchestration recovery
- **Issue:** The original execution stopped after writing tests and part of the repository layer, leaving service and worker implementations missing.
- **Fix:** Completed the missing implementation files directly from the existing failing tests, then re-ran full verification.
- **Files modified:** `cmd/scan/main.go`, `internal/repository/*.go`, `internal/service/*.go`, `internal/worker/*.go`
- **Verification:** `go test ./...`, `go build ./...`
- **Committed in:** `a96fad6`, `646f3bb`

**2. [Rule 3 - Blocking] Replaced placeholder module imports with the real module path**
- **Found during:** full-repo verification after initial 01-03 commits
- **Issue:** Existing code still imported `github.com/yourusername/acgwarehouse-backend`, which broke `go test ./...` once all packages were compiled together.
- **Fix:** Updated all internal imports to `github.com/wonichan/acgwarehouse-backend`.
- **Files modified:** `cmd/server/main.go`, `cmd/scan/main.go`, `internal/repository/*.go`, `internal/service/*.go`, `internal/worker/*.go`
- **Verification:** `go test ./...`, `go build ./...`
- **Committed in:** `5166a7a`

**3. [Rule 3 - Blocking] Replaced the plan's undefined watcher binary verification target**
- **Found during:** task 2 (文件夹监控服务)
- **Issue:** The plan verified `./cmd/watcher`, but no such deliverable was defined in plan artifacts or must-haves.
- **Fix:** Kept watcher behavior as a service-level component and proved it with a regression test that imports a newly created image and queues a job.
- **Files modified:** `internal/service/watcher_service.go`, `internal/service/watcher_service_test.go`
- **Verification:** `go test -v ./internal/service/... -run TestWatcherImportsNewImageAndQueuesJob`
- **Committed in:** `646f3bb`

---

**Total deviations:** 3 auto-fixed (3 blocking)
**Impact on plan:** These fixes preserved the intended scanner/watcher/job outcomes while removing execution-breakers and keeping the repository buildable.

## Issues Encountered

- The original 01-03 execution lost its background task handle before writing summary/tracker artifacts, so recovery had to continue from local filesystem evidence.
- Full-repo verification exposed legacy placeholder import paths that partial package builds had not surfaced earlier.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 1 now has a working import path: scan CLI, recursive watcher behavior, persisted image rows, and queued import events.
- The codebase is ready for phase-level verification and transition into Phase 2 planning/discussion.

---
*Phase: 01-foundation-scan-tag-base*
*Completed: 2026-03-14*
