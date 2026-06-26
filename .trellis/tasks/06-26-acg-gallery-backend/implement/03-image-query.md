## 阶段 3 · 图片查询 / 详情 / 搜索 / 软删除

- [ ] service/image：列表（tag/filename 过滤 + created_at/size/tag 排序 + 分页）、详情聚合、相似推荐（共享标签）、软删除/恢复。
- [ ] handler/image.go + 路由：GET /images、GET /images/:id（记 view 事件，走内存缓冲+批量 flush）、GET /search（中文/拼音/首字母）、DELETE /images/:id [admin]、restore。
- [ ] view 事件缓冲器：内存队列 + 定时/按量批量写入 image_event（单写事务），避免逐条写锁竞争。
- 验证：AC-R1 / AC-R11 / AC-R12 / AC-R13 / AC-R9。

- [ ] **codegraph sync**：本阶段完成后同步索引（`codegraph sync`）。
