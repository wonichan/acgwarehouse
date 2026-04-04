---
phase: 15
slug: compute-sidecar-infrastructure
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-04-04
---

# Phase 15 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` + `flutter test` |
| **Config file** | `go.mod`, `flutter_app/pubspec.yaml` |
| **Quick run command** | `go test ./internal/app/... ./internal/service/... ./internal/handler/...` |
| **Full suite command** | `go test ./...; if ($?) { flutter test }` |
| **Estimated runtime** | ~45 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/app/... ./internal/service/... ./internal/handler/...`
- **After every plan wave:** Run `go test ./...; if ($?) { flutter test }`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 45 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 15-01-01 | 01 | 1 | COMP-01, COMP-02 | unit | `go test ./internal/sidecar/... -run "Runtime|Lifecycle|Degraded|Shutdown" -count=1` | ✅ | ⬜ pending |
| 15-01-02 | 01 | 1 | COMP-01, COMP-02 | integration | `go test ./internal/app/... -run "Sidecar|Shutdown|Degraded" -count=1` | ✅ | ⬜ pending |
| 15-01-03 | 01 | 1 | COMP-01 | regression | `go test ./internal/handler/... -run "Health|Ready|Routes" -count=1` | ✅ | ⬜ pending |
| 15-02-01 | 02 | 1 | COMP-06 | unit | `go test ./internal/app/... -run Manifest -count=1` | ✅ | ⬜ pending |
| 15-02-02 | 02 | 1 | COMP-06 | widget/unit | `flutter test test/bootstrap/runtime_manifest_loader_test.dart test/config/api_config_test.dart` | ✅ | ⬜ pending |
| 15-03-01 | 03 | 2 | COMP-02 | unit/integration | `go test ./internal/service/... ./internal/handler/... -run "Admin|Overview|Sidecar|Health|Ready" -count=1` | ✅ | ⬜ pending |
| 15-03-02 | 03 | 2 | COMP-01, COMP-02 | regression | `go test ./internal/app/... ./internal/handler/... -run "Degraded|Sidecar|Health|Ready" -count=1` | ✅ | ⬜ pending |
| 15-03-03 | 03 | 2 | COMP-01, COMP-02, COMP-06 | cross-stack | `go test ./internal/app/... ./internal/service/... ./internal/handler/... -run "Sidecar|Manifest|Admin|Degraded|Health|Ready" -count=1; if ($?) { flutter test test/bootstrap/runtime_manifest_loader_test.dart test/config/api_config_test.dart }` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/app/*_test.go` — sidecar lifecycle and manifest test coverage scaffolds
- [ ] `internal/service/admin_service_test.go` — sidecar diagnostics assertions
- [ ] `internal/handler/*_test.go` — `/health` and `/ready` boundary assertions
- [ ] `flutter_app/test/config/*_test.dart` — manifest-driven base URL bootstrap tests

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Desktop boot sequence feels operationally acceptable | COMP-01 | Startup timing and desktop packaging boot flow are environment-sensitive | Launch desktop app in local desktop flow, verify Flutter waits for Go manifest, Go reports ready, and sidecar status is visible without blocking gallery usability |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 45s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
