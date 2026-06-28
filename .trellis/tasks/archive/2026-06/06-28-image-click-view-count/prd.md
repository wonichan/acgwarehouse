# 点击图片未增加浏览数排查

## Goal

确认用户反馈“点击图片后图片浏览数没有增加”是产品限制、展示延迟，还是代码缺陷；产出可验证的原因说明，并在用户批准后再进入修复。

## Background / User Report

- 用户观察到：点击图片没有增加图片的浏览数。
- 用户想知道：这是系统限制还是 bug。

## Requirements

- R1: 排查图片点击、详情加载、浏览事件记录、浏览数展示之间的数据流。
- R2: 点击进入详情页后，当前详情页展示的“浏览 N 次”必须立即包含本次浏览。
- R3: 修复应保持后端浏览事件异步缓冲落库机制，不因为展示即时 +1 而改为每次同步写库。
- R4: 修复应覆盖详情接口返回值，并避免影响列表、搜索、热榜已有计数读取语义。

## Acceptance Criteria

- [x] 能说明点击图片后是否会调用后端详情接口。
- [x] 能说明后端是否记录浏览事件并更新 `view_count`。
- [x] 能说明前端当前显示的浏览数为什么没有立即变化。
- [x] 进入图片详情页的首个响应中，`image.view_count` 比记录前数据库快照多 1。
- [x] 浏览事件仍会通过 `ViewBuffer` 异步 flush 并最终累加数据库 `image.view_count`。
- [x] 有后端测试覆盖详情返回的即时浏览数语义。
- [x] 验证相关 Go 包测试通过。

## Confirmed Facts

- `frontend/vue-gallery/src/pages/DetailPage.vue:130-137` 在详情页挂载和 `imageId` 变化时调用 `loadDetail()`。
- `frontend/vue-gallery/src/pages/DetailPage.vue:71-74` 的 `loadDetail()` 调用 `getImage(imageId)`，并把响应保存到 `detail`。
- `frontend/vue-gallery/src/pages/DetailPage.vue:193-195` 直接显示响应里的 `image.view_count`：`浏览 {{ image.view_count }} 次`。
- `frontend/vue-gallery/src/api/client.ts:115-118` 的 `getImage(id)` 请求 `/images/${id}`。
- `internal/handler/image.go:43-55` 的详情 handler 调用 `imageService.Detail(...)` 并返回结果。
- `internal/service/image.go:111-128` 的 `ImageService.Detail()` 先用 `FindActiveByID()` 读取图片，再调用 `views.RecordView(...)` 记录浏览事件。
- `internal/service/image.go:128-132` 随后用一开始读出的 `image` 构造详情响应；因此本次响应里的 `view_count` 是记录浏览前读取到的旧值。
- `internal/service/view_buffer.go:50-68` 的 `RecordView()` 只把浏览事件放入内存缓冲，达到 100 条才立即 flush。
- `internal/service/view_buffer.go:96-112` 会按固定间隔 flush；`internal/conf/conf.go:28` 默认 `VIEW_FLUSH_INTERVAL` 是 `1s`。
- `internal/repository/image.go:161-179` 的 `CreateImageEvents()` flush 后会写入 `image_event` 并执行 `view_count = view_count + count`。

## Current Finding

确认是用户可见体验 bug：详情接口已经记录本次浏览，但响应仍返回记录前的旧 `view_count`，导致当前详情页不会立即显示 +1。

计划按最小后端修复处理：在详情响应构造前为本次返回值补上本次 view 的展示增量，同时保留 `ViewBuffer` 的异步落库机制，避免把每次详情访问变成同步写库。

## Out of Scope

- 不改变去重策略：当前每次详情请求都会计为一次 view，本任务不新增同用户/IP/时间窗口去重。
- 不改热榜预计算逻辑；热榜仍依赖已有异步写库和定时重算。
- 不新增前端乐观计数逻辑，优先让详情接口返回正确展示值。

## Open Questions

无阻塞问题。等待用户批准开始实现。