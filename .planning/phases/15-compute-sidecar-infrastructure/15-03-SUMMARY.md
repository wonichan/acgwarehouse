---
phase: 15-compute-sidecar-infrastructure
plan: 03
subsystem: infra
tags: [sidecar, admin-overview, degraded-availability, manifest, flutter, go]

requires:
  - phase: 15-01
    provides: Go-owned sidecar lifecycle and degraded startup semantics
  - phase: 15-02
    provides: runtime manifest Go discovery path for Flutter bootstrap
provides:
  - Sidecar diagnostics contract in admin task-platform overview (`state`, `last_probe_at`, `last_probe_result`, `last_error_summary`)
  - Regression coverage proving Go service remains usable under sidecar degraded conditions
  - Phase-closure verification matrix across Go diagnostics/health and Flutter manifest bootstrap
affects: [phase-16-duplicate-migration, phase-20-operations-monitoring, sidecar-observability]

tech-stack:
  added: []
  patterns: [admin diagnostics DTO, app-internal sidecar status provider, degraded probe-result normalization]

key-files:
  created: []
  modified:
    - internal/service/admin_service.go
    - internal/service/admin_service_test.go
    - internal/handler/admin_handler_test.go
    - internal/handler/routes_test.go
    - internal/app/app.go
    - internal/app/app_test.go

key-decisions:
  - "Expose sidecar observability only through admin overview, while keeping /health and /ready Go-scoped"
  - "Map sidecar degraded/stopped runtime states to failed probe diagnostics for operator-facing clarity"

patterns-established:
  - "Admin diagnostics extension pattern: add bounded DTO fields instead of inflating base health endpoints"
  - "App-to-service observability injection pattern: Go-internal provider passes runtime state snapshots"

requirements-completed: [COMP-01, COMP-02, COMP-06]

duration: 6 min
completed: 2026-04-04
---

# Phase 15 Plan 03: Observability and Degraded Availability Summary

**Admin overview now exposes bounded sidecar diagnostics while degraded sidecar scenarios keep Go serving paths usable and verifiable across Go+Flutter checks.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-04-04T01:08:34+08:00
- **Completed:** 2026-04-04T01:14:13+08:00
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments
- Extended `TaskPlatformOverview` with sidecar diagnostics fields and provider-based population in `AdminService`
- Added degraded regressions for sidecar startup failure and post-start crash while preserving Go health/readiness behavior
- Locked phase closure with cross-stack verification matrix (`go test` diagnostics/health + `flutter test` runtime manifest bootstrap)

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: Extend admin overview contract with sidecar diagnostics** - `c3eb2f8` (test)
2. **Task 1 GREEN: Extend admin overview contract with sidecar diagnostics** - `4799ab0` (feat)
3. **Task 2 RED: Add degraded-availability regressions** - `38ff2a2` (test)
4. **Task 2 GREEN: Preserve degraded availability + diagnostics propagation** - `7374478` (fix)
5. **Task 3 RED: Add phase-closure cross-stack checks** - `698ed80` (test)
6. **Task 3 GREEN: Lock phase-15 cross-stack verification matrix** - `bb47d96` (test)

**Plan metadata:** pending final docs commit

## Files Created/Modified
- `internal/service/admin_service.go` - Added sidecar diagnostics DTO + provider interface and overview wiring
- `internal/service/admin_service_test.go` - Added diagnostics contract assertions for overview payload
- `internal/handler/admin_handler_test.go` - Added endpoint serialization assertions for new sidecar diagnostics fields
- `internal/handler/routes_test.go` - Added regression checks that `/health` and `/ready` remain free of sidecar payloads
- `internal/app/app.go` - Added app-side sidecar status snapshot provider and degraded/stopped probe-result mapping
- `internal/app/app_test.go` - Added startup-failure, post-start crash, and stopped-sidecar diagnostics regressions

## Decisions Made
- Bound sidecar observability to admin overview contract; base `/health` and `/ready` remain Go-only.
- Normalize sidecar `degraded` and `stopped` to failed probe result to keep operator diagnostics deterministic.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Flutter verification command failed in repository root**
- **Found during:** Task 3 verification
- **Issue:** `flutter test` initially ran from repository root and failed due missing `pubspec.yaml`
- **Fix:** Re-ran the Flutter verification command with `workdir=flutter_app`
- **Files modified:** None
- **Verification:** `flutter test test/bootstrap/runtime_manifest_loader_test.dart test/config/api_config_test.dart` passed
- **Committed in:** N/A (verification-command correction only)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** No scope change; correction only affected verification command context.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 15 closure checks now protect sidecar diagnostics visibility and degraded availability invariants.
- Ready for phase aggregation/verification and Phase 16 duplicate-detection migration planning.

## Self-Check: PASSED
- Summary file exists on disk.
- All six task commit hashes are present in git history.
