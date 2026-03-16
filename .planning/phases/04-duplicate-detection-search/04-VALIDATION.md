---
phase: 04
slug: duplicate-detection-search
status: ready
nyquist_compliant: true
wave_0_complete: false
created: 2026-03-16
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go test (标准库) + Flutter test |
| **Config file** | `go.mod` / `flutter_app/pubspec.yaml` |
| **Quick run command** | `go test ./internal/... -short` |
| **Full suite command** | `go test ./... && cd flutter_app && flutter test` |
| **Estimated runtime** | ~30 seconds (后端) + ~10 seconds (前端) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/... -short`
- **After every plan wave:** Run `go test ./... && cd flutter_app && flutter test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 45 seconds

---

## Per-task Verification Map

| task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 1 | DUPD-01 | unit | `go test ./internal/service/... -run TestFileHash` | ❌ W0 | ⬜ pending |
| 04-01-02 | 01 | 1 | DUPD-02 | unit | `go test ./internal/service/... -run TestPHash` | ❌ W0 | ⬜ pending |
| 04-01-03 | 01 | 1 | DUPD-02 | unit | `go test ./internal/service/... -run TestHammingDistance` | ❌ W0 | ⬜ pending |
| 04-01-04 | 01 | 1 | DUPD-03 | unit | `go test ./internal/service/... -run TestDuplicateDetection` | ❌ W0 | ⬜ pending |
| 04-01-05 | 01 | 1 | DUPD-04 | api | `go test ./internal/handler/... -run TestDuplicateAPI` | ❌ W0 | ⬜ pending |
| 04-02-01 | 02 | 1 | SRCH-01 | unit | `go test ./internal/service/... -run TestFTSSearch` | ❌ W0 | ⬜ pending |
| 04-02-02 | 02 | 1 | SRCH-02 | unit | `go test ./internal/service/... -run TestTagSearch` | ❌ W0 | ⬜ pending |
| 04-02-03 | 02 | 1 | SRCH-03 | unit | `go test ./internal/service/... -run TestCombinedSearch` | ❌ W0 | ⬜ pending |
| 04-02-04 | 02 | 1 | SRCH-05 | api | `go test ./internal/handler/... -run TestSearchAPI` | ❌ W0 | ⬜ pending |
| 04-03-01 | 03 | 2 | DUPD-05 | widget | `flutter test test/widgets/duplicate_group_test.dart` | ❌ W0 | ⬜ pending |
| 04-04-01 | 04 | 2 | SRCH-04 | unit | `go test ./internal/handler/... -run TestImageSearchAPI` | ❌ W0 | ⬜ pending |
| 04-04-02 | 04 | 2 | SRCH-04 | widget | `flutter test test/screens/search_screen_test.dart` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/service/duplicate_service_test.go` — 测试文件哈希、pHash、汉明距离
- [ ] `internal/service/search_service_test.go` — 测试 FTS 搜索
- [ ] `internal/handler/duplicate_handler_test.go` — 测试重复检测 API
- [ ] `internal/handler/search_handler_test.go` — 测试搜索 API
- [ ] `flutter_app/test/services/search_service_test.dart` — 测试搜索服务
- [ ] `flutter_app/test/screens/search_screen_test.dart` — 测试搜索界面

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 以图搜图拖拽上传 | SRCH-04 | 桌面端交互测试 | 1. 启动 Flutter 应用 2. 进入搜索页面 3. 拖拽图片到上传区域 4. 验证搜索结果 |
| 重复图片删除确认 | DUPD-05 | 用户交互流程 | 1. 进入重复管理页面 2. 选择要删除的图片 3. 确认删除操作 4. 验证文件实际被删除 |
| 搜索结果排序切换 | SRCH-05 | UI 交互测试 | 1. 执行搜索 2. 切换排序方式 3. 验证结果顺序变化 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 45s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending