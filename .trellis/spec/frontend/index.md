# Frontend Spec Index

Layer: frontend/vue-gallery
Language: TypeScript + Vue 3 + Vite

## Available Specs

- `api-client.md` - API调用层封装、类型定义、composables使用
- `design.md` - UI设计规范（如果存在）

## Quick Reference

### API Integration Pattern

```typescript
// 1. Import API methods and types
import { getImages, ApiError } from '@/api/client'
import type { ImageItem } from '@/api/client'

// 2. Use in composables or pages
const { user, login, isLoggedIn } = useAuth()

// 3. Call API with error handling
try {
  const data = await getImages({ limit: 20 })
} catch (e) {
  if (e instanceof ApiError) {
    showError(e.message)
  }
}
```

### Type Import Rule (CRITICAL)

All type imports must use `import type` syntax:

```typescript
import type { ImageItem, UserResponse } from '@/api/client'
```

### API Endpoints

All backend APIs are under `/api/v1/*`:
- Auth/Profile: `/api/v1/users/login`, `/api/v1/users/register`, `/api/v1/users/me`, `/api/v1/users/password`
- Images: `/api/v1/images`, `/api/v1/images/:id`, `/api/v1/search`
- Tags: `/api/v1/tags`
- Rankings: `/api/v1/rankings`
- Collections: `/api/v1/collections` (auth required)
