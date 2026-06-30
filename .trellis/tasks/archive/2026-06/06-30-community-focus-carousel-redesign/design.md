# 设计方案

## Scope

重设计 `frontend/vue-gallery/src/components/Carousel.vue` 的结构与 scoped 样式，并把后端已有图片宽高从 `RankingResponse.image` 透传到 `CarouselSlide`。保留 `useCarousel`、`GalleryPage.vue` 调用方式与后端接口。

## Design Direction

- 采用“Soft Structuralism + Double-Bezel”思路：外层面板承载层次，内部焦点卡片再形成独立内容核心，避免浅色背景与 UI 融在一起。
- 保持本项目既有 token：`--space-*`、`--radius-*`、`--surface`、`--fg`、`--accent`、`--motion-*`，不新增第三方字体、图标或动画库。
- 用克制的非纯黑阴影、内边框、浅色层叠背景提升质感；避免大面积紫蓝渐变、强发光和纯装饰图形。
- 动效仅使用 `transform`、`opacity`、`background`、`border-color` 等低风险属性，使用项目已有 motion token 或定制 cubic-bezier。

## Component Structure

- Header：左侧标题，右侧上一张/下一张按钮。移动端按钮变为等宽一行，避免压缩标题。
- Viewport：保留轨道和 slide 结构，slide 内改为双层卡片。
- Focus card：桌面为图文两列；图片列作为详情入口，使用后端 `width / height` 计算 CSS `aspect-ratio`，并用 `object-fit: contain` 保留完整图片；文案列不显示文件名，改用“本周第 N 位”“热度分”“均分”等信息承载内容重点。
- Stats：热度、收藏改为小型数据格，数字使用等宽字体与 tabular nums。
- Footer：由拥挤 chip 横滑改为“进度线 + 紧凑分页点/数字 + 状态文本”。分页按钮可点、可聚焦，当前项清晰。

## Responsive Contract

- Desktop：图文两列，图片与文案至少保持 token 化间距，卡片宽度随容器伸缩。
- Tablet：两列比例收窄，图片高度用 `clamp()`，文案不溢出。
- Mobile：单列布局，图片在上、文字在下，footer 垂直排列；所有按钮最小触达尺寸不小于 40px；图片比例使用自身宽高并通过 `max-height` / `min-height` 控制极端比例。

## Compatibility

- 不引入新库；允许使用 CSS 容器查询处理桌面窄侧栏。
- 保留键盘左右键、ARIA region、slide label、分页按钮 aria-current。
- 图片详情链接使用现有 `/detail?id=<imageId>` 路由，不新增路由。
- 若浏览器不支持 `color-mix()`，现有项目已经广泛使用；本次不引入新的兼容边界。

## Risks

- 旧的全局 carousel 样式仍在 `app.css`，组件 scoped 样式需覆盖到足够具体，避免大范围改全局样式。
- 截图中的真实图片可能比例差异大，图片容器需用后端宽高生成动态 aspect-ratio，并对极端比例设置高度边界。
