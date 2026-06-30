# 技术设计：收藏夹封面与详情页重构

## 架构与边界

### 改动分层

```
后端（Go）
├── model/po/collection.go        — 新增 CoverImageID 字段
├── model/do/collection.go        — 新增 CoverImageID + CollectionItem.Image
├── model/dto/collection.go       — Response/UpdateRequest/ItemResponse 扩展
├── repository/collection_mapper.go — 传递封面 + Image 关联
├── repository/collection.go      — Preload Items.Image
├── service/collection.go         — 注入 cosBase，校验封面，构造 URL
├── handler/collection.go          — collectionToResponse 构造完整响应
└── handler/router/router.go      — CollectionService 构造注入 cosBase

前端（Vue + TS）
├── api/types.ts                  — 类型扩展
├── api/client.ts                 — normalize 传递新字段 + updateCollectionCover
├── router/index.ts               — 新增 /collections/:id 路由
├── pages/CollectionsPage.vue     — 列表页重构
└── pages/CollectionDetailPage.vue — 新增详情页
```

## 数据契约

### 后端 DTO 变更

**dto.CollectionResponse**（新增 2 字段）：
```go
type CollectionResponse struct {
    ID            int64                    `json:"id"`
    UserID        int64                    `json:"user_id"`
    Name          string                   `json:"name"`
    Visibility    string                   `json:"visibility"`
    CoverImageID  int64                    `json:"cover_image_id"`       // 新增：0=未设置
    CoverImageURL string                   `json:"cover_image_url"`      // 新增：fallback 第一张
    CreatedAt     string                   `json:"created_at"`
    Items         []CollectionItemResponse `json:"items,omitempty"`
}
```

**dto.CollectionItemResponse**（新增嵌套 image）：
```go
type CollectionItemResponse struct {
    CollectionID int64           `json:"collection_id"`
    ImageID      int64           `json:"image_id"`
    Image        *dto.ImageResponse `json:"image,omitempty"`  // 新增
    CreatedAt    string          `json:"created_at"`
}
```

**dto.CollectionUpdateRequest**（新增可选字段）：
```go
type CollectionUpdateRequest struct {
    Name         string `json:"name" vd:"len($) > 0 && len($) <= 64"`
    Visibility   string `json:"visibility" vd:"$ == 'private' || $ == 'public'"`
    CoverImageID *int64 `json:"cover_image_id"`  // 新增：nil=不更新，0=清空，>0=设置
}
```

### 前端类型变更

**CollectionResponse**（types.ts）：
```typescript
export interface CollectionResponse {
  readonly id: number
  readonly user_id: number
  readonly name: string
  readonly visibility: CollectionVisibility
  readonly cover_image_id: number        // 新增
  readonly cover_image_url: string      // 新增
  readonly created_at: string
  readonly updated_at?: string
  readonly items: readonly CollectionItemResponse[]
}
```

**CollectionItemResponse**（types.ts）：
```typescript
export interface CollectionItemResponse {
  readonly collection_id: number
  readonly image_id: number
  readonly image?: ImageItem            // 新增
  readonly created_at: string
}
```

## 数据流

### 列表页数据流

```
GET /collections (Auth)
  → CollectionService.ListByOwner
    → repo.ListByOwner (Preload Items.Image)
    → for each collection:
        → determine cover URL (cover_image_id > 0 ? that image's URL : first item's URL)
        → build CollectionResponse with cover_image_url + items[].image
  → handler returns list
```

### 详情页数据流

```
GET /collections/:id (Public)
  → CollectionService.FindVisible
    → repo.FindVisible (Preload Items.Image)
    → build response (same as list)
  → handler returns single collection
```

### 设置封面数据流

```
PUT /collections/:id (Auth, Owner)
  body: { cover_image_id: 42 }
  → CollectionService.Update
    → validate cover_image_id in items (if > 0)
    → repo.Update
  → handler returns updated collection
  → frontend refreshes detail page
```

## 关键设计决策

### D1: CollectionService 注入 cosBase

`CollectionService` 需要生成图片 URL。参照 `RankingService` 和 `DailyRecommendationService` 的模式，注入 `cosBase string`。

**代价**：`NewCollectionService` 签名变更，`router.go` 构造需更新。
**替代方案**：注入 `ImageService` — 过度耦合，不需要。

### D2: do.CollectionItem 新增 Image 字段

`do.CollectionItem` 当前只有 `CollectionID`、`ImageID`、`CreatedAt`。新增 `Image do.Image` 字段，由 repository 在 Preload 后通过 mapper 填充。

**代价**：`collectionItemToDO` 需要接收 `po.CollectionItem`（含 Image 关联）。
**替代方案**：在 service 层单独查询图片 — 多一次 DB 查询，不必要。

### D3: 封面 fallback 逻辑位置

在 service 层构造 `CollectionResponse` 时计算 `cover_image_url`：
- `cover_image_id > 0` → 找到对应 item 的 image URL
- `cover_image_id == 0` 且 items 非空 → items[0] 的 image URL
- items 为空 → 空字符串

**理由**：前端不需要额外逻辑，直接用 `cover_image_url`。

### D4: 详情页复用 ArtCard + masonry

参照 GalleryPage 的 masonry 布局，但简化：
- 复用 `ArtCard` 组件和 `imageToArtItem` 转换
- 移除无限滚动、ResizeObserver 可简化为 CSS columns 或保留 JS masonry
- 保留 JS masonry 与 GalleryPage 一致（用户要求"参照图库页面"）

### D5: updateCollectionCover 复用现有 PUT 接口

前端不新增 API 方法，直接在 `updateCollection` 中传递 `cover_image_id`。但当前 `updateCollection` 不存在（只有 `createCollection`）。

**决策**：新增 `updateCollection(id, input)` 前端 API 方法，支持 name/visibility/cover_image_id。

## 兼容性与迁移

- **数据库迁移**：`po.Collection` 新增 `CoverImageID *int64`，GORM AutoMigrate 自动添加 nullable 列，老数据 cover_image_id 为 NULL，service 层转为 0。
- **API 兼容**：新增 JSON 字段，旧前端忽略不认识的字段，向后兼容。
- **前端类型扩展**：新字段加到 interface，旧组件不使用即可。

## 风险点

- `ListByOwner` 预加载 `Items.Image` 会增加查询开销（N+1 → JOIN）。收藏夹列表通常不多，可接受。
- `po.CollectionItem.Image` 关联已有 gorm tag（po/collection_item.go:13），只需 Preload。
- 前端 masonry 逻辑从 GalleryPage 复制简化版，需注意生命周期清理（ResizeObserver、IntersectionObserver）。
