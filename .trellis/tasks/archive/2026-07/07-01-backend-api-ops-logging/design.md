# Backend operations and API logging design

## Architecture

继续以 `pkg/logger` 作为唯一日志入口。底层仍使用 Zap，但输出从 stdout/stderr 改为两个 `gookit/rotatefile` 文件 writer：应用日志 writer 与接口访问日志 writer。

新增访问日志中间件只挂载到 `/api/v1` 路由组，避免非 API 请求进入 access 日志。中间件在 `ctx.Next(c)` 前记录开始时间，在后置阶段读取 Hertz RequestContext 和 Response 字段，并调用专用 access logger 写入 `access.YYYYMMDD.log`。

## Boundaries

- `pkg/logger` 负责：Zap 初始化、文件 writer 创建、全局 app logger 替换、access logger 输出函数、测试隔离辅助能力。
- `internal/handler/middleware` 负责：从 Hertz 请求上下文计算 access log 字段，不关心文件路径和 Zap core 细节。
- `internal/handler/router` 负责：将 access log 中间件挂到 `/api/v1` group。
- `cmd/web` 负责：在服务启动阶段调用日志初始化，并补充生命周期/后台任务状态日志。

## Logger output design

- app 日志文件：`data/log/app.YYYYMMDD.log`。
- access 日志文件：`data/log/access.YYYYMMDD.log`。
- 输出目标：仅文件，不写 stdout/stderr。
- 轮转：`gookit/rotatefile` `ModeCreate`，每日创建日期文件。
- 大小：单文件最大 100MB。
- 压缩：历史轮转文件启用压缩。
- 目录：启动时自动创建 `data/log`。

`logger.Info/Warn/Error` 仍写 app 日志。新增 `logger.Access(ctx, fields...)` 或等价专用函数写 access 日志，避免 access 日志混入 app 日志。

## Access log contract

每条 `/api/v1` 请求完成后输出一条 Info access 日志，字段固定为：

- `route`：`ctx.FullPath()`；为空时降级为 `path`。
- `method`：`string(ctx.Method())`。
- `path`：`string(ctx.Path())`，不包含 query。
- `client_ip`：`ctx.ClientIP()`。
- `user_agent`：`string(ctx.UserAgent())` 或等价 User-Agent header。
- `request_body_bytes`：`ctx.Request.Header.ContentLength()`，负值归零。
- `response_body_bytes`：`len(ctx.Response.BodyBytes())`。
- `duration_ms`：`time.Since(start).Milliseconds()`。
- `status_code`：`ctx.Response.StatusCode()`。

禁止记录 query、请求体原文、响应体原文、Authorization、Cookie、密码、token。

## Operations log contract

补齐服务生命周期和后台任务状态日志，不新增业务审计日志。建议覆盖：

- logger 初始化完成。
- 配置加载完成后记录非敏感配置摘要，如监听地址、log level、log dir。
- SQLite 打开成功。
- Search index 打开成功。
- RankingJob 启动/停止。
- ViewBuffer 启动/flush 完成。
- graceful shutdown 开始/完成。
- 资源关闭失败仍通过现有 error/warn 路径记录。

敏感配置如 JWT secret、COS secret、token、密码不得进入日志。

## Compatibility

- API 路由、响应体、状态码、认证、限流、CORS 不变。
- `LOG_LEVEL` 仍控制 app/access logger 的最低级别；不新增 access log 开关。
- `LOG_LEVEL=warn` 会自然关闭 Info 级 access 日志。
- 不引入 hertz-contrib accesslog；自定义中间件更容易满足安全字段和文件分流要求。

## Rollback

若文件日志初始化失败，服务启动应返回错误而不是静默退回 stdout，避免与“不保留 stdout”要求冲突。回滚时删除 logger 文件输出初始化、access middleware 注册、运维日志补充和新增依赖即可恢复原 stdout 行为。
