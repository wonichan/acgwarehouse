# 收藏夹封面与详情页重构

## Goal

重构前端收藏夹页面：支持为收藏夹设置封面（从夹内图片中选一张作为封面），移除无意义的 ID/Owner/public 标签和 imageID 列表，改为点击收藏夹卡片进入详情页查看所有收藏图片。

## Background

### 当前状态（代码事实）

**前端 `CollectionsPage.vue`**：
- 封面区域（`album-cover--meta`）显示收藏数量数字，非图片封面（CollectionsPage.vue:187-190）
- `span.tag` 显示 `ID {{ collection.id }}`、`Owner {{ collection.user_id }}`、`{{ collection.visibility }}`（CollectionsPage.vue:196-198）
- `collection-item` 列表显示 `Image {{ item.image_id }}` 和收藏时间（CollectionsPage.vue:200-210）
- 路由只有 `/collections`，无 `/collections/:id` 详情页路由（router/index.ts:29-34）

**前端 API 层**：
- 已有 `getCollection(id)` 方法（client.ts:201-204），但 `CollectionDetailResponse` 是空扩展（types.ts:111）
- `CollectionItemResponse` 只有 `collection_id`、`image_id`、`created_at`，缺图片 URL/尺寸等信息（types.ts:95-99）

**后端模型**：
- `po.Collection` 无 `CoverImageID` 字段（po/collection.go:8-15）
- `do.Collection` 无 `CoverImageID` 字段（do/collection.go:16-23）
- `dto.CollectionResponse` 无 `cover_image_id` 字段（dto/collection.go:21-28）
- `dto.CollectionUpdateRequest` 只接收 `name` 和 `visibility`（dto/collection.go:10-13）
- `dto.CollectionItemResponse` 只有 `CollectionID`、`ImageID`、`CreatedAt`（dto/collection.go:31-35）
- `collectionToDO` / `collectionToPO` 不传递封面字段（collection_mapper.go:18-37）

**后端路由与服务**：
- `GET /collections/:id` (Detail) 已存在，公开访问（router.go:126）
- `PUT /collections/:id` (Update) 已存在，Auth required（router.go:127）
- `CollectionRepository.FindVisible` 预加载 Items 但不预加载 Image 关联
- `CollectionService` 无 `cosBase` 依赖，无法生成图片 URL
- `po.CollectionItem` 已有 `Image Image` gorm 关联（po/collection_item.go:13），只需 Preload

**GalleryPage 展示方式**：
- masonry 瀑布流列布局 + `ArtCard` 组件（GalleryPage.vue:62-134）
- `imageToArtItem` 将 `ImageItem` 转为 `ArtItem`（imagePresentation.ts:11-33）
- 通过 ResizeObserver 响应式调整列数，支持无限滚动

## Requirements

### R1: 收藏夹封面字段（后端数据模型）

- `po.Collection` 新增 `CoverImageID *int64` 字段（nullable，GORM 自动迁移）
- `do.Collection` 新增 `CoverImageID int64` 字段（0 表示未设置）
- `dto.CollectionResponse` 新增 `cover_image_id` 和 `cover_image_url` 字段
- `dto.CollectionUpdateRequest` 新增可选 `CoverImageID *int64` 字段
- `collection_mapper.go` 传递封面字段

### R2: 封面设置 API（后端业务逻辑）

- `PUT /collections/:id` 支持更新 `cover_image_id`
- `CollectionService` 注入 `cosBase`，生成封面图片 URL
- service 层校验：`cover_image_id > 0` 时对应的图片必须在当前收藏夹的 items 中
- `cover_image_id = 0` 或 `null` 时清空封面
- 未设置封面时：`cover_image_url` fallback 为 items 中第一张图片的 URL（空收藏夹则空字符串）

### R3: 收藏夹详情 API 返回图片信息（后端业务逻辑）

- `dto.CollectionItemResponse` 扩展嵌套 `Image *dto.ImageResponse` 字段
- `CollectionRepository.FindVisible` 和 `ListByOwner` 预加载 `Items.Image` 关联
- `collectionToDO` 传递嵌套 `do.Image` 到 `do.CollectionItem`
- handler `collectionToResponse` 构造 item 的图片 URL（复用 `imageURL` 逻辑）

### R4: 收藏夹列表页重构（前端）

- 卡片封面区域显示图片：有封面显示封面图，无封面 fallback 第一张，空收藏夹显示占位图
- 移除 `span.tag`（ID/Owner/visibility 三行）
- 移除 `collection-item` 列表（不在列表页展示条目）
- 卡片整体可点击，RouterLink 跳转到 `/collections/:id` 详情页
- 保留收藏夹名称、数量、可见性、创建时间 meta 信息

### R5: 收藏夹详情页（前端）

- 新增路由 `/collections/:id`
- 新页面 `CollectionDetailPage.vue` 展示收藏夹内所有收藏图片
- 参照 GalleryPage 的 masonry + ArtCard 展示方式（复用 ArtCard 组件和 imageToArtItem）
- 不需要无限滚动（收藏夹图片数量有限）
- 每张图片卡片提供"设为封面"操作（owner 才能操作，非 owner 不显示）
- 已设为封面的图片显示标记（如角标）
- 图片可点击进入图片详情 `/detail?id=`
- 提供返回列表页导航

## Acceptance Criteria

- [ ] AC1: 后端 `collection` 表新增 `cover_image_id` 列，GORM AutoMigrate 成功
- [ ] AC2: `PUT /collections/:id` 传入有效 `cover_image_id` 能更新封面，传入不在此收藏夹的 image_id 返回错误
- [ ] AC3: `GET /collections/:id` 返回的每个 item 包含完整图片信息（id, url, filename, width, height 等）
- [ ] AC4: `GET /collections` 列表和 `GET /collections/:id` 详情均返回 `cover_image_id` 和 `cover_image_url`
- [ ] AC5: 未设置封面时 `cover_image_url` 返回 items 中第一张图片的 URL
- [ ] AC6: 前端 `/collections` 列表页卡片显示封面图片（有封面时）
- [ ] AC7: 前端列表页无 ID/Owner/visibility tag、无 imageID 条目列表
- [ ] AC8: 前端 `/collections/:id` 详情页列出所有收藏图片
- [ ] AC9: 详情页中可设置封面，设置后列表页封面同步更新
- [ ] AC10: 非收藏夹 owner 访问详情页不显示"设为封面"按钮
- [ ] AC11: `go test ./...` 通过
- [ ] AC12: `cd frontend/vue-gallery && npm run build` 通过

## Confirmed Decisions

- **封面设置入口**：收藏夹详情页内，每张图片卡片提供"设为封面"操作（owner 才能）
- **封面 fallback**：未设置封面时自动使用 items 中第一张图片的 URL，空收藏夹返回空字符串
- **详情页布局**：参照 GalleryPage 的 masonry + ArtCard 展示方式，复用 `imageToArtItem` 和 `ArtCard` 组件，不含无限滚动
- **设置封面请求方式**：前端先 GET 当前 collection 全量数据，再 PUT 提交（保持 name/visibility 必填校验不变）

## Out of Scope

- 收藏夹排序/拖拽
- 收藏夹批量操作
- 收藏夹描述/标签字段
- 公开收藏夹的社交分享
- 详情页移除收藏功能（后端已有接口，但本任务不实现前端移除 UI）
- 无限滚动（收藏夹图片数量有限，一次加载全部）
