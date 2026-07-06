# Implement similar image recommendations

## Goal

在图片详情页返回真实的「相似推荐」图片列表，让用户从一张作品能继续发现相关作品。当前后端 `ImageDetailResponse.similar_images` 永远返回空数组，前端组件已就绪但无数据可展示。

## User Value

用户在详情页看到与当前作品相关的其他作品，延长浏览路径；右侧「相似推荐」面板不再永远显示调试味空状态。

## Confirmed Facts

### 前端
- `frontend/vue-gallery/src/pages/DetailPage.vue:274` 已渲染 `<SimilarImagesPanel :images="detail.similar_images" />`，位于右侧 aside 栈底部。
- `frontend/vue-gallery/src/api/types.ts:58` 已定义 `similar_images: readonly ImageItem[]`；`client.ts:166` `getImage(id)` 返回含 `similar_images` 的 `ImageDetailResponse`。
- `frontend/vue-gallery/src/components/SimilarImagesPanel.vue` 已实现网格渲染 + 空状态，但空状态文案带调试味（`"后端返回 similar_images 为空时显示此状态"`），「更多」链接指向通用 `/search`。
- 网格使用 `.grid-2` class（推测 2 列布局）。
- `frontend/vue-gallery/src/pages/GalleryPage.vue` 当前不读取 URL 查询参数，用内部 `activeFilter` 状态管理过滤；`galleryImageQuery`（:139）不传 `tag` 字段。要让 `/?tag=<tag>` 深链生效，需给 GalleryPage 增加读取 `route.query.tag` 的小改动。

### 后端
- `internal/model/dto/image.go:30` `ImageDetailResponse.SimilarImages []ImageResponse` 字段已存在（JSON `similar_images`）。
- `internal/service/image.go:208` `newDetailResponse` 硬编码 `SimilarImages: []dto.ImageResponse{}` —— 始终空。
- `ImageRepository` 接口（`internal/ports/repositories.go:31` 与 `internal/service/image.go:40`）无相似度查询方法。
- 可用相似度信号：`image.category`（字符串）、`image_tag` 关联表（`internal/model/po/image_tag.go`，多对多）、`avg_score`/`favorite_count`/`view_count`（热度）。
- `ImageTagReader` 接口（`internal/service/image.go:62`）仅暴露 `ListByImageID(ctx, imageID) ([]do.Tag, error)`，返回 `do.Tag`（含 ID + Name）。
- 现有 `ListActive`（`internal/repository/image.go:78`）支持按单个标签名过滤（:195-198 JOIN image_tag + tag），但不支持多标签重叠打分。
- `TagRepository.ListByImageID`（`internal/repository/tag.go:143`）可查单张图片的标签列表。
- 详情服务在 `ImageService.Detail` → `newDetailResponse`（:194）中组装，已有 `tags` 字段填充逻辑（`imageTagNames` :213）可复用。
- 图片有软删除（`status` + `deleted_at`），查询须走 `activeImages` 限定（:182）。

### 架构约束
- 后端分层：handler → service → repository（接口在 `ports` 包），service 通过 `ImageTagReader` 读标签。
- SQLite WAL + GORM，读库 `readDB` / 写库 `writeDB` 分离；相似推荐是只读查询，走 `readDB`。
- 前端 Vue 3 + TS + Vite，无 UI 组件库依赖。

## Decisions

1. **相似度策略**：标签重叠为主 + category 回退。优先按与当前图片共享的标签数量降序排列；不足 6 张时用同 `category` 图片补足。标签是人工语义分组、相关性最高；category 兜底保证无标签或冷门标签的图片也有推荐。
2. **返回数量**：6 张。
3. **category 回退排序**：`view_count desc`，填充时排除当前图片 + 已选中的标签重叠结果（去重）。
4. **「更多」链接**：当前图片有标签时指向 `/?tag=<首个标签>`，无标签时回退 `/search`。需给 `GalleryPage` 增加读取 `route.query.tag` 的小改动（约 10 行）以支持 tag 深链。

## Requirements

- 在详情接口 `GET /api/v1/images/:id` 的响应中填充真实的 `similar_images` 列表（最多 6 张）。
- 相似推荐须排除当前图片自身。
- 相似推荐须只含可公开展示（未软删除）的图片。
- 标签重叠为主：按共享标签数降序；同数时按 `view_count desc` 二级排序。
- category 回退：标签重叠不足 6 张时，用同 `category` 图片按 `view_count desc` 补足，去重排除已选结果与当前图片。
- 当前图片无标签且无 category（或 category 命中也为空）时，`similar_images` 可为空数组，前端显示友好空状态。
- 前端 `SimilarImagesPanel` 空状态文案去除调试味，改为面向用户的友好文案。
- 前端「更多」链接：有标签时指向 `/?tag=<首个标签>`，无标签时指向 `/search`。
- `GalleryPage` 读取 `route.query.tag` 并传入 `getImages` 的 `tag` 参数，支持 tag 深链。
- 不改变 `ImageDetailResponse` 的 JSON 契约形状（字段名、类型不变）。
- 不引入新的外部依赖。

## Acceptance Criteria

- [ ] 调用 `GET /api/v1/images/:id`，当图片有标签且库内有共享标签的其他图片时，`similar_images` 非空且按共享标签数降序排列。
- [ ] `similar_images` 不包含当前 `:id` 对应的图片。
- [ ] `similar_images` 中不出现已软删除的图片。
- [ ] 标签重叠不足 6 张时，同 `category` 图片按 `view_count desc` 补足至最多 6 张，且不与标签重叠结果或当前图片重复。
- [ ] 当前图片无标签且无有效 category 时，`similar_images` 为空数组，前端显示友好空状态（非调试文案）。
- [ ] 前端「更多」链接：当前图片有标签时指向 `/?tag=<首个标签>`，无标签时指向 `/search`。
- [ ] 从「更多」链接进入 `/?tag=<tag>` 后，GalleryPage 显示对应标签的图片列表。
- [ ] 后端单测覆盖：标签重叠排序、排除自身、软删除过滤、category 回退填充与去重。
- [ ] `go test ./...` 通过。
- [ ] `cd frontend/vue-gallery && npm run build` 通过。

## Out of Scope

- 个性化推荐（基于用户历史的推荐）。
- 新增独立「相似推荐」API 端点（继续复用详情接口的 `similar_images` 字段）。
- 前端 `SimilarImagesPanel` 视觉重设计（仅清理调试文案 + 调整「更多」链接）。
- 引入向量嵌入 / 协同过滤等重型推荐算法。
- 热度加权的多因子排序（v1 仅标签重叠数 + view_count 二级排序）。

## Open Questions

- 无（所有产品决策已敲定，剩余为实现细节）。
