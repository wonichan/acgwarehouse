---
phase: 20-operations-monitoring
plan: 02
subsystem: ui
tags: [flutter, provider, websocket, monitoring, admin-api]
requires:
  - phase: 20-01
    provides: authenticated monitoring websocket, sidecar restart endpoint, and monitoring REST contracts
provides:
  - typed Flutter monitoring models for overview, batches, tasks, restart impact, and websocket events
  - admin monitoring service methods for overview, batch, task, and sidecar restart endpoints
  - provider-owned monitoring state with websocket lifecycle, reconnect backoff, retry, and batch drilldown
affects: [phase-20-03, flutter-monitoring-page, desktop-monitoring-workspace]
tech-stack:
  added: [web_socket_channel]
  patterns:
    - typed admin monitoring contracts in the Flutter data layer
    - provider-managed websocket lifecycle with explicit disconnect and exponential backoff reconnect
    - retry-driven unavailable state without caching stale monitoring snapshots
key-files:
  created:
    - flutter_app/lib/models/monitoring_models.dart
    - flutter_app/lib/services/monitoring_service.dart
    - flutter_app/lib/providers/monitoring_provider.dart
    - flutter_app/test/services/monitoring_service_test.dart
    - flutter_app/test/providers/monitoring_provider_test.dart
  modified:
    - flutter_app/pubspec.yaml
    - flutter_app/pubspec.lock
key-decisions:
  - "Monitoring REST access stays under ApiConfig.hostUrl because admin endpoints live outside /api/v1."
  - "MonitoringProvider owns websocket connection state and reconnects only when the provider was not explicitly disconnected."
  - "REST failures surface a serviceUnavailable edge state with retry instead of preserving stale monitoring data."
patterns-established:
  - "Monitoring contract parsing: decode backend admin payloads into typed models before provider state updates."
  - "Monitoring reconnect flow: websocket error/close -> mark disconnected -> sleep with exponential backoff -> reconnect."
requirements-completed: [OPS-01, OPS-02]
duration: session
completed: 2026-04-05
---

# Phase 20 Plan 02: Monitoring Flutter data layer summary

**Typed Flutter monitoring contracts now cover admin overview, batch/task drilldown, sidecar restart impact, and provider-managed realtime websocket state.**

## Performance

- **Duration:** session
- **Completed:** 2026-04-05T00:00:00Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- Added typed monitoring models for backend overview, batches, tasks, restart impact, and websocket events.
- Added `MonitoringService` admin REST methods that consume `/admin/api` endpoints with optional Basic Auth headers.
- Added `MonitoringProvider` with initial load, batch drilldown, websocket lifecycle, reconnect backoff, restart refresh, and unavailable-state retry.

## Task Commits

1. **Task 1 RED** — `636d71d` `test(20-02): add failing monitoring service coverage`
2. **Task 1 GREEN** — `c783ca3` `feat(20-02): add typed monitoring models and service contracts`
3. **Task 1 deps** — `851ee59` `deps(20-02): add web_socket_channel to flutter_app`
4. **Task 2 RED** — `3cbc863` `test(20-02): add failing monitoring provider coverage`
5. **Task 2 GREEN** — `6789638` `feat(20-02): add monitoring provider with WebSocket lifecycle and reconnection`

## Files Created/Modified

- `flutter_app/lib/models/monitoring_models.dart` - typed contracts for overview, batches, tasks, sidecar status, runtime metrics, websocket events, and restart impact.
- `flutter_app/lib/services/monitoring_service.dart` - admin monitoring REST client for overview, batches, tasks, and sidecar restart.
- `flutter_app/lib/providers/monitoring_provider.dart` - provider-owned monitoring state with connect/disconnect, retry, reconnect, restart refresh, and drilldown loading.
- `flutter_app/test/services/monitoring_service_test.dart` - RED/GREEN contract coverage for parsing and admin endpoint usage.
- `flutter_app/test/providers/monitoring_provider_test.dart` - RED/GREEN coverage for provider lifecycle, websocket updates, reconnect, retry, restart, and sidecar-stopped behavior.
- `flutter_app/pubspec.yaml` - monitoring websocket dependency declaration.
- `flutter_app/pubspec.lock` - resolved lockfile for the websocket dependency.

## Decisions Made

- Used optional auth-header injection in `MonitoringService` so the admin client can attach Basic Auth without coupling the data layer to a specific settings provider yet.
- Kept websocket URI creation injectable from the provider constructor so the workspace can decide host translation and tests can drive deterministic channels.
- Refreshed overview and batches after `restartSidecar()` while keeping the existing websocket session active, matching the plan’s no-auto-disconnect requirement.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Captured `pubspec.lock` with the websocket dependency update**
- **Found during:** Task 1 dependency install
- **Issue:** Adding `web_socket_channel` changed both `flutter_app/pubspec.yaml` and `flutter_app/pubspec.lock`; committing only the manifest would leave dependency resolution non-reproducible.
- **Fix:** Ran `flutter pub get` and committed the refreshed lockfile with the dependency declaration.
- **Files modified:** `flutter_app/pubspec.yaml`, `flutter_app/pubspec.lock`
- **Verification:** `flutter test test/services/monitoring_service_test.dart` and `flutter test test/providers/monitoring_provider_test.dart`
- **Committed in:** `851ee59`

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** The auto-fix was required for a valid Flutter dependency update. No scope creep.

## Issues Encountered

- Existing unrelated workspace changes were already present in planning/docs/config files and `data/`; task commits staged only Phase 20-02 files.

## User Setup Required

None - no external service configuration required.

## Verification Evidence

- `flutter test test/services/monitoring_service_test.dart` → `All tests passed!`
- `flutter test test/providers/monitoring_provider_test.dart` → `All tests passed!`
- `lsp_diagnostics` on touched Dart files → no diagnostics found

## Self-Check

PASSED

- Plan 20-02 tasks were implemented with the requested atomic commits.
- Touched-scope tests passed for service and provider coverage.
- `.planning/phases/20-operations-monitoring/20-02-SUMMARY.md` was generated.
- Tracking files (`.planning/ROADMAP.md`, `.planning/STATE.md`, `.planning/REQUIREMENTS.md`) were intentionally left untouched per user instruction.

## Next Phase Readiness

- Phase 20-03 can consume typed provider state instead of raw maps when building the monitoring workspace UI.
- Backend monitoring contracts from 20-01 now have matching Flutter-side parsing, retry, and websocket orchestration.

---

*Phase: 20-operations-monitoring*
*Completed: 2026-04-05*
