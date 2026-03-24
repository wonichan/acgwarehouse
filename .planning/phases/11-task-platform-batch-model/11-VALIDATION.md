---
phase: 11
slug: task-platform-batch-model
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-03-24
---

# Phase 11 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — uses repository/service/handler package tests |
| **Quick run command** | `go test ./internal/... -run "Task|Batch|Scanner|Admin|AITag" -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~30-60 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/... -run "Task|Batch|Scanner|Admin|AITag" -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-task Verification Map

| task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 11-01-01 | 01 | 1 | PIPE-01 | repository/unit | `go test ./internal/repository/... -run "TaskBatch|PlatformTask|Schema" -count=1` | ❌ W0 | ⬜ pending |
| 11-01-02 | 01 | 1 | PIPE-03 | service/unit | `go test ./internal/service/... -run "TaskPlatform|Lifecycle" -count=1` | ❌ W0 | ⬜ pending |
| 11-02-01 | 02 | 2 | PIPE-03 | repository+service | `go test ./internal/repository/... -run "PlatformTask|Job" -count=1 && go test ./internal/service/... -run "TaskPlatform|Duplicate" -count=1` | ❌ W0 | ⬜ pending |
| 11-02-02 | 02 | 2 | SAFE-03 | service/unit | `go test ./internal/service/... -run "Duplicate|Dedupe|Skip" -count=1` | ❌ W0 | ⬜ pending |
| 11-03-01 | 03 | 3 | PIPE-01 | service/integration | `go test ./internal/service/... -run "Scanner|TaskPlatform" -count=1` | ✅ | ⬜ pending |
| 11-03-02 | 03 | 3 | PIPE-03 | handler/integration | `go test ./internal/handler/... -run "AITag|Admin" -count=1` | ✅ | ⬜ pending |
| 11-04-01 | 04 | 4 | PIPE-01 | service/unit | `go test ./internal/service/... -run "Admin|TaskRead" -count=1` | ❌ W0 | ⬜ pending |
| 11-04-02 | 04 | 4 | PIPE-03, SAFE-03 | handler/integration | `go test ./internal/handler/... -run "Admin" -count=1` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/repository/task_batch_repository_test.go` — batch schema and repository coverage
- [ ] `internal/repository/platform_task_repository_test.go` — task persistence and dedupe coverage
- [ ] `internal/service/task_platform_service_test.go` — lifecycle and duplicate protection coverage
- [ ] `internal/service/task_read_service_test.go` — batch/task read model aggregation coverage

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 后台批次/任务读模型字段是否满足运营可读性 | PIPE-01, PIPE-03 | 需要结合真实管理视角判断摘要字段是否够用 | 启动服务后请求 `/admin/api/task-batches` 和 `/admin/api/tasks`，检查来源摘要、状态统计、跳过原因摘要是否可读 |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
