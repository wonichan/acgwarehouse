# 社区精选轮播优化：3s自动播放 + 淡入淡出动画

## Goal

前端图库首页（`GalleryPage`）的"社区精选轮播"（`Carousel.vue` + `useCarousel.ts`）目前无自动播放，切换为横向 `translateX` 位移。改造为：默认 3 秒自动切换下一张，切换过渡改为淡入淡出（opacity + 轻微 scale/blur），保留现有交互与无障碍能力，并在受限运动偏好下降级。

## Scope

- **修改文件**：
  - `frontend/vue-gallery/src/composables/useCarousel.ts`
  - `frontend/vue-gallery/src/components/Carousel.vue`
- **不改**：`GalleryPage.vue` 的调用方式、`CarouselSlide` 类型、后端接口、其它组件。

## Requirements

### 1. 自动播放（useCarousel.ts）

- `useCarousel(slides, options?)` 新增可选参数 `options: { interval?: number; autoplay?: boolean }`，默认 `interval = 3000`、`autoplay = true`。
- 组件挂载后启动定时器，每 `interval` 毫秒调用 `next()`。
- 用户主动交互（`next()` / `prev()` / `goto(i)`）后需**重置**计时器，避免刚点完又立刻切走。
- 暴露 `pause()` / `resume()` 供模板绑定。
- 以下情形不启动或暂停：
  - `slides.length <= 1`
  - `window.matchMedia('(prefers-reduced-motion: reduce)').matches`
  - `document.visibilityState === 'hidden'`（浏览器后台标签页时）
- `onBeforeUnmount` 与 `onDeactivated`（`<KeepAlive>` 场景）必须清理定时器与事件监听。

### 2. 淡入淡出动画（Carousel.vue）

- 所有 slide 使用 CSS Grid 堆叠在同一格（`grid-area: 1 / 1`），取代当前的横向 flex + `translateX(--carousel-offset)`。
- 非活动 slide：`opacity: 0`、`transform: scale(0.985)`、`filter: blur(4px)`、`pointer-events: none`、`aria-hidden="true"`。
- 活动 slide：`opacity: 1`、`transform: scale(1)`、`filter: blur(0)`。
- 过渡：`opacity/transform/filter` 时长 720ms，缓动 `cubic-bezier(0.32, 0.72, 0, 1)`（禁止 `linear` / `ease-in-out`）。
- `<aside>` 根元素上：`mouseenter → pause`、`mouseleave → resume`、`focusin → pause`、`focusout → resume`（键盘用户可看清停留内容）。
- 视口区域高度：`.carousel-viewport` 需容纳最高 slide，不能因绝对定位导致坍塌（用 grid 天然继承最大子项尺寸）。

### 3. 无障碍与降级

- 保留现有 `role="region"`、`aria-roledescription="carousel"`、`aria-live` 状态文本、圆点导航、键盘左右切换。
- `prefers-reduced-motion: reduce`：
  - 不启动自动播放；
  - 淡入淡出仅保留 `opacity` 且过渡 ≤ 120ms，禁用 `blur` 与 `scale`。
- `slides.length <= 1`：不启动自动播放（现状已在 `next()` 上自然满足，需在启动时先行校验）。

### 4. 兼容性

- 不引入新依赖；仅 Vue 3 `ref` / `computed` / `onMounted` / `onBeforeUnmount` / `onDeactivated`（Vue 已具备）。
- `offset` 与 `--carousel-offset` 变量在改为叠层动画后不再驱动位移，`useCarousel` 可保留返回值以最小影响 API，但 `Carousel.vue` 模板不再消费。

## Acceptance Criteria

- [ ] 打开图库首页，社区精选轮播在无交互 3s 后自动切换到下一张，且循环回到首张。
- [ ] slide 切换时视觉表现为淡入淡出（对旧的横向滑动为可感知的替换），非当前 slide 无法被 Tab 到、无法被指针点击。
- [ ] 悬停轮播区域时自动播放暂停，鼠标移开 3s 内自动恢复推进。
- [ ] 键盘 ← / → 仍可切换，且切换后计时器重置。
- [ ] 圆点导航点击后切换到目标 slide，计时器重置。
- [ ] 切换过渡时长约 720ms，缓动为自定义 `cubic-bezier`，无 `linear` / `ease-in-out`。
- [ ] 系统设置"减少动效"时：不自动播放、无 blur/scale、opacity 过渡 ≤ 120ms。
- [ ] 浏览器后台标签页时不推进；返回前台后恢复。
- [ ] `<KeepAlive>` 场景切走再回来不重复注册定时器（无内存泄漏、无双倍推进）。
- [ ] `npm run build`（vue-tsc + vite build）通过。

## Notes

- 现有 `.carousel-track { transform: translateX(...) }` 与 `.carousel-slide { opacity: 0.5 → 1 }` 相关 CSS 将被 grid 叠层替代，其它 `.focus-card` / `.focus-art` / `.carousel-footer` / `.carousel-rail-dot` 样式保持不变。
- `useCarousel` 保留 `offset` 返回值以避免破坏其它潜在调用（当前仅 `Carousel.vue` 一处调用者）。
