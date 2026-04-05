---
phase: 21-windows-packaging
plan: 03
subsystem: infra
tags: [windows, packaging, powershell, pyinstaller, flutter, portable-zip]
requires:
  - phase: 21-01
    provides: packaging-aware Go runtime layout and sidecar executable contract
  - phase: 21-02
    provides: packaged Flutter bootstrap and bundle-relative startup expectations
provides:
  - authoritative Windows portable packaging pipeline
  - onedir PyInstaller sidecar spec for the packaged Python runtime
  - smoke verification for bundle layout and packaging docs
  - operator documentation for launch, overwrite upgrades, and troubleshooting
affects: [phase-21-04, windows-packaging, operator-docs]
tech-stack:
  added: [PyInstaller]
  patterns: [bundle-local Windows packaging, PowerShell smoke verification, ZIP-first portable distribution]
key-files:
  created:
    - deploy/windows/package.ps1
    - deploy/windows/package-smoke.ps1
    - services/python-sidecar/sidecar.spec
    - docs/windows-portable-package.md
  modified:
    - Makefile
    - .gitignore
key-decisions:
  - "The package root keeps the Flutter executable at `ACGWarehouse.exe` and leaves Flutter support DLLs adjacent to it while runtime-managed services stay under `runtime/`."
  - "The Python sidecar is packaged with a PyInstaller onedir layout so overwrite upgrades and support inspection stay debuggable inside the portable bundle."
  - "The packaging script bootstraps PyInstaller when missing so the documented packaging command remains repeatable on the current Windows environment."
patterns-established:
  - "Windows packaging command: `deploy/windows/package.ps1` owns build, assembly, and ZIP creation."
  - "Packaging validation: `deploy/windows/package-smoke.ps1` verifies bundle layout and required operator documentation strings."
requirements-completed: [OPS-03]
duration: 27 min
completed: 2026-04-05
---

# Phase 21 Plan 03: Windows portable bundle assembly and overwrite-upgrade operator guidance Summary

**Windows x64 portable packaging now builds the Flutter launcher, Go runtime, Python sidecar, and ZIP artifact through one PowerShell command with smoke-checked operator docs.**

## Performance

- **Duration:** 27 min
- **Started:** 2026-04-05T14:57:42Z
- **Completed:** 2026-04-05T15:24:42Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Added `deploy/windows/package.ps1` to clean outputs, build Go, package the sidecar via PyInstaller, build Flutter Windows release, assemble the portable layout, and emit `dist/windows-zip/ACGWarehouse-windows-x64-portable.zip`.
- Added `deploy/windows/package-smoke.ps1` to verify the assembled bundle layout and to enforce required documentation headings and overwrite-upgrade guidance.
- Added `services/python-sidecar/sidecar.spec`, `Makefile` packaging entry, and `docs/windows-portable-package.md` so operators have one packaging command and explicit launch/overwrite/troubleshooting instructions.

## Task Commits

No task commits were created in this run. The plan's staged commit steps were not executed because this workspace already had unrelated in-progress changes and no git commit was requested.

## Files Created/Modified
- `deploy/windows/package.ps1` - Builds and assembles the Windows portable bundle, then creates the ZIP artifact.
- `deploy/windows/package-smoke.ps1` - Verifies required bundle paths and docs coverage.
- `services/python-sidecar/sidecar.spec` - Defines the PyInstaller onedir sidecar package with the packaged executable name.
- `Makefile` - Exposes `package-windows-portable` as the documented packaging entry point.
- `docs/windows-portable-package.md` - Documents bundle layout, build command, first launch, overwrite upgrades, and troubleshooting.
- `.gitignore` - Ignores generated `dist/` packaging outputs.

## Decisions Made
- Kept the portable root aligned with the packaged Flutter bootstrap by renaming the Flutter release runner to `ACGWarehouse.exe` while preserving adjacent Flutter runtime files.
- Used a PyInstaller spec that emits `runtime/python-sidecar/acgwarehouse-sidecar.exe` inside a onedir folder named `python-sidecar`.
- Used a custom ZIP writer so the portable archive preserves the required empty directories such as `storage/`, `library/`, `runtime/logs/`, and `runtime/diagnostics/`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Bootstrapped missing PyInstaller dependency**
- **Found during:** Task 1 (Red-green the Windows portable packaging pipeline and bundle layout)
- **Issue:** `python -m PyInstaller` was not available in the active Windows Python environment, so the authoritative packaging command could not complete.
- **Fix:** Added a bootstrap step in `deploy/windows/package.ps1` that installs `PyInstaller` with `python -m pip install` when the module is missing.
- **Files modified:** `deploy/windows/package.ps1`
- **Verification:** `powershell -ExecutionPolicy Bypass -File deploy/windows/package.ps1 -SkipTests; if ($?) { powershell -ExecutionPolicy Bypass -File deploy/windows/package-smoke.ps1 -VerifyBundleLayout }`
- **Committed in:** Not committed in this run

**2. [Rule 2 - Missing Critical] Ignored generated packaging outputs**
- **Found during:** Task 1 (Red-green the Windows portable packaging pipeline and bundle layout)
- **Issue:** The packaging pipeline generates `dist/` artifacts that should remain out of version control while preserving the existing working tree.
- **Fix:** Added `/dist/` to `.gitignore`.
- **Files modified:** `.gitignore`
- **Verification:** Packaging verification completed successfully without leaving tracked build artifacts in the repository.
- **Committed in:** Not committed in this run

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 missing critical)
**Impact on plan:** Both deviations were necessary to make the documented Windows packaging command repeatable and to keep generated artifacts out of source control. No product scope changed.

## Issues Encountered
- `System.IO.Path.GetRelativePath` was unavailable in the active PowerShell/.NET runtime during ZIP creation, so the packaging script switched to a substring-based relative-path helper.
- `lsp_diagnostics` has no configured server for `.ps1` and `.spec` files in this workspace, so diagnostics for those changed files could only be attempted and recorded as unsupported.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- The repo now has the required packaging command, smoke checks, and operator docs needed for packaged-bundle validation in Plan 21-04.
- The current branch still contains unrelated pre-existing working tree changes outside this plan; they were preserved.

---
*Phase: 21-windows-packaging*
*Completed: 2026-04-05*
