# Design 索引: ACG 二次元图库后端服务

> 技术设计按模块拆分至 `design/` 子目录，避免单文档膨胀。本文件为导航索引。
> 实现执行计划见 `implement.md` -> `implement/`。需求与验收见 `prd.md`。

## 文档地图

| 文档 | 范围 |
|---|---|
| [design/00-foundation.md](design/00-foundation.md) | **地基**：分层架构、技术选型、SQLite 并发（WAL+双池）、数据模型全景、bleve/SQLite 一致性、接口契约(CORS/JWT/分页/时区/校验)、生命周期(优雅关闭/配置/admin引导)、API 总览、回滚 |
| [design/01-user-auth.md](design/01-user-auth.md) | 用户、JWT 认证、RequireAdmin 中间件 |
| [design/02-image-sync.md](design/02-image-sync.md) | COS 集成、cmd/sync 同步（宽高解析、--reindex） |
| [design/03-search.md](design/03-search.md) | bleve CJK 分词 + go-pinyin 全拼/首字母 |
| [design/04-ranking.md](design/04-ranking.md) | 热榜：事件流水、窗口、贝叶斯、预计算缓存 |
| [design/05-tag.md](design/05-tag.md) | 标签全局共享、多对多、suggest、bleve 同步 |
| [design/06-rating.md](design/06-rating.md) | 评分 upsert、冗余聚合 |
| [design/07-collection.md](design/07-collection.md) | 收藏夹私有/公开、去重收藏数 |

## 阅读顺序

先读 `00-foundation.md`（含所有跨模块约定），再按需查阅功能模块。模块文档只写该模块特有逻辑，全局约定一律引用地基，不重复。
