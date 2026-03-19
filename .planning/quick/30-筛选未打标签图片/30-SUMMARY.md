# Quick Task 30: 添加筛选未打标签图片功能 - 摘要

**完成日期:** 2026-03-20
**状态:** 已完成

## 变更概述

实现了"筛选未打标签图片"功能，允许用户在图片库中快速找到没有打过标签的图片。

## 修改的文件

### 后端 (Go)

| 文件 | 变更 |
|------|------|
| `internal/repository/image_repository.go` | 添加 `FindUntagged` 和 `CountUntagged` 方法 |
| `internal/handler/image_handler.go` | 添加 `has_tags` 查询参数支持，验证与 `tag_ids` 互斥 |

### 前端 (Flutter)

| 文件 | 变更 |
|------|------|
| `flutter_app/lib/services/api_service.dart` | 添加 `hasTags` 参数到 `fetchImages` |
| `flutter_app/lib/providers/image_provider.dart` | 添加 `hasTagsFilter` 状态和 `setHasTagsFilter` 方法 |
| `flutter_app/lib/widgets/tag_filter_drawer.dart` | 添加"未打标签"开关 UI |
| `flutter_app/lib/screens/gallery_screen.dart` | 连接 hasTagsFilter 回调 |

## 技术实现

### 后端

1. **Repository 层**: 使用 LEFT JOIN 查询没有关联标签的图片
   ```sql
   SELECT i.* FROM images i
   LEFT JOIN image_tags it ON it.image_id = i.id
   WHERE it.image_id IS NULL
   ```

2. **Handler 层**: 支持 `has_tags` 查询参数
   - `has_tags=false`: 返回未打标签的图片
   - `has_tags=true`: 返回所有图片（同默认行为）
   - 与 `tag_ids` 参数互斥，同时使用返回 400 错误

### 前端

1. **API Service**: 扩展 `fetchImages` 支持 `hasTags` 参数
2. **Provider**: 管理 `hasTagsFilter` 状态，与 `selectedTagIds` 互斥
3. **UI**: 在标签筛选抽屉中添加 SwitchListTile 开关

## 测试

### 后端测试
- `TestFindUntaggedReturnsOnlyImagesWithoutTags` - 验证只返回无标签图片
- `TestFindUntaggedSupportsPagination` - 验证分页
- `TestFindUntaggedSupportsSorting` - 验证排序
- `TestCountUntaggedReturnsCorrectCount` - 验证计数
- `TestFindUntaggedReturnsEmptyWhenAllImagesHaveTags` - 边界情况
- `TestImageHandlerListImagesFiltersByHasTagsFalse` - Handler 测试
- `TestImageHandlerListImagesHasTagsFalseWithTagIDsReturnsError` - 互斥验证

### 前端验证
- `flutter analyze` 无错误

## 使用方法

1. 打开图片库页面
2. 点击左上角筛选按钮打开标签筛选抽屉
3. 打开"未打标签"开关
4. 列表将只显示没有标签的图片
5. 关闭开关或选择特定标签可恢复正常显示

## API 变更

新增查询参数:

```
GET /api/v1/images?has_tags=false
```

返回格式不变，包含 `images`, `next_cursor`, `has_more`, `total` 字段。