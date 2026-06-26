# 设计 · 热榜

> 依赖地基 §3（image_event/ranking 表）。

## 7. 热榜算法（job/ranking_job）

- 源：`image_event`。窗口 day=24h/week=7d/month=30d（`created_at >= now-window`）。
- 收藏数=窗口内 favorite 去重 user_id；点击数=view 计数（不去重）。
- 贝叶斯评分 `bayes = (C*m + sum_score)/(C + n)`，C 先验票数（可配）、m 全站均分。
- 热度 `score = w1*bayes + w2*log(1+fav) + w3*log(1+view)`，权重 conf 可配。
- 默认 10min 计算 day/week/month top-N 写 ranking，排除 status=deleted；查询读 ranking join image。
