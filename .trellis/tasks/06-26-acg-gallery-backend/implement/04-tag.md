## 阶段 4 · 标签

- [x] po/do/dto: tag、image_tag；repository/tag。
- [x] service/tag：CRUD（删/改限 admin）、批量打/取消标签（同步 bleve tags 字段 + usage_count）、suggest（前缀 + 频次）。
- [x] handler/tag.go + 路由。
- 验证：AC-R2 / AC-R7（非 admin 删改 403）/ AC-R8 已通过：TDD RED 先暴露缺少标签仓储/服务；补充 tag 查询过滤回归测试暴露 join 后 `created_at` 歧义并修复。`go test ./...`、`go build ./...`、`go vet ./...`、`gofmt -s -l .` 均通过。

- [x] **codegraph sync**：本阶段完成后同步索引（`codegraph sync`）。
