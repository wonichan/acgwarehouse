# Frontend Spec Index

Layer: frontend/vue-gallery
Language: TypeScript + Vue 3 + Vite

## Available Specs

- `api-client.md` - API调用层封装、类型定义、composables使用
- `component-guidelines.md` - 组件结构、Lucide/`AppIcon`、影院衬、移动折叠菜单 a11y、样式模式、响应式/masonry/loading 契约、UI/UX design skills
- 根目录 `DESIGN.md` - 色板、字号、间距、motion、组件视觉身份（token 源）
- 根目录 `AGENTS.md` frontend 段 - 双轨视觉规则（主身份 + 旗舰页例外）

## Pre-Development Checklist

改 `frontend/vue-gallery` UI 前至少阅读：

1. `DESIGN.md`（token / 身份）
2. `component-guidelines.md`（图标、影院衬、响应式、loading、masonry、skills）
3. 若动导航/Header：Mobile Disclosure Menu 契约
4. 若动详情主图：Cinema Backdrop 契约

## Quality Check

- [ ] 无 emoji；交互图标来自 Lucide + `AppIcon`（按名 import）
- [ ] 新色仅 token `color-mix`；feature 样式优先 scoped
- [ ] 三端布局可用；图流保持 stable masonry + aspect-ratio
- [ ] 折叠菜单：aria-expanded / controls / Esc / inert-when-closed
- [ ] 详情影院衬未污染全站 `:root`
- [ ] `npm run build`（vue-tsc + vite）通过

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

### Responsive Layout Rule

When modifying frontend UI code, keep desktop, tablet, and mobile layouts working. Check `component-guidelines.md` for the responsive layout contract before changing page, form, list, modal, or navigation styles.

### API Endpoints

All backend APIs are under `/api/v1/*`:
- Auth/Profile: `/api/v1/users/login`, `/api/v1/users/register`, `/api/v1/users/me`, `/api/v1/users/password`
- Images: `/api/v1/images`, `/api/v1/images/:id`, `/api/v1/search`
- Tags: `/api/v1/tags`
- Image tags: `/api/v1/images/tags` (`POST` assign, `DELETE` unassign)
- Rankings: `/api/v1/rankings`
- Daily recommendations: `/api/v1/daily-recommendations`
- Collections: `/api/v1/collections` (auth required)

### Deployment Fallback Contract

Vue Router uses history mode, so every frontend-only route must be served by the SPA entrypoint on static hosts.

Keep `frontend/vue-gallery/public/_redirects` in sync with route/deployment changes:

```text
/api/* /api/:splat 200
/* /index.html 200
```

- The `/api/*` rule must appear before the catch-all rule so backend API paths are not served as `index.html`.
- The `/* /index.html 200` rule is required for direct public access to clean URLs such as `/account`, `/search`, `/trending`, `/collections`, and `/detail`.
- The production frontend service must run the Go frontend proxy server (`bin/frontend-server`) so `/api/v1/*` reaches the backend before the SPA fallback; do not use plain `serve -s dist -l 2017` because it serves API paths as `index.html`.
- Do not switch to hash routing (`/#/...`) to hide missing static-host fallback configuration.


**Language**: 所有 documentation 都应使用 **Chinese** 编写。