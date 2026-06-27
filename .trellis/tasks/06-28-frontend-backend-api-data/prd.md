# Use backend API data in frontend

## Goal

当前 Vue 前端仍有页面使用本地 mock 或 hard-coded 动态数据；本任务要把已有后端接口覆盖的数据改为消费本地 `2018` 端口后端返回的真实 API 数据，让图库、热榜、详情、收藏夹等页面展示与后端数据保持一致。

## Confirmed Facts

- 前端项目为 `frontend/vue-gallery`，技术栈是 TypeScript + Vue 3 + Vite。
- 前端 API 调用约定集中在 `frontend/vue-gallery/src/api/client.ts`，前端代码统一请求 `/api/v1/*`。
- 开发环境应通过 Vite proxy 将 `/api` 转发到 `http://localhost:2018`，前端代码不应硬编码后端端口。
- 后端统一响应格式为 `{ code, data, msg }`。
- 后端图片、搜索、热榜等列表响应约定为 `{ total, page, size, list }`，前端 API client 可归一化为 `{ items, total, page, limit }`。
- `frontend/vue-gallery/vite.config.ts` 已配置 `/api` proxy 到 `http://localhost:2018`。
- 本地后端 `GET /api/v1/images?size=2` 返回 200，响应中 `data.size=2`、`data.list` 含真实图片数据；`GET /api/v1/images?limit=2` 仍按默认 `size=20` 返回，说明前端 `limit` 参数与后端 `size` 参数不一致。
- 本地后端 `GET /api/v1/rankings?period=day&size=3` 返回 200 和 3 条热榜数据；`GET /api/v1/rankings?limit=3` 返回默认 20 条，说明现有 `getRankings(limit)` 参数未被后端消费。
- 后端图片搜索处理器读取查询参数 `q`，现有前端搜索调用使用 `keyword`，需要修正为后端契约。
- 后端标签建议接口读取 `q` 和 `limit`，现有前端 API client 使用 `prefix`，需要修正为后端契约。
- `GET /api/v1/images/:id` 已确认返回详情结构 `{ image, tags, avg_score, rating_count, favorite_count, my_rating, is_collected, similar_images }`。
- `GET /api/v1/collections` 和 `GET /api/v1/users/me` 未登录时返回 401 / `请先登录`；收藏夹真实数据接入必须通过现有登录/注册流程获得 token，或在未登录时显示认证提示。
- 用户已确认评分展示采用后端百分制（`0-100`），例如 `62.3/100`。

## Mock / Placeholder Inventory

- `frontend/vue-gallery/src/pages/CollectionsPage.vue`: `Mock albums` 和 `Mock collection items` 为静态相册/作品数据；创建相册按钮只显示 toast。
- `frontend/vue-gallery/src/pages/TrendingPage.vue`: 热榜行是静态 HTML，period 切换只改变本地状态，不请求 `/api/v1/rankings`。
- `frontend/vue-gallery/src/pages/DetailPage.vue`: 作品标题、标签、评分、相似推荐均为固定示例；保存评分/收藏/下载只显示 toast。
- `frontend/vue-gallery/src/pages/AccountPage.vue`: 认证已接入真实 API；资料和偏好表单标注为 placeholder，后端暂无资料/偏好保存接口。
- `frontend/vue-gallery/src/pages/GalleryPage.vue`: 已调用 `getImages` / `getRankings`，但受 API client 参数契约问题影响。
- `frontend/vue-gallery/src/pages/SearchPage.vue`: 已调用 `searchImages`，但受 `keyword` vs `q` 参数契约问题影响。

## API Client Contract Gaps

- 后端错误消息字段为 `msg`，前端类型和错误读取逻辑需要对齐。
- 图片详情响应是嵌套结构，不是 `ImageItem` 的扁平扩展。
- `TagResponse` 后端字段为 `usage_count`，不是 `category/count`。
- `CollectionResponse` 后端字段为 `user_id/visibility/items`，没有 `description/item_count`。
- `CollectionDetailResponse.items` 后端返回收藏条目 ID，不是完整图片列表。
- `createCollection` 后端 body 为 `{name, visibility}`，不是 `{name, description}`。
- 评分提交后端要求 `0-100` 整数。

## Requirements

- R1. 定位并移除当前前端仍在使用的 mock / hard-coded 动态数据；对后端尚未支持的功能必须显示诚实状态，不能继续伪造成功。
- R2. 对已有后端接口覆盖的数据，前端页面必须通过统一 API client 或 composable 读取真实后端数据。
- R3. 开发环境请求必须保持相对路径 `/api/v1/*`，依赖 Vite proxy 转发到本地 `2018` 后端。
- R4. 页面必须保留现有 loading、empty、error 等可见状态，不因真实接口集成降低用户体验。
- R5. TypeScript 类型导入必须使用 `import type`，不得引入 `as any`、`@ts-ignore` 或弱化类型检查。
- R6. 需要验证真实后端接口返回结构与前端类型/归一化逻辑一致。
- R7. 前端 API client 必须对齐后端实际参数名：分页使用 `size`，搜索关键词使用 `q`，标签建议使用 `q`，热榜支持 `period/page/size`。
- R8. 详情页应优先根据路由中的图片 ID 读取 `GET /api/v1/images/:id`；没有有效 ID 时才显示明确的空/错误状态，而不是固定示例作品。
- R9. 收藏夹页必须移除静态相册/作品列表 mock；未登录时显示需要登录的状态，已登录时读取 `/collections` 并按后端响应渲染。
- R10. 热榜页必须调用 `/rankings` 渲染真实 day/week/month 数据，不再使用静态排名 HTML。

## Acceptance Criteria

- [ ] 前端 mock 数据使用点有清单，包含文件路径、数据用途、替换接口或保留原因。
- [ ] 已有后端接口可支持的数据页面改为调用真实 API，并正确渲染后端返回内容。
- [ ] `getImages`、`searchImages`、`getRankings`、`suggestTags` 等 API client 方法的请求参数与本地后端实际消费的参数一致。
- [ ] API client 的 response/types 对齐后端 DTO，包括 detail、tag、collection、ranking、error envelope。
- [ ] 热榜页按 period 从真实 `/rankings` 数据渲染，并覆盖 loading/error/empty 状态。
- [ ] 详情页可通过真实图片 ID 展示后端详情数据，不再固定展示示例标题、标签、评分和相似图。
- [ ] 收藏夹页不再渲染 mock 相册/作品列表；未登录时呈现 401/登录提示，登录后使用真实收藏夹响应。
- [ ] `/api/v1/*` 开发代理确认指向 `http://localhost:2018`，前端源码不硬编码 `localhost:2018`。
- [ ] API 错误、空数据、加载中状态在涉及页面均可见且不崩溃。
- [ ] TypeScript 检查、相关构建/测试命令通过，或明确记录与本任务无关的既有失败。
- [ ] 使用真实本地后端完成至少一个浏览器级页面验证。

## Out of Scope

- 不新增后端接口，除非前端接入暴露出已存在接口契约与实现不一致的阻塞问题。
- 不重做视觉设计或调整页面风格。
- 不实现新的业务功能；只把已有 mock 数据替换为真实后端数据。
- 不为验证主动注册或修改后端用户/收藏夹数据，除非用户明确同意创建临时测试数据。

## Decisions

- 评分展示与提交统一使用后端百分制（`0-100`）；前端不再显示五分制评分文案。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
