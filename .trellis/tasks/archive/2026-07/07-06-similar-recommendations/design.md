# Design: Similar Image Recommendations

## 概述

在 `ImageService.newDetailResponse` 中填充真实的 `similar_images`，策略为标签重叠为主 + category 回退，最多 6 张。前端清理调试文案 + 调整「更多」链接 + 给 GalleryPage 加 tag 深链支持。

## 架构与边界

遵循现有分层（`handler → service → ports 接口 ← repository 实现`）：

```
service/image.go (newDetailResponse)
  ├── s.tags.ListByImageID(ctx, imageID)   // 已有，复用以拿 tagIDs + names
  ├── s.repo.FindSimilarByTagIDs(...)       // 新增仓储方法
  └── s.repo.FindSimilarByCategory(...)     // 新增仓储方法（回退）
```

不新增 handler、不新增 API 端点、不改 DTO JSON 形状。改动集中在 service + repository + 两个接口定义 + 前端组件/页面。

## 数据流

```
GET /api/v1/images/:id
  → ImageService.Detail(ctx, id, userID)
    → repo.FindActiveByID(ctx, id)              // 已有
    → views.RecordView(...)                     // 已有
    → newDetailResponse(ctx, image)
      ├── tags.ListByImageID(ctx, image.ID)     // 已有，返回 []do.Tag
      │   → 派生 tagNames []string (给 Tags 字段)
      │   → 派生 tagIDs []int64  (给相似查询)
      ├── repo.FindSimilarByTagIDs(ctx, tagIDs, image.ID, 6)   // 新增
      │   → []do.Image (按重叠数 desc, view_count desc)
      ├── 若 len < 6:
      │   remaining = 6 - len
      │   excludeIDs = [image.ID] + 已选 image IDs
      │   repo.FindSimilarByCategory(ctx, image.Category, excludeIDs, remaining)  // 新增
      │   → []do.Image (按 view_count desc)
      └── 合并 → []dto.ImageResponse → SimilarImages 字段
```

## 契约

### 新增仓储接口方法

加到 `internal/ports/repositories.go` 的 `ImageRepository` 接口 和 `internal/service/image.go` 的 `ImageRepository` 接口（两者保持同步）：

```go
// FindSimilarByTagIDs 按标签重叠数查询相似图片，排除 excludeImageID，按重叠数降序、view_count 降序排序，limit 控制数量。
FindSimilarByTagIDs(ctx context.Context, tagIDs []int64, excludeImageID int64, limit int) ([]do.Image, error)

// FindSimilarByCategory 按分类查询相似图片，排除 excludeImageIDs，按 view_count 降序排序，limit 控制数量。
FindSimilarByCategory(ctx context.Context, category string, excludeImageIDs []int64, limit int) ([]do.Image, error)
```

### 边界行为
- `tagIDs` 为空时，`FindSimilarByTagIDs` 直接返回空切片（不查询）。
- `category` 为空字符串时，`FindSimilarByCategory` 直接返回空切片（不查询）。
- `limit <= 0` 时返回空切片。
- `excludeImageIDs` 为空时，`FindSimilarByCategory` 不加 `NOT IN` 条件。
- 两方法都只返回 `active` 且 `deleted_at IS NULL` 的图片。

## SQL 设计

### FindSimilarByTagIDs

```sql
SELECT image.*
FROM image
JOIN image_tag ON image_tag.image_id = image.id
WHERE image_tag.tag_id IN (?)
  AND image.id != ?
  AND image.status = 'active'
  AND image.deleted_at IS NULL
GROUP BY image.id
ORDER BY COUNT(image_tag.tag_id) DESC, image.view_count DESC
LIMIT ?
```

GORM 实现：
```go
func (r *ImageRepository) FindSimilarByTagIDs(ctx context.Context, tagIDs []int64, excludeImageID int64, limit int) ([]do.Image, error) {
    if len(tagIDs) == 0 || limit <= 0 || excludeImageID < 1 {
        return []do.Image{}, nil
    }
    var images []po.Image
    err := activeImages(r.readDB.WithContext(ctx)).
        Joins("JOIN image_tag ON image_tag.image_id = image.id").
        Where("image_tag.tag_id IN ?", tagIDs).
        Where("image.id != ?", excludeImageID).
        Group("image.id").
        Order("COUNT(image_tag.tag_id) DESC, image.view_count DESC").
        Limit(limit).
        Find(&images).Error
    if err != nil {
        return nil, pkgerrors.WithMessage(err, "find similar images by tag ids")
    }
    return imagesToDO(images), nil
}
```

### FindSimilarByCategory

```sql
SELECT image.*
FROM image
WHERE image.category = ?
  AND image.id NOT IN (?)
  AND image.status = 'active'
  AND image.deleted_at IS NULL
ORDER BY image.view_count DESC
LIMIT ?
```

GORM 实现：
```go
func (r *ImageRepository) FindSimilarByCategory(ctx context.Context, category string, excludeImageIDs []int64, limit int) ([]do.Image, error) {
    category = strings.TrimSpace(category)
    if category == "" || limit <= 0 {
        return []do.Image{}, nil
    }
    query := activeImages(r.readDB.WithContext(ctx)).Where("image.category = ?", category)
    if len(excludeImageIDs) > 0 {
        query = query.Where("image.id NOT IN ?", excludeImageIDs)
    }
    var images []po.Image
    err := query.Order("image.view_count DESC").Limit(limit).Find(&images).Error
    if err != nil {
        return nil, pkgerrors.WithMessage(err, "find similar images by category")
    }
    return imagesToDO(images), nil
}
```

## Service 编排

重构 `newDetailResponse`（`internal/service/image.go:194`）：

```go
func (s *ImageService) newDetailResponse(ctx context.Context, image do.Image) (dto.ImageDetailResponse, error) {
    response := s.toImageResponse(image)

    // 复用单次标签查询，同时派生 names 和 IDs
    tags, err := s.imageTags(ctx, image.ID)
    if err != nil {
        return dto.ImageDetailResponse{}, err
    }
    tagNames := make([]string, 0, len(tags))
    tagIDs := make([]int64, 0, len(tags))
    for _, tag := range tags {
        tagNames = append(tagNames, tag.Name)
        tagIDs = append(tagIDs, tag.ID)
    }

    similar, err := s.findSimilarImages(ctx, image, tagIDs, similarImageLimit)
    if err != nil {
        return dto.ImageDetailResponse{}, err
    }

    return dto.ImageDetailResponse{
        Image:         response,
        Tags:          tagNames,
        AvgScore:      image.AvgScore,
        RatingCount:   image.RatingCount,
        FavoriteCount: image.FavoriteCount,
        MyRating:      nil,
        IsCollected:   false,
        SimilarImages: similar,
    }, nil
}
```

新增编排方法：

```go
const similarImageLimit = 6

func (s *ImageService) findSimilarImages(ctx context.Context, image do.Image, tagIDs []int64, limit int) ([]dto.ImageResponse, error) {
    // 1. 标签重叠
    byTag, err := s.repo.FindSimilarByTagIDs(ctx, tagIDs, image.ID, limit)
    if err != nil {
        return nil, pkgerrors.WithMessage(err, "find similar by tags")
    }
    result := byTag
    if len(result) >= limit {
        return s.toImageResponseList(result[:limit]), nil
    }
    // 2. category 回退
    remaining := limit - len(result)
    excludeIDs := make([]int64, 0, len(result)+1)
    excludeIDs = append(excludeIDs, image.ID)
    for _, img := range result {
        excludeIDs = append(excludeIDs, img.ID)
    }
    byCategory, err := s.repo.FindSimilarByCategory(ctx, image.Category, excludeIDs, remaining)
    if err != nil {
        return nil, pkgerrors.WithMessage(err, "find similar by category")
    }
    result = append(result, byCategory...)
    return s.toImageResponseList(result), nil
}
```

`imageTags` 替代原 `imageTagNames`（返回 `[]do.Tag` 而非 `[]string`），原 `imageTagNames` 改为薄封装或删除。

## 前端改动

### SimilarImagesPanel.vue
- 新增可选 prop `moreLinkTag?: string`。
- 「更多」链接：`moreLinkTag` 非空时指向 `/?tag=<encoded tag>`，否则 `/search`。
- 空状态文案：`"暂无相似作品"` + `"还没有相关的作品推荐"`（去除调试味）。

### DetailPage.vue:274
```vue
<SimilarImagesPanel :images="detail.similar_images" :more-link-tag="detail.tags[0]" />
```

### GalleryPage.vue
- 新增 `import { useRoute } from 'vue-router'` + `const route = useRoute()`。
- 读取 `route.query.tag`（string 类型），存为 `activeTag` ref。
- `galleryImageQuery` 增加 `tag` 字段：当 `activeTag` 非空时传入。
- `onMounted` / `onActivated` 时若 URL 有 tag，触发带 tag 的加载。
- 可选：在过滤栏显示当前 tag chip（v1 可略，仅保证查询生效）。

## 兼容性

- DTO JSON 形状不变，`similar_images` 从永远 `[]` 变为可能有数据，前端已处理非空渲染。
- 不改路由形状、不改 API 端点。
- 新增仓储方法是接口扩展，`repository.ImageRepository` 已实现其他方法，新增方法需同步实现（否则编译失败 → 强制实现）。

## 测试策略

### 后端单测
- `internal/repository/image_test.go`：新增 `TestFindSimilarByTagIDs`、`TestFindSimilarByCategory`，覆盖排序、排除自身、软删除过滤、空入参边界。
- `internal/service/image_test.go`（若不存在则新建）：用 mock `ImageRepository` + mock `ImageTagReader` 测试 `findSimilarImages` 编排逻辑（标签重叠足够、不足触发回退、无标签无 category 返回空）。

### 前端
- `npm run build` 类型检查通过（无单测框架）。

## 风险与回退

- **风险**：`GROUP BY image.id` + `COUNT` 在 SQLite 上数据量大时可能慢。**缓解**：`LIMIT 6` 限制结果集；`image_tag` 表已有 `image_id`/`tag_id` 索引。若未来慢可加 `(tag_id, image_id)` 复合索引。
- **回退点**：若相似查询失败，`newDetailResponse` 应返回错误（不静默吞），handler 层已有错误处理。可考虑相似查询失败时降级为空数组而非整体失败 —— v1 选择整体失败以暴露问题，后续可调整。
