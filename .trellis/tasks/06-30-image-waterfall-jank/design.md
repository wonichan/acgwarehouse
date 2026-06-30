# 排查图片瀑布流加载抖动 - Design

## Problem

当前图库页使用 CSS multi-column 作为瀑布流容器。追加新卡片后，浏览器会重新平衡列内容；同时图片预览区只用 `previewVariant` 固定高度，未使用真实宽高占位，图片懒加载完成后仍可能造成局部尺寸变化。二者叠加导致下拉加载更多时可见区域抖动。

## Decision

用前端稳定列布局替代 CSS multi-column：

- `GalleryPage.vue` 根据 `ArtItem` 的预估高度把卡片分配到列数组。
- 首屏和追加页使用同一分配函数，新增项只追加到当前最短列，既有项不被重新分配。
- 响应式列数由 `ResizeObserver` 根据容器宽度计算，桌面最多 4 列，平板 2-3 列，窄屏 1 列。
- 列数变化时允许重算全量列，因为视口变化本身是显式布局变化；普通下拉追加不重排既有项。
- `ArtItem` 增加 `imageWidth` / `imageHeight`，由 `imageToArtItem()` 从后端 `ImageItem.width/height` 填充。
- `ArtCard.vue` 给 `.art-preview` 写入 `aspect-ratio` style；无有效宽高时沿用 `previewVariant` 的 fallback 高度。

## Research Influence

- 参考 web.dev 的 CLS 建议，图片尺寸占位是必做项，不是附属优化。
- 参考 MDN 与 Chrome 当前说明，不采用原生 CSS masonry；其兼容性和规范形态尚不足以作为生产依赖。
- 参考 Masonic 这类高性能 masonry 实现，核心思路是 positioner/cache、容器测量、无限加载和必要时虚拟化。本任务先实现轻量版 positioner：稳定列高缓存与最短列追加。
- 参考 Unsplash / Pexels / Pinterest / Flickr 等图片产品形态，本项目更接近探索型图片流，继续保留 masonry 视觉密度；不改成相册式 justified row。

## Boundaries

- 只改主图库瀑布流相关文件：`GalleryPage.vue`、`ArtCard.vue`、`types/index.ts`、`imagePresentation.ts`、`app.css`，以及必要测试。
- 不改后端 API，不改搜索/热榜/详情逻辑。
- 不引入第三方 masonry 包。

## Data Flow

`getImages()` -> `ImageItem(width,height,url,...)` -> `imageToArtItem()` -> `ArtItem(imageWidth,imageHeight,previewVariant,...)` -> `GalleryPage` 稳定列分配 -> `ArtCard` aspect-ratio 占位。

## Trade-offs

- JS 分列比纯 CSS 多少许前端逻辑，但能保证追加时既有卡片位置稳定。
- 只按预估高度分配列，不等图片真实渲染后回流重排；列高度可能不是像素级最优，但滚动稳定性优先。
- 视口宽度变化时重排是可接受行为；无限滚动追加时不重排是核心约束。

## Validation

- 静态测试覆盖：存在稳定分列逻辑、`ArtItem` 传递真实宽高、主图库不再使用 CSS columns。
- 构建验证：`npm run build`。
- 若本地服务可启动，再用浏览器检查滚动加载、console error、可见卡片位置是否稳定。
