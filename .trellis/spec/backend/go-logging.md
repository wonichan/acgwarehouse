# Logging Guidelines

> How logging is done in this project.

---

## Overview

根据场景严格调用封装库（新站：`git.ppdaicorp.com/intl/intl-common-go/log`）

---

## Log Levels

 info

 warn

 error

---

## Structured Logging

```go
func Info(ctx context.Context, msg string, fields ...zap.Field)
func Infof(ctx context.Context, format string, args ...interface{})
func Warn(ctx context.Context, msg string, fields ...zap.Field)
func Warnf(ctx context.Context, format string, args ...interface{})
func Error(ctx context.Context, msg string, fields ...zap.Field)
func Errorf(ctx context.Context, format string, args ...interface{})
```

日志方法必须传递三个参数：`ctx` (上下文), `message` (简要描述), `field` (通过 zap 包装的结构化键值对)。
 ```go
log.Info(ctx, "request param", zap.Any("RiskFlowRequest", req))
 ```

---

## What to Log

- `log.Info` / `log.Infof`：记录业务关键节点（如出入参、DB状态、请求响应），用于业务进度追踪。
- `log.Warn` / `log.Warnf`：记录不影响业务主流程的偶发性异常问题。
- `log.Error` / `log.Errorf`：记录破坏了业务主流程、需要关注的错误。**注意：触发此类日志会直接导致企业微信自动告警！请谨慎调用。**
记录 Field 时，**必须包含标识性变量**（如 `userId`、`flowId`），以便于分布式全链路追踪。

---

## What NOT to Log

**硬性禁止**：
- 代码中**绝对禁止出现 `fmt.Println`** 或原生的 `log.Print`。
- 日志中**绝对禁止记录敏感信息**（如密码、身份证号、手机号等）。如果确实需要记录敏感信息，必须进行脱敏处理。
- 日志中**绝对禁止记录过于冗长的信息**（如大型 JSON、完整的请求/响应体等）。如果确实需要记录此类信息，必须进行适当的截断或摘要处理。

---

## 场景：Web 服务落盘日志与接口访问日志

### 1. Scope / Trigger
- 触发：`cmd/web` HTTP 服务需要将结构化日志输出到文件，并按类型拆分应用日志与 `/api/v1` 接口访问日志。
- 目标：应用日志和接口访问日志分别落盘、按日期切分、按大小轮转并压缩历史文件；不再写 stdout/stderr。

### 2. Signatures
- 配置：`logger.FileConfig{Level, LogDir}`；`LogDir` 未设置时默认 `data/log`。
- 构造：`logger.NewFiles(cfg logger.FileConfig) (logger.Loggers, error)`。
- 注入：`logger.ReplaceGlobals(loggers logger.Loggers) func()`。
- 应用日志：`logger.Info/Warn/Error(ctx, msg, zap.Field...)` 写入 `data/log/app.YYYYMMDD.log`。
- 接口日志：`logger.Access(ctx, zap.Field...)` 写入 `data/log/access.YYYYMMDD.log`，级别固定 `Info`，消息固定 `api access`。
- 中间件：`middleware.AccessLog()` 只挂载到 `/api/v1` 路由组。

### 3. Contracts
- 日志目录：`data/log`；启动时用 `os.MkdirAll(logDir, 0o755)` 自动创建。
- 轮转配置：`gookit/rotatefile` `ModeCreate` + `EveryDay` + `MaxSize=100*OneMByte` + `Compress=true`。
- 文件命名：`app.YYYYMMDD.log`、`access.YYYYMMDD.log`；不采用带连字符的日期格式。
- 输出目标：不写 stdout/stderr；`cmd/web` 必须使用 `logger.NewFiles + logger.ReplaceGlobals` 而不是旧的 `logger.New`。
- 接口日志字段：`route`、`method`、`path`、`client_ip`、`user_agent`、`request_body_bytes`、`response_body_bytes`、`duration_ms`、`status_code`。
- 禁记字段：`query`、请求体原文、响应体原文、`Authorization`、`Cookie`、密码、token。
- 日志级别：接口访问日志统一使用 `Info`，不因 HTTP 4xx/5xx 自动升级为 `Warn`/`Error`。
- 运维日志范围：配置加载后服务启动信息、SQLite/Search 初始化、RankingJob/ViewBuffer 启动、优雅关闭开始/完成；不新增登录、收藏、评分等业务审计日志。

### 4. Validation & Error Matrix
- `LogDir` 无权限或磁盘不可写 -> `logger.NewFiles` 返回带堆栈错误；`cmd/web` `run` 直接失败退出，不静默回落 stdout。
- `Level` 非法 -> `logger.NewFiles` 返回带堆栈错误。
- `Content-Length` < 0（chunked/unknown）-> `request_body_bytes` 归零后再写入日志。
- `ctx.FullPath()` 为空 -> `route` 降级为 `string(ctx.Path())`。

### 5. Good/Base/Bad Cases
- Good：`GET /api/v1/images/42` 完成后产生一条 access 日志，`route=/api/v1/images/:id`、`path=/api/v1/images/42`、`status_code=200`、`response_body_bytes` 为实际响应字节数。
- Good：非 `/api/v1` 请求（如 `/health`）不产生任何 access 日志。
- Base：`GET /api/v1/ping` 无 body 时 `request_body_bytes=0`；`Access-Log` 依然写入 access 文件。
- Bad：将 access 日志字段（如 `route`、`status_code`）写入应用 logger，导致 app.log 与 access.log 混流。
- Bad：将查询串、Authorization、Cookie、请求体或响应体原文写入 access 日志。
- Bad：`cmd/web` 保留 `logger.New` stdout 输出，或用文件轮转失败后回退 stdout。

### 6. Tests Required
- `pkg/logger`：`TestFileLoggersWriteAppAndAccessToSeparateDateFiles` 验证 app/access 文件分流；`TestFileLoggersCreateLogDirectory` 验证目录自动创建。
- `internal/handler/middleware`：`TestAccessLogWritesRequiredFields_whenAPIRequestCompletes` 验证必需字段与敏感字段排除；`TestAccessLogNormalizesUnknownRequestBodySize_whenContentLengthMissing` 验证 `request_body_bytes` 归零。
- `internal/handler/router`：`TestRegisterWithOptionsWritesAccessLogForAPIV1Route_whenPingCompletes`、`TestRegisterWithOptionsSkipsAccessLogForNonAPIRoute_whenOutsideGroupCompletes` 验证 `/api/v1` 挂载范围。

### 7. Wrong vs Correct

#### Wrong
```go
zapLogger, err := logger.New(cfg.Log.Level) // 仍写 stdout/stderr
if err != nil {
    return err
}
logger.ReplaceGlobal(zapLogger)
```

#### Correct
```go
loggers, err := logger.NewFiles(logger.FileConfig{Level: cfg.Log.Level})
if err != nil {
    return pkgerrors.WithMessage(err, "create logger")
}
logger.ReplaceGlobals(loggers)
```

#### Wrong
```go
// 将 access 字段写入应用 logger，会混入 app.log。
logger.Info(c, "api access",
    zap.String("route", ctx.FullPath()),
    zap.Int("status_code", ctx.Response.StatusCode()),
)
```

#### Correct
```go
logger.Access(c,
    zap.String("route", ctx.FullPath()),
    zap.Int("status_code", ctx.Response.StatusCode()),
)
```

#### Wrong
```go
// 记录完整 query 与 Authorization，泄露敏感信息。
logger.Access(c,
    zap.String("query", string(ctx.Request.QueryString())),
    zap.String("authorization", string(ctx.GetHeader("Authorization"))),
)
```

#### Correct
```go
logger.Access(c,
    zap.String("route", route),
    zap.String("path", string(ctx.Path())),
    zap.String("client_ip", ctx.ClientIP()),
    zap.String("user_agent", string(ctx.UserAgent())),
)
```

