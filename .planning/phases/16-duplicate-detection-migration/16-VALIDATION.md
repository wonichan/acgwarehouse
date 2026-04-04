---
phase: 16
slug: duplicate-detection-migration
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-04
---

# Phase 16 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) + pytest (Python) |
| **Config file** | None for Go (uses `go test`). Python: to be created |
| **Quick run command** | `go test ./internal/service/... ./internal/handler/... -run TestDuplicate -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/service/... ./internal/handler/... -run TestDuplicate -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 16-01-01 | 01 | 1 | COMP-03a | unit | `pytest services/python-sidecar/tests/test_hashing.py -x` | ❌ W0 | ⬜ pending |
| 16-01-02 | 01 | 1 | COMP-03b | unit | `pytest services/python-sidecar/tests/test_grouping.py -x` | ❌ W0 | ⬜ pending |
| 16-02-01 | 02 | 1 | COMP-03c | integration | `go test ./internal/service/ -run TestDuplicateDetectionPython -count=1` | ❌ W0 | ⬜ pending |
| 16-02-02 | 02 | 1 | COMP-03d | integration | `go test ./internal/service/ -run TestDuplicateSaveResults -count=1` | ❌ W0 | ⬜ pending |
| 16-02-03 | 02 | 2 | COMP-03e | unit | `go test ./internal/handler/ -run TestDuplicateDetectSidecarUnavailable -count=1` | ❌ W0 | ⬜ pending |
| 16-03-01 | 03 | 1 | COMP-04a | unit | `pytest services/python-sidecar/tests/test_scoring.py -x` | ❌ W0 | ⬜ pending |
| 16-03-02 | 03 | 1 | COMP-04b | unit | `pytest services/python-sidecar/tests/test_scoring.py::test_rationale_structure -x` | ❌ W0 | ⬜ pending |
| 16-03-03 | 03 | 2 | COMP-04c | integration | `go test ./internal/repository/ -run TestDuplicateRelationRationale -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `services/python-sidecar/tests/test_hashing.py` — stubs for COMP-03a
- [ ] `services/python-sidecar/tests/test_grouping.py` — stubs for COMP-03b
- [ ] `services/python-sidecar/tests/test_scoring.py` — stubs for COMP-04a, COMP-04b
- [ ] `services/python-sidecar/tests/conftest.py` — shared fixtures (test images)
- [ ] Python test framework install: `pip install pytest` — if not already available
- [ ] `internal/service/duplicate_service_test.go` — extend for Python integration tests (COMP-03c, COMP-03d)
- [ ] `internal/handler/duplicate_handler_test.go` — extend for sidecar unavailable error (COMP-03e)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Full-library pHash recomputation performance | COMP-03 | Requires real large image library | Trigger detection with 1000+ images, observe completion time and progress reporting |
| End-to-end desktop app startup with sidecar | COMP-03 | Requires real Windows desktop environment | Launch app, trigger duplicate detection, verify Python sidecar processes images |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
