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
│   └── client.ts        # API调用封装、类型定义
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
  code?: string
  
  constructor(message: string, status: number, code?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}
```

---

## 后端API契约

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
{ code: 0, data: { id, username, role, created_at }, msg: "" }
```

### 图片接口

**图片列表**: `GET /api/v1/images`
```typescript
// Query: page, limit, tag, category
// Response
{
  code: 0,
  data: {
    items: ImageItem[],
    total: number,
    page: number,
    limit: number
  },
  msg: ""
}
```

**图片详情**: `GET /api/v1/images/:id`

**搜索**: `GET /api/v1/search`
```typescript
// Query: keyword, tags, min_score, page, limit
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
