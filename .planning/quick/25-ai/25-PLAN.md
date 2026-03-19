# Quick Task 25: 修复AI生成标签的数据一致性问题

## 问题描述

AI生成标签后，用户点击接受，再点击删除标签，退回到首页标签筛选时出现问题：
1. 生成的标签修改后，原始标签仍出现在数据库中（脏数据）
2. 删除标签后，标签筛选中的计数仍为1，但点进去发现没有图片
3. 生成后的标签操作与标签数据库/筛选功能没有联动

## 根因分析

**核心问题**：标签删除时 `usage_count` 未同步递减

**代码位置**：
- `RemoveImageTag` (`image_tag_handler.go:143-159`)：只删除 image_tags 记录，**没有减少 tags.usage_count**
- `AddImageTag` (`image_tag_handler.go:130-133`)：正确增加了 usage_count
- `MergeImageTag`：没有正确处理 source 和 target 标签的 usage_count
- `ReviewTag`/`BatchReview`：拒绝标签时没有减少 usage_count

**结果**：usage_count 只增不减，导致筛选时显示错误的计数

## 修复方案

### 1. 添加 DecrementUsageCount 方法
- 文件：`internal/repository/tag_repository.go`
- 添加 `DecrementUsageCount` 方法到 TagRepository 接口和实现
- 使用 `MAX(usage_count - 1, 0)` 防止负数

### 2. 修复 RemoveImageTag
- 文件：`internal/handler/image_tag_handler.go`
- 在删除 image_tags 记录后，调用 `tagRepo.DecrementUsageCount`

### 3. 修复 MergeImageTag
- 文件：`internal/handler/image_tag_handler.go`
- 检查 target 标签是否已存在于该图片
- source 标签 usage_count 减1
- target 标签只在不存在时才增加 usage_count

### 4. 修复 ReviewTag/BatchReview
- 文件：`internal/handler/image_tag_handler.go`
- 拒绝标签时减少 usage_count
- 确认标签时不修改 usage_count（因为添加时已经增加了）

## 修复验证

所有测试通过：
- `go test ./internal/handler/...` ✓
- `go test ./internal/repository/...` ✓
- `go test ./... -short` ✓
