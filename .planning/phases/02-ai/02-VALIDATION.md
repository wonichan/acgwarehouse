---
phase: 02
slug: ai
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-14
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go test (后端) + Flutter test (前端) |
| **Config file** | Go: 无需额外配置; Flutter: `pubspec.yaml` |
| **Quick run command** | `go test ./internal/... -short` |
| **Full suite command** | `go test ./... && cd flutter_app && flutter test` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/... -short`
- **After every plan wave:** Run `go test ./...` (后端全量)
- **Before `/gsd-verify-work`:** 全量测试必须通过
- **Max feedback latency:** 30 秒

---

## Per-task Verification Map

| task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 1 | IMPT-05 | unit | `go test ./internal/service/thumbnail_test.go` | ❌ W0 | ⬜ pending |
| 02-01-02 | 01 | 1 | IMPT-05 | unit | `go test ./internal/service/cos_test.go` | ❌ W0 | ⬜ pending |
| 02-02-01 | 02 | 1 | IMPT-06 | unit | `go test ./internal/service/phash_test.go` | ❌ W0 | ⬜ pending |
| 02-03-01 | 03 | 2 | GALR-01,02,03,04,05 | widget | `flutter test test/widgets/` | ❌ W0 | ⬜ pending |
| 02-03-02 | 03 | 2 | GALR-03 | widget | `flutter test test/screens/image_detail_test.dart` | ❌ W0 | ⬜ pending |
| 02-04-01 | 04 | 2 | IMPT-07, GALR-05 | integration | `go test ./internal/handler/images_test.go` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

### Go 后端测试文件

- [ ] `internal/service/thumbnail_test.go` — 缩略图生成测试桩
- [ ] `internal/service/phash_test.go` — 感知哈希计算测试桩
- [ ] `internal/service/cos_test.go` — COS 上传测试桩 (使用 mock)
- [ ] `internal/handler/images_test.go` — 图片 API 端点测试桩
- [ ] `internal/repository/image_repository_test.go` — 扩展分页查询测试

### Flutter 前端测试文件

- [ ] `flutter_app/test/widgets/image_grid_test.dart` — 网格组件测试桩
- [ ] `flutter_app/test/widgets/image_masonry_test.dart` — 瀑布流组件测试桩
- [ ] `flutter_app/test/screens/gallery_screen_test.dart` — 图片库页面测试桩
- [ ] `flutter_app/test/screens/image_detail_screen_test.dart` — 详情页测试桩

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| COS 实际上传 | IMPT-05 | 需要真实凭证和存储桶 | 使用 `COS_SECRET_ID` 和 `COS_SECRET_KEY` 环境变量运行 `go test -run TestCOSUpload ./internal/service/` |
| Flutter UI 渲染 | GALR-01,02 | Widget 测试不完全反映真实渲染 | 在模拟器/真机上运行 `flutter run`，检查网格和瀑布流布局 |
| 图片缩放交互 | GALR-03 | 需要触摸手势测试 | 在详情页测试双指缩放和拖动 |
| 内存使用 | Pitfall 3 | 需要性能分析工具 | 使用 Flutter DevTools 监控内存，滚动大量图片 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending