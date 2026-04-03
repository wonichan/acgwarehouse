---
phase: 15-compute-sidecar-infrastructure
plan: 02
subsystem: infra
tags: [go, flutter, runtime-manifest, bootstrap, dynamic-port]
requires:
  - phase: 15-compute-sidecar-infrastructure
    provides: sidecar runtime lifecycle baseline from 15-01
provides:
  - Go runtime manifest writer with atomic publish and cleanup
  - Flutter startup manifest loader that applies go.base_url pre-bootstrap
  - Development-only localhost fallback path separated from runtime contract
affects: [phase-16-compute-migration, flutter-startup, deployment-runtime-discovery]
tech-stack:
  added: []
  patterns: [temp-file-rename atomic manifest write, startup runtime URL discovery]
key-files:
  created:
    - internal/app/runtime_manifest.go
    - flutter_app/lib/bootstrap/runtime_manifest_loader.dart
    - flutter_app/lib/bootstrap/runtime_manifest_loader_io.dart
    - flutter_app/lib/bootstrap/runtime_manifest_loader_stub.dart
    - flutter_app/test/bootstrap/runtime_manifest_loader_test.dart
    - flutter_app/test/config/api_config_test.dart
  modified:
    - internal/app/app.go
    - cmd/server/main.go
    - internal/app/runtime_manifest_test.go
    - flutter_app/lib/config/api_config.dart
    - flutter_app/lib/providers/config_provider.dart
    - flutter_app/lib/main.dart
key-decisions:
  - "Manifest schema keeps Flutter contract limited to go.base_url plus metadata"
  - "Flutter startup resolves runtime URL before runApp and keeps localhost as dev-only fallback"
patterns-established:
  - "Runtime discovery pattern: Go publishes manifest, Flutter consumes before first API usage"
  - "Platform-safe bootstrap: conditional IO loader prevents non-IO targets from hard dependency"
requirements-completed: [COMP-06]
duration: 8min
completed: 2026-04-03
---

# Phase 15 Plan 02: Manifest Discovery Bootstrap Summary

**Go now atomically publishes runtime `go.base_url`, and Flutter consumes it at startup so API traffic no longer relies on fixed Go ports in product behavior.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-04-03T16:50:10Z
- **Completed:** 2026-04-03T16:58:34Z
- **Tasks:** 2
- **Files modified:** 12

## Accomplishments
- Added Go runtime manifest payload builder and atomic writer (temp file + rename) with shutdown cleanup.
- Wired Go startup to publish actual listener-derived URL and kept schema free of Python endpoint leakage.
- Added Flutter bootstrap loader to apply manifest `go.base_url` before app initialization.
- Preserved `http://localhost:8080` as explicit development fallback only.

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: Build atomic Go runtime manifest writer with schema guarantees** - `1b16f42` (test)
2. **Task 1 GREEN: Build atomic Go runtime manifest writer with schema guarantees** - `f98eb0f` (feat)
3. **Task 2 RED: Add Flutter bootstrap manifest loader and dev-only fallback path** - `a0a4c7f` (test)
4. **Task 2 GREEN: Add Flutter bootstrap manifest loader and dev-only fallback path** - `d6b12e0` (feat)

## Files Created/Modified
- `internal/app/runtime_manifest.go` - manifest schema, path/base-url resolution, atomic write and cleanup.
- `internal/app/runtime_manifest_test.go` - schema, anti-leakage, and atomic-write behavior tests.
- `internal/app/app.go` - listener-based startup wiring and manifest publish/remove lifecycle.
- `cmd/server/main.go` - graceful close handling without fatal on `http.ErrServerClosed`.
- `flutter_app/lib/bootstrap/runtime_manifest_loader.dart` - startup manifest parsing/apply flow.
- `flutter_app/lib/bootstrap/runtime_manifest_loader_io.dart` - IO-backed manifest read/path resolver.
- `flutter_app/lib/bootstrap/runtime_manifest_loader_stub.dart` - non-IO fallback implementation.
- `flutter_app/lib/config/api_config.dart` - explicit dev fallback API and runtime override support.
- `flutter_app/lib/providers/config_provider.dart` - runtime URL sync with `ApiConfig`.
- `flutter_app/lib/main.dart` - bootstrap loader invocation before `runApp`.
- `flutter_app/test/bootstrap/runtime_manifest_loader_test.dart` - manifest success/fallback precedence coverage.
- `flutter_app/test/config/api_config_test.dart` - URL normalization and fallback behavior coverage.

## Decisions Made
- Keep manifest JSON contract minimal (`version`, `generated_at`, `go.base_url`, `go.ready`) to satisfy D-04/D-06 boundaries.
- Resolve runtime endpoint from actual listener address and normalize wildcard hosts (`0.0.0.0/::`) to loopback for Flutter consumption.
- Use conditional Dart imports for IO manifest access to avoid forcing IO dependency on non-desktop targets.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Existing parallel-wave changes currently make `go test ./internal/app/... -run "Manifest|Runtime" -count=1` fail from unrelated `internal/app/app_test.go` sidecar symbols. Recorded in `.planning/phases/15-compute-sidecar-infrastructure/deferred-items.md` and not modified in this plan per scope boundary.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Runtime manifest discovery contract is in place and test-backed.
- Ready for follow-up plans to consume sidecar diagnostics and broader phase-15 verification.

## Self-Check: PASSED

- FOUND: `.planning/phases/15-compute-sidecar-infrastructure/15-02-SUMMARY.md`
- FOUND commits: `1b16f42`, `f98eb0f`, `a0a4c7f`, `d6b12e0`
