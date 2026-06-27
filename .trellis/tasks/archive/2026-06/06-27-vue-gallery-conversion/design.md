# Vue.js 图库原型转换 - 技术设计

## Architecture Overview

```
/opt/acgwarehouse/frontend/vue-gallery/
├── src/
│   ├── assets/
│   │   └── app.css          # 复用原原型 CSS
│   ├── composables/
│   │   ├── useCarousel.ts   # 轮播逻辑
│   │   ├── useSelection.ts  # 批量选择逻辑
│   │   ├── useZoom.ts       # 图片缩放逻辑
│   │   ├── useToast.ts      # Toast 通知
│   │   ├── useForm.ts       # 表单验证与提交
│   │   ├── useTabs.ts       # 标签页切换
│   │   └── useMotion.ts     # 入场动效
│   ├── components/
│   │   ├── AppHeader.vue    # 顶部导航
│   │   ├── AppFooter.vue    # 页脚
│   │   ├── Toast.vue        # Toast 通知组件
│   │   ├── BatchPanel.vue   # 批量操作面板
│   │   ├── ArtCard.vue      # 作品卡片
│   │   ├── Carousel.vue     # 轮播组件
│   │   ├── SegmentedControl.vue  # 分段切换
│   │   └── Panel.vue        # 面板容器
│   ├── pages/
│   │   ├── GalleryPage.vue  # 图库首页
│   │   ├── DetailPage.vue   # 图片详情
│   │   ├── SearchPage.vue   # 智能搜索
│   │   ├── TrendingPage.vue # 热榜
│   │   ├── CollectionsPage.vue  # 收藏夹
│   │   └── AccountPage.vue  # 账户中心
│   ├── router/
│   │   └── index.ts         # 路由配置
│   ├── types/
│   │   └── index.ts         # TypeScript 类型定义
│   ├── App.vue              # 根组件
│   └── main.ts              # 入口文件
├── index.html
├── vite.config.ts
├── tsconfig.json
├── package.json
└── README.md
```

## Data Flow

### 全局状态

```typescript
// src/composables/useToast.ts
const toastState = reactive({
  message: '',
  isOpen: false
})

// src/composables/useSelection.ts
const selectionState = reactive({
  selectedIds: new Set<string>()
})
```

### 页面数据

```typescript
// src/types/index.ts
interface ArtItem {
  id: string
  title: string
  tags: string[]
  score: number
  favorites: number
  previewVariant: 'default' | 'tall' | 'wide'
}

interface CarouselSlide {
  id: string
  title: string
  description: string
  tag: string
  tagType: 'hot' | 'normal'
  score: number
  favorites: number
  artVariant: 'rain' | 'character' | 'album'
}

interface Album {
  id: string
  name: string
  tags: string[]
  count: number
  lastUpdated: string
}
```

## Component Contracts

### AppHeader.vue

```vue
<script setup lang="ts">
defineProps<{
  activeRoute?: string
}>()
</script>
```

- Props: `activeRoute` - 当前激活的导航项
- Slots: 默认 slot 用于扩展导航
- Events: 无

### Carousel.vue

```vue
<script setup lang="ts">
import type { CarouselSlide } from '@/types'

const slides: CarouselSlide[] = [...]
const currentIndex = ref(0)
</script>
```

- Props: `slides` - 轮播数据数组
- Events: 无（内部状态）
- 键盘导航: ArrowLeft / ArrowRight

### ArtCard.vue

```vue
<script setup lang="ts">
import type { ArtItem } from '@/types'

defineProps<{
  item: ArtItem
  selectable?: boolean
}>()

const emit = defineEmits<{
  select: [id: string]
}>()
</script>
```

- Props: `item` - 作品数据, `selectable` - 是否可选择
- Events: `select` - 选择事件
- 点击行为: 点击卡片触发选择，点击链接进入详情页

### BatchPanel.vue

```vue
<script setup lang="ts">
import { useSelection } from '@/composables/useSelection'

const { selectedCount, clearSelection } = useSelection()
</script>
```

- Props: 无（使用全局 selectionState）
- Events: 无（按钮操作直接调用 composable）
- 显示条件: `selectedCount > 0`

### Toast.vue

```vue
<script setup lang="ts">
import { useToast } from '@/composables/useToast'

const { message, isOpen } = useToast()
</script>
```

- Props: 无（使用全局 toastState）
- 自动关闭: 1800ms 后隐藏

## Composables Design

### useCarousel

```typescript
export function useCarousel(slides: CarouselSlide[]) {
  const currentIndex = ref(0)
  
  const next = () => currentIndex.value = (currentIndex.value + 1) % slides.length
  const prev = () => currentIndex.value = (currentIndex.value - 1 + slides.length) % slides.length
  const goto = (index: number) => currentIndex.value = index
  
  const currentSlide = computed(() => slides[currentIndex.value])
  const statusText = computed(() => `第 ${currentIndex.value + 1} 张，共 ${slides.length} 张：${currentSlide.value.title}`)
  
  return { currentIndex, currentSlide, statusText, next, prev, goto }
}
```

### useSelection

```typescript
const selectionState = reactive({
  selectedIds: new Set<string>()
})

export function useSelection() {
  const selectedCount = computed(() => selectionState.selectedIds.size)
  
  const toggle = (id: string) => {
    if (selectionState.selectedIds.has(id)) {
      selectionState.selectedIds.delete(id)
    } else {
      selectionState.selectedIds.add(id)
    }
  }
  
  const clear = () => selectionState.selectedIds.clear()
  
  return { selectedIds: selectionState.selectedIds, selectedCount, toggle, clear }
}
```

### useZoom

```typescript
export function useZoom(min = 0.75, max = 1.7, step = 0.15) {
  const zoom = ref(1)
  
  const zoomIn = () => zoom.value = Math.min(max, zoom.value + step)
  const zoomOut = () => zoom.value = Math.max(min, zoom.value - step)
  const reset = () => zoom.value = 1
  
  return { zoom, zoomIn, zoomOut, reset }
}
```

### useToast

```typescript
const toastState = reactive({
  message: '',
  isOpen: false
})

let timer: number | null = null

export function useToast() {
  const show = (message: string, duration = 1800) => {
    toastState.message = message
    toastState.isOpen = true
    if (timer) clearTimeout(timer)
    timer = window.setTimeout(() => {
      toastState.isOpen = false
    }, duration)
  }
  
  return { ...toRefs(toastState), show }
}
```

### useForm

```typescript
export function useForm<T extends Record<string, string>>(initialValues: T) {
  const values = reactive<T>(initialValues)
  const errors = reactive<Record<string, string>>({})
  const status = ref<'idle' | 'loading' | 'success' | 'error'>('idle')
  
  const validate = () => {
    // 实现表单验证逻辑
  }
  
  const submit = async (onSubmit: (values: T) => Promise<void>) => {
    if (!validate()) return
    status.value = 'loading'
    try {
      await onSubmit(values)
      status.value = 'success'
    } catch {
      status.value = 'error'
    }
  }
  
  return { values, errors, status, validate, submit }
}
```

## Router Configuration

```typescript
// src/router/index.ts
import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/', name: 'gallery', component: () => import('@/pages/GalleryPage.vue') },
  { path: '/detail', name: 'detail', component: () => import('@/pages/DetailPage.vue') },
  { path: '/search', name: 'search', component: () => import('@/pages/SearchPage.vue') },
  { path: '/trending', name: 'trending', component: () => import('@/pages/TrendingPage.vue') },
  { path: '/collections', name: 'collections', component: () => import('@/pages/CollectionsPage.vue') },
  { path: '/account', name: 'account', component: () => import('@/pages/AccountPage.vue') },
]

export const router = createRouter({
  history: createWebHistory(),
  routes
})
```

## CSS Migration Strategy

1. **直接复制**: 将 `assets/app.css` 复制到 `src/assets/app.css`
2. **全局引入**: 在 `main.ts` 中 `import './assets/app.css'`
3. **无样式冲突**: 不引入 UI 库，保持原有样式体系
4. **组件复用**: 使用原有 CSS 类名（`.btn`、`.panel`、`.art-card` 等）

## Accessibility Preservation

### 保持的 aria 属性

- `role="tablist"` / `role="tab"` / `role="tabpanel"` - 标签页
- `aria-selected` / `aria-controls` / `aria-hidden` - 标签页状态
- `aria-label` / `aria-labelledby` - 区域标注
- `aria-current` - 导航当前位置
- `aria-live="polite"` - Toast 状态播报
- `aria-invalid` / `aria-describedby` - 表单错误关联

### 键盘导航

- Tab: 焦点移动
- ArrowLeft / ArrowRight: 轮播切换、标签页切换
- Enter / Space: 按钮激活、卡片选择

## Build Configuration

### vite.config.ts

```typescript
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  }
})
```

### tsconfig.json

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "strict": true,
    "jsx": "preserve",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "esModuleInterop": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "skipLibCheck": true,
    "paths": {
      "@/*": ["./src/*"]
    }
  },
  "include": ["src/**/*.ts", "src/**/*.vue"],
  "exclude": ["node_modules"]
}
```

## Rollback Strategy

如果转换过程中遇到严重问题：
1. 原原型 `/opt/acgwarehouse/frontend/example` 保持不变
2. Vue 项目可独立删除，不影响原项目
3. 可随时回退到静态 HTML 方案