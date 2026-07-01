# Design: 图片详情页操作刷新与图库滚动位置记忆

## 问题根因

### 根因 1: 详情页操作后整页闪烁

`DetailPage.vue` 的 `loadDetail()` 函数在入口设置 `loading.value = true`（line 66）。模板中 `v-else-if="loading"`（line 161-167）会渲染加载骨架屏，替换掉整个详情视图。

三个 mutation handler 在成功后都调用 `await loadDetail()`：
- `handleSaveRating`（line 95）：评分成功后
- `onPickerSuccess`（line 131）：收藏/标签成功后

每次调用都触发 `loading=true` → 骨架屏闪现 → `loading=false` → 详情重新渲染，造成视觉刷新。

### 根因 2: 图库滚动位置丢失

`GalleryPage.vue` 在 `onMounted` 中调用 `loadInitialGallery()`，每次组件挂载都从 `page=1` 重新加载全部数据。组件没有被缓存，导航到详情页时卸载，返回时重新挂载，所有状态（artItems、masonryColumns、currentPage、scrollY）全部丢失。

`App.vue` 的 `<RouterView />` 没有 `<KeepAlive>` 包裹。`router/index.ts` 没有 `scrollBehavior`。

## 方案

### 方案 1: 详情页静默刷新

**核心思路**：mutation 成功后仍然重新 `getImage(id)`（满足 spec "最终状态以 getImage 为准"），但不设置 `loading=true`，直接用新数据 patch `detail.value`，模板始终停留在详情视图分支，不闪骨架屏。

**新增函数** `refreshDetailSilently()`：
```typescript
async function refreshDetailSilently(): Promise<void> {
  if (imageId.value === null) return
  try {
    const response = await getImage(imageId.value)
    detail.value = response
    selectedScore.value = clampRatingScore(response.my_rating ?? response.avg_score)
  } catch {
    // 静默刷新失败不影响已展示的数据，用户可手动重试或继续操作
  }
}
```

**评分乐观更新**：`handleSaveRating` 在 `rateImage` 成功后，先用 `RatingResponse.score` 立即 patch `detail.value.my_rating`，给用户即时反馈，然后调用 `refreshDetailSilently()` 同步 `avg_score`/`rating_count`。

```typescript
async function handleSaveRating(): Promise<void> {
  // ... 登录校验 ...
  savingRating.value = true
  try {
    const result = await rateImage(imageId.value, clampRatingScore(selectedScore.value))
    // 乐观更新 my_rating，立即反馈
    if (detail.value) {
      detail.value = { ...detail.value, my_rating: result.score }
    }
    show(`评分已更新为 ${result.score}/100`)
    // 静默刷新 avg_score / rating_count
    await refreshDetailSilently()
  } catch (e) { /* ... */ }
  finally { savingRating.value = false }
}
```

**收藏/标签**：`onPickerSuccess` 不再调用 `loadDetail()`，改为 `refreshDetailSilently()`。

**保留 `loadDetail()`**：仅用于 `onMounted`（首次加载/路由 id 变化）和错误重试按钮，这些场景需要显示 loading 骨架屏。

**关键不变量**：
- `loading.value` 仅在首次加载、id 变化、手动重试时为 true。
- mutation 后的刷新永不设置 `loading=true`。
- `detail.value` 始终最终以 `getImage(id)` 返回为准。

### 方案 2: 图库页 KeepAlive

**核心思路**：用 `<KeepAlive include="GalleryPage">` 缓存图库页组件实例。导航到详情页时组件 deactivated（DOM 保留，观察者断开），返回时 activated（观察者重连，不重新加载）。

**App.vue 改动**：
```vue
<RouterView v-slot="{ Component }">
  <KeepAlive :include="['GalleryPage']">
    <component :is="Component" />
  </KeepAlive>
</RouterView>
```

**GalleryPage.vue 改动**：

1. 声明组件名（KeepAlive `include` 按名称匹配）：
```typescript
defineOptions({ name: 'GalleryPage' })
```

2. 引入 `onActivated` / `onDeactivated`：
```typescript
import { onMounted, onBeforeUnmount, onActivated, onDeactivated, nextTick } from 'vue'
```

3. 生命周期重构：
```typescript
let initialized = false

onMounted(() => {
  void loadInitialGallery().then(() => { initialized = true })
})

onActivated(() => {
  if (!initialized) return  // 首次挂载由 onMounted 负责
  // DOM 已保留，只需重新挂载观察者
  nextTick(() => {
    observeMasonryContainer()
    observeGallerySentinel()
  })
})

onDeactivated(() => {
  // 断开观察者，保留数据/DOM
  galleryObserver?.disconnect()
  galleryObserver = null
  masonryResizeObserver?.disconnect()
  masonryResizeObserver = null
})

onBeforeUnmount(() => {
  // 真正销毁时清理（保险）
  galleryObserver?.disconnect()
  galleryObserver = null
  masonryResizeObserver?.disconnect()
  masonryResizeObserver = null
})
```

**为什么 KeepAlive 优于手动状态持久化**：
- DOM 保留 → 浏览器自然保留滚动位置，无需手动 save/restore scrollY。
- 瀑布流 DOM 节点保留 → 无需重建 masonryColumns，无布局抖动。
- 组件实例保留 → artItems、currentPage、hasMore 等状态全部保留。
- 只需处理观察者的断开/重连，改动最小。

**风险与缓解**：
- **风险**：KeepAlive 下 `onMounted` 只触发一次，首次进入时 `onActivated` 也会触发但 `initialized=false`，被 guard 跳过，不会重复加载。✓
- **风险**：返回图库后 sentinel IntersectionObserver 需要重新 observe。`observeGallerySentinel` 内部有 `galleryObserver !== null` guard，deactivated 时已置 null，activated 时可重新创建。✓
- **风险**：ResizeObserver 同理，`observeMasonryContainer` 有 `masonryResizeObserver !== null` guard。✓
- **风险**：数据过期。用户显式切换筛选时会调用 `handleFilter` → `loadGallery`，会重新加载。这是期望行为。✓

## 涉及文件

| 文件 | 改动 |
|------|------|
| `src/pages/DetailPage.vue` | 新增 `refreshDetailSilently`；`handleSaveRating` 加乐观更新；`onPickerSuccess` 改用静默刷新 |
| `src/pages/GalleryPage.vue` | 加 `defineOptions({ name: 'GalleryPage' })`；加 `onActivated`/`onDeactivated` 生命周期管理观察者 |
| `src/App.vue` | RouterView 包裹 KeepAlive include GalleryPage |

## 兼容性

- 不改 API 调用签名、不改路由路径、不改后端契约。
- 不影响其他页面（搜索/热榜/收藏/账户）的正常挂载/卸载。
- KeepAlive 仅缓存 GalleryPage，其他页面照常销毁重建。

## 回滚

- DetailPage：将 `refreshDetailSilently` 调用改回 `loadDetail` 即可。
- GalleryPage：删除 `defineOptions` 和 `onActivated`/`onDeactivated` 即可。
- App.vue：删除 KeepAlive 包裹即可。
- 三处改动互相独立，可单独回滚。
