---
phase: 6
slug: optimization-deployment
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-03-18
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test + flutter test + docker compose smoke commands |
| **Config file** | none |
| **Quick run command** | `go test ./internal/... ./cmd/server/...` |
| **Full suite command** | `go test ./... && cd flutter_app && flutter test` |
| **Estimated runtime** | ~45 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/... ./cmd/server/...` or the task-specific narrower command
- **After every plan wave:** Run `go test ./... && cd flutter_app && flutter test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-task Verification Map

| task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 06-01-01 | 01 | 1 | DEPL-01 | config | `docker compose config` | ✅ | ⬜ pending |
| 06-01-02 | 01 | 1 | DEPL-01 | smoke | `docker compose up -d && curl http://localhost:8080/health` | ✅ | ⬜ pending |
| 06-02-01 | 02 | 1 | DEPL-02 | unit | `go test ./internal/handler/... ./internal/service/... -run Admin -count=1` | ✅ | ⬜ pending |
| 06-02-02 | 02 | 1 | DEPL-02 | smoke | `go test ./cmd/server/... -run Admin -count=1` | ✅ | ⬜ pending |
| 06-03-01 | 03 | 1 | DEPL-01 | unit | `go test ./internal/repository/... ./internal/handler/... -run Image -count=1` | ✅ | ⬜ pending |
| 06-03-02 | 03 | 1 | DEPL-01 | widget | `cd flutter_app && flutter test test/providers test/screens -r compact` | ✅ | ⬜ pending |
| 06-04-01 | 04 | 2 | DEPL-01 | benchmark | `go test ./test/perf/... -run ^$ -bench . -benchmem -count=1` | ✅ | ⬜ pending |
| 06-04-02 | 04 | 2 | DEPL-01 | smoke | `docker compose config && go test ./... && cd flutter_app && flutter test` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 管理后台首页在浏览器中可读、卡片状态易区分 | DEPL-02 | 视觉可用性无法完全靠 CLI 判断 | 启动 Compose 后访问 `/admin`，确认状态卡片、错误列表和操作按钮可见且布局未错乱 |
| 10k+ 图片日常滚动浏览体感顺畅 | DEPL-01 | 需要真实滚动交互与主观体验 | 使用基准数据启动应用，在 Flutter 客户端连续滚动 30 秒并切换排序，确认没有明显卡顿或重复加载抖动 |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
