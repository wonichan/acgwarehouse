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
export async function createCollection(name: string, visibility?: 'private' | 'public'): Promise<CollectionResponse>
export async function addImageToCollection(collectionId: number, imageId: number): Promise<void>
export async function createTag(name: string): Promise<TagResponse>
export async function assignTagsToImages(imageIds: readonly number[], tagIds: readonly number[]): Promise<readonly ImageTagResponse[]>
export async function unassignTagsFromImages(imageIds: readonly number[], tagIds: readonly number[]): Promise<readonly ImageTagResponse[]>
export async function rateImage(imageId: number, score: number): Promise<RatingResponse>
export async function getCurrentUser(): Promise<UserResponse>
export async function updateCurrentUserProfile(input: UserProfileUpdateRequest): Promise<UserResponse>
export async function changeCurrentUserPassword(input: UserPasswordUpdateRequest): Promise<void>
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
- `suggestTags(text, limit)` -> `GET /api/v1/tags/suggest?q=<text>&limit=<limit>`；不要发送旧 `prefix`。
- `getRankings({period,page,limit})` -> `GET /api/v1/rankings?period=day|week|month&page=&size=`。
- 面向图片展示的排名列表必须在页面边界过滤不可展示图片项后再渲染：`image.size > 0`、`image.width > 0`、`image.height > 0`、`image.url.trim().length > 0`、且 `!image.url.trim().endsWith('/')`。目录占位项（例如 `filename=thumbnails`、URL 以 `/` 结尾）不能进入轮播、瀑布流或详情推荐卡片。
- `getImage(id)` 返回嵌套 detail：`{image,tags,avg_score,rating_count,favorite_count,my_rating,is_collected,similar_images}`。
- `TagResponse` 字段是 `{id,name,usage_count,created_at,updated_at}`。
- `CollectionResponse` 字段是 `{id,user_id,name,visibility,created_at,updated_at?,items}`；`items` 是 `{collection_id,image_id,created_at}`，不是完整图片卡片。
- `createCollection()` body 是 `{name,visibility}`，`visibility` 默认 `private`；不要发送旧 `description`。
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
- Good：热榜 period UI `每日/每周/每月` 映射为 `day/week/month`，切换后重新请求 `/rankings`。
- Good：社区焦点需要 10 张可展示作品时，可以请求 `getRankings({period: 'week', limit: 20})` 作为缓冲，先过滤不可展示图片项，再 `slice(0, 10)` 渲染真实作品。
- Good：未登录收藏页展示登录 required 状态，并说明 `/collections` 需要 Bearer token。
- Good：详情页点击“收藏到相册/管理标签/保存评分”且未登录时，渲染 inline `role="alert"` 登录状态与 `/account` 链接，不发送 mutation 请求。
- Good：批量打标签的移除操作使用真实 `DELETE /images/tags`，body 同时包含 `image_ids` 与 `tag_ids`。
- Good：账户中心注册成功后自动登录，`GET /users/me` 同步 expanded `UserResponse`；保存资料/偏好后刷新仍保留。
- Good：20 个中文字符 nickname 前端校验通过，后端也按字符数接受。
- Base：后端 `similar_images: []` 时详情页显示 empty state。
- Base：排名接口返回 `filename=thumbnails`、`size=0`、`width=0`、`height=0` 或 URL 以 `/` 结尾时，该条目被跳过，剩余真实图片继续展示。
- Bad：搜索页展示“标签/评分筛选”并发送后端不消费的 `tags/min_score`，会误导用户以为筛选生效。
- Bad：收藏夹详情只返回 image_id 时，用静态图片卡片填充 masonry，这是 mock fallback。
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

#### 4. Validation & Error Matrix

| Condition | Frontend behavior |
|-----------|-------------------|
| 未登录点击 mutation action | 不调用 mutation API；显示 inline 登录状态与 `/account` 链接 |
| 批量 selection 含非正整数 ID | mutation 前过滤；过滤后为空则显示 inline error |
| 创建收藏夹名称为空 | 禁用创建提交，不发送 API |
| 标签选择为空 | 禁用添加/移除提交，不发送 API |
| mutation 返回 `ApiError` | 保持 picker 打开，显示后端错误，不清空 selection |
| mutation 成功 | 详情页重新加载 detail；批量面板只在成功后清空 selection |

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
- `GET /api/v1/collections` - 收藏夹（需认证）
- `PUT /api/v1/images/:id/rating` - 评分（需认证）

---

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
