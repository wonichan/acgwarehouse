---
phase: 03
slug: ai
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-15
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — existing infrastructure |
| **Quick run command** | `go test -v ./internal/ai/... ./internal/repository/... ./internal/service/...` |
| **Full suite command** | `go test -v ./...` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -v ./internal/ai/... ./internal/repository/...`
- **After every plan wave:** Run `go test -v ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-task Verification Map

| task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 03-01-01 | 01 | 1 | AIRE-01 | unit | `go test -v ./internal/ai/...` | ❌ W0 | ⬜ pending |
| 03-01-02 | 01 | 1 | AIRE-01 | unit | `go test -v ./internal/ai/... -run TestRateLimited` | ❌ W0 | ⬜ pending |
| 03-01-03 | 01 | 1 | AIRE-06 | unit | `go test -v ./internal/worker/... -run TestAI` | ✅ | ⬜ pending |
| 03-02-01 | 02 | 2 | TAGS-01 | unit | `go test -v ./internal/repository/... -run TestTag` | ❌ W0 | ⬜ pending |
| 03-02-02 | 02 | 2 | TAGS-03 | unit | `go test -v ./internal/service/... -run TestTagGovernance` | ❌ W0 | ⬜ pending |
| 03-03-01 | 03 | 3 | TAGS-02 | integration | `go test -v ./internal/handler/... -run TestTag` | ❌ W0 | ⬜ pending |
| 03-03-02 | 03 | 3 | TAGS-04 | integration | `go test -v ./internal/handler/... -run TestSearch` | ❌ W0 | ⬜ pending |
| 03-04-01 | 04 | 4 | AIRE-05 | unit | `flutter test test/widgets/tag_filter_test.dart` | ❌ W0 | ⬜ pending |
| 03-04-02 | 04 | 4 | TAGS-05 | unit | `flutter test test/screens/tag_confirm_test.dart` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

### Go Backend

- [ ] `internal/ai/ai_test.go` — AI provider tests
- [ ] `internal/ai/rate_limiter_test.go` — Rate limiter tests
- [ ] `internal/repository/tag_repository_test.go` — Tag repository tests
- [ ] `internal/repository/tag_alias_repository_test.go` — Alias repository tests
- [ ] `internal/repository/image_tag_repository_test.go` — Image-tag relation tests
- [ ] `internal/service/tag_governance_service_test.go` — Tag governance tests
- [ ] `internal/handler/tag_handler_test.go` — Tag API handler tests

### Flutter Frontend

- [ ] `flutter_app/test/widgets/tag_filter_test.dart` — Tag filter widget tests
- [ ] `flutter_app/test/providers/tag_provider_test.dart` — Tag provider tests
- [ ] `flutter_app/test/screens/tag_confirm_test.dart` — Tag confirmation screen tests

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| AI 实际调用返回有效标签 | AIRE-01 | 需要 API 密钥和网络 | 1. 配置 AI_API_KEY 2. 运行 `go run ./cmd/server` 3. POST /api/v1/images/1/ai-tags 4. 检查返回标签 |
| Flutter 标签确认 UI 交互 | AIRE-05 | 视觉验证 | 1. 启动 Flutter 应用 2. 打开图片详情页 3. 确认/拒绝标签按钮正常工作 |
| 标签筛选结果正确 | TAGS-03 | 需要 UI + 数据集成 | 1. 选择多个标签 2. 验证结果为 AND 交集 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending