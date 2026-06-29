# Go 安全配置与 HTTP 防护规范

## Scenario: 代码层公网防护

### 1. Scope / Trigger
- Trigger: 后端服务可被公网访问，且安全能力由 Go/Hertz 代码承担一部分时。
- Scope: `internal/conf`、`internal/handler/middleware`、`internal/handler/router`。
- Out: 云安全组、Nginx、TLS 证书、WAF、SSH 防护不在代码层 spec 内。

### 2. Signatures
- Config:
  - `JWT_SECRET`: 必填，至少 32 字符，不得为空或占位符。
  - `CORS_ALLOW_ORIGIN`: 可选，逗号分隔 Origin 白名单；空值表示不写跨域放行头。
  - `MAX_REQUEST_BODY_BYTES`: 可选，默认 `1048576`。
  - `RATE_LIMIT_RPS`: 可选，默认 `2`。
  - `RATE_LIMIT_BURST`: 可选，默认 `5`。
- Middleware:
  - `middleware.CORS(origins ...string)`
  - `middleware.SecurityHeaders()`
  - `middleware.RequestBodyLimit(maxBytes int)`
  - `middleware.NewRateLimiter(rate float64, burst int)`
  - `middleware.RateLimit(limiter *RateLimiter)`
- Router:
  - `router.New(cfg, services)` 必须装配安全头、body 限制、CORS、登录注册限流。
  - `router.Register(...)` 保持测试兼容；需要限流时用 `RegisterWithOptions(...)`。

### 3. Contracts
- JWT 弱密钥是启动错误，不得降级为 warn。
- CORS 默认安全：未配置白名单时，不得返回 `Access-Control-Allow-Origin: *`。
- 只有显式配置 `*` 才允许任意 Origin；普通部署应配置真实前端域名。
- 安全头至少包含：
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY`
  - `Referrer-Policy: no-referrer`
  - `Content-Security-Policy`
- 超过 body 上限返回 HTTP 413。
- 登录、注册高频请求返回 HTTP 429。
- 限流是单实例内存防护，不可宣称替代网关/WAF。

### 4. Validation & Error Matrix
- `JWT_SECRET` 为空、占位符或少于 32 字符 -> 配置加载失败。
- `MAX_REQUEST_BODY_BYTES <= 0` -> 配置加载失败。
- `RATE_LIMIT_RPS <= 0` 或 `RATE_LIMIT_BURST <= 0` -> 配置加载失败。
- Origin 不在白名单 -> 不写 CORS 放行头。
- 请求体超过上限 -> HTTP 413 / Code 40001。
- 限流桶耗尽 -> HTTP 429 / Code 40001。

### 5. Good/Base/Bad Cases
- Good: 生产环境设置强 `JWT_SECRET` 与单个真实 `CORS_ALLOW_ORIGIN`。
- Good: 多前端域名用逗号分隔白名单。
- Base: 无 `Origin` 的服务端请求不写 CORS 头，但业务仍可正常处理。
- Base: 测试中使用 `Register(...)` 保持旧路由行为；安全选项测试用 `RegisterWithOptions(...)`。
- Bad: 默认 CORS 为 `*`，导致公网服务无意放行任意浏览器来源。
- Bad: `JWT_SECRET_PLACEHOLDER` 可启动服务。
- Bad: 只在云侧限流，却不对登录/注册保留代码层兜底。

### 6. Tests Required
- Config: 默认 JWT 占位符会失败；强 JWT 密钥可加载；非法 body/限流数值会失败。
- Middleware: 空 CORS 白名单不返回放行头；命中 Origin 返回 CORS 头；OPTIONS 返回 204。
- Middleware: 安全头存在；body 超限返回 413；限流超出 burst 返回 429。
- Router: 登录/注册在 `RegisterWithOptions` 下触发 429。

### 7. Wrong vs Correct

#### Wrong
```go
const defaultCORSAllowOrigin = "*"
const defaultJWTSecret = "JWT_SECRET_PLACEHOLDER"
```

#### Correct
```go
const defaultCORSAllowOrigin = ""
const minJWTSecretLength = 32

if secret == "" || secret == defaultJWTSecret || len(secret) < minJWTSecretLength {
	return errors.New("jwt secret must be configured with at least 32 characters")
}
```

#### Wrong
```go
engine.Use(middleware.CORS("*"))
```

#### Correct
```go
engine.Use(middleware.SecurityHeaders())
engine.Use(middleware.RequestBodyLimit(cfg.Security.MaxRequestBodyBytes))
engine.Use(middleware.CORS(cfg.CORS.AllowOrigins...))
```
