# Frontend Backend API Data Design

## Scope

把 `frontend/vue-gallery` 中仍使用 mock / hard-coded 动态数据的页面改为读取本地后端 `2018` 端口提供的 `/api/v1/*` 数据。前端源码继续使用相对路径，由 Vite proxy 转发，不直接写入 `localhost:2018`。

## Boundaries

- 前端边界：只修改 `frontend/vue-gallery/src` 下 API client、页面和必要 display mapper/type。
- 后端边界：不新增接口，不改后端行为；以当前本地后端和 Go handler DTO 为真实契约。
- UI 边界：不重做视觉设计，只替换数据来源并补齐 loading / error / empty 状态。
- 认证边界：collections、rating、tag assignment 等需要 Bearer token 的接口必须复用现有 `useAuth` / token 管理；未登录时显示登录提示，不伪造数据。

## Backend Contracts To Consume

### Shared Response

所有后端响应外层为：

```json
{"code":0,"data":{},"msg":""}
```

错误响应使用 `msg`，不是 `message`。API client 的 `ApiResponse` / `ApiError` 读取逻辑必须对齐这一点。

### Pagination

后端统一分页查询参数：

- `page`: 默认 `1`
- `size`: 默认 `20`，最大 `100`
- `sort`: 默认 `created_at`
- `order`: 默认 `desc`

后端列表响应数据为：

```json
{"total":0,"page":1,"size":20,"list":[]}
```

前端可继续暴露 display-friendly 的 `{ items, total, page, limit }`，但请求必须发送 `size`，不是 `limit`。

### Images

- `GET /api/v1/images?filename=&tag=&page=&size=&sort=&order=` 返回图片列表。
- `GET /api/v1/images/:id` 返回：
  - `image`: 图片主数据
  - `tags`: 字符串数组
  - `avg_score`, `rating_count`, `favorite_count`, `my_rating`, `is_collected`, `similar_images`
- `GET /api/v1/search?q=&page=&size=` 返回图片列表。

### Tags

- `GET /api/v1/tags` 返回 `{id,name,usage_count,created_at,updated_at}` 数组。
- `GET /api/v1/tags/suggest?q=&limit=` 返回同一 tag 数组。
- 批量打标需要登录：`POST /api/v1/images/tags`，body 为 `{image_ids,tag_ids}`。

### Rankings

- `GET /api/v1/rankings?period=day|week|month&page=&size=` 返回分页热榜列表。
- 旧前端 `daily/weekly/monthly` 分段需要映射为后端 `day/week/month`。

### Collections

- `GET /api/v1/collections` 需要登录，返回当前用户收藏夹数组：`{id,user_id,name,visibility,created_at,items}`。
- `POST /api/v1/collections` 需要登录，body 为 `{name,visibility}`；后端没有 `description` 字段。
- `GET /api/v1/collections/:id` 可匿名读取公开收藏夹，但返回的 `items` 只有 `{collection_id,image_id,created_at}`，不是完整 `ImageItem`。
- 若收藏页需要展示图片卡片，必须额外按 `image_id` 调 `GET /api/v1/images/:id` 或先只展示收藏夹元数据与 item 数量。

## Frontend Data Flow

### API Client First

先修正 `frontend/vue-gallery/src/api/client.ts`，让页面层只依赖统一 API client：

- `ApiResponse<T>` 使用 `msg` 字段。
- `apiCall` 在非 2xx 或业务 `code !== 0` 时抛出 `ApiError`，错误消息来自 `msg`。
- `getImages` 将 `limit` 映射为后端 `size`，新增 `filename/sort/order` 支持；不要发送后端不消费的 `category`。
- `searchImages` 将 `keyword` 映射为 `q`，分页使用 `size`；`tags/minScore` 当前后端不消费，页面不应假装它们生效。
- `suggestTags` 使用 `q`。
- `getRankings` 改为接收 `{period,page,limit}` 或类似对象，并发送 `period/page/size`。
- `getImage` 类型改为真实详情响应，不再 `extends ImageItem`。
- `TagResponse`、`CollectionResponse`、`CollectionDetailResponse` 类型对齐后端 DTO。
- `createCollection` 发送 `{name, visibility}`。

### Display Mappers

页面可保留 `ArtItem` / `CarouselSlide` / `Album` 作为显示层类型，但映射函数必须从真实 API 类型计算：

- `ImageItem -> ArtItem`: 使用 `id`、`filename`、`url`、`width/height`、`category`、`avg_score`、`favorite_count`。
- `RankingResponse -> CarouselSlide` / 热榜行：使用 `rank`、`score`、`image`。
- `CollectionResponse -> Album`: `count` 从 `items.length` 计算；`lastUpdated` 可先由 `created_at` 格式化或直接展示创建时间。

### Page Integration

1. `TrendingPage.vue`
   - 用 `getRankings({ period, limit })` 替换硬编码 rank rows。
   - period UI 映射：`daily -> day`、`weekly -> week`、`monthly -> month`。
   - 切换分段时重新加载；保留 loading/error/empty。

2. `DetailPage.vue`
   - 从 route query `id` 读取图片 ID。
   - 有效 ID 时调用 `getImage(id)` 渲染真实详情。
   - 无有效 ID 时展示“请选择一张作品”的空状态，不再渲染固定示例。
   - 相似推荐使用 `similar_images`。
   - 评分保存仅在登录后调用 `rateImage`；评分展示和提交均使用后端百分制（`0-100`）。
   - 收藏/下载/打标若缺少完整后端/UI 流程，不能假装成功；改为明确提示需要选择收藏夹或功能尚未接入。

3. `CollectionsPage.vue`
   - 用 `getCollections()` 替换 `albums` mock。
   - 未登录或 401 时显示登录提示。
   - 创建收藏夹调用 `createCollection(name, visibility)`；当前 UI 无 visibility 控件时默认 `private`。
   - 先展示收藏夹元数据和 `items.length`；不要用 mock `collectionItems` 伪造图片卡片。
   - 若要展示收藏夹图片，作为后续增强：选中收藏夹后按 item image_id 拉取详情。

4. `AccountPage.vue`
   - 认证已经使用真实 API。
   - 资料/偏好保存没有后端接口，保留“功能开发中”但不能作为本任务的真实数据接入目标。
   - 可选：登录后拉取 `getCollections()` 显示真实收藏夹数量；失败时保持非阻塞。

5. `GalleryPage.vue` / `SearchPage.vue`
   - 已经调用 API，但受 client 参数 bug 影响。
   - 修正 client 后验证图库数量、热榜轮播、搜索结果是否符合后端真实响应。

## Compatibility And Rollback

- 不保留 mock fallback。真实接口失败时显示错误/重试或登录提示。
- 不加入旧参数兼容层；当前代码尚未发布为外部 API，直接改为后端契约。
- 回滚点：API client 参数和类型修正应独立验证；每个页面替换完成后可单独回退对应页面改动。

## Validation Strategy

- 静态：`npm run build` in `frontend/vue-gallery`。
- API smoke：用 `curl` 验证 `/api/v1/images?size=2`、`/api/v1/rankings?period=day&size=3`、`/api/v1/search?q=...&size=2`、`/api/v1/tags`。
- Browser：启动 Vite dev server，通过代理访问图库、热榜、详情、收藏页未登录状态；确认 network 请求为 `/api/v1/*`。
- 认证相关：不主动创建测试用户，除非用户批准；未登录状态必须可验证。

## Product Decision

评分单位已确认使用后端百分制，例如 `62.3/100`。不要继续使用五分制文案或筛选条件。
