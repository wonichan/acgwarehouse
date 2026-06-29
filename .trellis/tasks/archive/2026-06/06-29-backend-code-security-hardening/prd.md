# 代码层后端安全加固

## 目标

后端已公网暴露。此任务只做代码层防护，降低被扫、被刷、弱默认配置、跨域误用与大请求压垮的风险。

## 已确认事实

- 后端使用 Go + Hertz，入口为 `cmd/web/main.go`，路由在 `internal/handler/router/router.go`。
- 当前全局中间件只有 `middleware.CORS(cfg.CORS.AllowOrigin)`。
- 已有 JWT 鉴权与管理员校验，但公开接口、登录、注册仍可被高频访问。
- `CORS_ALLOW_ORIGIN` 默认值为 `*`。
- `JWT_SECRET` 默认值为 `JWT_SECRET_PLACEHOLDER`，启动时未拒绝弱密钥。
- 目前未见安全响应头、请求体大小限制、全局/敏感接口限流测试覆盖。

## 需求

- 启动配置必须拒绝弱 JWT 密钥，禁止公网服务继续使用占位符或过短密钥。
- CORS 默认必须收紧；未显式配置时不应对任意 Origin 放行凭证型跨域。
- 增加全局安全响应头，降低浏览器侧基础攻击面。
- 增加请求体大小限制，拒绝异常大 body。
- 增加内存级限流，至少覆盖登录、注册等高风险写接口；不得引入外部服务依赖。
- 保持现有 API 响应格式与业务路由兼容。
- 本任务不处理云安全组、Nginx、TLS 证书、WAF、服务器 SSH、防火墙。

## 验收标准

- [x] 使用默认 `JWT_SECRET_PLACEHOLDER` 启动配置校验失败。
- [x] 合法强 JWT 密钥可正常加载配置。
- [x] 未配置 CORS 时，不再返回 `Access-Control-Allow-Origin: *`。
- [x] 配置允许源时，匹配 Origin 才返回 CORS 头；不匹配时不放行。
- [x] OPTIONS 预检请求仍可正常返回。
- [x] 响应包含基础安全头：`X-Content-Type-Options`、`X-Frame-Options`、`Referrer-Policy`、`Content-Security-Policy`。
- [x] 超过请求体上限时返回 HTTP 413 或等价拒绝响应。
- [x] 登录/注册在短时间高频请求时触发 HTTP 429。
- [x] `go test ./internal/conf ./internal/handler/middleware ./internal/handler/router` 通过。

## 范围外

- 不改云服务安全组与公网端口。
- 不部署 Nginx/HTTPS/WAF。
- 不改数据库结构。
- 不重做用户权限模型。
