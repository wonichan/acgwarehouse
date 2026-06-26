## 阶段 8 · 收尾

- [ ] 核心 service 单测：评分聚合、贝叶斯热榜、收藏去重、权限校验；COS 走接口 mock。
- [ ] 最小 Dockerfile + README（启动、sync/--reindex 用法、env 清单：COS/JWT/ADMIN/端口/权重）。

- [ ] 全量 `go build ./...` + `go vet ./...` + `gofmt -s -l .`（无输出）。
- [ ] 对照 prd.md 全部 AC 逐条核验。
- [ ] 适配/更新 `.trellis/spec/backend/go-project-layout.md`（MongoDB -> SQLite，记录 bleve/COS 约定）via trellis-update-spec。
- [ ] README：启动、sync 用法、env 配置（COS 凭证、JWT secret）。

- [ ] **codegraph sync**：本阶段完成后同步索引（`codegraph sync`）。
