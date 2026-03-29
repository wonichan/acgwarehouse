# 修复重试失败任务时AI标签计数错误增加的bug

## Bug描述
在重试失败的AI标签生成任务时，`MergeTags` 函数会重复增加标签的 `UsageCount`，导致计数不准确。

## 根因分析
`MergeTags` 函数（第34-103行）存在以下问题：
1. **第78行**：当标签已存在时，无条件调用 `IncrementUsageCount` 增加计数
2. **第90-99行**：使用 `INSERT OR REPLACE` 替换图片-标签关联（覆盖已存在记录）

**问题流程**：
- 第一次AI标签生成 → 创建标签 → 计数+1 → 创建关联 ✓
- 任务失败（部分数据已创建）
- 重试 → 发现标签存在 → 计数+1 → 替换关联 → 计数重复增加 ✗

## 修复方案

### Task 1: 添加 `Exists` 方法到 ImageTagRepository
**文件**: `internal/repository/image_tag_repository.go`

在 `ImageTagRepository` 接口中添加：
```go
Exists(ctx context.Context, imageID, tagID int64) (bool, error)
```

实现检查指定图片-标签关联是否已存在的查询。

### Task 2: 修改 MergeTags 函数逻辑
**文件**: `internal/service/tag_governance_service.go`

在第78行 `IncrementUsageCount` 调用前，先检查图片-标签关联是否已存在：
- 如果关联已存在 → 跳过计数增加，只保存关联
- 如果关联不存在 → 增加计数，创建关联

## 实现细节

### Task 1: 添加 Exists 方法

**接口变更** (`image_tag_repository.go` 第12-24行):
```go
type ImageTagRepository interface {
    // ... existing methods
    Exists(ctx context.Context, imageID, tagID int64) (bool, error)  // 新增
}
```

**实现** (`image_tag_repository.go`):
```go
func (r *imageTagRepository) Exists(ctx context.Context, imageID, tagID int64) (bool, error) {
    var count int64
    err := r.db.QueryRowContext(ctx, `
        SELECT COUNT(*) FROM image_tags WHERE image_id = ? AND tag_id = ?
    `, imageID, tagID).Scan(&count)
    if err != nil {
        return false, err
    }
    return count > 0, nil
}
```

### Task 2: 修改 MergeTags 逻辑

**修改点** (`tag_governance_service.go` 第66-99行):
```go
// 原逻辑（第78行）: 无条件增加计数
if err := s.tagRepo.IncrementUsageCount(ctx, tag.ID); err != nil {
    return err
}

// 新逻辑: 先检查关联是否存在
exists, err := s.imageTagRepo.Exists(ctx, imageID, tag.ID)
if err != nil {
    return err
}
if !exists {
    // 只有关联不存在时才增加计数
    if err := s.tagRepo.IncrementUsageCount(ctx, tag.ID); err != nil {
        return err
    }
}
```

## 向后兼容性
- 新增 `Exists` 方法不破坏现有接口实现
- `MergeTags` 行为变更仅影响计数增加逻辑，其他功能保持不变

## 验证方式
1. 运行现有单元测试
2. 手动测试场景：
   - 首次调用 MergeTags → 计数正确+1
   - 相同参数再次调用 → 计数不应再增加