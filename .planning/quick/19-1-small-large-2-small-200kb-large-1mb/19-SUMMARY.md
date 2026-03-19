# Quick Task 19: 缩略图任务修改

**Created:** 2026-03-19
**Completed:** 2026-03-19
**Duration:** ~4 minutes

## Summary

修改缩略图生成任务的文件名格式和大小控制逻辑：
1. 文件名格式改为 `{原文件名}-{size}.jpg`
2. small缩略图最小200KB，large缩略图最大1MB

## Changes

### Task 1: 修改缩略图文件名格式

**Files modified:**
- `internal/service/scanner_service.go` - 添加 `filename` 字段到 `image_imported` 任务 payload
- `cmd/server/main.go` - 传递 `filename` 到 `thumbnail_generate` 任务
- `internal/worker/thumbnail_handler.go` - 添加 `Filename` 字段，传递到 `Upload` 方法
- `internal/service/cos_service.go` - `Upload` 方法签名改为 `filename` 参数，使用新格式

**Before:** `thumbnails/{image_id}_{size}.jpg`
**After:** `thumbnails/{filename}-{size}.jpg`

### Task 2: 实现动态缩略图大小调整

**Files modified:**
- `internal/service/thumbnail_service.go` - 实现动态大小调整逻辑

**Logic:**
- **small 缩略图:** 如果生成后小于 200KB，逐步增加宽度（每次+100px）或质量（每次+5）
- **large 缩略图:** 如果生成后大于 1MB，逐步降低质量（每次-5）
- 设置最大调整迭代次数（10次），避免无限循环

## Commits

| Commit | Description |
|--------|-------------|
| `50d4728` | feat(quick-19): 修改缩略图文件名格式为 {原文件名}-{size}.jpg |
| `3f30825` | feat(quick-19): 实现动态缩略图大小调整 |

## Verification

- [x] Go build passes
- [x] All related tests pass
  - `TestCOSServiceUploadReturnsURLAndUsesKeyFormat`
  - `TestCOSServiceUploadReturnsErrorOnFailure`
  - `TestThumbnailServiceGenerateThumbnail*`
  - `TestThumbnailHandler*`

## Deviations

None - plan executed exactly as written.