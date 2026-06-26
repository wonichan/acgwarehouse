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
