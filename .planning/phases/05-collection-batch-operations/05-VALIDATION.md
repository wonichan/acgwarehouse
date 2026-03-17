---
phase: 05
slug: collection-batch-operations
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-17
---

# Phase 05 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (后端) + flutter_test (前端) |
| **Config file** | Go: go.mod / Flutter: pubspec.yaml |
| **Quick run command** | `go test ./internal/... -short -count=1` |
| **Full suite command** | `go test ./internal/... -v && cd flutter_app && flutter test` |
| **Estimated runtime** | ~60 seconds (后端) + ~30 seconds (前端) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/... -short -count=1`
- **After every plan wave:** Run full suite
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 90 seconds

---

## Per-task Verification Map

| task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 05-01-01 | 01 | 1 | COLL-01, COLL-05 | unit | `go test ./internal/repository/... -run Collection -v` | ❌ W0 | ⬜ pending |
| 05-01-02 | 01 | 1 | COLL-02 | unit | `go test ./internal/repository/... -run CollectionImage -v` | ❌ W0 | ⬜ pending |
| 05-02-01 | 02 | 2 | COLL-01~04 | unit | `go test ./internal/service/... -run Collection -v` | ❌ W0 | ⬜ pending |
| 05-02-02 | 02 | 2 | BTCH-02~04 | unit | `go test ./internal/service/... -run Batch -v` | ❌ W0 | ⬜ pending |
| 05-03-01 | 03 | 2 | COLL-01~05 | integration | `go test ./internal/handler/... -run Collection -v` | ❌ W0 | ⬜ pending |
| 05-03-02 | 03 | 2 | BTCH-01~04 | integration | `go test ./internal/handler/... -run Batch -v` | ❌ W0 | ⬜ pending |
| 05-04-01 | 04 | 3 | COLL-05 | widget | `cd flutter_app && flutter test test/widgets/collection_drawer_test.dart` | ❌ W0 | ⬜ pending |
| 05-04-02 | 04 | 3 | BTCH-01 | widget | `cd flutter_app && flutter test test/widgets/batch_selection_test.dart` | ❌ W0 | ⬜ pending |
| 05-04-03 | 04 | 3 | BTCH-02~04 | widget | `cd flutter_app && flutter test test/widgets/batch_operation_sheet_test.dart` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/repository/collection_repository_test.go` — stubs for COLL-01~05
- [ ] `internal/service/collection_service_test.go` — stubs for COLL-01~04
- [ ] `internal/service/batch_service_test.go` — stubs for BTCH-02~04
- [ ] `internal/handler/collection_handler_test.go` — stubs for API 端点
- [ ] `internal/handler/batch_handler_test.go` — stubs for 批量操作 API
- [ ] `flutter_app/test/providers/selection_provider_test.dart` — stubs for 批量选择状态
- [ ] `flutter_app/test/providers/collection_provider_test.dart` — stubs for 收藏夹状态
- [ ] `flutter_app/test/widgets/collection_drawer_test.dart` — stubs for 收藏夹列表 UI
- [ ] `flutter_app/test/widgets/batch_selection_test.dart` — stubs for 批量选择 UI
- [ ] `flutter_app/test/widgets/batch_operation_sheet_test.dart` — stubs for 操作面板 UI

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 长按触发选择模式 | BTCH-01 | 触摸交互测试 | 在真机或模拟器上长按图片，验证选择模式激活 |
| 批量删除确认对话框 | BTCH-04 | UI 交互测试 | 选择多张图片点击删除，验证对话框显示正确数量和警告 |
| 收藏夹封面自动更新 | COLL-04 | 集成验证 | 添加图片后验证封面更新，移除封面图片后验证自动更换 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending