# 执行计划：收藏夹封面与详情页重构

## 实施顺序

### Phase A: 后端数据模型与 mapper（无行为变更，可独立编译验证）

- [ ] A1. `po/collection.go`：新增 `CoverImageID *int64` 字段
- [ ] A2. `do/collection.go`：`Collection` 新增 `CoverImageID int64`；`CollectionItem` 新增 `Image do.Image`
- [ ] A3. `dto/collection.go`：`CollectionResponse` 加 `CoverImageID`/`CoverImageURL`；`CollectionItemResponse` 加 `Image *dto.ImageResponse`；`CollectionUpdateRequest` 加 `CoverImageID *int64`
- [ ] A4. `collection_mapper.go`：`collectionToDO`/`collectionToPO` 传递 `CoverImageID`；`collectionItemToDO` 传递 `Image`
- [ ] A5. 验证：`go build ./...`

### Phase B: 后端 repository 与 service（业务逻辑变更）

- [ ] B1. `repository/collection.go`：`ListByOwner`/`FindVisible` 添加 `Preload("Items.Image")`
- [ ] B2. `service/collection.go`：`CollectionService` 注入 `cosBase string`；新增 `imageURL(cosKey)` 方法
- [ ] B3. `service/collection.go`：`Update` 方法校验 `cover_image_id` 在 items 中（或为 nil/0）
- [ ] B4. `service/collection.go`：新增内部方法计算 `CoverImageURL`（fallback 第一张）
- [ ] B5. `handler/collection.go`：`collectionToResponse` 构造 `CoverImageURL` 和每个 item 的 `Image`（含 URL）
- [ ] B6. `handler/router/router.go`：`Services` 结构中 `CollectionService` 构造注入 `cosBase`
- [ ] B7. `cmd/web/main.go`：初始化 `CollectionService` 时传入 `cosBase`
- [ ] B8. 验证：`go test ./...`

### Phase C: 前端 API 层与类型（无 UI 变更）

- [ ] C1. `api/types.ts`：`CollectionResponse` 加 `cover_image_id`/`cover_image_url`；`CollectionItemResponse` 加 `image?: ImageItem`
- [ ] C2. `api/client.ts`：`BackendCollectionResponse` 传递新字段；新增 `updateCollection(id, input)` 方法
- [ ] C3. `api/client.ts`：新增 `setCollectionCover(collection, imageId)` 辅助方法（GET → PUT 全量）
- [ ] C4. 验证：`cd frontend/vue-gallery && npx tsc --noEmit`

### Phase D: 前端列表页重构

- [ ] D1. `CollectionsPage.vue` 模板：封面区域改为 `<img>` 显示 `cover_image_url`（空则占位）
- [ ] D2. `CollectionsPage.vue` 模板：删除 `span.tag` 三行（ID/Owner/visibility）
- [ ] D3. `CollectionsPage.vue` 模板：删除 `collection-item` 列表块
- [ ] D4. `CollectionsPage.vue` 模板：卡片整体用 `<RouterLink :to="/collections/${id}">` 包裹
- [ ] D5. `CollectionsPage.vue` 样式：调整 `.album-cover` 样式为图片展示；删除 `.collection-item` 相关样式
- [ ] D6. 验证：`npm run build` + 浏览器手动检查 `/collections` 页面

### Phase E: 前端详情页新增

- [ ] E1. `router/index.ts`：新增 `/collections/:id` 路由指向 `CollectionDetailPage.vue`
- [ ] E2. 新建 `pages/CollectionDetailPage.vue`：
  - `onMounted` 调用 `getCollection(id)` 加载
  - 复用 GalleryPage 的 masonry 布局逻辑（简化版，无无限滚动）
  - 每个 item 用 `imageToArtItem(item.image)` 转为 ArtItem
  - 使用 `ArtCard` 组件展示
  - owner 时每张图片卡片上有"设为封面"按钮
  - 已设为封面的图片显示角标
  - 图片点击跳转 `/detail?id=`
  - 返回按钮跳转 `/collections`
- [ ] E3. `CollectionDetailPage.vue`：设置封面后重新加载详情
- [ ] E4. 验证：`npm run build` + 浏览器手动检查 `/collections/:id` 页面

### Phase F: 最终验证

- [ ] F1. `go test ./...` 全部通过
- [ ] F2. `cd frontend/vue-gallery && npm run build` 通过
- [ ] F3. 手动端到端验证：
  - 列表页显示封面图片
  - 列表页无 tag/imageID
  - 点击卡片进入详情页
  - 详情页列出所有图片
  - 设置封面后返回列表页封面更新
  - 非登录用户访问公开收藏夹详情页正常（无设置按钮）

## 验证命令

```bash
# 后端编译与测试
go build ./...
go test ./...

# 前端类型检查与构建
cd frontend/vue-gallery && npx tsc --noEmit
cd frontend/vue-gallery && npm run build
```

## 风险文件

| 文件 | 风险 | 回滚点 |
|------|------|--------|
| `service/collection.go` | 构造函数签名变更 | 保留旧签名为包装方法 |
| `router.go` | Services 结构变更 | 同步修改 cmd/web 初始化 |
| `collection_mapper.go` | DO/PO 映射遗漏 | A5 编译验证 |

## 回滚方案

每个 Phase 独立可回滚：
- Phase A/B 失败 → 回滚后端文件，前端未改动不受影响
- Phase C/D/E 失败 → 回滚前端文件，后端已验证不受影响
- 全部完成后发现问题 → git revert 整个 commit
