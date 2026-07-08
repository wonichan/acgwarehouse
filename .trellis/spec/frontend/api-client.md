# Frontend API Client 规范

> 前端API调用层的实现约定

---

## 概述

前端使用 `fetch` API 与后端通信，通过统一的 API client 模块封装所有HTTP请求。

---

## 文件结构

```
frontend/vue-gallery/src/
├── api/
│   ├── client.ts        # API方法封装与前端友好归一化
│   ├── transport.ts     # fetch、token、Response envelope、ApiError 边界
│   └── types.ts         # 后端 DTO / query 类型定义
├── composables/
│   └── useAuth.ts       # 认证状态管理
└── pages/
    └── *.vue            # 页面组件调用API
```

---

## API Client 实现

### 基础配置

```typescript
const API_BASE = '/api/v1'  // 生产环境由nginx代理，开发环境由vite代理

// JWT Token 管理
const TOKEN_KEY = 'acgwarehouse_token'

export function setToken(token: string): void
export function clearToken(): void
export function isAuthenticated(): boolean
```

### 请求封装

```typescript
async function apiCall<T>(path: string, options?: RequestInit & { skipAuth?: boolean }): Promise<T> {
  const token = getToken()
  
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  
  // 自动添加JWT认证头
  if (token && !options?.skipAuth) {
    headers['Authorization'] = `Bearer ${token}`
  }
  
  const response = await fetch(`${API_BASE}${path}`, { ...options, headers })
  
  if (!response.ok) {
    throw new ApiError(message, status, code)
  }
  
  return response.json()
}
```

### 错误处理

```typescript
export class ApiError extends Error {
  status: number
  code?: number
  
  constructor(message: string, status: number, code?: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}
```

---

## 后端API契约

### 场景：Vue Gallery API Client 对接真实后端 DTO

#### 1. Scope / Trigger

- Trigger：前端页面或 composable 调用 `/api/v1/*` 后端接口，尤其是认证/账户资料、图片列表、搜索、热榜、详情、标签、收藏夹、评分。
- Scope：`frontend/vue-gallery/src/api/client.ts` 只暴露前端友好的方法；`transport.ts` 只处理 fetch/envelope/token/error；`types.ts` 只声明后端 DTO 与 query 类型。
- Goal：页面层不直接知道后端端口、响应 envelope、后端字段名差异，也不能伪造后端未支持的筛选或业务成功。

#### 2. Signatures

```typescript
const API_BASE = '/api/v1'

export interface ApiResponse<T> {
  readonly code: number
  readonly msg: string
  readonly data: T
}

export class ApiError extends Error {
  readonly status: number
  readonly code?: number
}

export async function getImages(params?: ImageQuery): Promise<ImageListResponse>
export async function searchImages(params: SearchQuery): Promise<ImageListResponse>
export async function getImage(id: number): Promise<ImageDetailResponse>
export async function getTags(): Promise<readonly TagResponse[]>
export async function suggestTags(q: string, limit?: number): Promise<readonly TagResponse[]>
export async function getRankings(params?: RankingQuery): Promise<readonly RankingResponse[]>
export async function getCollections(): Promise<readonly CollectionResponse[]>
export async function getCollection(id: number): Promise<CollectionDetailResponse>
export async function createCollection(name: string, visibility?: 'private' | 'public'): Promise<CollectionResponse>
export async function updateCollection(id: number, input: CollectionUpdateInput): Promise<CollectionResponse>
export async function addImageToCollection(collectionId: number, imageId: number): Promise<void>
export async function createTag(name: string): Promise<TagResponse>
export async function assignTagsToImages(imageIds: readonly number[], tagIds: readonly number[]): Promise<readonly ImageTagResponse[]>
export async function unassignTagsFromImages(imageIds: readonly number[], tagIds: readonly number[]): Promise<readonly ImageTagResponse[]>
export async function rateImage(imageId: number, score: number): Promise<RatingResponse>
export async function getCurrentUser(): Promise<UserResponse>
export async function updateCurrentUserProfile(input: UserProfileUpdateRequest): Promise<UserResponse>
export async function changeCurrentUserPassword(input: UserPasswordUpdateRequest): Promise<void>
export async function getMonthlyCheckIns(year: number, month: number): Promise<MonthlyCheckInsResponse>
```

#### 3. Contracts

- 所有前端源码请求必须使用相对路径 `/api/v1/*`；开发环境只允许在 `vite.config.ts` proxy 中出现 `http://localhost:2018`。
- 后端成功/错误 envelope 是 `{code,data,msg}`。前端错误消息优先读取 `msg`，不是 `message`。
- 后端列表响应是 `{total,page,size,list}`。前端可归一化为 `{items,total,page,limit}`，但请求必须发送 `size`。
- `getImages({limit})` -> `GET /api/v1/images?size=<limit>`；可发送 `filename/tag/sort/order/page/size`，不要发送旧 `category`。
- 图库瀑布流自动分页必须显式发送 `getImages({page, limit})`：首屏传 `page: 1`，下一页传 `currentPage + 1`；下一页结果追加到现有列表，不能替换首屏列表。
- 无限滚动分页必须有 `loadingMore` 或等价锁，避免同一页并发重复请求；必须用列表响应的 `total/page/size` 计算 `hasMore`，全部加载后停止请求。
- 底部 `IntersectionObserver` / sentinel 必须在首屏加载完成并且 DOM 更新后注册，避免 sentinel 首次相交时因 `loading=true` 被丢弃且不再触发。
- `searchImages({keyword,limit})` -> `GET /api/v1/search?q=<keyword>&size=<limit>`；不要发送旧 `keyword/tags/min_score/limit`。
- 搜索页的可分享状态必须由路由 query 驱动：`/search?q=<keyword>` 进入页面时应同步搜索输入并自动发起 `searchImages({keyword, limit})`；页面内搜索提交关键词时应先更新 route query，再由 route watcher 触发请求，避免 URL 与结果不一致。
- 顶栏或其他全局“快速搜索”输入不能只渲染静态 `<input>`；必须是可提交的 `role="search"` 表单或等价可访问控件，非空提交导航到 `/search?q=<keyword>`，空提交导航到 `/search`。
- 同步搜索输入时 watcher 不能只监听 `route.query.q`；如果同一个组件跨路由常驻（例如全局 header），应同时关注 `route.path` 和 `route.query.q`，避免从其他页面进入 `/search` 时保留旧关键词。
- `suggestTags(text, limit)` -> `GET /api/v1/tags/suggest?q=<text>&limit=<limit>`；不要发送旧 `prefix`。
- `getRankings({period,page,limit})` -> `GET /api/v1/rankings?period=day|week|month&page=&size=`；热榜页按周期显式请求展示数量：`day=20`、`week=50`、`month=100`，并与后端省略 `size` 时的默认数量保持一致。
- 面向图片展示的排名列表必须在页面边界过滤不可展示图片项后再渲染：`image.size > 0`、`image.width > 0`、`image.height > 0`、`image.url.trim().length > 0`、且 `!image.url.trim().endsWith('/')`。目录占位项（例如 `filename=thumbnails`、URL 以 `/` 结尾）不能进入轮播、瀑布流或详情推荐卡片。
- `getImage(id)` 返回嵌套 detail：`{image,tags,avg_score,rating_count,favorite_count,my_rating,is_collected,similar_images}`。
- `TagResponse` 字段是 `{id,name,usage_count,created_at,updated_at}`。
- `CollectionResponse` 字段是 `{id,user_id,name,visibility,created_at,updated_at?,cover_image_id,cover_image_url,items}`。
- `CollectionItemResponse` 字段是 `{collection_id,image_id,created_at,image?}`；`image` 存在时是完整 `ImageItem`，用于收藏详情页和列表封面，不要再用静态图片或 mock 卡片填充。
- `cover_image_id=0` 表示未显式设置封面；`cover_image_url` 是后端计算后的最终封面 URL，前端列表页直接用它渲染封面。空字符串表示空收藏夹/无可展示图片，显示占位状态。
- `getCollection(id)` -> `GET /api/v1/collections/:id`，公开收藏夹可匿名读取；私有收藏夹按后端权限返回错误。
- `createCollection()` body 是 `{name,visibility}`，`visibility` 默认 `private`；不要发送旧 `description`。
- `updateCollection(id,input)` -> `PUT /api/v1/collections/:id`。`input` 必须包含当前 `name` 与 `visibility`；`cover_image_id` 省略表示不改封面，`0` 表示清空显式封面，`>0` 表示设为该图片。
- `addImageToCollection(collectionId, imageId)` -> `POST /api/v1/collections/:id/items`，body 是 `{image_id}`。
- `createTag(name)` -> `POST /api/v1/tags`，body 是 `{name}`。
- `assignTagsToImages(imageIds, tagIds)` -> `POST /api/v1/images/tags`，body 是 `{image_ids,tag_ids}`。
- `unassignTagsFromImages(imageIds, tagIds)` -> `DELETE /api/v1/images/tags`，body 是 `{image_ids,tag_ids}`。
- 评分使用后端百分制整数 `0-100`；展示为 `/100`，不要在真实数据页面显示五分制如 `4.8 分`。
- 用户资料字段必须与后端字符计数语义一致：nickname 1-20 个字符、favorite_tags 0-120 个字符、bio 0-200 个字符。前端校验 CJK 时用字符计数（如 `Array.from(value).length`），不要用字节长度假设。
- 账户中心保存资料/偏好必须调用 `PUT /api/v1/users/me`；不得用 localStorage 伪造 profile 持久化。密码修改必须调用 `PUT /api/v1/users/password`。

#### 4. Validation & Error Matrix

| Condition | Frontend behavior |
|-----------|-------------------|
| HTTP 非 2xx，body 有 `msg/code` | throw `ApiError(msg, status, code)` |
| HTTP 非 2xx，body 非 JSON | throw `ApiError('API Error: <status>', status)` |
| HTTP 200 但 `code !== 0` | throw `ApiError(msg || '请求失败', 200, code)` |
| `/collections` 未登录返回 401 / 40101 | 页面显示登录提示，不渲染 mock 收藏夹 |
| `GET /collections/:id` 返回 `items[].image` | 收藏详情页用真实 `ImageItem` 渲染 ArtCard，不显示 raw imageID |
| `cover_image_url` 为空 | 收藏夹列表显示占位封面，不显示数字封面或空白破图 |
| `PUT /collections/:id` 设置封面成功 | 详情页刷新返回的 collection，并让当前封面按钮变为 disabled 状态 |
| `PUT /collections/:id` 返回 400（cover 不在夹内） | 显示后端错误，不本地伪造封面已更新 |
| 收藏夹/标签/评分 mutation 未登录 | 阻止调用 mutation API，显示持久 inline 登录提示与 `/account` 链接；toast 只能作为补充 |
| 批量标签移除 | 发送选中图片 ID 和选中标签 ID 到 `DELETE /images/tags`；不要伪造本地成功 |
| 后端返回空 `list` | 页面显示 empty state，不使用 fallback mock |
| 瀑布流下一页请求失败 | 保留已加载图片，显示可重试的 next-page 错误状态，不把首屏 `error` 覆盖为整页失败 |
| 瀑布流已加载数量达到 `total` | `hasMore=false`，停止自动请求下一页并显示已加载全部状态 |
| 瀑布流下一页仍在请求中 | 忽略新的 sentinel 触发，避免同一页重复请求 |
| 排名/推荐列表包含目录占位或零尺寸图片 | 页面过滤该条目；如果过滤后为空，显示 empty state，不渲染 broken image 或目录名 |
| 缺少有效 detail `id` | 详情页显示“请选择一张作品”，不渲染固定示例 |
| 评分值来自均分小数 | 提交前 round 并 clamp 到 `0-100` |
| profile nickname/tags/bio 超出字符限制 | 阻止提交，显示字段级错误，不发送 API |
| `PUT /users/me` 返回 400/401/500 | 保留表单输入，显示 inline status，不写 localStorage fallback |
| `PUT /users/password` 旧密码错误 | 显示后端错误，旧/新密码字段仍可编辑 |

#### 5. Good / Base / Bad Cases

- Good：`getImages({limit: 2})` 的 network query 是 `size=2`，返回数据归一化为 `items.length === data.list.length`。
- Good：图库瀑布流首屏请求 `/images?page=1&size=20`，滚动到底部后请求 `/images?page=2&size=20`，并把第二页追加到已有卡片后面。
- Good：顶栏快速搜索输入 `miku` 后提交，URL 变为 `/search?q=miku`，搜索页输入同步为 `miku`，network query 是 `/api/v1/search?q=miku&size=20`。
- Good：直接访问 `/search?q=miku`，页面无需额外点击即可请求 `/api/v1/search?q=miku&size=20` 并更新搜索摘要。
- Good：热榜 period UI `每日/每周/每月` 映射为 `day/week/month`，切换后重新请求 `/rankings`。
- Good：热榜页请求每日/每周/每月时分别发送 `size=20/50/100`，侧栏“返回数量”和列表长度来自真实响应，不使用静态补齐。
- Good：社区焦点需要 10 张可展示作品时，可以请求 `getRankings({period: 'week', limit: 20})` 作为缓冲，先过滤不可展示图片项，再 `slice(0, 10)` 渲染真实作品。
- Good：未登录收藏页展示登录 required 状态，并说明 `/collections` 需要 Bearer token。
- Good：收藏夹列表页使用 `cover_image_url` 渲染封面，卡片跳转 `/collections/:id`，不显示 `ID/Owner/raw imageID`。
- Good：收藏详情页使用 `items[].image` 经 `imageToArtItem` 转成 ArtCard，owner 可调用 `updateCollection(...,{cover_image_id})` 设置封面。
- Good：未登录访问公开收藏详情页可浏览图片，但不显示“设为封面”按钮。
- Good：详情页点击“收藏到相册/管理标签/保存评分”且未登录时，渲染 inline `role="alert"` 登录状态与 `/account` 链接，不发送 mutation 请求。
- Good：批量打标签的移除操作使用真实 `DELETE /images/tags`，body 同时包含 `image_ids` 与 `tag_ids`。
- Good：账户中心注册成功后自动登录，`GET /users/me` 同步 expanded `UserResponse`；保存资料/偏好后刷新仍保留。
- Good：20 个中文字符 nickname 前端校验通过，后端也按字符数接受。
- Base：后端 `similar_images: []` 时详情页显示 empty state。
- Base：排名接口返回 `filename=thumbnails`、`size=0`、`width=0`、`height=0` 或 URL 以 `/` 结尾时，该条目被跳过，剩余真实图片继续展示。
- Bad：搜索页展示“标签/评分筛选”并发送后端不消费的 `tags/min_score`，会误导用户以为筛选生效。
- Bad：顶栏显示“快速搜索”输入框但没有 `v-model`、提交处理、路由导航或 API 请求，用户输入后无任何搜索行为。
- Bad：搜索页只在按钮点击时调用 API，直接打开 `/search?q=miku` 不自动搜索，导致可分享 URL 与页面结果脱节。
- Bad：收藏夹详情只返回 image_id 时，用静态图片卡片填充 masonry，这是 mock fallback。
- Bad：收藏夹列表页把 `items[].image_id` 渲染成可见文本，暴露实现 ID 而不是让用户进入详情页看图。
- Bad：设置封面时只在前端替换图片 URL，不调用 `PUT /collections/:id`。
- Bad：点击“加入收藏夹/批量打标签”后只显示成功 toast，不调用真实收藏夹或标签 API。
- Bad：批量移除标签只更新本地 chip，不调用 `DELETE /images/tags`。
- Bad：将目录占位排名项直接映射成轮播 slide，导致主图显示 broken image alt 文本（如 `thumbnails`）。

#### 6. Tests Required

- Build/type：`npm run build` 必须通过，覆盖 `vue-tsc -b`。
- API smoke：`curl http://localhost:2018/api/v1/images?size=2` 断言 `data.size=2`；`/rankings?period=day&size=3` 断言 envelope/list shape；需要固定展示数量的页面额外断言过滤后 displayable 数量和首个展示项不是目录占位；`/search?q=<term>&size=2` 断言 envelope/list shape；`/collections` 未登录断言 401/`msg`。
- Proxy smoke：Vite dev server 下访问 `/api/v1/images?size=1` 必须代理到后端，前端源码不得硬编码端口。
- Page/E2E：浏览图库、热榜、`/detail?id=<known id>`、未登录收藏页，断言真实 API 请求和 loading/error/empty/login-required 状态。
- Account E2E：注册自动登录、保存资料、保存偏好、修改密码、退出后用新密码登录并恢复资料；断言 `/users/me` 与 `/users/password` 真实请求返回 200。
- Favorites/tags E2E：未登录点击详情/批量 mutation action 时断言 inline 登录状态；登录后断言收藏夹列表、创建收藏夹、加入收藏夹、创建标签、批量添加标签、批量移除标签均发出对应 `/collections`、`/tags`、`/images/tags` 请求。
- Collections cover E2E：登录后访问 `/collections` 断言卡片有封面 `<img>` 且无 ID/Owner/imageID 文本；点击卡片进入 `/collections/:id`；点击“设为封面”断言 `PUT /collections/:id` 200；清 token 后访问公开详情页断言可读但无设置按钮。
- Gallery pagination：图库页必须验证滚动到底部会发出 `page=2&size=<pageSize>` 请求；重复 sentinel 触发不会并发请求同一页；`items.length >= total` 后不会继续请求；下一页失败时已加载卡片仍保留并出现重试入口。

#### 7. Wrong vs Correct

##### Wrong

```typescript
// 后端不消费这些参数；错误消息字段也不是 message。
await apiCall<ApiResponse<ImageListResponse>>('/search?keyword=miku&tags=雨景&limit=20')
throw new ApiError(response.message, response.status)

// 直接渲染排名结果会把目录占位项显示成 broken image。
carouselSlides.value = rankingsData.map(rankingToSlide)

// 瀑布流分页不能一直请求默认第一页，也不能用下一页覆盖已有列表。
const imagesData = await getImages({ limit: 20 })
artItems.value = imagesData.items.map(imageToArtItem)
```

##### Correct

```typescript
const query = new URLSearchParams()
query.set('q', keyword)
query.set('size', String(limit))

const response = await unwrapResponse(
  apiCall<ApiResponse<BackendListResponse<ImageItem>>>(`/search?${query.toString()}`)
)

return {
  items: response.list,
  total: response.total,
  page: response.page,
  limit: response.size,
}

function hasDisplayableImage(ranking: RankingResponse): boolean {
  const image = ranking.image
  const imageUrl = image.url.trim()
  return image.size > 0 && image.width > 0 && image.height > 0 && imageUrl.length > 0 && !imageUrl.endsWith('/')
}

carouselSlides.value = rankingsData.filter(hasDisplayableImage).slice(0, 10).map(rankingToSlide)

const imagesData = await getImages({ page: currentPage.value + 1, limit: 20 })
const nextItems = imagesData.items.map(imageToArtItem)
artItems.value = [...artItems.value, ...nextItems]
```

##### Infinite scroll setup

```typescript
async function loadInitialGallery(): Promise<void> {
  await loadGallery()
  await nextTick()
  observeGallerySentinel()
}
```

### 场景：收藏夹与普通用户标签 Mutation

#### 1. Scope / Trigger

- Trigger：前端实现收藏夹选择、创建收藏夹、图片打标/取消打标、批量图片标签操作。
- Scope：只覆盖普通用户收藏/标签 mutation；管理员标签 CRUD、收藏夹 rename/delete、收藏夹 item removal 不在此场景内。

#### 2. Signatures

```typescript
export async function createCollection(name: string, visibility?: CollectionVisibility): Promise<CollectionResponse>
export async function addImageToCollection(collectionId: number, imageId: number): Promise<void>
export async function createTag(name: string): Promise<TagResponse>
export async function assignTagsToImages(imageIds: readonly number[], tagIds: readonly number[]): Promise<readonly ImageTagResponse[]>
export async function unassignTagsFromImages(imageIds: readonly number[], tagIds: readonly number[]): Promise<readonly ImageTagResponse[]>
```

#### 3. Contracts

- `POST /collections` request：`{name:string, visibility:'private'|'public'}`。
- `POST /collections/:id/items` request：`{image_id:number}`。
- `POST /tags` request：`{name:string}`。
- `POST /images/tags` request：`{image_ids:number[], tag_ids:number[]}`。
- `DELETE /images/tags` request：`{image_ids:number[], tag_ids:number[]}`。
- `ImageTagResponse` response：`{image: ImageItem, tags: string[]}`；前端可用 `tags` 刷新可移除标签状态，但详情页最终状态必须以重新 `getImage(id)` 为准。
- 详情页 mutation 成功后的静默刷新必须保留本次已确认成功的用户态字段（例如 `rateImage()` 返回的 `score`、收藏添加成功后的 `is_collected=true`），同时采用 `getImage(id)` 返回的服务端聚合字段（`avg_score`、`rating_count`、`favorite_count`、`tags`、`similar_images`）。不要让刷新响应里的 `my_rating:null` 或 `is_collected:false` 覆盖刚刚成功的用户操作反馈。

#### 4. Validation & Error Matrix

| Condition | Frontend behavior |
|-----------|-------------------|
| 未登录点击 mutation action | 不调用 mutation API；显示 inline 登录状态与 `/account` 链接 |
| 批量 selection 含非正整数 ID | mutation 前过滤；过滤后为空则显示 inline error |
| 创建收藏夹名称为空 | 禁用创建提交，不发送 API |
| 标签选择为空 | 禁用添加/移除提交，不发送 API |
| mutation 返回 `ApiError` | 保持 picker 打开，显示后端错误，不清空 selection |
| mutation 成功后静默刷新详情 | 调用 `getImage(id)` 刷新服务端聚合字段；保留本次 mutation 已确认成功的 `my_rating` / `is_collected` 用户态反馈，不出现详情骨架屏闪烁 |
| 后端详情刷新返回 `my_rating:null` / `is_collected:false` 但本次 mutation 已成功 | 不覆盖刚确认的用户态反馈；继续使用响应中的 `avg_score` / `rating_count` / `favorite_count` 等聚合字段 |

#### 5. Good / Base / Bad Cases

- Good：登录后在详情页创建私有/公开收藏夹并添加当前图片；成功后重新加载 detail。
- Good：批量选择图片后添加标签或移除标签；请求体同时包含所有选中图片 ID 和所有选中标签 ID。
- Base：没有收藏夹时显示 empty state，并提供创建并添加路径。
- Bad：用 toast 文案“已加入收藏夹/已打标签”替代真实 API 调用。
- Bad：把标签写入 localStorage 或 Pinia 来伪造服务器状态。

#### 6. Tests Required

- Build/type：`npm run build` 必须通过。
- Browser smoke：未登录详情页和批量面板 mutation action 显示持久登录提示。
- API smoke：登录态下断言 `POST /collections`、`POST /collections/:id/items`、`POST /tags`、`POST /images/tags`、`DELETE /images/tags` 的 method/body 正确。

#### 7. Wrong vs Correct

##### Wrong

```typescript
show(`${selectedCount.value} 张图片已批量打标签`)
clear()
```

##### Correct

```typescript
const result = await unassignTagsFromImages(validImageIds.value, selectedTagIds)
if (result.length > 0) {
  clear()
}
```

### 认证接口

**登录**: `POST /api/v1/users/login`
```typescript
// Request
{ username: string, password: string }

// Response
{ code: 0, data: { token: string }, msg: "" }
```

**注册**: `POST /api/v1/users/register`
```typescript
// Request
{ username: string, password: string }

// Response
{ code: 0, data: UserResponse, msg: "" }
```

**当前用户**: `GET /api/v1/users/me` (需认证)
```typescript
// Response
{ code: 0, data: UserResponse, msg: "" }
```

**更新当前用户资料与偏好**: `PUT /api/v1/users/me` (需认证)
```typescript
// Request
{
  nickname: string,
  favorite_tags: string,
  bio: string,
  public_profile: boolean,
  email_notifications: boolean,
  sync_collections: boolean
}

// Response
{ code: 0, data: UserResponse, msg: "" }
```

**修改密码**: `PUT /api/v1/users/password` (需认证)
```typescript
// Request
{ old_password: string, new_password: string }

// Response
{ code: 0, data: null, msg: "" }
```

### 图片接口

**图片列表**: `GET /api/v1/images`
```typescript
// Query sent to backend: page, size, filename, tag, sort, order
// Frontend method may accept `limit`, but must map it to backend `size`.
// Response
{
  code: 0,
  data: {
    list: ImageItem[],
    total: number,
    page: number,
    size: number
  },
  msg: ""
}
```

**图片详情**: `GET /api/v1/images/:id`

**搜索**: `GET /api/v1/search`
```typescript
// Query sent to backend: q, page, size
```

### 其他接口

- `GET /api/v1/tags` - 标签列表
- `GET /api/v1/rankings` - 热榜
- `GET /api/v1/daily-recommendations` - 每日随机推荐
- `GET /api/v1/collections` - 收藏夹（需认证）
- `PUT /api/v1/images/:id/rating` - 评分（需认证）
- `GET /api/v1/users/me/check-ins?year=&month=` - 月度签到记录（需认证），返回 `{dates: string[], total_points: number}`；`UserResponse.points` 为累计积分，`GET /users/me` 触发幂等自动签到

---

---

### 场景：每日随机推荐首页区块

#### 1. Scope / Trigger

- Trigger：前端接入 `/api/v1/daily-recommendations`，在图库首页展示与社区精选/热榜不同的每日随机推荐。
- Scope：API client 类型与方法、首页加载编排、图片展示过滤、独立 loading/empty/error UI。
- Goal：每日推荐以图片为主、必要文字辅助；daily API 失败不影响图库瀑布流或社区精选。

#### 2. Signatures

```typescript
export interface DailyRecommendationListResponse {
  readonly date: string
  readonly timezone: string
  readonly total: number
  readonly list: readonly ImageItem[]
}

export async function getDailyRecommendations(): Promise<DailyRecommendationListResponse>
```

Backend endpoint：`GET /api/v1/daily-recommendations`，返回 `{date,timezone,total,list}`，外层仍为 `{code,data,msg}`。

#### 3. Contracts

- `getDailyRecommendations()` 必须请求相对路径 `/daily-recommendations`，由 transport 拼成 `/api/v1/daily-recommendations`。
- `client.ts` 作为业务 API facade，应 re-export `getDailyRecommendations`，并在 `api` 对象中包含该方法，避免调用方必须知道拆分文件路径。
- 首页 daily block 使用接口返回的 `list`，先过滤不可展示图片，再 `slice(0, 10)`，再映射为 `ArtItem`。
- 不可展示图片判定与热榜/图库一致：`size > 0`、`width > 0`、`height > 0`、`url.trim()` 非空且不以 `/` 结尾。
- Daily block 与社区精选并存：社区精选继续走 `/rankings?period=week&size=20`，daily block 走 `/daily-recommendations`。
- Daily block 文案保持最小：允许 “北京时间今日更新”“每日随机推荐” 和一条短说明；不得展示 inline note、timezone 字符串、刷新机制等运营/调试文字。
- Daily API 错误只写入 daily-local state；不得把整个 GalleryPage 置为 full-page error。
- Daily card 使用现有 `ArtCard` 时应传 `selectable=false`，避免推荐区点击触发批量选择。

#### 4. Validation & Error Matrix

| Condition | Frontend behavior |
|-----------|-------------------|
| `/daily-recommendations` 返回 200 且 `list=[]` | 显示每日推荐 empty panel，不使用 mock 图片 |
| `/daily-recommendations` 返回 4xx/5xx | 显示 daily-local error panel 和可选重试；图库主体仍继续展示 |
| daily list 含目录占位/零尺寸/空 URL | 过滤该项；正常 UI 不展示调试说明 |
| daily 请求慢于图库/热榜 | daily block 保持自身 loading，不阻塞图库主体渲染 |
| 移动端 375px | 无水平滚动；daily grid 单列展示 |

#### 5. Good / Base / Bad Cases

- Good：首页同时发起 `/daily-recommendations`、`/images?page=1&size=20...`、`/rankings?period=week&size=20`，三个区域各自渲染。
- Good：后端 daily 返回空 list 时，页面显示“今日还没有可展示作品”，而不是隐藏整个区块或用示例图填充。
- Good：daily API 500 时，社区精选和图库仍可用，daily 区块显示“每日推荐暂时不可用”。
- Base：daily 返回 3 张 displayable 图片时只展示 3 张，不补 mock 卡片。
- Bad：把 daily 请求放进主 `Promise.all`，导致 daily 失败触发整页 `error`。
- Bad：在 daily 区块展示 `Asia/Shanghai`、`每日 00:00 刷新` 等非必要 inline note，破坏图片优先的信息层级。
- Bad：复用社区精选轮播文案或 hot/rank 标签，让用户误以为 daily 来自热榜算法。

#### 6. Tests Required

- Build/type：`npm run build` 必须通过，覆盖 `vue-tsc -b`。
- Browser smoke：生产前端代理下访问首页，断言 daily section 可见、请求 `/api/v1/daily-recommendations`、console errors 为 0。
- Responsive smoke：375px 宽度断言 `document.body.scrollWidth === window.innerWidth`。
- API smoke：`curl /api/v1/daily-recommendations` 返回 200/envelope，并包含 `date/timezone/total/list`。
- Regression：确认社区精选仍请求 `/rankings`，不被 daily endpoint 替代。

#### 7. Wrong vs Correct

##### Wrong

```typescript
const [images, rankings, daily] = await Promise.all([
  getImages(query),
  getRankings({ period: 'week', limit: 20 }),
  getDailyRecommendations(),
])
// daily 失败会进入主 catch，整页图库失败。
```

##### Correct

```typescript
const dailyResult = dailyRecommendations.load() // catches into daily-local state
const [imagesData, slides] = await Promise.all([
  getImages(galleryImageQuery(1)),
  getCommunityFocusSlides(),
])
await dailyResult
```

##### Wrong

```vue
<p>今日随机池 · Asia/Shanghai · 每日 00:00 刷新</p>
```

##### Correct

```vue
<p class="daily-random-copy">随机抽取今日可展示作品，让冷门作品也有机会被看见。</p>
```

## Composables 使用

### useAuth

```typescript
import { useAuth } from '@/composables/useAuth'

const { user, loading, error, isLoggedIn, isAdmin, login, register, logout } = useAuth()

// 登录
const success = await login(username, password)
if (success) {
  // 登录成功
} else {
  // error.value 包含错误信息
}

// 退出
logout()
```

---

## 页面集成示例

### 加载数据

```typescript
import { getImages, ApiError } from '@/api/client'
import type { ImageItem } from '@/api/client'

const loading = ref(true)
const error = ref<string | null>(null)
const items = ref<ImageItem[]>([])

async function loadData() {
  loading.value = true
  error.value = null
  
  try {
    const data = await getImages({ limit: 20 })
    items.value = data.items
  } catch (e) {
    if (e instanceof ApiError) {
      error.value = e.message
    } else {
      error.value = '加载失败'
    }
  } finally {
    loading.value = false
  }
}

onMounted(() => loadData())
```

---

## TypeScript类型导入

**重要**: 项目启用 `verbatimModuleSyntax`，类型必须使用 `type` 导入：

```typescript
// ✅ 正确
import { getImages, ApiError } from '@/api/client'
import type { ImageItem } from '@/api/client'

// ❌ 错误 - 会导致编译失败
import { getImages, ImageItem, ApiError } from '@/api/client'
```

---

## 开发 vs 生产环境

### 开发环境 (vite dev server)

```typescript
// vite.config.ts
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:2018',
      changeOrigin: true
    }
  }
}
```

### 生产环境 (nginx)

```nginx
location /api/ {
    proxy_pass http://127.0.0.1:2018;
    # ... proxy headers
}
```

前端代码无需区分环境，API路径统一为 `/api/v1/*`。

---

## 常见错误

### 1. 忘记添加认证头

```typescript
// ❌ 错误 - 需要认证的接口会返回401
const response = await fetch('/api/v1/collections')

// ✅ 正确 - 使用apiCall自动添加Authorization头
const collections = await getCollections()
```

### 2. 类型导入错误

```typescript
// ❌ 错误 - TypeScript编译失败
import { ImageItem } from '@/api/client'

// ✅ 正确
import type { ImageItem } from '@/api/client'
```

### 3. 未处理错误状态

```typescript
// ❌ 错误 - 未捕获API错误
const data = await getImages()

// ✅ 正确
try {
  const data = await getImages()
  // 使用data
} catch (e) {
  if (e instanceof ApiError) {
    showError(e.message)
  }
}
```

---

## 文件权限

- `.env` 文件: `chmod 600` - 仅所有者可读写
- 数据目录 (`data/`): `www:www` 用户权限
- 后端二进制: 可执行权限

---

## 相关文档

- [后端HTTP API规范](../backend/go-http-api.md)
- [后端错误处理](../backend/go-error-handling.md)
