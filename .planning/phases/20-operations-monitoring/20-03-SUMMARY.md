---
phase: 20-operations-monitoring
plan: 03
subsystem: ui
tags: [flutter, fluent_ui, monitoring, websocket, desktop-shell]
requires:
  - phase: 20-02
    provides: typed monitoring contracts, provider-owned monitoring state, and websocket lifecycle handling
provides:
  - desktop navigation integration for the operations monitoring workspace
  - dual-area monitoring workspace with batch-first drilldown and sidecar diagnostics
  - sidecar restart confirmation UX with impact messaging and runtime metrics
affects: [phase-20, desktop-monitoring-workspace, operations-monitoring-shell]
tech-stack:
  added: []
  patterns:
    - fluent desktop page entry points resolve to workspace widgets backed by providers
    - monitoring workspace uses provider-owned websocket connect/disconnect at page lifetime boundaries
    - monitoring cards use semantic status bands plus compact metric tiles for operator diagnostics
key-files:
  created:
    - flutter_app/lib/widgets/monitoring/monitoring_workspace.dart
    - flutter_app/lib/widgets/monitoring/batch_list_section.dart
    - flutter_app/lib/widgets/monitoring/sidecar_diagnostic_section.dart
    - flutter_app/test/widgets/monitoring/monitoring_workspace_test.dart
    - flutter_app/test/widgets/monitoring/sidecar_diagnostic_section_test.dart
  modified:
    - flutter_app/lib/providers/navigation_provider.dart
    - flutter_app/lib/app/fluent_app_shell.dart
    - flutter_app/lib/app/fluent_screens.dart
    - flutter_app/lib/config/api_config.dart
    - flutter_app/lib/models/monitoring_models.dart
    - flutter_app/lib/main.dart
    - flutter_app/test/providers/navigation_provider_test.dart
    - flutter_app/test/navigation_provider_test.dart
    - flutter_app/test/app/fluent_app_shell_test.dart
key-decisions:
  - "Kept 运营监控 as a desktop-only 6th shell entry while leaving the existing Material navigation structure unchanged."
  - "Extended BatchRow with created/finished timestamps so the monitoring list can honor the UI contract without introducing new endpoints."
  - "Registered MonitoringProvider in the main app tree so the shell-mounted monitoring page can connect through the real provider lifecycle."
patterns-established:
  - "Monitoring batch rows: semantic badge + linear progress + formatted timestamp + inline drilldown details."
  - "Sidecar diagnostics: top color band, display-sized state text, compact runtime metrics, and destructive-confirm restart flow."
requirements-completed: [OPS-01, OPS-02]
duration: session
completed: 2026-04-05
---

# Phase 20 Plan 03: Operations monitoring workspace summary

**Desktop operations monitoring now ships as a real 6th Fluent shell page with batch drilldown, sidecar diagnostics, reconnect/retry handling, and restart confirmation UX.**

## Performance

- **Duration:** session
- **Started:** 2026-04-05T00:00:00Z
- **Completed:** 2026-04-05T00:00:00Z
- **Tasks:** 3
- **Files modified:** 14

## Accomplishments

- Added the 6th Fluent shell destination, monitoring endpoints, and real workspace entry wiring.
- Added the dual-area monitoring workspace with websocket status, unavailable/reconnect states, batch progress rows, and task drilldown.
- Added the sidecar diagnostic card with semantic status bands, runtime metrics, error summary, and restart confirmation.

## Task Commits

Each task was committed atomically:

1. **Task 1 RED** — `a166a0e` `test(20-03): add failing navigation and API config coverage`
2. **Task 1 GREEN** — `eefb1f9` `feat(20-03): add 运营监控 navigation entry, screens, and API config`
3. **Task 2** — `18eb456` `feat(20-03): add monitoring workspace with dual-area layout and batch list`
4. **Task 3** — `27ef26a` `feat(20-03): add sidecar diagnostic section with restart confirmation`
5. **Verification fix** — `3941698` `fix(20-03): wire monitoring provider into app bootstrap`

## Files Created/Modified

- `flutter_app/lib/providers/navigation_provider.dart` - expands the desktop navigation contract to 6 entries.
- `flutter_app/lib/app/fluent_app_shell.dart` - adds the 运营监控 pane item to the Fluent shell.
- `flutter_app/lib/app/fluent_screens.dart` - routes the Fluent monitoring page to the workspace widget.
- `flutter_app/lib/config/api_config.dart` - exposes admin monitoring REST and websocket URLs.
- `flutter_app/lib/models/monitoring_models.dart` - adds batch timestamp parsing for monitoring rows.
- `flutter_app/lib/widgets/monitoring/monitoring_workspace.dart` - owns page lifecycle, reconnect banner, unavailable state, and dual-area layout.
- `flutter_app/lib/widgets/monitoring/batch_list_section.dart` - renders batch rows, progress, timestamps, semantic tags, and task drilldown.
- `flutter_app/lib/widgets/monitoring/sidecar_diagnostic_section.dart` - renders status diagnostics, metrics, recent error summary, and restart confirmation.
- `flutter_app/lib/main.dart` - registers monitoring service/provider in the application provider tree.
- `flutter_app/test/providers/navigation_provider_test.dart` - RED/GREEN coverage for the navigation/API config contract.
- `flutter_app/test/navigation_provider_test.dart` - aligns the legacy shared navigation expectations to the 6-item contract.
- `flutter_app/test/app/fluent_app_shell_test.dart` - verifies the monitoring pane exists and can mount.
- `flutter_app/test/widgets/monitoring/monitoring_workspace_test.dart` - covers workspace lifecycle, reconnect/retry states, batch rendering, and selection.
- `flutter_app/test/widgets/monitoring/sidecar_diagnostic_section_test.dart` - covers state bands, restart confirmation, loading state, metrics, and error empty state.

## Decisions Made

- Reused the Phase 17/19 workspace pattern by keeping page composition in `fluent_screens.dart` and moving monitoring behavior into dedicated widgets.
- Derived sidecar metrics directly from `MonitoringOverview` instead of introducing another provider layer.
- Kept restart impact fallback tied to running task count so the dialog still explains impact before any restart call returns.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added batch timestamp parsing to the Flutter monitoring model**
- **Found during:** Task 2 (monitoring workspace with dual-area layout and batch list)
- **Issue:** The existing Flutter `BatchRow` model did not expose timestamps, but the UI contract required rendered batch timestamps.
- **Fix:** Added `createdAt` and `finishedAt` parsing/storage on `BatchRow` and used those values in the batch list UI.
- **Files modified:** `flutter_app/lib/models/monitoring_models.dart`, `flutter_app/lib/widgets/monitoring/batch_list_section.dart`, `flutter_app/test/widgets/monitoring/monitoring_workspace_test.dart`
- **Verification:** `flutter test test/widgets/monitoring/monitoring_workspace_test.dart`
- **Committed in:** `18eb456`

**2. [Rule 3 - Blocking] Wired MonitoringProvider into the app bootstrap after shell verification failed**
- **Found during:** Final verification
- **Issue:** The new Fluent monitoring page mounted from the real app shell without a registered `MonitoringProvider`, causing `ProviderNotFoundException`.
- **Fix:** Registered `MonitoringService` + `MonitoringProvider` in `flutter_app/lib/main.dart` and updated the shell/navigation tests to match the shared 6-item contract.
- **Files modified:** `flutter_app/lib/main.dart`, `flutter_app/test/app/fluent_app_shell_test.dart`, `flutter_app/test/navigation_provider_test.dart`
- **Verification:** `flutter test test/providers/navigation_provider_test.dart test/navigation_provider_test.dart test/app/fluent_app_shell_test.dart test/services/monitoring_service_test.dart test/providers/monitoring_provider_test.dart test/widgets/monitoring/monitoring_workspace_test.dart test/widgets/monitoring/sidecar_diagnostic_section_test.dart`
- **Committed in:** `3941698`

---

**Total deviations:** 2 auto-fixed (1 missing critical, 1 blocking)
**Impact on plan:** Both auto-fixes were required for contract-complete UI behavior and shell integration. No scope creep beyond the monitoring workspace delivery.

## Issues Encountered

- `flutter test` for the entire `flutter_app` still reports unrelated pre-existing failures outside the touched monitoring scope, including `test/app/material_app_shell_test.dart`, `test/providers/theme_provider_test.dart`, and adaptive navigation tests.

## User Setup Required

None - no external service configuration required.

## Verification Evidence

- `flutter test test/providers/navigation_provider_test.dart test/app/fluent_app_shell_test.dart` → `All tests passed!`
- `flutter test test/widgets/monitoring/monitoring_workspace_test.dart` → `All tests passed!`
- `flutter test test/widgets/monitoring/sidecar_diagnostic_section_test.dart` → `All tests passed!`
- `flutter test` → fails outside touched scope; monitoring-triggered integration issue was fixed, remaining failures are pre-existing/unrelated.
- `flutter test test/providers/navigation_provider_test.dart test/navigation_provider_test.dart test/app/fluent_app_shell_test.dart test/services/monitoring_service_test.dart test/providers/monitoring_provider_test.dart test/widgets/monitoring/monitoring_workspace_test.dart test/widgets/monitoring/sidecar_diagnostic_section_test.dart` → `All tests passed!`
- `lsp_diagnostics` on all touched Dart files → `No diagnostics found`

## Self-Check

PASSED

- All plan 20-03 tasks were implemented and committed atomically.
- Touched-scope Flutter verification passed after final integration fixes.
- `.planning/phases/20-operations-monitoring/20-03-SUMMARY.md` was created.
- Tracking files (`.planning/ROADMAP.md`, `.planning/STATE.md`, `.planning/REQUIREMENTS.md`) were intentionally left untouched by this work.

## Next Phase Readiness

- Phase 20 now has desktop navigation, monitoring workspace, and sidecar diagnostics end-to-end in Flutter.
- Remaining verification noise is outside the touched monitoring scope and can be addressed separately if a repo-wide green baseline is required.

---

*Phase: 20-operations-monitoring*
*Completed: 2026-04-05*
