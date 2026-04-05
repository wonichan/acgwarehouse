---
phase: 21-windows-packaging
plan: 01
subsystem: infra
tags: [windows-packaging, portable-runtime, sidecar, python, diagnostics]
requires:
  - phase: 15-compute-sidecar-infrastructure
    provides: Go-orchestrated sidecar lifecycle and runtime manifest flow
provides:
  - portable bundle-local runtime path resolution for manifest, logs, diagnostics, and mutable data
  - packaging-aware Go sidecar bootstrap using explicit executable and port inputs
  - Python sidecar CLI startup entrypoint with no-console-safe stdio initialization
affects: [21-02, bundle-assembly, packaged-bootstrap]
tech-stack:
  added: []
  patterns: [portable runtime layout contract, env-driven sidecar bootstrap, structured startup diagnostics]
key-files:
  created:
    - internal/app/packaged_runtime.go
    - internal/app/packaged_runtime_test.go
    - internal/app/bootstrap_test.go
    - services/python-sidecar/tests/test_main_startup.py
  modified:
    - internal/app/runtime_manifest.go
    - internal/app/bootstrap.go
    - services/python-sidecar/main.py
key-decisions:
  - "Portable mode resolves bundle-local runtime, diagnostics, config, storage, and library paths from the launcher directory or ACG_RUNTIME_ROOT."
  - "Go uses ACG_SIDECAR_EXECUTABLE and ACG_SIDECAR_PORT for packaged sidecar startup, while keeping the python script fallback for development."
  - "Packaged Python startup failures write structured python-classified diagnostics into runtime/diagnostics/startup-error.json."
patterns-established:
  - "Portable runtime layout: runtime artifacts stay under runtime/, mutable app data stays under storage/, Flutter assets stay under data/."
  - "Packaged sidecar startup: Go passes --host/--port explicitly and Python parses them through argparse."
requirements-completed: [OPS-03]
duration: not-recorded
completed: 2026-04-05
---

# Phase 21 Plan 01: Windows Packaging Summary

**Portable runtime layout resolution, packaged sidecar bootstrap wiring, and Python CLI startup safety for the Windows bundle path.**

## Performance

- **Duration:** not recorded in session metadata
- **Started:** not recorded in session metadata
- **Completed:** 2026-04-05T21:53:14.8250142+08:00
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Added `PortableRuntimeLayout`, bundle-local manifest resolution, and structured startup diagnostic writing in Go.
- Updated Go bootstrap to honor packaged sidecar executable/port/log/diagnostic environment inputs while preserving development fallback behavior.
- Added a real Python `main()` entrypoint with `argparse` host/port parsing and no-console-safe `sys.stdout` / `sys.stderr` initialization.

## Task Commits

No commits were created because this execution request did not include a commit instruction.

## Files Created/Modified
- `internal/app/packaged_runtime.go` - Defines portable runtime layout resolution and structured startup diagnostics.
- `internal/app/packaged_runtime_test.go` - Covers bundle-local layout, manifest path, and diagnostic validation behavior.
- `internal/app/runtime_manifest.go` - Prefers `ACG_RUNTIME_ROOT` bundle-local manifest resolution before temp fallback.
- `internal/app/bootstrap.go` - Reads packaged sidecar env vars, builds explicit sidecar args, and writes startup diagnostics on launch failure.
- `internal/app/bootstrap_test.go` - Verifies explicit executable/port bootstrap behavior and python-classified startup diagnostics.
- `services/python-sidecar/main.py` - Adds CLI parsing and no-console-safe stdout/stderr initialization.
- `services/python-sidecar/tests/test_main_startup.py` - Verifies Python CLI argument handling and no-console startup safety.

## Decisions Made
- Used `ACG_RUNTIME_ROOT` as the portable bundle anchor for runtime manifest, logs, and diagnostics when packaged mode is active.
- Kept development fallback on `python services/python-sidecar/main.py` while still passing explicit `--host` and `--port` arguments.
- Scoped packaged startup diagnostics to structured JSON containing component, message, log paths, and timestamp only.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- `internal/app/bootstrap_test.go` did not exist yet, so packaged bootstrap coverage was added as a new focused test file.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Portable runtime path and sidecar executable contracts are now in place for packaged Flutter bootstrap work.
- Bundle assembly can now rely on fixed `runtime/`, `runtime/logs/`, and `runtime/diagnostics/` locations.

---
*Phase: 21-windows-packaging*
*Completed: 2026-04-05*
