# Quick Task 25 Summary

## 任务完成
修复了AI生成标签的数据一致性问题：删除标签后筛选计数未更新，脏数据残留

## 修改文件

### 1. internal/repository/tag_repository.go
- 添加 `DecrementUsageCount` 方法到 TagRepository 接口
- 实现 `DecrementUsageCount`，使用 `MAX(usage_count - 1, 0)` 防止负数

### 2. internal/handler/image_tag_handler.go
- **RemoveImageTag**: 删除标签后调用 `DecrementUsageCount` 减少 usage_count
- **MergeImageTag**: 
  - 检查 target 标签是否已存在
  - source 标签 usage_count 减1
  - target 标签只在不存在时才增加 usage_count
- **ReviewTag**: 拒绝标签时减少 usage_count
- **BatchReview**: 批量拒绝时逐个减少 usage_count

## 测试结果
所有测试通过：
```
ok  	github.com/wonichan/acgwarehouse-backend/internal/handler	2.472s
ok  	github.com/wonichan/acgwarehouse-backend/internal/repository	3.783s
ok  	github.com/wonichan/acgwarehouse-backend/...
```

## 修复效果
- 删除标签后，tags.usage_count 正确递减
- 标签筛选时显示的计数与实际图片数量一致
- 合并标签时正确处理两个标签的 usage_count
- 拒绝标签时正确减少 usage_count

## 技术细节
- 使用 `MAX(usage_count - 1, 0)` 确保 usage_count 不会变成负数
- 批量操作时逐个处理，确保每个标签的 usage_count 都被正确更新
- 合并标签时检查 target 是否已存在，避免重复增加
