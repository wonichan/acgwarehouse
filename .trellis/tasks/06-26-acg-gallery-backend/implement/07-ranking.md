## 阶段 7 · 热榜

- [ ] po/do: image_event、ranking；repository/event、repository/ranking。
- [ ] job/ranking_job.go：窗口聚合 + 贝叶斯 + 热度公式 -> 写 ranking，排除软删除；定时调度（默认 10min，可配）。
- [ ] handler/ranking.go + 路由 GET /rankings?period=。
- [ ] cmd/web 启动时拉起 job。
- 验证：AC-R4（三榜窗口、读缓存、软删除排除）。

- [ ] **codegraph sync**：本阶段完成后同步索引（`codegraph sync`）。
