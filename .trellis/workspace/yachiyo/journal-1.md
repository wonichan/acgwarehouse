# Journal - yachiyo (Part 1)

> AI development session journal
> Started: 2026-06-26

---



## Session 1: ACG gallery landing page on 2017 behind nginx

**Date**: 2026-06-26
**Task**: ACG gallery landing page on 2017 behind nginx
**Branch**: `master`

### Summary

Built a self-contained static anime gallery landing page (landing/index.html, styles.css, app.js) served by nginx on 127.0.0.1:2017 and reverse-proxied at https://acgwarehouse.cloud. Replaced the MinIO S3 root proxy and removed /console/. Fixed a Cloudflare Flexible-mode ERR_TOO_MANY_REDIRECTS loop by making port 80 honor X-Forwarded-Proto instead of unconditionally 301->https; user switched Cloudflare to Full mode and public access verified 200. git init'd the repo (first commit 23923b1). Also updated local Go toolchain 1.26.0->1.26.4 earlier in the session.

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `23923b1` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete

---

## Session 2: ACG gallery backend stages 00-01

**Date**: 2026-06-26
**Task**: ACG图库后端服务 (`.trellis/tasks/06-26-acg-gallery-backend`)
**Branch**: `master`

### Summary

Started the backend Trellis task and implemented stages 00/01 only. Stage 00 added the Go module, Hertz web skeleton, env config, zap logger wrapper, unified error codes/response helpers, CORS middleware, SQLite WAL read/write pool setup, `/api/v1/ping`, and graceful shutdown hooks. Stage 01 added user po/do/dto, user repository/service, bcrypt registration/login, local HS256 JWT sign/parse with strict `typ=JWT` + `alg=HS256` header validation, Auth/RequireAdmin middleware, register/login/me routes, admin bootstrap, and unit tests for repository/service/JWT behavior.

### Main Changes

- Created Go backend module `github.com/yachiyo/acgwarehouse` with `cmd/`, `internal/`, and `pkg/` layout.
- Implemented foundation files under `internal/conf`, `internal/handler`, `internal/infra/db`, `pkg/errors`, and `pkg/logger`.
- Implemented authentication files under `internal/model/{do,dto,po}`, `internal/repository`, `internal/service`, `internal/handler`, `internal/handler/middleware`, and `pkg/jwt`.
- Addressed review blocker: `pkg/jwt.Manager.Parse` now decodes and validates JOSE header `typ=JWT` and `alg=HS256`; tests cover invalid alg/type/malformed header.
- Updated stage 00/01 implementation checklist and ran `codegraph sync` after stages.

### Verification

- [OK] `go mod tidy`
- [OK] `go test ./pkg/jwt`
- [OK] `go test ./...`
- [OK] `go build ./...`
- [OK] `go vet ./...`
- [OK] `gofmt -s -l .` produced no output
- [OK] Runtime smoke test on temp DB/port: `/api/v1/users/me` without token returned 401; register returned 200; login returned JWT; `/me` with Bearer token returned 200; admin bootstrap login returned 200; SIGINT graceful shutdown exited 0.
- [INFO] `gopls_go_diagnostics` returned `no views` in this environment.

### Status

[OK] **Stages 00 and 01 completed; stages 02+ not started.**

### Next Steps

- Continue with stage 02 image sync when requested.
- Before commit/finish, run full Trellis check and decide whether to update backend spec for the local JWT implementation / deferred dependency strategy.



## Session 2: ACG gallery backend stages 04-07 complete

**Date**: 2026-06-27
**Task**: ACG gallery backend stages 04-07 complete
**Branch**: `master`

### Summary

Implemented tag, rating, collection, and ranking modules with TDD. Each stage added model/DTO/PO, repository/service/handler/tests, HTTP routes, DB migration, updated code-spec, and recorded task artifact. All Go tests/build/vet/gofmt pass.

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `f49f074` | (see git log) |
| `b6fffe3` | (see git log) |
| `f930458` | (see git log) |
| `cdc237f` | (see git log) |
| `3a6d5a8` | (see git log) |
| `ceaec96` | (see git log) |
| `ffc5c6e` | (see git log) |
| `f4c3eb8` | (see git log) |
| `0756506` | (see git log) |
| `a596f9d` | (see git log) |
| `446b1e3` | (see git log) |
| `d83e6cc` | (see git log) |
| `58ea428` | (see git log) |
| `2cc948e` | (see git log) |
| `ec18b38` | (see git log) |
| `52d36af` | (see git log) |
| `88ac855` | (see git log) |
| `b3cc4ac` | (see git log) |
| `a4fa436` | (see git log) |
| `bbecbd6` | (see git log) |
| `822f363` | (see git log) |
| `bd800ff` | (see git log) |
| `252805a` | (see git log) |
| `73008e2` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 3: Vue.js Gallery SPA Implementation

**Date**: 2026-06-27
**Task**: Vue.js Gallery SPA Implementation
**Branch**: `master`

### Summary

Converted static HTML prototype to Vue 3 + TypeScript SPA. Created 6 pages, 7 composables, 7 components. Configured Vite proxy for backend API. Preserved Colorful design system CSS.

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `3175a13` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 4: 生产环境部署：前端2017+nginx反代、后端2018+HTTPS

**Date**: 2026-06-27
**Task**: 生产环境部署：前端2017+nginx反代、后端2018+HTTPS
**Branch**: `master`

### Summary

完成前端API层开发（从Mock切换到真实API）、后端构建、systemd服务配置、nginx HTTPS反代配置、环境变量配置。部署已验证成功：HTTPS正常、API代理工作、管理员登录成功。

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `d8ac0ac` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 5: 修复生产前端真实数据渲染

**Date**: 2026-06-27
**Task**: 修复生产前端真实数据渲染
**Branch**: `master`

### Summary

修复生产前端接口成功但页面无数据显示的问题：将后端 data.list/size/avg_score 响应归一化为前端 items/limit/avg_score，修复 Gallery/Search 数据映射；为轮播添加空数据保护；将后端图片 url 接入 ArtCard 展示真实缩略图，并完成生产构建与公网接口验证。

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `47524a6` | (see git log) |
| `3627639` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 6: Use backend API data in frontend

**Date**: 2026-06-28
**Task**: Use backend API data in frontend
**Branch**: `master`

### Summary

Replaced Vue gallery mock and hard-coded dynamic data with real backend API responses, documented frontend API contracts, and verified build plus API/proxy smoke checks.

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `414b19f` | (see git log) |
| `80ac5c5` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 7: Gallery community focus carousel redesign

**Date**: 2026-06-28
**Task**: Gallery community focus carousel redesign
**Branch**: `master`

### Summary

Redesigned the Vue gallery home community focus carousel to use weekly ranking data, filter non-displayable ranking image rows, and show a 10-item feature-plus-rail carousel. Documented the frontend ranking display contract and verified build, API smoke, responsive Playwright checks, Trellis check, visual QA, and post-implementation review.

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `042172f` | (see git log) |
| `9170e74` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 8: Fix gallery waterfall pagination

**Date**: 2026-06-28
**Task**: Fix gallery waterfall pagination
**Branch**: `master`

### Summary

Implemented automatic infinite scroll pagination for the Vue gallery waterfall, verified page query behavior in browser, and documented the frontend pagination contract.

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `1e8490c` | (see git log) |
| `29b55cc` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 9: Fix image detail view count display

**Date**: 2026-06-28
**Task**: Fix image detail view count display
**Branch**: `master`

### Summary

Fixed image detail responses so the displayed view_count includes the current detail view while preserving asynchronous ViewBuffer persistence. Added regression coverage and documented the backend API contract.

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `6c10b76` | (see git log) |
| `a36e16a` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 10: Complete login and profile center

**Date**: 2026-06-29
**Task**: Complete login and profile center
**Branch**: `master`

### Summary

Implemented end-to-end login, registration, profile, preference, and password flows across backend and Vue account center; added tests, browser QA, and account API specs.

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `26dd02d` | (see git log) |
| `94cf1e9` | (see git log) |
| `6fa7ea4` | (see git log) |
| `3b29fd1` | (see git log) |
| `c6b1bc1` | (see git log) |
| `8d89679` | (see git log) |
| `7e2180b` | (see git log) |
| `91c3762` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 11: Favorites and tag frontend workflows

**Date**: 2026-06-29
**Task**: Favorites and tag frontend workflows
**Branch**: `master`

### Summary

Implemented real Vue Gallery favorites and tag management workflows, including typed collection/tag API helpers, reusable picker panels, detail and batch mutation flows, collection visibility selection, updated frontend specs, and verified with build, trellis-check, and browser smoke.

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `c806474` | (see git log) |
| `2c2df69` | (see git log) |
| `d3a1ea8` | (see git log) |
| `850de3d` | (see git log) |
| `468114e` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 12: Fix frontend public route 404

**Date**: 2026-06-29
**Task**: Fix frontend public route 404
**Branch**: `master`

### Summary

Added SPA fallback metadata for Vue history routes, documented the fallback and serve -s production requirement, rebuilt and redeployed the frontend service, and verified public subroutes return the SPA while /api/v1/ping remains backend JSON.

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `64448b4` | (see git log) |
| `05ad482` | (see git log) |
| `1842727` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 13: 修复前端快速搜索

**Date**: 2026-06-29
**Task**: 修复前端快速搜索
**Branch**: `master`

### Summary

确认前端搜索能力已存在，修复顶栏快速搜索未接线问题：提交关键词跳转到 /search?q=...，搜索页从路由 query 自动请求 /api/v1/search?q=...&size=20，并将 route-driven search 约定写入前端 API spec。

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `c51901f` | (see git log) |
| `57883c0` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 14: 完成每日推荐

**Date**: 2026-06-30
**Task**: 完成每日推荐
**Branch**: `master`

### Summary

实现全站统一的每日随机推荐：新增后端数据模型、公平随机池、服务和公开 API；前端首页增加图片优先的每日随机推荐区块；补充 backend/frontend code-spec 与 Trellis 规划上下文。验证通过 go test ./...、frontend npm run build、gopls、API smoke 与 Playwright smoke。

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `4035e26` | (see git log) |
| `f4142c3` | (see git log) |
| `c46507c` | (see git log) |
| `5e8bc22` | (see git log) |
| `f31f669` | (see git log) |
| `eaec4ad` | (see git log) |
| `a20354b` | (see git log) |
| `09d02f7` | (see git log) |
| `d323270` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 15: 收藏夹封面与详情页重构

**Date**: 2026-06-30
**Task**: 收藏夹封面与详情页重构
**Branch**: `master`

### Summary

后端新增 cover_image_id/cover_image_url 字段与 CoverImageIDSet 区分省略/清空语义，CollectionItem 嵌套 Image 并 Preload 过滤不可展示图片，CollectionService 注入 cosBase 生成封面 URL fallback。前端新增 /collections/:id 详情页（masonry + ArtCard + 设为封面），重构列表页卡片显示封面、移除 ID/Owner/imageID 标签。通过 trellis-check 两轮复查后拆分 CollectionDetailPage 为 CollectionMasonryGrid + useMasonryLayout 以满足 250 LOC 上限，并修复设计 token 与 focus 可访问性。Playwright E2E 在本地 2017/2018 端口验证通过。

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `96c041a` | (see git log) |
| `f08eae7` | (see git log) |
| `998919b` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 16: 修复详情刷新与图库滚动恢复

**Date**: 2026-07-01
**Task**: 修复详情刷新与图库滚动恢复
**Branch**: `master`

### Summary

修复图片详情页评分/收藏/标签操作后的整页刷新闪烁；使用 KeepAlive 保留图库返回时的滚动位置与已加载瀑布流；补充详情静默刷新用户态字段保留约定，并完成构建、Playwright 浏览器验证与 trellis-check 复核。

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `f242b26` | (see git log) |
| `d87130d` | (see git log) |
| `d6a8e95` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 17: 实现标签管理 CLI

**Date**: 2026-07-01
**Task**: 实现标签管理 CLI
**Branch**: `master`

### Summary

新增 tagctl 命令行工具，支持按名称新增、删除、更新标签；删除复用仓储事务清理 image_tag 关联；新增 DB-only 配置入口并记录本地维护 CLI 配置规范。

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `ee2511e` | (see git log) |
| `a2a26df` | (see git log) |
| `70e95db` | (see git log) |
| `fef5c9f` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 18: 后端运维与接口访问日志落盘

**Date**: 2026-07-01
**Task**: 后端运维与接口访问日志落盘
**Branch**: `master`

### Summary

为 Go 后端接入 gookit/rotatefile，日志按日期切分落盘到 data/log，单文件 100MB 触发轮转并压缩历史文件；应用日志与 /api/v1 接口访问日志分别写入 app.YYYYMMDD.log 与 access.YYYYMMDD.log，接口日志记录 route/method/path/client_ip/user_agent/request_body_bytes/response_body_bytes/duration_ms/status_code，禁记 query/请求体/响应体/Authorization/Cookie；cmd/web 切换到文件日志并补充生命周期日志，同时更新后端日志规范文档。

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `722625a` | (see git log) |
| `93e496e` | (see git log) |
| `56b05dd` | (see git log) |
| `d0135ba` | (see git log) |
| `c42c12c` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete
