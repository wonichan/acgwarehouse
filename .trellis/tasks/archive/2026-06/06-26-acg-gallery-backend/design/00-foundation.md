# 设计 · 地基（Foundation）

> 全局架构、技术选型、数据模型全景、bleve一致性、生命周期、跨模块契约、API 总览。
> 各功能模块详细设计见 `design/0X-*.md`。

## 1. 架构总览

单体 Go 服务，遵循 `.trellis/spec/backend` 分层。原 spec 示例的 MongoDB 适配为 **SQLite**，公司内部 log 包适配为**本地 logger 封装**。

```
acgwarehouse/
├── cmd/
│   ├── web/main.go               # HTTP 服务入口
│   └── sync/main.go              # COS -> SQLite + bleve 同步任务（可重跑）
├── internal/
│   ├── conf/                     # 配置（COS 凭证占位/DB/bleve/JWT/热榜权重）
│   ├── handler/
│   │   ├── {user,image,tag,rating,collection,ranking}.go
│   │   ├── common.go             # Response 封装 + 分页解析
│   │   ├── middleware/auth.go    # JWT 校验 + RequireAdmin
│   │   └── router/router.go
│   ├── job/ranking_job.go        # 定时预计算热榜
│   ├── infra/
│   │   ├── db/sqlite.go          # modernc.org/sqlite（纯 Go 免 CGO）+ GORM
│   │   ├── search/bleve.go       # bleve 封装（CJK + 拼音）
│   │   └── client/cos/client.go  # COS SDK 封装
│   ├── model/{do,dto,po}/
│   ├── repository/               # image tag rating collection user event ranking
│   └── service/                  # 接口 + 实现子包
└── pkg/{errors,logger,jwt,pinyin}/
```

**对象流转（强制）**：handler 收 `dto` -> 转 `do` -> 调 `service`；repository/infra 用 `po` 向上返回 `do`；禁止 `po` 穿透 handler。

## 2. 技术选型

| 关注点 | 选型 | 理由 |
|---|---|---|
| SQLite 驱动 | `modernc.org/sqlite` 纯 Go 免 CGO + GORM | 部署简单；GORM 提供 upsert/迁移 |
| ORM | GORM | spec 禁 service 写原生 SQL，repository 解耦 |
| 搜索 | bleve v2 + CJK + 拼音字段 | 见 §6 |
| HTTP | Hertz `server.Default()` | spec 指定 |
| 认证 | JWT(HS256) + bcrypt | 轻量无状态 |
| 错误 | `github.com/pkg/errors` 带堆栈 | spec 强制 |

### SQLite 并发模型（多用户读多写少）

SQLite 在**文件级串行化所有写**：同一时刻只有一个写事务持锁，多开写连接不会并行写，只会互相抢锁抛 `SQLITE_BUSY`/`database is locked`。因此采用 **WAL + 双连接池** 模式：

| 池 | MaxOpenConns | 用途 |
|---|---|---|
| 读池 | N（如 CPU 核数，可调高） | WAL 下多读者并发；浏览/查询/详情/搜索全走读池，随用户数扩展 |
| 写池 | 1 | 评分/收藏/打标签/事件落库走写池，进程内排队串行，避免锁错误 |

要点：
- `PRAGMA journal_mode=WAL`：读者不阻塞写者、写者不阻塞读者，契合图库"读多写少"。
- `PRAGMA busy_timeout=5000`：偶发争锁时等待而非立即报错。
- 写池=1 是把不可避免的串行做干净（进程内排队），不是限流；读完全并发。
- **view 事件高频写优化**：详情访问产生的 view 事件不逐条写事务，改为**内存缓冲 + 批量 flush**（每秒或每 N 条一次事务）。热榜定时预计算容忍秒级延迟，view 计数最终一致可接受。评分/收藏等用户期望实时的低频写直接走写池单条事务。
- 同步任务与热榜 job 的批量写用事务包裹。
- 扩展信号：若非 view 的写并发也成为瓶颈，才考虑迁移 PostgreSQL（超出当前 MVP）。

## 3. 数据模型（po）

```
user            (id PK, username UNIQUE, password_hash, role[user|admin], created_at)
image           (id PK, cos_key UNIQUE, filename, size, last_modified,
                 width INT, height INT, category NULL,
                 avg_score REAL, rating_count INT, favorite_count INT, view_count INT,
                 status[active|deleted], deleted_at NULL, created_at)
tag             (id PK, name UNIQUE, usage_count INT, created_at)
image_tag       (image_id, tag_id, PK(image_id,tag_id))            -- 多对多
rating          (user_id, image_id, score[0..100], updated_at, PK(user_id,image_id))
collection      (id PK, user_id, name, visibility[private|public], created_at)
collection_item (collection_id, image_id, created_at, PK(collection_id,image_id))
image_event     (id PK, image_id, user_id NULL, type[view|favorite|rating], value, created_at)
ranking         (period[day|week|month], image_id, score REAL, rank INT, computed_at,
                 PK(period,image_id))                              -- 预计算缓存
```

`image_event.created_at` 建索引（热榜窗口查询）。`avg_score/rating_count/favorite_count` 在写事务内更新；`view_count` 最终一致可接受。

## 6.1 bleve / SQLite 一致性策略

- **SQLite 为唯一真相源，bleve 为可重建的派生索引**。
- 写路径：先提交 SQLite 事务（真相），再更新 bleve（派生）。
- bleve 更新失败**不回滚 SQLite**，仅记 warn 日志（数据未丢，索引暂时漂移）。
- **兜底**：`cmd/sync --reindex` 从 SQLite 全量重建 bleve 索引，用于对账/灾后恢复。
- 不引入两阶段提交（bleve 无事务语义，本地文件场景属过度设计）。

## 9.0 接口契约约定（B6-B10）

- **CORS**：全局 CORS 中间件，允许 origin 走 env（开发默认 `*`，生产收紧）。
- **JWT**：access token 有效期默认 7 天（env 可调），不做 refresh token，过期重新登录。
- **分页**：`page` 默认 1；`size` 默认 20、上限 100（超出截断）。列表默认排序 `created_at desc`。
- **时间/时区**：DB 存 UTC，API 输出 RFC3339（带时区），热榜窗口按 UTC 计算；前端负责本地化。
- **输入校验**：handler 层 dto + binding tag 做基础校验（必填/长度/范围）；业务规则（唯一/去重）在 service 层。规则：密码≥6 位、用户名 3-32 字符、标签名 1-32 字符且去首尾空格、评分 0-100 整数。

## 9.1 服务生命周期与配置

- **配置（conf）**：环境变量 + 合理默认值，不引入 yaml/viper。凭证类（`COS_SECRET_ID`/`COS_SECRET_KEY`/`JWT_SECRET`）必须走 env；其余（DB 路径、bleve 路径、监听端口、热榜权重/周期、view flush 间隔、JWT 有效期）env 可覆盖默认。
- **优雅关闭**：`cmd/web` 用 `signal.NotifyContext` 监听 SIGINT/SIGTERM；收到信号按序：flush view 缓冲 -> 停止热榜 job -> close bleve -> close db -> 关 HTTP server（Hertz `OnShutdown` 钩子）。保证缓冲不丢、索引文件不损坏。
- **首个管理员引导**：启动时若配置 `ADMIN_USERNAME`/`ADMIN_PASSWORD` 且该用户不存在，则自动创建为 `admin`（幂等，已存在则跳过）。避免手工改库，可文档化。

## 9. API 草案（/api/v1，Response{Code,Data,Msg}，列表 {total,page,size,list}）

```
# 用户
POST   /api/v1/users/register      {username,password}
POST   /api/v1/users/login         -> {token}
GET    /api/v1/users/me            [auth]

# 图片
GET    /api/v1/images              ?tag=&filename=&sort=created_at|size|tag&order=&page=&size=
GET    /api/v1/images/:id          聚合(元数据+标签+均分+我的评分+是否收藏+相似图)，记 view
GET    /api/v1/search              ?q= (中文/拼音/首字母) &page=&size=
DELETE /api/v1/images/:id          [admin] 软删除
POST   /api/v1/images/:id/restore  [admin]

# 标签
GET    /api/v1/tags                列表
GET    /api/v1/tags/suggest        ?q= 前缀建议(按 usage_count)
POST   /api/v1/tags                [auth] 创建
PUT    /api/v1/tags/:id            [admin] 更新
DELETE /api/v1/tags/:id            [admin] 删除
POST   /api/v1/images/tags         [auth] 批量打标签 {image_ids,tag_ids}
DELETE /api/v1/images/tags         [auth] 批量取消标签

# 评分
PUT    /api/v1/images/:id/rating   [auth] {score:0..100} upsert

# 收藏
GET    /api/v1/collections         [auth] 我的收藏夹
POST   /api/v1/collections         [auth] {name,visibility}
PUT    /api/v1/collections/:id     [auth][owner]
DELETE /api/v1/collections/:id     [auth][owner]
GET    /api/v1/collections/:id     公开或 owner 可见
POST   /api/v1/collections/:id/items            [auth][owner] {image_id}
DELETE /api/v1/collections/:id/items/:imageId   [auth][owner]

# 热榜
GET    /api/v1/rankings            ?period=day|week|month&page=&size=
```

## 10. 兼容/回滚/运维

- 全新项目，无迁移负担。SQLite + bleve 均为本地文件，删除即重置；`cmd/sync` 可重建数据与索引。
- 回滚点：每阶段（见 implement.md）可独立 `go build` 验证；地基阶段失败不影响后续设计。
- 风险文件：`infra/db/sqlite.go`（WAL/连接池）、`infra/search/bleve.go`（分析器注册）、`job/ranking_job.go`（贝叶斯权重）。
