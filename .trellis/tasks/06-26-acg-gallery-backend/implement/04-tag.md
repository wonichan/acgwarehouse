## 阶段 4 · 标签

- [ ] po/do/dto: tag、image_tag；repository/tag。
- [ ] service/tag：CRUD（删/改限 admin）、批量打/取消标签（同步 bleve tags 字段 + usage_count）、suggest（前缀 + 频次）。
- [ ] handler/tag.go + 路由。
- 验证：AC-R2 / AC-R7（非 admin 删改 403）/ AC-R8。

- [ ] **codegraph sync**：本阶段完成后同步索引（`codegraph sync`）。
