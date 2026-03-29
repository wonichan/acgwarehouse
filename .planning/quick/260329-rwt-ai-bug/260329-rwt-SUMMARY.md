# Quick Task 260329-rwt: 修复重试失败任务时AI标签计数错误增加的bug

## Summary

成功修复了重试失败任务时AI标签计数错误增加的bug。

## Bug根因

在 `tag_governance_service.go` 的 `MergeTags` 函数中：
- 第78行无条件调用 `IncrementUsageCount`
- 当图片-标签关联已存在时（如重试失败任务），计数仍被增加
- 导致同一张图片被重复计数

## 修复方案

1. 在 `ImageTagRepository` 添加 `Exists(ctx, imageID, tagID) (bool, error)` 方法
2. 在 `MergeTags` 中增加计数前先检查关联是否已存在
   - 关联已存在 → 跳过计数增加
   - 关联不存在 → 正常增加计数

## Changed Files

| File | Change |
|------|--------|
| `internal/repository/image_tag_repository.go` | 添加 `Exists` 方法（接口+实现） |
| `internal/service/tag_governance_service.go` | 修改 `MergeTags` 逻辑，添加关联存在性检查 |
| `internal/service/tag_governance_service_test.go` | 添加回归测试 |

## Verification

- 所有标签相关测试通过（12个）
- 新增回归测试验证：重复调用 `MergeTags` 时计数不再增加
- 编译成功，无错误

## Test Case

`TestTagGovernanceMergeTagsDoesNotIncrementCountWhenAssociationExists`:
- 第一次 `MergeTags`: 计数从5增加到6 ✓
- 第二次 `MergeTags`（模拟重试）: 计数保持6，不再增加 ✓