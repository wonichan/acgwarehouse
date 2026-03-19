# Quick Task 19: 缩略图任务修改

**Created:** 2026-03-19
**Status:** Ready for execution

## Overview

修改缩略图生成任务：
1. 文件名格式改为 `{原文件名}-{size}.jpg`
2. small缩略图最小200KB，large缩略图最大1MB

## Tasks

### Task 1: 修改缩略图文件名格式

**Files:**
- `internal/service/scanner_service.go`
- `cmd/server/main.go`
- `internal/worker/thumbnail_handler.go`
- `internal/service/cos_service.go`

**Action:**
1. 修改 `scanner_service.go`：在创建 `image_imported` 任务时，payload 中添加 `filename` 字段
2. 修改 `main.go`：在 `image_imported` 处理器创建 `thumbnail_generate` 任务时，传递 `filename`
3. 修改 `thumbnail_handler.go`：
   - 在 `thumbnailJobPayload` 结构体添加 `Filename` 字段
   - 修改 `COSService.Upload` 调用，传递 `filename`
4. 修改 `cos_service.go`：
   - `Upload` 方法签名添加 `filename` 参数
   - 文件名格式改为 `thumbnails/{filename}-{size}.jpg`

**Verify:**
- 编译通过
- 缩略图上传后文件名符合新格式

**Done:**
- 缩略图文件名格式为 `{原文件名}-small.jpg` 和 `{原文件名}-large.jpg`

---

### Task 2: 实现动态缩略图大小调整

**Files:**
- `internal/service/thumbnail_service.go`

**Action:**
1. 修改 `GenerateThumbnail` 方法，实现动态调整：
   - small: 生成后检查大小，如果小于200KB则逐步增大宽度（每次+100px）或质量（每次+5）
   - large: 生成后检查大小，如果大于1MB则逐步降低质量（每次-5）
2. 添加 `minSmallSize` (200KB) 和 `maxLargeSize` (1MB) 常量
3. 设置调整上限，避免无限循环

**Verify:**
- 编译通过
- 运行单元测试

**Done:**
- small缩略图 >= 200KB
- large缩略图 <= 1MB

---

## Summary

| Task | Files | Complexity |
|------|-------|------------|
| 1 | 4 files | Medium |
| 2 | 1 file | Medium |

**Estimated effort:** 1-2 hours