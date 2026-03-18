---
phase: 06-optimization-deployment
plan: 02
subsystem: admin-dashboard-backend
tags: [admin, http-api, basic-auth, task-management]
dependency_graph:
  requires:
    - internal/config/config.go (AdminConfig)
    - internal/service/admin_service.go (AdminService)
    - internal/worker/job_manager.go (pause/resume)
  provides:
    - internal/handler/admin_handler.go (HTTP endpoints)
    - /admin/api/* routes
  affects:
    - cmd/server/main.go (bootstrap wiring)
tech_stack:
  - Go + Gin for HTTP
  - Basic Auth for protection
  - Service/Handler pattern
key_files:
  created:
    - internal/handler/admin_handler.go
    - internal/handler/admin_handler_test.go
  modified:
    - internal/handler/routes.go
    - cmd/server/main.go
decisions:
  - Basic Auth over JWT/sessions (simple local/internal use)
  - Interface-based admin service (testable with mocks)
  - Config-driven auth (skip if no credentials configured)
---

# Phase 6 Plan 2: Admin HTTP Endpoints Summary

## Objective

Expose protected admin HTTP endpoints and wire them into server bootstrap, completing the backend contract for the admin dashboard.

## Implementation

### 1. Admin Handler (`internal/handler/admin_handler.go`)

Created the `AdminHandler` with the following endpoints:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/api/summary` | GET | Returns health, config, tasks, and library stats |
| `/admin/api/jobs` | GET | Returns recent jobs (limit query param) |
| `/admin/api/actions/scan` | POST | Triggers manual scan |
| `/admin/api/actions/jobs/pause` | POST | Pauses job processing |
| `/admin/api/actions/jobs/resume` | POST | Resumes job processing |
| `/admin/api/actions/jobs/retry-failed` | POST | Retries all failed jobs |

### 2. Basic Auth Protection

- Config-driven username/password from `config.yaml` (or env vars)
- Skips auth if both username and password are empty (dev mode)
- Returns 401 with `WWW-Authenticate` header when credentials missing/invalid

### 3. Route Registration (`internal/handler/routes.go`)

- Added `AdminSvc` and `AdminCfg` to `Dependencies` struct
- Registered `/admin/api` group with Basic Auth middleware
- All admin endpoints protected under `/admin/api/*`

### 4. Server Bootstrap (`cmd/server/main.go`)

- Created `collectionRepo` for admin service
- Instantiated `AdminService` with all required dependencies
- Passed `AdminSvc` and `AdminCfg` to `SetupRoutes`

### 5. Tests (`internal/handler/admin_handler_test.go`)

All 6 tests pass:
- `TestAdminHandler_Summary_AuthRequired` - 401 without auth, 200 with valid credentials
- `TestAdminHandler_GetSummary` - Returns summary data
- `TestAdminHandler_TriggerScan` - Triggers scan job
- `TestAdminHandler_RetryFailedJobs` - Retries failed jobs
- `TestAdminHandler_Pause` - Pauses background tasks
- `TestAdminHandler_Resume` - Resumes background tasks

## Verification

```bash
# Admin handler tests
go test ./internal/handler/... -run Admin -count=1
# PASS

# Service/repository tests (from Task 1)
go test ./internal/repository/... ./internal/service/... -run "Admin|JobManager|JobRepository" -count=1
# PASS

# Build verification
go build ./cmd/server
# PASS
```

## Deviations from Plan

None - plan executed exactly as written.

## Auth Gates

None - Basic Auth is self-contained in the handler, no external services required.

## Metrics

| Metric | Value |
|--------|-------|
| Duration | ~15 min |
| Tasks | 1 (Task 2 only, Task 1 pre-completed) |
| Files Created | 2 |
| Files Modified | 2 |
| Tests | 6 passing |

---

**Plan Status:** Complete ✓

**Task 2 Complete:** Admin HTTP endpoints exposed and wired into server bootstrap.