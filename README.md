# ACGWarehouse

ACGWarehouse 是一套图库仓库：后端管用户、图片、标签、评分、收藏、热榜与 COS 同步；前端供浏览、搜索、详情、收藏与账户操作。

前后分明，接口以 `/api/v1` 为界。

## 技术栈

- 后端：Go、Hertz、GORM、SQLite WAL、Bleve、Tencent COS、JWT、Zap
- 前端：Vue 3、TypeScript、Vite、Vue Router
- 存储：默认 `data/acgwarehouse.db`，搜索索引默认 `data/bleve`

## 目录

```text
cmd/web                  HTTP 服务入口
cmd/sync                 COS 同步与 Bleve 重建入口
internal/conf            环境变量配置
internal/handler         HTTP handler 与路由
internal/service         业务服务
internal/repository      SQLite 持久化
internal/infra           DB、搜索、COS 等基础设施
pkg                      可复用公共包
frontend/vue-gallery     Vue 前端
tests                    集成与压测脚本
```

## 后端启动

须有 Go，版本以 `go.mod` 为准。

PowerShell：

```powershell
$env:PORT="8080"
$env:JWT_SECRET="dev-secret-change-me"
$env:ADMIN_USERNAME="admin"
$env:ADMIN_PASSWORD="AdminPass123"
go run ./cmd/web
```

验之：

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/ping
```

首次启动会自动创建 SQLite 表与索引目录。

## 后端配置

未设则取默认值。生产勿用占位密钥。

| 变量 | 默认值 | 用途 |
| --- | --- | --- |
| `PORT` | `8080` | HTTP 端口 |
| `SQLITE_PATH` | `data/acgwarehouse.db` | SQLite 文件 |
| `SQLITE_BUSY_TIMEOUT_MS` | `5000` | SQLite busy timeout |
| `SQLITE_READ_MAX_OPEN_CONNS` | `CPU * 4` | 读连接池大小 |
| `BLEVE_PATH` | `data/bleve` | 搜索索引目录 |
| `COS_SECRET_ID` | 占位值 | COS SecretId |
| `COS_SECRET_KEY` | 占位值 | COS SecretKey |
| `COS_BUCKET` | `acgwarehouse-1301393037` | COS bucket |
| `COS_REGION` | `ap-shanghai` | COS region |
| `COS_DOMAIN` | 项目 COS 域名 | 图片访问域名 |
| `COS_PREFIX` | `/thumbnails` | COS 扫描前缀 |
| `JWT_SECRET` | 占位值 | JWT 签名密钥 |
| `JWT_DURATION` | `168h` | 登录有效期 |
| `ADMIN_USERNAME` | 空 | 首个管理员用户名 |
| `ADMIN_PASSWORD` | 空 | 首个管理员密码 |
| `CORS_ALLOW_ORIGIN` | `*` | CORS 来源 |
| `LOG_LEVEL` | `info` | 日志级别 |
| `RANKING_RECOMPUTE_INTERVAL` | `10m` | 热榜重算间隔 |
| `VIEW_FLUSH_INTERVAL` | `1s` | 浏览量缓冲刷新间隔 |

## 同步与索引

从 COS 拉取图片元数据并写入 SQLite、Bleve：

```powershell
$env:COS_SECRET_ID="your-secret-id"
$env:COS_SECRET_KEY="your-secret-key"
go run ./cmd/sync
```

仅按 SQLite 重建 Bleve 索引：

```powershell
go run ./cmd/sync -reindex
```

## API 概览

统一前缀：`/api/v1`。

| 资源 | 路径 |
| --- | --- |
| 健康检查 | `GET /ping` |
| 用户 | `POST /users/register`、`POST /users/login`、`GET/PUT /users/me`、`PUT /users/password` |
| 图片 | `GET /images`、`GET /images/:id`、`GET /search` |
| 标签 | `GET /tags`、`GET /tags/suggest`、`POST /tags`、`PUT/DELETE /tags/:id` |
| 图片标签 | `POST /images/tags`、`DELETE /images/tags` |
| 评分 | `PUT /images/:id/rating` |
| 收藏夹 | `GET/POST /collections`、`GET/PUT/DELETE /collections/:id`、`POST /collections/:id/items` |
| 热榜 | `GET /rankings` |

认证接口使用 `Authorization: Bearer <token>`。

## 前端启动

```powershell
cd frontend/vue-gallery
npm install
npm run dev
```

构建：

```powershell
npm run build
npm run preview
```

前端请求固定走 `/api/v1`。开发代理在 `frontend/vue-gallery/vite.config.ts` 的 `server.proxy` 中配置；若要连本地后端，把 target 改为 `http://localhost:8080`。

Vue Router 使用 history mode。静态部署须保留 `frontend/vue-gallery/public/_redirects`，使前端路由回落到 `index.html`，且 `/api/*` 不被前端吞掉。

## 验证

后端单元测试：

```powershell
go test ./...
```

前端类型检查与构建：

```powershell
cd frontend/vue-gallery
npm run build
```

集成脚本在 `tests/integration_test.sh`。脚本内含 Linux 路径与 bash 语法，Windows 下宜在 WSL 或相应部署环境运行。
