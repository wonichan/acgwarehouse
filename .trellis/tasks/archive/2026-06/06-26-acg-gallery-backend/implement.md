# Implement 索引: ACG 二次元图库后端服务

> 执行计划按阶段拆分至 `implement/` 子目录。本文件为导航 + 全局执行约束。
> 技术设计见 `design.md` -> `design/`。

## 执行原则

- **codegraph 工作流**：实现开始前确认索引就绪（已 `codegraph init`，索引存在则跳过；如需重建用 `codegraph index`）。**每完成一个阶段/模块后运行 `codegraph sync`** 更新索引，再进入下一阶段。

- 严格按阶段顺序；每阶段结束必须 `go build ./...` + `go vet ./...` 通过、`gofmt -s` 无 diff。
- 阶段间为回滚点；前阶段未过不进下阶段。
- 每个文件遵守 `.trellis/spec/backend`（分层、对象流转、错误处理、日志、命名）。

## 阶段地图

| 阶段 | 文档 | 交付 |
|---|---|---|
| 0 | [implement/00-foundation.md](implement/00-foundation.md) | go.mod、依赖、conf、logger、errors、sqlite(WAL+双池)、common(分页/CORS)、优雅关闭骨架；**含全局验证命令 + 风险点** |
| 1 | [implement/01-user-auth.md](implement/01-user-auth.md) | user 模型、bcrypt、JWT、Auth/RequireAdmin、admin 引导 |
| 2 | [implement/02-image-sync.md](implement/02-image-sync.md) | COS client、bleve 封装、pinyin、image 模型、cmd/sync(+reindex) |
| 3 | [implement/03-image-query.md](implement/03-image-query.md) | 查询/排序/分页、详情聚合、搜索、软删除、view 缓冲 |
| 4 | [implement/04-tag.md](implement/04-tag.md) | 标签 CRUD、批量打标签、suggest、bleve tags 同步 |
| 5 | [implement/05-rating.md](implement/05-rating.md) | 评分 upsert + 冗余聚合 + 事件 |
| 6 | [implement/06-collection.md](implement/06-collection.md) | 收藏夹 CRUD、可见性、去重收藏数 |
| 7 | [implement/07-ranking.md](implement/07-ranking.md) | 事件/ranking 表、热榜 job、查询接口 |
| 8 | [implement/08-finish.md](implement/08-finish.md) | 单测、Dockerfile、README、spec 更新、全量校验 |
