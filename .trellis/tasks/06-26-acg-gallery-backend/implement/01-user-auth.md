## 阶段 1 · 用户与认证

- [x] po/do/dto: user。
- [x] repository/user：按 username 查、创建。
- [x] service/user：注册（bcrypt）、登录（校验 + 签 JWT）、取当前用户。
- [x] pkg/jwt：签发/解析 HS256。
- [x] handler/middleware/auth.go：`Auth`（注入 user_id/role）+ `RequireAdmin`。
- [x] handler/user.go + 路由：register/login/me。
- [x] 启动引导：若 env 配置 ADMIN_USERNAME/PASSWORD 且不存在则创建 admin（幂等）。
- 验证：注册→登录拿 token→/me 带 token 200、不带 401（AC-R6）。

- [x] **codegraph sync**：本阶段完成后同步索引（`codegraph sync`）。
