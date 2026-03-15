---
phase: 03-ai
plan: 03
subsystem: api
tags: [gin, sqlite, tags, async-jobs, ai]
requires:
  - phase: 03-02
    provides: tag repositories, image-tag repository, governance service
provides:
  - tag CRUD and alias management endpoints
  - image tag review and association endpoints
  - AI tag trigger and status endpoints
affects: [03-04, tag-review-ui, image-detail-api]
tech-stack:
  added: []
  patterns: [gin handler dependency injection, sqlite-backed handler tests, async job status lookup by payload]
key-files:
  created:
    [internal/handler/tag_handler.go, internal/handler/image_tag_handler.go, internal/handler/ai_tag_handler.go, internal/handler/routes_test.go]
  modified:
    [cmd/server/main.go, internal/handler/routes.go, internal/repository/tag_repository.go, internal/repository/image_repository.go, internal/repository/job_repository.go]
key-decisions:
  - "Route registration accepts optional dependencies so existing server callers stay compatible while real handlers can be wired in."
  - "Server startup now builds repositories, governance service, and async job manager so tag APIs are functional instead of placeholder-only."
  - "Tag deletion explicitly removes image-tag and alias associations before deleting the tag record."
patterns-established:
  - "Handler tests use sqlite fixtures plus Gin test routers to verify request and response behavior."
  - "AI status handlers resolve the latest image job by scanning ai_tag_generation payloads from the async_jobs table."
requirements-completed: [TAGS-02, TAGS-04, TAGS-05, AIRE-05]
duration: 16 min
completed: 2026-03-15
---

# Phase 03 Plan 03: 标签管理 API 层 Summary

**Gin-based tag CRUD, image tag review workflows, and AI tag job endpoints backed by sqlite repositories and async job wiring**

## Performance

- **Duration:** 16 min
- **Started:** 2026-03-15T16:57:19+08:00
- **Completed:** 2026-03-15T17:13:28+08:00
- **Tasks:** 4
- **Files modified:** 12

## Accomplishments
- Added `TagHandler` for tag listing, search, CRUD, and alias management.
- Added `ImageTagHandler` for grouped image tag queries, manual tag attachment, and review workflows.
- Added `AITagHandler` for queueing AI tag jobs, batch triggers, and status polling.
- Registered all tag-related routes and wired server startup dependencies for runtime use.

## task Commits

Each task was committed atomically:

1. **task 1: 实现标签管理 API** - `e90c792`, `87dda2f` (test, feat)
2. **task 2: 实现图片标签关联 API** - `513b338`, `d7ee964` (test, feat)
3. **task 3: 实现 AI 标签触发 API** - `1f82d9f`, `c7515d6` (test, feat)
4. **task 4: 更新路由注册** - `64fa3d3` (feat)

**Plan metadata:** pending

## Files Created/Modified
- `internal/handler/tag_handler.go` - Tag CRUD, alias management, and merged search handling.
- `internal/handler/tag_handler_test.go` - Route-level TDD coverage for tag endpoints.
- `internal/handler/image_tag_handler.go` - Image tag grouping, mutation, and review handlers.
- `internal/handler/image_tag_handler_test.go` - Route-level TDD coverage for image-tag endpoints.
- `internal/handler/ai_tag_handler.go` - AI job queue trigger, batch queue, and status handlers.
- `internal/handler/ai_tag_handler_test.go` - Route-level TDD coverage for AI trigger endpoints.
- `internal/handler/routes.go` - Dependency-aware route registration for all tag APIs.
- `internal/handler/routes_test.go` - Route registration verification.
- `cmd/server/main.go` - Runtime repository, governance service, and job manager bootstrap.
- `internal/repository/tag_repository.go` - Tag delete support for handler-driven cleanup.
- `internal/repository/image_repository.go` - Nullable `phash` reads and image lookup by ID.
- `internal/repository/job_repository.go` - Async job lookup by type for AI status queries.

## Decisions Made
- Used optional dependency injection in `SetupRoutes` so the previous `handler.SetupRoutes(r)` call site stays valid while the server can pass real repositories and services.
- Returned grouped image tag payloads with tag labels resolved from `TagRepository`, matching the review UI's confirmed/pending/rejected sections.
- Mapped `async_jobs.status = ready` to API response `queued` so the REST contract stays user-facing even though the queue internals use `ready`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed nullable image hash reads breaking image lookups**
- **Found during:** task 2 (实现图片标签关联 API)
- **Issue:** seeded and real image rows can leave `images.phash` as `NULL`, but `ImageRepository` scanned directly into `int64`, causing handler lookups to fail with 500 responses.
- **Fix:** added `FindByID` and switched image selects to `COALESCE(phash, 0)`.
- **Files modified:** `internal/repository/image_repository.go`
- **Verification:** `go test -v ./internal/handler/... -run TestImageTag`
- **Committed in:** `d7ee964`

**2. [Rule 2 - Missing Critical] Extended repositories for handler-side cleanup and status lookup**
- **Found during:** task 1 and task 3
- **Issue:** planned handlers needed tag deletion and image-specific async job lookup, but repositories exposed neither capability.
- **Fix:** added `TagRepository.Delete` and `JobRepository.FindByType` to support delete cleanup and AI status resolution.
- **Files modified:** `internal/repository/tag_repository.go`, `internal/repository/job_repository.go`
- **Verification:** `go test -v ./internal/handler/...` and `go build ./...`
- **Committed in:** `87dda2f`, `c7515d6`

**3. [Rule 2 - Missing Critical] Wired runtime server dependencies instead of leaving new routes inert**
- **Found during:** task 4 (更新路由注册)
- **Issue:** registering routes alone would leave the main server using placeholder handlers because no repositories, services, or job manager were injected.
- **Fix:** added dependency-aware `SetupRoutes` and server bootstrap for repositories, governance service, job manager, and AI worker registration.
- **Files modified:** `internal/handler/routes.go`, `cmd/server/main.go`
- **Verification:** `go test -v ./internal/handler/... -run TestRoutes` and `go build ./...`
- **Committed in:** `64fa3d3`

---

**Total deviations:** 3 auto-fixed (1 bug, 2 missing critical)
**Impact on plan:** All deviations were required to make the planned APIs functional in tests and at runtime. No unrelated scope was added.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Tag management, image review, and AI trigger APIs are ready for the Phase 03-04 Flutter tag frontend.
- AI background jobs still depend on valid provider credentials at runtime, but the API and queue integration are now in place.

## Self-Check
PASSED

---
*Phase: 03-ai*
*Completed: 2026-03-15*
