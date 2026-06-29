# 技术设计

## 边界

代码层加固放在 `internal/conf` 与 `internal/handler/middleware`。路由层只负责装配中间件，不把安全逻辑散落到业务 handler。

## 配置

- `JWT_SECRET`：加载后执行校验。空值、占位符、长度不足 32 字符直接报错。
- `CORS_ALLOW_ORIGIN`：默认空字符串。支持逗号分隔白名单；支持 `*`，但只有显式配置 `*` 才允许任意源。
- `MAX_REQUEST_BODY_BYTES`：默认 1MiB，可用环境变量调整。
- `RATE_LIMIT_RPS`、`RATE_LIMIT_BURST`：全局默认值，用于写接口基础限流。

## 中间件

- `SecurityHeaders()`：所有响应加安全头。
- `RequestBodyLimit(maxBytes)`：请求体超限即拒绝，避免大 body 占用内存。
- `CORS(config)`：按 Origin 精确匹配；无 Origin 的同源或非浏览器请求不额外写跨域头。
- `RateLimit(limiter)`：内存 token bucket，key 以客户端 IP + 路由族为主；登录注册单独挂载。

## 路由装配

- 全局：安全头、请求体限制、CORS。
- 敏感公开写接口：`POST /api/v1/users/register`、`POST /api/v1/users/login` 加限流。
- 鉴权接口保持既有 `Auth` / `RequireAdmin` 规则。

## 取舍

- 内存限流可防单实例刷接口；多实例部署时各实例独立计数，不能替代网关/WAF。
- 收紧 CORS 会要求部署环境显式设置前端 Origin；这是安全默认值，避免无意裸奔。
- JWT 弱密钥启动失败可能暴露旧部署配置问题，但这是必要失败。

## 回滚

若部署受阻，可临时显式设置 `CORS_ALLOW_ORIGIN` 为现有前端域名，并设置足够强的 `JWT_SECRET`。不建议回滚到占位符密钥或默认 `*`。
