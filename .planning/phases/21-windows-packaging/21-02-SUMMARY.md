---
phase: 21-windows-packaging
plan: 02
subsystem: ui
tags: [windows-packaging, flutter, startup, manifest, shutdown, diagnostics]
requires:
  - phase: 21-windows-packaging
    provides: portable runtime layout, packaged sidecar executable contract, and startup diagnostics schema
provides:
  - Flutter-first packaged desktop bootstrap that starts the bundled Go runtime and waits for runtime readiness
  - classified startup failure surface with bundle-local log paths for go, python, and startup-chain failures
  - packaged manifest-path handoff and graceful Go shutdown with bounded kill fallback
affects: [21-03, 21-04, packaged-bootstrap, smoke-verification]
tech-stack:
  added: []
  patterns: [flutter-first packaged bootstrap, bundle-local manifest handoff, startup failure screen, bounded shutdown fallback]
key-files:
  created:
    - flutter_app/lib/bootstrap/packaged_desktop_bootstrap.dart
    - flutter_app/lib/widgets/startup/startup_failure_screen.dart
    - flutter_app/test/bootstrap/packaged_desktop_bootstrap_test.dart
  modified:
    - flutter_app/lib/bootstrap/runtime_manifest_loader_io.dart
    - flutter_app/lib/main.dart
    - flutter_app/test/bootstrap/runtime_manifest_loader_test.dart
key-decisions:
  - "Flutter remains the single user-facing launcher and blocks normal shell rendering until the packaged Go runtime writes the bundle-local runtime manifest or startup diagnostics."
  - "Packaged manifest resolution now prefers ACG_RUNTIME_MANIFEST_PATH, then a bundle-local runtime path derived from the executable directory, before falling back to the temp path used for development compatibility."
  - "App shutdown posts to /shutdown when a packaged runtime base URL is known, then force-kills the Go child if it does not exit within five seconds."
patterns-established:
  - "Packaged bootstrap: classify go/python/startup_chain failures from startup-error.json and always surface bundle-local log paths."
  - "Desktop startup wiring: main.dart gates normal app startup on packaged bootstrap success and routes bootstrap failure to a dedicated blocking screen."
requirements-completed: [OPS-03]
duration: not-recorded
completed: 2026-04-05
---

# Phase 21 Plan 02: Windows Packaging Summary

**Flutter-first packaged startup orchestration with manifest wait, classified startup failure UI, manifest path handoff, and bounded Go shutdown.**

## Performance

- **Duration:** not recorded in session metadata
- **Started:** not recorded in session metadata
- **Completed:** 2026-04-05T00:00:00Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Added `PackagedDesktopBootstrap` to resolve bundle-local runtime paths, launch the packaged Go executable, wait for `runtime/runtime-manifest.json`, and classify startup diagnostics.
- Added `StartupFailureScreen` and updated `main.dart` so packaged startup failures block normal shell rendering with clear go/python/startup-chain messaging plus log locations.
- Updated manifest path resolution and shutdown handling so packaged startup uses `ACG_RUNTIME_MANIFEST_PATH` or a bundle-local runtime path and app exit triggers graceful Go shutdown with kill fallback.

## Task Commits

No commits were created because this execution request did not include a commit instruction.

## Files Created/Modified
- `flutter_app/lib/bootstrap/packaged_desktop_bootstrap.dart` - Implements packaged desktop startup, failure classification, manifest wait, and shutdown behavior.
- `flutter_app/lib/widgets/startup/startup_failure_screen.dart` - Renders the blocking packaged startup failure surface with failure class and log paths.
- `flutter_app/lib/bootstrap/runtime_manifest_loader_io.dart` - Prefers env-driven or bundle-local manifest resolution before the development temp fallback.
- `flutter_app/lib/main.dart` - Runs packaged bootstrap before normal shell startup and routes failures to `StartupFailureScreen`.
- `flutter_app/test/bootstrap/packaged_desktop_bootstrap_test.dart` - Covers packaged startup, failure mapping, failure-screen rendering, and shutdown behavior.
- `flutter_app/test/bootstrap/runtime_manifest_loader_test.dart` - Covers env preference and bundle-local manifest path fallback.

## Decisions Made
- Kept the Flutter executable as the only visible launch entry and treated the runtime manifest as the readiness barrier before normal shell rendering.
- Derived packaged manifest fallback from the executable directory only when the packaged Go binary exists, preserving temp-path behavior for development compatibility.
- Reused the manifest-reported base URL for shutdown instead of introducing a second packaged control channel.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- `flutter test` emitted a transient `git fetch --tags` network error during one red-phase run before the actual test compilation errors; the plan verification commands still passed successfully once implementation was complete.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- The packaged Flutter launcher now owns Go startup, manifest readiness gating, startup error surfacing, and shutdown cleanup expected by bundle assembly work.
- Packaging pipeline work can rely on the fixed runtime contract: `runtime/runtime-manifest.json`, `runtime/diagnostics/startup-error.json`, and bundle-local log paths.

---
*Phase: 21-windows-packaging*
*Completed: 2026-04-05*
