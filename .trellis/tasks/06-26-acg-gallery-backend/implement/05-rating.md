## 阶段 5 · 评分

- [x] po/do/dto: rating；repository/rating（upsert + 聚合）。
- [x] service/rating：0-100 校验、upsert、事务内更新 image.avg_score/rating_count、发 rating 事件。
- [x] handler/rating.go + 路由 PUT /images/:id/rating。
- 验证：AC-R3（越界 400、覆盖、冗余字段正确）。

- [x] **codegraph sync**：本阶段完成后同步索引（`codegraph sync`）。
