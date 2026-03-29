---
phase: 13
slug: backend-monitoring-queue-control
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-03-27
---

# Phase 13 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — use package-local Go tests |
| **Quick run command** | `go test ./internal/service/... -run "Admin|TaskRead" -count=1` |
| **Full suite command** | `go test ./internal/service/... ./internal/handler/... -count=1` |
| **Estimated runtime** | ~20 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/service/... -run "Admin|TaskRead" -count=1` or the task-specific handler variant
- **After every plan wave:** Run `go test ./internal/service/... ./internal/handler/... -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 20 seconds

---

## Per-task Verification Map

| task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 13-01-01 | 01 | 1 | PIPE-02, OPS-01 | markup/style smoke | `rg -n "batch|task-detail|error|refresh|filter" web/admin/index.html web/admin/styles.css` | ✅ | ⬜ pending |
| 13-01-02 | 01 | 1 | PIPE-02, OPS-01 | handler | `go test ./internal/handler/... -run "AdminHandler_(GetTaskBatches|GetTasks)" -count=1` | ✅ | ⬜ pending |
| 13-02-01 | 02 | 2 | PIPE-02, OPS-01 | manual+backend | `go test ./internal/service/... ./internal/handler/... -run "Admin|TaskRead" -count=1` | ✅ | ⬜ pending |
| 13-03-01 | 03 | 3 | OPS-02, OPS-03, OPS-05, OPS-06 | unit | `go test ./internal/service/... -run "PauseResume|Clear|Cancel" -count=1` | ❌ W0-in-plan | ⬜ pending |
| 13-03-02 | 03 | 3 | OPS-02, OPS-03, OPS-05, OPS-06 | handler | `go test ./internal/handler/... -run "Pause|Resume|Clear|Cancel" -count=1` | ❌ W0-in-plan | ⬜ pending |
| 13-04-01 | 04 | 4 | OPS-04 | unit | `go test ./internal/service/... -run "RetryFailed|RetryBatch|RetryTask" -count=1` | ❌ W0-in-plan | ⬜ pending |
| 13-04-02 | 04 | 4 | OPS-04 | handler | `go test ./internal/handler/... -run "Retry" -count=1` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] Existing infrastructure covers all phase requirements for Go backend tests.
- [ ] `internal/service/admin_service_test.go` — add cancel / clear / retry-new-batch coverage stubs inside plans 03-04.
- [ ] `internal/handler/admin_handler_test.go` — add new route/action coverage stubs inside plans 03-04.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 批次列表在上、任务明细在下，且默认突出异常与进行中任务 | PIPE-02, OPS-01 | 静态 `web/admin` 页面暂无现成 JS 自动化基建 | 启动服务后访问 `/admin-ui/`，确认顶部概览、批次表、明细区、异常区和自动刷新行为符合 Context 锁定决策 |
| 破坏性动作强确认文案与影响数量展示 | OPS-05, OPS-06 | 需要人工确认交互文案与风险提示质量 | 在后台页触发取消批次、取消任务、清空队列，确认弹窗列出影响范围和数量，取消后页面刷新正确 |
| 重试 toast 提供新批次跳转入口 | OPS-04 | 需要人工确认交互反馈与跳转体验 | 在后台页对失败批次执行重试，确认 toast 展示重试数量，并可跳转到新批次 |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 20s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
