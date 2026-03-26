---
phase: "12"
slug: import-task-auto-scheduling
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-03-26"
---

# Phase 12 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — existing test infrastructure |
| **Quick run command** | `go test ./internal/service/... -run "AITagAuto" -count=1` |
| **Full suite command** | `go test ./internal/... -count=1` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run quick command for affected package
- **After every plan wave:** Run full suite
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-task Verification Map

| task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 12-01-01 | 01 | 1 | AIQ-02 | migration | `go test ./internal/repository/... -run "ImageTagSource" -count=1` | ✅ W0 | ⬜ pending |
| 12-01-02 | 01 | 1 | AIQ-02 | unit | `go test ./internal/repository/... -run "FindImagesWithoutAITags" -count=1` | ✅ W0 | ⬜ pending |
| 12-02-01 | 02 | 1 | AIQ-01 | unit | `go test ./internal/service/... -run "AITagAutoScheduler" -count=1` | ✅ W0 | ⬜ pending |
| 12-03-01 | 03 | 2 | AIQ-01, AIQ-02 | unit | `go test ./internal/service/... -run "AutoEnqueueCondition" -count=1` | ✅ W0 | ⬜ pending |
| 12-04-01 | 04 | 3 | AIQ-01, AIQ-02 | integration | `go test ./internal/... -run "AutoSchedulingE2E" -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/repository/image_repository_test.go` — add `FindImagesWithoutAITags` test
- [ ] `internal/service/ai_tag_auto_scheduler_test.go` — scheduler test stubs
- [ ] `internal/config/config_test.go` — auto_ai_tag_on_import config test

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 定时扫描实际执行 | AIQ-01 | 涉及 goroutine + ticker 时序 | 启动服务后观察日志确认每5分钟扫描 |
| 大批量导入后队列积压 | AIQ-01 | 需要 100+ 图片导入 | 导入测试图片集后检查队列状态 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

---

*Phase: 12-import-task-auto-scheduling*
*Created: 2026-03-26*