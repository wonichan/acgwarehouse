# 设计 · 标签管理

> 决策见 prd.md R2/R7/R8。详细字段/接口见地基 §3 数据模型(tag/image_tag) 与 §9 API。

## 要点
- 标签全局共享；image_tag 多对多。
- 打/取消标签需在同事务后**同步更新该图 bleve `tags` 字段**（一致性见地基 §6.1）。
- usage_count 随打/取消标签增减，供 suggest 排序。
- 删除/更新标签需 RequireAdmin。
- suggest：bleve 前缀查询 + usage_count 排序。
