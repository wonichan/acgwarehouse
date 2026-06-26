# 设计 · 评分

> 决策见 prd.md R3。表见地基 §3 (rating)。

## 要点
- 0-100 整数，handler dto binding 校验，越界 400。
- `(user_id,image_id)` 唯一，upsert 覆盖。
- 同事务更新 image.avg_score / rating_count；发 rating 事件（image_event）。
- 贝叶斯均分由热榜 job 计算（见 design/04-ranking.md），评分模块只维护原始分与冗余聚合。
