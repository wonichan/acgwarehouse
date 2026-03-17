---
phase: 05
slug: collection-batch-operations
status: draft
nyquist_compliant: true
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
| **Quick run command** | go test ./internal/... -short -count=1 |
| **Full suite command** | go test ./internal/... -v && cd flutter_app && flutter test |
| **Estimated runtime** | ~60 seconds (后端) + ~30 seconds (前端) |

---

## Sampling Rate

- **After every task commit:** Run go test ./internal/... -short -count=1
- **After every plan wave:** Run full suite
- **Before /gsd-verify-work:** Full suite must be green
- **Max feedback latency:** 90 seconds

---

## Per-task Verification Map

| task ID | Plan | Wave | Requirement | Test Type | Automated Command | Status |
|---------|------|------|-------------|-----------|-------------------|--------|
| 05-01-01 | 01 | 1 | COLL-01 | unit | go build ./internal/domain/... | pending |
| 05-01-02 | 01 | 1 | COLL-02 | unit | go build ./internal/domain/... | pending |
| 05-01-03 | 01 | 1 | COLL-01~05 | unit | go build ./internal/repository/... | pending |
| 05-01-04 | 01 | 1 | COLL-01~05 | unit | go build ./internal/repository/... | pending |
| 05-01-05 | 01 | 1 | COLL-01~05 | unit | go build ./internal/repository/... | pending |
| 05-01-06 | 01 | 1 | COLL-01~05 | unit | go test ./internal/repository/... -run Collection -v | pending |
| 05-01-07 | 01 | 1 | COLL-01~05 | unit | go test ./internal/repository/... -run Collection -v | pending |
| 05-02-01 | 02 | 2 | COLL-01~04 | unit | go build ./internal/service/... | pending |
| 05-02-02 | 02 | 2 | COLL-01~04 | unit | go build ./internal/service/... | pending |
| 05-02-03 | 02 | 2 | COLL-01~04 | unit | go test ./internal/service/... -run Collection -v | pending |
| 05-02-04 | 02 | 2 | BTCH-02~04 | unit | go build ./internal/service/... | pending |
| 05-02-05 | 02 | 2 | BTCH-02~04 | unit | go build ./internal/service/... | pending |
| 05-02-06 | 02 | 2 | BTCH-02~04 | unit | go test ./internal/service/... -run Batch -v | pending |
| 05-02-07~12 | 02 | 2 | COLL-01~05, BTCH-01~04 | integration | go test ./internal/handler/... -v | pending |
| 05-02-13 | 02 | 2 | COLL-01~05, BTCH-01~04 | integration | go test ./internal/handler/... -run Routes -v | pending |
| 05-03-01~05 | 03 | 2 | COLL-01~05 | unit | cd flutter_app && flutter analyze lib/providers/ | pending |
| 05-03-06~07 | 03 | 2 | BTCH-01 | unit | cd flutter_app && flutter analyze lib/providers/ | pending |
| 05-03-08 | 03 | 2 | COLL-01~05 | unit | cd flutter_app && flutter test test/providers/collection_provider_test.dart | pending |
| 05-03-09 | 03 | 2 | BTCH-01 | unit | cd flutter_app && flutter test test/providers/selection_provider_test.dart | pending |
| 05-04-01~09 | 04 | 3 | COLL-01~05, BTCH-01~04 | widget | cd flutter_app && flutter analyze | pending |
| 05-04-10 | 04 | 3 | COLL-05 | widget | cd flutter_app && flutter test test/widgets/collection_list_item_test.dart | pending |
| 05-04-11 | 04 | 3 | BTCH-01 | widget | cd flutter_app && flutter test test/widgets/selectable_image_tile_test.dart | pending |
| 05-04-12 | 04 | 3 | BTCH-02~04 | widget | cd flutter_app && flutter test test/widgets/batch_operation_sheet_test.dart | pending |

*Status: pending / green / red / flaky*

---

## Wave 0 Requirements

- [ ] internal/repository/collection_repository_test.go - 测试存根
- [ ] internal/service/collection_service_test.go - 测试存根
- [ ] internal/service/batch_service_test.go - 测试存根
- [ ] internal/handler/collection_handler_test.go - 测试存根
- [ ] internal/handler/batch_handler_test.go - 测试存根
- [ ] flutter_app/test/providers/selection_provider_test.dart - 测试存根
- [ ] flutter_app/test/providers/collection_provider_test.dart - 测试存根
- [ ] flutter_app/test/widgets/collection_list_item_test.dart - 测试存根
- [ ] flutter_app/test/widgets/selectable_image_tile_test.dart - 测试存根
- [ ] flutter_app/test/widgets/batch_operation_sheet_test.dart - 测试存根

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 长按触发选择模式 | BTCH-01 | 触摸交互测试 | 在真机或模拟器上长按图片，验证选择模式激活 |
| 批量删除确认对话框 | BTCH-04 | UI 交互测试 | 选择多张图片点击删除，验证对话框显示正确数量和警告 |
| 收藏夹封面自动更新 | COLL-04 | 集成验证 | 添加图片后验证封面更新，移除封面图片后验证自动更换 |
| 侧边栏收藏夹列表展示 | COLL-05 | UI 验证 | 打开侧边栏，验证收藏夹列表显示正确（名称、数量、封面） |

---

## Validation Sign-Off

- [x] All tasks have <automated> verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 90s
- [x] nyquist_compliant: true set in frontmatter

**Approval:** pending

---

## Plan Execution Order



---

*Validation strategy created: 2026-03-17*
