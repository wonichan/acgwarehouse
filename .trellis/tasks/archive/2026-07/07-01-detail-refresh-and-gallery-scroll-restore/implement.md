# Implement: 图片详情页操作刷新与图库滚动位置记忆

## 执行顺序

按依赖关系分三步，每步独立可验证。

---

## Step 1: DetailPage 静默刷新

**文件**: `src/pages/DetailPage.vue`

**改动**:

1. 新增 `refreshDetailSilently()` 函数（在 `loadDetail` 之后）：
   - 不设置 `loading.value = true`
   - 调用 `getImage(imageId.value)` 获取最新详情
   - 成功后 patch `detail.value` 和 `selectedScore`
   - 失败时静默（不覆盖已有数据，不设 error）

2. 修改 `handleSaveRating()`：
   - `rateImage` 成功后，先用 `result.score` 乐观 patch `detail.value.my_rating`
   - 然后调用 `await refreshDetailSilently()` 替代 `await loadDetail()`

3. 修改 `onPickerSuccess()`：
   - 将 `await loadDetail()` 改为 `await refreshDetailSilently()`

4. 保留 `loadDetail()` 不变（onMounted / watch imageId / 重试按钮仍用它）。

**验证**:
- `npm run build` 通过
- 手动：详情页打分 → 无骨架屏闪烁，"我的评分"立即更新
- 手动：详情页收藏 → 无骨架屏闪烁，"已收藏"出现
- 手动：详情页标签 → 无骨架屏闪烁，标签列表更新

---

## Step 2: GalleryPage KeepAlive 生命周期

**文件**: `src/pages/GalleryPage.vue`

**改动**:

1. 在 `<script setup>` 顶部加 `defineOptions({ name: 'GalleryPage' })`

2. import 补充 `onActivated`, `onDeactivated`：
   ```typescript
   import { ref, onMounted, onBeforeUnmount, onActivated, onDeactivated, nextTick } from 'vue'
   ```

3. 新增模块级 `let initialized = false` 标志

4. `onMounted` 末尾将 `initialized = true`：
   ```typescript
   onMounted(() => {
     void loadInitialGallery().then(() => { initialized = true })
   })
   ```

5. 新增 `onActivated`：仅当 `initialized === true` 时重新挂载观察者：
   ```typescript
   onActivated(() => {
     if (!initialized) return
     nextTick(() => {
       observeMasonryContainer()
       observeGallerySentinel()
     })
   })
   ```

6. 新增 `onDeactivated`：断开观察者，保留数据：
   ```typescript
   onDeactivated(() => {
     galleryObserver?.disconnect()
     galleryObserver = null
     masonryResizeObserver?.disconnect()
     masonryResizeObserver = null
   })
   ```

7. 保留 `onBeforeUnmount` 不变（作为最终保险）。

**验证**:
- `npm run build` 通过
- 手动：图库滚动 3 屏 → 进详情 → 返回 → 滚动位置恢复，卡片仍在
- 手动：返回后继续滚动 → sentinel 能触发下一页加载
- 手动：返回后 resize 窗口 → masonry 列数重算正常
- 手动：切换筛选 → 仍重新加载

---

## Step 3: App.vue KeepAlive 包裹

**文件**: `src/App.vue`

**改动**:

将 `<RouterView />` 改为：
```vue
<RouterView v-slot="{ Component }">
  <KeepAlive :include="['GalleryPage']">
    <component :is="Component" />
  </KeepAlive>
</RouterView>
```

**验证**:
- `npm run build` 通过
- 手动：图库 → 详情 → 返回，确认 KeepAlive 生效（Network 无 page=1 重复请求）
- 手动：其他页面（搜索/热榜/收藏）正常切换，未被缓存

---

## 最终验证

```bash
cd frontend/vue-gallery
npm run build
```

构建通过后，启动 dev server 做完整 E2E 手动验证：

1. **打分不闪烁**：登录 → 详情页 → 保存评分 → 无骨架屏，"我的评分"立即更新
2. **收藏不闪烁**：详情页 → 收藏到相册 → 无骨架屏，"已收藏"出现
3. **标签不闪烁**：详情页 → 管理标签 → 添加标签 → 无骨架屏，标签列表更新
4. **滚动恢复**：图库滚动 3 屏 → 进详情 → 返回 → 位置恢复
5. **无限滚动恢复**：返回后继续下滑 → 下一页正常加载
6. **筛选不退化**：返回后切换筛选 → 正常重新加载
7. **其他页面不受影响**：搜索/热榜/收藏/账户页正常挂载卸载

## 回滚点

- Step 1 回滚：`refreshDetailSilently` → `loadDetail`
- Step 2 回滚：删除 `defineOptions` + `onActivated`/`onDeactivated`
- Step 3 回滚：删除 KeepAlive 包裹
