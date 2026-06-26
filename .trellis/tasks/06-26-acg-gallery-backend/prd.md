# PRD: ACG 二次元图库后端服务

## Goal / User Value

为二次元图库提供 Golang 后端 HTTP 服务，支撑前端的图片浏览、条件查询、标签管理、评分、热榜、收藏与用户系统。图片实体存储在腾讯云 COS，后端只管理元数据与业务逻辑。

## Confirmed Facts (from request + repo inspection)

- **技术栈**：Golang + Hertz（HTTP 框架），本地 SQLite 存储，blevesearch/bleve 作为搜索引擎。
- **图片来源**：腾讯云 COS，已上传完毕。
  - 访问域名：`https://acgwarehouse-1301393037.cos.ap-shanghai.myqcloud.com`
  - 存储桶：`acgwarehouse-1301393037`
  - 路径前缀：`/thumbnails`
- **仓库现状**：尚无 `go.mod`，仅有 `landing/` 静态落地页（分类标签：插画/头像/壁纸/角色）。
- **已有 Go 后端 spec**（`.trellis/spec/backend/`、`.trellis/spec/guides/`）：
  - 强制分层架构 `cmd / internal{conf,handler,infra,model(do/dto/po),repository,service} / pkg`。
  - 对象流转：handler 收 dto → 转 do → 调 service；repository/infra 用 po；禁止 po 穿透到 handler。
  - 统一响应体 `Response{Code int, Data interface{}, Msg string}`。
  - RESTful 路由：全小写、中划线分隔、`v1` 版本号、深度 ≤5 层。
  - 注意：现有 layout 示例用 MongoDB，本项目改用 SQLite，spec 需适配。

## Technical Decisions (摘要，细节见 design.md)

- **Module path**：`github.com/yachiyo/acgwarehouse`。**SQLite 驱动**：`modernc.org/sqlite`（纯 Go 免 CGO）+ GORM。
- **并发模型**：WAL + 双连接池（读池 N 并发 / 写池 1 串行）+ busy_timeout；view 高频写走内存缓冲批量 flush。
- **bleve 一致性**：SQLite 为真相源，bleve 尽力同步（失败仅 warn 不回滚），`cmd/sync --reindex` 全量重建兜底。
- **搜索**：bleve v2 + CJK 分析器（必须 blank import cjk）+ go-pinyin 全拼/首字母字段。
- **生命周期**：env 配置 + 默认值；优雅关闭（signal.NotifyContext，按序 flush/close）；首个 admin 经 env 引导幂等创建。
- **接口契约**：CORS（origin 走 env）；JWT 7 天无 refresh；分页 size 默认 20 上限 100；时间 UTC 存储 RFC3339 输出；handler dto binding 校验 + service 业务校验。

## Requirements (converged)

### R0 图片元数据同步（Ingestion）【已确认】
- 实现可重跑的 COS 同步任务（独立 `cmd/sync` 入口，亦可作为后台 job）。
- 通过腾讯云 COS SDK `ListObjects` 遍历 `/thumbnails` 前缀。
- 将每个对象的 key/filename、size、last-modified 等元数据幂等 upsert 进 SQLite `image` 表。
- 同步建立/更新 bleve 索引。
- COS 凭证（SecretId/SecretKey）通过配置注入，初期使用占位符，后续由开发者配置。

### R0.1 图片展示元数据（宽高/分类）【已确认】
- 宽高：`cmd/sync` 同步时用 Go 标准库 `image.DecodeConfig`（只读文件头）解析每张图 `width/height` 入库，schema 预留。
- 分类：`image.category` 字段预留；从 COS 对象 key 路径/文件名规则推断填充。命名规则当前未知（landing 为占位无真实 key），规则未提供时同步留空，后续经标签/管理接口补。

### R1 图片访问与查询【已确认】
- 提供前端图片访问接口（返回元数据 + 完整 COS URL + 宽高，供前端瀑布流）。
- 条件查询：按标签查询、按文件名查询。
- 排序：按时间、文件大小、标签；默认 `created_at desc`。

### R2 标签管理【已确认】
- 标签**全局共享**，不绑定用户：任何登录用户创建的标签进入公共标签库，打在图片上后所有人可见、可查询。
- 标签 CRUD（创建、更新、删除），登录后可操作（后续可加管理员权限收口）。
- 单张/批量给图片打自定义标签；图片与标签为多对多关系。
- 暂不做预置分类标签；前端的固定分类（插画/头像/壁纸/角色）此阶段不映射为标签体系。

### R3 评分系统【已确认】
- 用户给单张图片评分，范围 0-100（整数）。
- 每个用户对每张图只有一条评分记录，主键 `(user_id, image_id)` 唯一；重复评分**覆盖**（upsert）。
- 在 image 表冗余维护 `avg_score`（平均分）与 `rating_count`（评分人数），写时更新，避免聚合扫表。
- 热榜的"评分因子"采用**贝叶斯平均**（加权平均），避免单票高分霸榜。

### R4 热榜排名【已确认】
- 提供日榜 / 周榜 / 月榜。
- **行为事件流水**：建 `image_event` 表（`image_id, user_id, type(view/favorite/rating), value, created_at`），记录带时间戳的行为。
- 榜单窗口：日榜统计最近 24h（或自然日）、周榜 7d、月榜 30d 窗口内的行为。
- 热度公式：`score = w1·贝叶斯评分 + w2·log(1+收藏数) + w3·log(1+点击数)`，权重可配置。
- **定时 job（如每 10 分钟）预计算**各榜单写入 `ranking` 缓存表；查询接口直接读缓存，避免实时扫流水（SQLite 性能关键）。
- 点击数（view）**不去重**：每次访问详情 +1 一条 view 事件；收藏因收藏夹天然每人一次。

### R5 收藏系统【已确认】
- 用户可创建**多个命名收藏夹**（`collection`：`id, user_id, name, visibility, created_at`），收藏夹 CRUD。
- 收藏项关联表 `collection_item`（`collection_id, image_id, created_at`），同一收藏夹内图片唯一。
- 收藏夹支持**私有/公开**可见性（owner 可设置）：私有仅 owner 可见可管理；公开则任何人（含未登录）可浏览，仅 owner 可管理。
- 提供浏览他人公开收藏夹的接口。
- 热榜"收藏数"= 某图片被**去重用户数**收藏（同一用户多夹收藏同图只算一次）；收藏/取消时同步更新 image 表冗余 `favorite_count` 并发 favorite 事件。

### R6 用户管理系统【已确认】
- 多用户 + JWT 认证的轻量方案。
- 用户表：`username / password_hash(bcrypt) / role / created_at`。
- 接口：注册、登录（返回 JWT）、获取当前用户信息。
- 中间件校验 JWT，将 `user_id` 与 `role` 注入 context。
- 权限边界：浏览/查询/看图**公开**（无需登录）；评分/收藏/打标签**需登录**；标签删除/改、删图（软删除）等受控操作**需管理员**。

### R7 管理员角色（F）【已确认】
- `user.role` 区分 `user` / `admin`。
- 管理员专属：标签删除/更新、图片软删除/恢复、（后续）封禁评分等受控操作。
- 鉴权中间件支持 `RequireAdmin`。

### R8 标签自动补全与热门标签（A）【已确认】
- 打标签 / 搜索时返回标签建议，基于已有标签 + 使用频次（`tag.usage_count` 冗余或实时统计）。
- 复用 bleve 前缀/模糊匹配能力。

### R9 相关推荐（B）【已确认】
- 图片详情提供"相似图片"，基于共享标签的简单推荐（共享标签数排序）。

### R10 统一分页（C）【已确认】
- 所有列表接口统一分页口径：`page`（从 1 起）+ `size`，响应返回 `total / page / size / list`。

### R11 图片详情聚合接口（D）【已确认】
- 单次返回：图片元数据 + 标签列表 + 平均分/评分人数 + 当前用户的评分（登录态）+ 是否已收藏（登录态）。
- 访问详情触发一条 view 事件（不去重）。

### R12 全文搜索增强（E）【已确认】
- bleve 索引文件名 + 标签，启用 CJK 分析器支持中文分词与模糊匹配。
- **拼音搜索**：索引时用拼音转换库（候选 `mozillazg/go-pinyin`）生成拼音字段，支持拼音/首字母检索。具体落地方式在 design 阶段调研确认。

### R13 图片软删除/隐藏（G）【已确认】
- image 表增加软删除标记（`deleted_at` 或 `status`）。
- 管理员可软删除/恢复；软删除后前端查询、搜索、热榜均不展示；COS 实体不动。

## Acceptance Criteria

> 以下为可测试验收标准。`v1` API 前缀统一为 `/api/v1`，响应体统一 `Response{Code,Data,Msg}`，列表统一分页 `{total,page,size,list}`。

### AC-R0 元数据同步
- [ ] 运行 `cmd/sync` 后，`/thumbnails` 下每个 COS 对象在 `image` 表存在一条记录（key、filename、size、last_modified）。
- [ ] 重复运行 `cmd/sync` 不产生重复记录（按 COS key 幂等 upsert），且不报错。
- [ ] 同步后 bleve 索引文档数与 `image` 表有效记录数一致。
- [ ] COS 凭证缺失/占位符时给出明确报错日志，不静默失败。

### AC-R0.1 展示元数据
- [ ] 同步后 `image` 表含 `width/height`（由 `image.DecodeConfig` 解析）。
- [ ] `category` 字段存在；无命名规则时为空不报错。

### AC-R1 图片查询与访问
- [ ] `GET /api/v1/images` 支持按 `tag`、`filename` 过滤，返回含 COS 完整 URL 的元数据列表。
- [ ] 支持 `sort` 参数按 `created_at`/`size`/`tag` 排序，`order=asc|desc`。
- [ ] 列表分页生效：`page`、`size` 改变返回数据与 `total` 正确。

### AC-R2 标签管理
- [ ] 登录用户可创建/更新/删除标签（删除/更新受管理员限制，见 AC-R7）。
- [ ] 可对单张或批量图片打标签/取消标签，图片-标签为多对多。
- [ ] 标签全局共享：A 用户创建并打的标签，B 用户查询可见可用。

### AC-R3 评分
- [ ] 登录用户对图片评分 0-100，越界返回 400。
- [ ] 同一用户对同图二次评分为覆盖，`rating` 表 `(user_id,image_id)` 唯一。
- [ ] 评分后 image 表 `avg_score`、`rating_count` 正确更新。

### AC-R4 热榜
- [ ] `GET /api/v1/rankings?period=day|week|month` 返回对应窗口榜单。
- [ ] 行为产生 `image_event` 流水（view/favorite/rating 带时间戳）。
- [ ] 定时 job 预计算后写入 `ranking` 缓存表，查询读缓存。
- [ ] 软删除图片不出现在任何榜单。

### AC-R5 收藏
- [ ] 登录用户可 CRUD 多个命名收藏夹，可设 `visibility=private|public`。
- [ ] 同一收藏夹内同图唯一；收藏/取消同步 `favorite_count`（去重用户数）。
- [ ] 公开收藏夹任何人可浏览，私有仅 owner 可见；非 owner 管理返回 403。

### AC-R6 用户/认证
- [ ] 注册（密码 bcrypt）、登录返回 JWT、获取当前用户信息。
- [ ] 受保护接口缺失/非法 JWT 返回 401。
- [ ] 浏览/查询/看图接口无需登录可访问。

### AC-R7 管理员
- [ ] 非管理员调用标签删除/更新、图片软删除返回 403。
- [ ] 管理员可软删除/恢复图片，软删除图片不出现在查询/搜索/热榜。

### AC-R8 标签补全/热门
- [ ] `GET /api/v1/tags/suggest?q=` 按前缀返回标签建议，按使用频次排序。

### AC-R9 相关推荐
- [ ] 图片详情返回基于共享标签的相似图片列表（按共享标签数排序）。

### AC-R11 详情聚合
- [ ] `GET /api/v1/images/:id` 单次返回元数据+标签+均分+评分人数+（登录态）我的评分+是否已收藏，并记录一条 view 事件。

### AC-R12 搜索增强
- [ ] 中文文件名/标签可被分词检索（CJK 分析器生效）。
- [ ] 拼音/首字母可检索到对应中文内容。

### AC-R13 软删除
- [ ] 软删除后 image 记录保留但带删除标记，所有公开列表/搜索/热榜均不展示。

### AC-契约
- [ ] 跨域请求带正确 CORS 响应头。
- [ ] `size>100` 被截断为 100；缺省 page=1/size=20。
- [ ] 时间字段输出为 RFC3339（带时区）。
- [ ] 优雅关闭：SIGTERM 后 view 缓冲落盘、索引正常 close。

### AC-全局
- [ ] 项目符合 `.trellis/spec/backend` 分层与对象流转约定（dto/do/po 不穿透）。
- [ ] 核心 service 单测（评分聚合/贝叶斯/收藏去重/权限）通过。
- [ ] `go build ./...` 与 `go vet ./...` 通过；`gofmt -s` 无 diff。

## Out of Scope (v2+)
- 评论/弹幕系统、用户关注/动态流。
- 多语言、图片 EXIF 深度解析、AI 自动打标签。
- CDN 防盗链签名 URL、图片上传接口（图片已手工上传至 COS）。
- 第三方 OAuth 登录、公开收藏夹的分享链接/社交功能。
- JWT refresh token（MVP 用 7 天 access token）。
- 接口限流/防刷（注册/登录预留中间件挂载点）。
- Swagger/OpenAPI 生成器（用 README 接口表替代）。
- 原图目录关联（当前 `/thumbnails` 即展示图，`image.cos_key` 存 thumbnail key；将来有原图目录再加字段）。
