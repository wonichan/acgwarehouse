---
phase: 21
slug: windows-packaging
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-05
---

# Phase 21 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test`, `pytest`, `flutter test`, PowerShell smoke scripts |
| **Config file** | Existing repo test setup; no new framework required |
| **Quick run command** | `go test ./internal/app/... -count=1` |
| **Full suite command** | `go test ./internal/app/... -count=1; if ($?) { python -m pytest services/python-sidecar/tests/test_main_startup.py -q; if ($?) { cd flutter_app; flutter test test/bootstrap -r compact; if ($?) { cd ..; powershell -ExecutionPolicy Bypass -File deploy/windows/package-smoke.ps1 -SkipBuild } } }` |
| **Estimated runtime** | ~45-60 seconds per full feedback cycle |

---

## Sampling Rate

- **After every task commit:** Run the task-local automated command from the plan.
- **After every plan wave:** Run the phase full suite command.
- **Before `/gsd-verify-work`:** The packaged smoke test must pass from the assembled bundle.
- **Max feedback latency:** 60 seconds.

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 21-01-01 | 01 | 1 | OPS-03 | T-21-01 / T-21-02 | Runtime files resolve inside bundle root only | unit | `go test ./internal/app/... -run TestPortableRuntimeLayout -count=1` | ✅ | ⬜ pending |
| 21-01-02 | 01 | 1 | OPS-03 | T-21-03 / T-21-04 | Go starts packaged sidecar with explicit executable + port contract | unit + pytest | `go test ./internal/app/... -run TestPackagedSidecarBootstrap -count=1; if ($?) { python -m pytest services/python-sidecar/tests/test_main_startup.py -q }` | ✅ | ⬜ pending |
| 21-02-01 | 02 | 2 | OPS-03 | T-21-05 / T-21-06 | Flutter bootstrap starts only bundle-local children and surfaces classified startup failures | flutter test | `cd flutter_app; flutter test test/bootstrap/packaged_desktop_bootstrap_test.dart -r compact` | ✅ | ⬜ pending |
| 21-02-02 | 02 | 2 | OPS-03 | T-21-07 | Flutter bootstrap shuts down Go on app exit and reads the packaged manifest path | flutter test | `cd flutter_app; flutter test test/bootstrap/packaged_desktop_bootstrap_test.dart test/bootstrap/runtime_manifest_loader_test.dart -r compact` | ✅ | ⬜ pending |
| 21-03-01 | 03 | 3 | OPS-03 | T-21-08 / T-21-09 | Packaging pipeline assembles stable portable layout and ZIP | smoke | `powershell -ExecutionPolicy Bypass -File deploy/windows/package.ps1 -SkipTests` | ✅ | ⬜ pending |
| 21-03-02 | 03 | 3 | OPS-03 | T-21-10 | Docs explicitly describe overwrite upgrade rules and log locations | grep/docs | `powershell -ExecutionPolicy Bypass -File deploy/windows/package-smoke.ps1 -VerifyDocsOnly` | ✅ | ⬜ pending |
| 21-04-01 | 04 | 4 | OPS-03 | T-21-11 / T-21-12 | Packaged bundle runs without external Python on PATH and duplicate detection updates the packaged sidecar log | smoke | `powershell -ExecutionPolicy Bypass -File deploy/windows/package-smoke.ps1` | ✅ | ⬜ pending |
| 21-04-02 | 04 | 4 | OPS-03 | T-21-13 | Human verifies unzip-and-run flow on Windows without Python | manual | `MISSING — human verification checkpoint` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Unzip on Windows machine without Python and launch successfully | OPS-03 | Clean-machine runtime absence cannot be proven from current dev box alone | Unzip the produced ZIP, run `ACGWarehouse.exe`, confirm main shell loads, then trigger duplicate detection |
| Startup dialog quality for real packaging failure | OPS-03 | Native dialog clarity and wording are user-facing | Temporarily rename packaged sidecar or Go binary, launch `ACGWarehouse.exe`, confirm dialog distinguishes component and shows log path |

---

## Validation Sign-Off

- [x] All tasks have automated verification or an explicit human-only checkpoint.
- [x] Sampling continuity avoids long stretches without executable checks.
- [x] Existing infrastructure covers the phase without Wave 0 scaffolding.
- [x] No watch-mode flags are used.
- [x] Feedback latency target stays under ~60 seconds.
- [x] `nyquist_compliant: true` is set in frontmatter.

**Approval:** pending
