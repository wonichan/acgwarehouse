# 排查图片瀑布流加载抖动 - Implementation

## Checklist

- [ ] 读取前端开发前规范。
- [ ] 给 `ArtItem` 增加可选真实图片宽高字段。
- [ ] 在 `imageToArtItem()` 中填充 `imageWidth` / `imageHeight`。
- [ ] 在 `ArtCard.vue` 中用 `aspect-ratio` 预留图片区域，保留 fallback 高度。
- [ ] 在 `GalleryPage.vue` 中实现稳定分列：
  - [ ] 维护 `masonryColumns` 与 `columnHeights`。
  - [ ] 根据容器宽度计算列数。
  - [ ] 首屏重建列，追加页只追加新项到最短列。
  - [ ] filter 切换时重建布局。
  - [ ] unmount 时断开 observer。
- [ ] 替换模板中的单层 `.masonry` v-for 为列式 v-for。
- [ ] 调整 `app.css`，移除主图库 CSS columns，改为 flex/grid 列容器。
- [ ] 更新或新增分页/布局静态测试。
- [ ] 运行前端构建或可用测试。

## Validation Commands

```powershell
npm run build
```

在 `frontend/vue-gallery` 目录执行。

## Risk Points

- `ArtCard` 被每日推荐复用，aspect-ratio 不能破坏其现有 grid 高度。
- 响应式列数变化要重算列，不能造成空列或重复卡片。
- `IntersectionObserver` sentinel 必须仍在所有列之后，不能放进某一列。

## Rollback

若稳定列实现导致显示错误，回退 `GalleryPage.vue` 模板/脚本与 `.masonry` CSS，即可恢复原 multi-column 行为。
