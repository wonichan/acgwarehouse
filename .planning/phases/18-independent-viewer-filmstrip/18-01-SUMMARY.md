---
phase: 18-independent-viewer-filmstrip
plan: 01
subsystem: ui
tags: [flutter, windows, desktop, multi-window, viewer]

requires:
  - phase: 17-desktop-shell-foundation
    provides: desktop shell top bar, gallery workspace, and real import baseline for the Windows app shell
provides:
  - Serializable result-set-scoped `ViewerSession` payloads for spawned viewer windows
  - `ViewerWindowService` launch coordination with dedicated secondary-window bootstrap translation
  - `ViewerWindowApp` placeholder host and Windows bootstrap branch for independent viewer windows
affects: [18-02, 18-03, independent-viewer-filmstrip]

tech-stack:
  added: [desktop_multi_window]
  patterns: [serialized viewer-session payloads, dedicated viewer bootstrap branch, shared desktop window policy helpers]

key-files:
  created:
    - .planning/phases/18-independent-viewer-filmstrip/18-01-SUMMARY.md
    - flutter_app/lib/models/viewer_session.dart
    - flutter_app/lib/services/viewer_window_service.dart
    - flutter_app/lib/app/viewer_window_app.dart
    - flutter_app/test/models/viewer_session_test.dart
    - flutter_app/test/services/viewer_window_service_test.dart
  modified:
    - .planning/ROADMAP.md
    - .planning/STATE.md
    - flutter_app/lib/main.dart
    - flutter_app/lib/utils/window_manager.dart
    - flutter_app/pubspec.yaml
    - flutter_app/pubspec.lock
    - flutter_app/windows/runner/main.cpp
    - flutter_app/windows/flutter/generated_plugin_registrant.cc
    - flutter_app/windows/flutter/generated_plugins.cmake
    - flutter_app/linux/flutter/generated_plugin_registrant.cc
    - flutter_app/linux/flutter/generated_plugins.cmake
    - flutter_app/macos/Flutter/GeneratedPluginRegistrant.swift

key-decisions:
  - "Use `desktop_multi_window` as the single true multi-window dependency for Windows viewer spawning."
  - "Keep spawned viewer windows isolated through serialized `ViewerSession` payloads instead of provider reach-through."
  - "Apply fixed centered viewer window defaults with no persisted window-memory restore in Phase 18-01."

patterns-established:
  - "Viewer launch flow: result set snapshot -> `ViewerSession` -> `ViewerWindowService` -> encoded multi-window bootstrap payload."
  - "Desktop startup branch: main shell uses `MyApp`, spawned viewer windows use `ViewerWindowApp`."
  - "Window policy split: main shell and viewer windows use distinct helpers from `window_manager.dart`."

requirements-completed: [VIEW-01]

duration: 15 min
completed: 2026-04-05
---

# Phase 18 Plan 01: Viewer Host Bootstrap Summary

**Windows desktop now has a real secondary viewer-window bootstrap path with serializable viewer sessions, a dedicated launch coordinator, and a placeholder spawned-window host ready for Phase 18-02 workspace mounting.**

## Performance

- **Duration:** 15 min
- **Started:** 2026-04-05T02:30:53+08:00
- **Completed:** 2026-04-05T02:45:00+08:00
- **Tasks:** 2
- **Files modified:** 13

## Accomplishments
- Added `ViewerSession` / `ViewerSessionItem` contracts so gallery or search result snapshots can cross window boundaries safely.
- Added `ViewerWindowService` seams for title formatting, viewer window policy, multi-window payload encoding, and bootstrap parsing.
- Added a Windows viewer bootstrap branch in `main.dart`, a dedicated `ViewerWindowApp` placeholder host, and the required multi-window dependency wiring.

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): failing viewer session and coordinator coverage** - `b57357a` (`test`)
2. **Task 1 (GREEN): viewer session model and launch seam** - `2bd6f3f` (`feat`)
3. **Task 2 (RED): failing desktop bootstrap coverage** - `ad46665` (`test`)
4. **Task 2 (GREEN): Windows secondary-window bootstrap** - `bdda89a` (`feat`)
5. **Task 2 (REFACTOR): shared desktop launch helpers** - `f1b7e75` (`refactor`)

## Files Created/Modified
- `flutter_app/lib/models/viewer_session.dart` - serializable viewer-session model built from `ImageModel` snapshots only
- `flutter_app/lib/services/viewer_window_service.dart` - launch coordinator, bootstrap payload parsing, and `desktop_multi_window` adapter
- `flutter_app/lib/app/viewer_window_app.dart` - placeholder spawned-window host for secondary viewer windows
- `flutter_app/lib/main.dart` - desktop bootstrap branch between main shell and viewer window host
- `flutter_app/lib/utils/window_manager.dart` - distinct main/viewer window policy helpers and viewer title builder
- `flutter_app/test/models/viewer_session_test.dart` - result-set snapshot and JSON round-trip coverage
- `flutter_app/test/services/viewer_window_service_test.dart` - launch translation and bootstrap parsing coverage
- `flutter_app/windows/runner/main.cpp` - viewer-aware initial window title selection during Windows startup

## Decisions Made
- Chose `desktop_multi_window` as the only new multi-window dependency to satisfy true secondary-window behavior without broader stack churn.
- Kept `ViewerWindowApp` intentionally minimal so 18-01 stops at host/bootstrap infrastructure and does not leak 18-02 workspace UI into this slice.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added generated plugin registrants after multi-window dependency install**
- **Found during:** Task 2 dependency integration
- **Issue:** Adding `desktop_multi_window` updated tracked Flutter desktop registrant files across supported platforms.
- **Fix:** Kept the generated registrants aligned with the new dependency so desktop builds remain consistent.
- **Files modified:** `flutter_app/windows/flutter/generated_plugin_registrant.cc`, `flutter_app/windows/flutter/generated_plugins.cmake`, `flutter_app/linux/flutter/generated_plugin_registrant.cc`, `flutter_app/linux/flutter/generated_plugins.cmake`, `flutter_app/macos/Flutter/GeneratedPluginRegistrant.swift`
- **Verification:** `flutter test` after `flutter pub get`
- **Committed in:** see follow-up docs/chore commit for phase closeout

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Required for dependency-consistent desktop bootstrap; no viewer UI scope creep introduced.

## Issues Encountered
- The refactor extraction briefly assumed all bootstrap payloads contained `logical_window_id` and `title`; restored fallback parsing so RED fixture payloads and future payload evolution remain compatible.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- 18-01 infrastructure is in place and verified.
- Ready for 18-02 reusable viewer workspace work on top of real spawned-window bootstrap.

---
*Phase: 18-independent-viewer-filmstrip*
*Completed: 2026-04-05*
