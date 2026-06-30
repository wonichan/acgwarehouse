# 实施计划

## Steps

- [x] 读取 `trellis-before-dev` 与前端相关规范，确认编辑边界。
- [x] 调整 `Carousel.vue` 模板：图片改为详情页链接，移除可见文件名，底部导航改为进度线和分页按钮，必要时增加局部结构类名。
- [x] 将后端图片 `width` / `height` 透传到 `CarouselSlide`，轮播图片按真实宽高比展示。
- [x] 在 `Carousel.vue` 增加 scoped CSS：外层层次、双层卡片、图文间距、标题换行、数据格、底部导航、响应式断点。
- [x] 避免触碰 `vite.config.ts`、`.env` 等已有未提交变更。
- [x] 运行 `npm run build` 于 `frontend/vue-gallery`。
- [x] 如构建通过，检查 `git diff -- frontend/vue-gallery/src/components/Carousel.vue .trellis/tasks/06-30-community-focus-carousel-redesign`，确认改动范围。

## Validation

```powershell
npm run build
```

Run from:

```text
E:\program\obsidian\acg\acgwarehouse\frontend\vue-gallery
```

Result: passed.

Additional visual checks on `http://localhost:5173/`:

- Desktop 1280x900: no horizontal overflow; carousel no longer exposes filename text; image link resolved to `/detail?id=172`.
- Tablet 820x900: no horizontal overflow; layout remains readable.
- Mobile 390x844: no horizontal overflow; carousel collapses to single column.
- Aspect check: first carousel image used backend dimensions `1200 / 2721`, rendered with `object-fit: contain`, no filename leak and no horizontal overflow.

## Review Gate

实现前需用户确认进入 Phase 2，然后执行：

```powershell
python .\.trellis\scripts\task.py start .trellis\tasks\06-30-community-focus-carousel-redesign
```
