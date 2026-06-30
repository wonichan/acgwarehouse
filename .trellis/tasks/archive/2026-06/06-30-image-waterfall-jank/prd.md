# 排查图片瀑布流加载抖动

## Goal

查明并彻底解决前端图库页下拉加载更多图片时页面抖动的问题，消除 CSS multi-column 追加重排与图片加载后尺寸变化带来的滚动跳动。

## Confirmed Facts

- 首页图库入口为 `frontend/vue-gallery/src/pages/GalleryPage.vue`。
- 下拉加载用 `IntersectionObserver` 观察底部 sentinel，触发 `loadNexalleryPage()`。
- 下一页请求使用 `currentPage + 1`，返回后映射为 `ArtItem` 并追加到 `artItems`。
- 图库容器使用 `.masonry { columns: 4 240px; column-gap: ... }`，属于 CSS multi-column 瀑布流。
- 卡片使用 `.art-card { break-inside: avoid; margin-bottom: ... }`。
- 图片预览区域只按 `previewVariant` 使用固定 `min-height`，未按后端真实 `width/height` 预留比例空间。
- `ImageItem` DTO 已包含真实 `width` 与 `height`。
- `ArtItem` 当前未携带真实图片宽高。
- 现有分页测试只覆盖 sentinel 触发与追加行为，不覆盖滚动抖动、布局稳定性或图片加载后的 CLS。

## Requirements

- 给出根因判断：是瀑布流布局固有重排、图片占位不足、分页追加 bug，或多因叠加。
- 证据必须来自代码、样式、运行表现或可复现实验，不凭感觉定论。
- 保持现有图库分页契约：首屏 `page=1`，追加 `currentPage+1`，保留已加载图片。
- 替换易重排的 CSS multi-column 瀑布流，不继续依赖 `.masonry { columns: ... }` 承载主图库。
- 使用后端真实图片宽高为卡片预览区预留稳定比例空间，图片未加载/懒加载时不改变卡片布局高度。
- 不引入 mock 图片数据，不改后端 API 合约。
- 不扩大到详情页、搜索页、热榜页重构。

## Acceptance Criteria

- [ ] 明确指出抖动根因与涉及文件/代码位置。
- [ ] 验证下拉加载是否存在重复请求、覆盖列表、错误页码、过早触发等代码 bug。
- [ ] 验证图片加载后是否因尺寸占位变化导致布局跳动。
- [ ] 验证 CSS multi-column 瀑布流追加卡片时是否会重排既有列内容。
- [ ] 主图库不再使用 CSS multi-column 作为瀑布流布局。
- [ ] 卡片图片区域按真实宽高或稳定 fallback 预留空间。
- [ ] `npm run build` 或等价前端检查通过。
- [ ] 保留无限滚动既有行为与 next-page 错误处理。

## Out of Scope

- 后端分页接口改造。
- 图片 CDN、缩略图生成、服务端裁剪策略。
- 全站 UI 重设计。
- 引入大型虚拟列表或第三方瀑布流库。

## Open Questions

- 无。用户已要求一步到位，允许替换瀑布流布局实现。
