# 生产环境部署准备 - 前端2017端口+nginx反代, 后端2018端口, 接口对齐检查, HTTPS配置

## Goal

完成前端（2017端口）、后端（2018端口）的生产环境部署准备，实现nginx反向代理和HTTPS访问（域名：acgwarehouse.cloud）。

## Requirements

- 前端构建产物部署在端口2017，通过nginx反代访问
- 后端Go服务（端口2018）启动于生产环境
- nginx配置HTTPS终止和域名路由
- 前端切换到真实后端API调用（非Mock数据）
- 环境变量配置完整（CORS、JWT、COS、数据库路径等）
- 部署文档完整，包含启动、回滚、健康检查步骤

## Confirmed Facts

- **项目技术栈**：Go (Hertz) 后端 + Vue 3 + Vite 前端
- **后端默认端口**：8080（需改为2018，通过`PORT`环境变量控制）
- **前端开发proxy**：`vite.config.ts` 中配置 `/api` → `http://localhost:2018`
- **后端API路径**：`/api/v1/*`（在 Hertz router中定义）
- **CORS配置**：`CORS_ALLOW_ORIGIN`环境变量控制（默认`*`）
- **部署现状**：仓库中没有nginx配置、systemd服务、Dockerfile等部署文件
- **TLS证书**：已有证书位于 `/etc/ssl/cloudflare/`
  - 证书：`/etc/ssl/cloudflare/cert.pem`
  - 私钥：`/etc/ssl/cloudflare/privkey.pem`
- **管理员账号**：`ADMIN_USERNAME=yachiyo`, `ADMIN_PASSWORD=YACHIYO`

## Requirements Breakdown

### Frontend (port 2017)
- 构建Vue应用并部署静态文件到nginx服务目录
- 实现真实API调用层（替换Mock数据）
- 添加生产环境的CSP和安全头（HSTS、X-Frame-Options等）

### Backend (port 2018)
- 环境变量配置：`PORT=2018`
- 环境变量配置：`CORS_ALLOW_ORIGIN=https://acgwarehouse.cloud`
- 环境变量配置：数据库路径、COS凭证、JWT密钥等
- 初始化SQLite数据库和bleve索引
- 管理员账号引导：`ADMIN_USERNAME=yachiyo`, `ADMIN_PASSWORD=YACHIYO`

### Nginx HTTPS
- 监听`80`端口（HTTP重定向到HTTPS）
- 监听`443`端口（HTTPS终止）
- 使用已有证书：`/etc/ssl/cloudflare/cert.pem` 和 `privkey.pem`
- 路由规则：
  - `/` → 前端静态文件（端口2017）
  - `/api/*` → 后端代理（端口2018）

### API Integration
- 前端实现API调用层（fetch/axios）
- API基本路径统一为 `/api/v1`
- CORS配置允许 `https://acgwarehouse.cloud`

## Acceptance Criteria

- [ ] 前端构建产物能通过`https://acgwarehouse.cloud`访问
- [ ] 后端服务能通过`https://acgwarehouse.cloud/api/v1`访问
- [ ] HTTPS正常工作，无浏览器安全警告
- [ ] 前端能成功调用后端API（登录、图片列表等）
- [ ] 管理员账号`yachiyo`能正常登录
- [ ] 环境变量配置完整，启动无配置错误
- [ ] 提供可执行的部署脚本和运维文档

## Out of Scope

- 数据迁移或现有数据备份
- 日志集中收集和ELK配置
- 监控告警系统配置
- 自动伸缩和容器化部署方案

## Technical Constraints

- 使用已有的SQLite数据库文件路径（需确保文件可读）
- Bleve索引路径需确保生产环境权限正确
- JWT密钥必须强密钥且 securely stored
- 防火墙需开放80、443端口（2018端口仅内网访问）
