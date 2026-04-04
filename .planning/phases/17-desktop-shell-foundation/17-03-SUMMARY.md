---
phase: 17-desktop-shell-foundation
plan: 03
subsystem: ui
tags: [go, gin, flutter, desktop-shell, import]

requires:
  - phase: 17-01
    provides: desktop shell top-bar contract with persistent search/import/settings affordances
  - phase: 17-02
    provides: grid-first gallery workspace and persistent right-side filter panel
provides:
  - Product-facing desktop import trigger endpoint at POST /api/v1/images/scan
  - Flutter ImportService client targeting product image scan endpoint
  - Shell Import Library action wired to backend with lightweight queued/failure feedback
affects: [desktop-shell-foundation, 18-independent-viewer-filmstrip, 20-operations-monitoring]

tech-stack:
  added: []
  patterns: [thin endpoint delegation to AdminService.TriggerScan, widget-independent import client, shell-level lightweight infobar feedback]

key-files:
  created:
    - .planning/phases/17-desktop-shell-foundation/17-03-SUMMARY.md
    - flutter_app/lib/services/import_service.dart
    - flutter_app/test/services/import_service_test.dart
  modified:
    - internal/handler/image_handler.go
    - internal/handler/image_handler_test.go
    - internal/handler/routes.go
    - flutter_app/lib/config/api_config.dart
    - flutter_app/lib/app/fluent_app_shell.dart
    - flutter_app/test/app/desktop_shell_top_bar_test.dart

key-decisions:
  - "Keep import product endpoint thin and delegate queueing to existing AdminService.TriggerScan path."
  - "Introduce a dedicated ImportService instead of embedding HTTP logic in shell widgets."
  - "Limit shell import feedback to queued/failure text-only infobar messages to stay within Phase 17 scope."

patterns-established:
  - "Backend import trigger: POST /api/v1/images/scan -> ImageHandler.TriggerImport -> AdminService.TriggerScan"
  - "Top-bar import action: ImportService.triggerImport -> show 'Library import queued' or 'Library import could not start'"

requirements-completed: [DSK-01]

duration: 1h 35m
completed: 2026-04-05
---

# Phase 17 Plan 03: Import Endpoint & Shell Feedback Summary

**Delivered a real desktop import action path from top-bar button to backend queue trigger, with bounded queued/failure user feedback and test coverage on both Go and Flutter sides.**

## Performance

- **Duration:** 1h 35m
- **Started:** 2026-04-05T05:00:00Z
- **Completed:** 2026-04-05T06:35:00Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- Added product-facing `POST /api/v1/images/scan` handler backed by existing manual scan queue orchestration.
- Added `ImportService` and endpoint config wiring so Flutter posts to product-facing scan endpoint and parses queued/error responses.
- Wired shell `Import Library` action to the import client and surfaced lightweight success/failure info-bar feedback only.

## Task Commits

1. **Task 1/2 RED coverage:** `432683e` (`test`)
2. **Task 1 backend + Task 2 client wiring:** `5062021` (`feat`)
3. **Task 2 shell feedback wiring:** `5e8c3bf` (`feat`)

## Files Created/Modified
- `internal/handler/image_handler.go` - add `TriggerImport` handler with 202 queued and structured failure JSON
- `internal/handler/routes.go` - wire `/api/v1/images/scan` to real image handler path when dependencies are available
- `internal/handler/image_handler_test.go` - add endpoint contract coverage for success, failure, and no-placeholder routing
- `flutter_app/lib/config/api_config.dart` - add product-facing `imageScan` endpoint getter
- `flutter_app/lib/services/import_service.dart` - add dedicated import trigger client and typed result/exception
- `flutter_app/test/services/import_service_test.dart` - verify endpoint target, queued parsing, and failure exception contract
- `flutter_app/lib/app/fluent_app_shell.dart` - wire top-bar import action to ImportService and show lightweight feedback
- `flutter_app/test/app/desktop_shell_top_bar_test.dart` - verify queued/failure feedback copy from top-bar import action

## Decisions Made
- Reused existing scan orchestration (`AdminService.TriggerScan`) instead of introducing a new import subsystem.
- Kept product API response minimal (`status`, `job_id`, `error`) and avoided Phase 20 monitoring concerns.
- Kept shell feedback lightweight and textual (`queued`/`failure`) without adding dashboarding or progress UI.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Route dependency typing prevented test-safe handler injection**
- **Found during:** Task 1 route wiring
- **Issue:** `Dependencies.AdminSvc` was typed as concrete `*service.AdminService`, making route-level contract testing and decoupled handler wiring unnecessarily rigid.
- **Fix:** Switched `Dependencies.AdminSvc` to `AdminServiceInterface` so the route can depend on the established handler contract instead of a concrete type.
- **Files modified:** `internal/handler/routes.go`
- **Verification:** `go test ./internal/handler ./internal/service -run 'Test(ImageHandler_TriggerImport|TestAdminService_TriggerScan)' -count=1`
- **Committed in:** `5062021`

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** No scope creep; change was required to keep route wiring aligned with existing interface-driven handler contracts.

## Issues Encountered
- Initial RED for Flutter import tests surfaced missing import service symbols and endpoint accessor; resolved via planned GREEN implementation.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 17 now has top-bar search/import/settings access, grid workspace, and persistent filters completed across 17-01/17-02/17-03.
- Ready for Phase 18 viewer and filmstrip work with shell import path already product-wired.

---
*Phase: 17-desktop-shell-foundation*
*Completed: 2026-04-05*
