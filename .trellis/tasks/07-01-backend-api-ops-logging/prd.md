# Add backend operations and API logs

## Goal

为 Go 后端补充可用于线上排查和运行观测的结构化文件日志：运维日志覆盖服务生命周期与后台任务状态，接口日志覆盖 `/api/v1` 请求完成事件，并按日志类型拆分落盘、自动轮转和压缩。

## Background

- 后端使用 Go + Hertz + Zap，HTTP 服务入口为 `cmd/web/main.go`。
- 全局日志封装在 `pkg/logger/logger.go`，当前通过 `logger.Info/Warn/Error(ctx, msg, zap.Field...)` 输出 JSON 到 stdout/stderr。
- 日志规范要求结构化日志、传入 `ctx/message/field`，禁止 `fmt.Println` / 原生 `log.Print`，禁止记录敏感信息和完整大请求/响应体。
- 路由集中在 `internal/handler/router/router.go`，API 统一挂载到 `/api/v1`。
- 现有全局中间件包括 `SecurityHeaders`、`RequestBodyLimit`、`CORS`；中间件目录已有测试 `internal/handler/middleware/security_test.go`。
- Hertz 支持在中间件 `ctx.Next(c)` 后读取访问日志字段：`ctx.FullPath()` 可取路由模板，`ctx.Path()` 可取实际路径，`ctx.Request.Header.ContentLength()` 可取请求 Content-Length，`ctx.Response.StatusCode()` 可取状态码，`len(ctx.Response.BodyBytes())` 可取响应体字节数，`time.Since(start)` 可取耗时。
- `go.mod` 已直接依赖 `go.uber.org/zap v1.28.0`，尚未依赖日志轮转库。
- `lumberjack` 活跃文件名固定，不能满足“活跃日志文件名体现日期”。
- `gookit/rotatefile` 支持 `ModeCreate` 按日期创建活跃文件，并支持大小限制与压缩；默认命名形如 `app.20260701.log`。

## Requirements

- 继续使用现有 Zap 结构化日志框架和 `pkg/logger` 包，不替换为新日志框架。
- 新增 `gookit/rotatefile` 作为文件轮转 writer。
- 日志只落盘到项目 `data/log` 目录，不再写 stdout/stderr；目录不存在时自动创建。
- 日志文件拆分为：
  - `data/log/app.YYYYMMDD.log`：运维日志、业务错误日志、现有 `logger.Info/Warn/Error`。
  - `data/log/access.YYYYMMDD.log`：仅 `/api/v1` 接口访问日志。
- 日志文件自动轮转并压缩历史文件，单个日志文件大小不得超过 100MB。
- 运维日志只覆盖服务生命周期和后台任务状态：配置加载后服务启动信息、SQLite/Search 初始化成功、RankingJob/ViewBuffer 启动、优雅关闭开始/完成、资源关闭失败/成功。
- 不新增业务审计日志，不在登录、收藏、评分等业务节点新增审计/埋点日志。
- 接口日志仅记录 `/api/v1` 下的业务 API 请求；`/api/v1/ping` 属于 API，默认记录；非 `/api/v1` 请求不记录 access 日志。
- 接口完成日志统一使用 `Info`，不按 HTTP 4xx/5xx 自动升级为 `Warn` 或 `Error`。
- 不新增单独接口日志开关；接口日志通过现有 `LOG_LEVEL` 控制，`LOG_LEVEL=warn` 时自然压掉 `Info` 级访问流水。
- 接口日志字段名和单位固定为：`route`、`method`、`path`、`client_ip`、`user_agent`、`request_body_bytes`、`response_body_bytes`、`duration_ms`、`status_code`。
- `route` 优先记录注册路由模板，例如 `/api/v1/images/:id`；无法取得模板时允许降级为实际 path。
- `path` 记录实际路径但不带 query。
- 请求体和响应体只记录字节大小，不记录原文。
- 不记录 query、Authorization、Cookie、完整请求体、完整响应体、密码、token 或其他敏感字段。
- 日志实现不得改变现有 API 响应结构、状态码、请求体限制、认证、限流和 CORS 行为。
- 为日志配置和接口日志中间件补充测试，证明必需字段、文件拆分、禁记字段和 `/api/v1` 范围正确。

## Acceptance Criteria

- [ ] `pkg/logger` 继续提供 `Info/Warn/Error` 结构化日志接口，应用日志只写入 `data/log/app.YYYYMMDD.log` 系列文件。
- [ ] 接口日志只写入 `data/log/access.YYYYMMDD.log` 系列文件，不混入 app 日志。
- [ ] 日志不写 stdout/stderr。
- [ ] `data/log` 不存在时运行时自动创建。
- [ ] app/access 日志文件均支持自动轮转、压缩历史文件，单文件大小不超过 100MB。
- [ ] 任意 `/api/v1` 请求完成后产生一条 Info access 日志，字段包含 `route/method/path/client_ip/user_agent/request_body_bytes/response_body_bytes/duration_ms/status_code`。
- [ ] 非 `/api/v1` 请求不产生 access 日志。
- [ ] access 日志不包含 query、请求体原文、响应体原文、Authorization、Cookie、密码或 token。
- [ ] 运维日志覆盖服务生命周期和后台任务状态，不新增业务审计日志。
- [ ] 接口日志不破坏正常响应：已有路由 smoke 测试或相关中间件测试通过。
- [ ] 新增或更新的 Go 测试覆盖日志文件拆分、access 日志核心字段和安全排除字段。
- [ ] `go test ./...` 通过，或明确说明与本改动无关的既有失败。

## Out of Scope

- 接入外部日志平台、APM、Tracing、Prometheus 指标或链路追踪系统。
- 修改前端、API 协议或业务响应体。
- 输出完整请求体/响应体。
- 新增审计日志、用户行为日志、登录/收藏/评分等业务节点日志或业务埋点体系。
- 自定义精确 `app-YYYY-MM-DD.log` / `access-YYYY-MM-DD.log` 文件名；本任务接受 `gookit/rotatefile` 默认 `app.YYYYMMDD.log` / `access.YYYYMMDD.log` 命名。

## Open Questions

无。
