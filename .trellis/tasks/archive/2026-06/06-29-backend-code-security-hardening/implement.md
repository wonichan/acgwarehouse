# 实施计划

## 步骤

1. [x] 扩展 `internal/conf`：增加安全配置、CORS 白名单、JWT 密钥校验、请求体与限流默认值。
2. [x] 改造 `internal/handler/middleware/cors.go`：支持白名单匹配与安全默认。
3. [x] 新增安全头中间件。
4. [x] 新增请求体大小限制中间件。
5. [x] 新增轻量内存限流中间件。
6. [x] 在 `internal/handler/router/router.go` 装配全局安全中间件，并给登录/注册挂载限流。
7. [x] 增加配置与中间件测试。
8. [x] 运行目标测试。

## 验证命令

```powershell
go test ./internal/conf ./internal/handler/middleware ./internal/handler/router
```

## 风险点

- Hertz 请求体 API 需要以当前版本为准，避免只测通过却运行时不生效。
- CORS 默认收紧会影响本地前端调试，需要在 `.env` 中配置本地 Origin。
- 限流测试要避免时间敏感导致偶发失败。
