# Quick Task 26: 调查Flutter图片搜索功能-标签名搜索无结果

**日期:** 2026-03-20
**状态:** 已完成
**提交:** 0d7cc54

## 问题诊断

用户报告在Flutter应用中通过搜索功能输入标签名时，无法搜索到对应图片。

### 根本原因

FTS (Full-Text Search) 全文搜索索引与标签数据未同步：

1. **初始化问题**: 图片插入时，FTS记录的`tags`字段初始化为空字符串
2. **同步缺失**: 当添加/删除标签时，`image_tag_repository.go`只更新`image_tags`表，从未更新FTS索引
3. **结果**: `images_fts.tags`字段始终为空，搜索标签名时FTS找不到任何匹配

## 修复内容

### 1. image_tag_repository.go - 添加FTS同步

```go
// 新增辅助方法：同步图片的FTS标签
func (r *imageTagRepository) syncImageFTS(ctx context.Context, imageID int64) error {
    // 聚合图片所有标签
    // 更新images_fts.tags字段
}

// Save() - 添加标签后同步FTS
// Delete() - 删除标签后同步FTS  
// MergeImageTag() - 合并标签后同步FTS
```

### 2. routes.go - 添加重建FTS API

```go
// POST /admin/api/actions/search/rebuild-fts
// 用于修复现有数据的FTS索引
```

### 3. main.go - 传递DB到Dependencies

```go
handler.SetupRoutes(r, &handler.Dependencies{
    // ...
    DB: db,  // 新增：用于FTS重建
})
```

### 4. fts_sync_test.go - 添加测试

- `TestImageTagSaveUpdatesFTS`: 验证添加标签后FTS同步
- `TestImageTagDeleteUpdatesFTS`: 验证删除标签后FTS同步

## 测试结果

```
=== RUN   TestImageTagSaveUpdatesFTS
--- PASS: TestImageTagSaveUpdatesFTS (0.26s)
=== RUN   TestImageTagDeleteUpdatesFTS  
--- PASS: TestImageTagDeleteUpdatesFTS (0.26s)
PASS
```

所有测试通过。

## 使用指南

### 修复现有数据

如果用户已有图片和标签数据，需要重建FTS索引：

```bash
curl -X POST http://localhost:8080/admin/api/actions/search/rebuild-fts \
  -u admin:password
```

### 新数据

修复后，新添加/删除的标签会自动同步到搜索索引。

## 搜索能力

| 搜索内容 | 是否支持 |
|---------|---------|
| 文件名 | ✅ |
| 标签名 | ✅ (已修复) |
| tag_ids筛选 | ✅ |