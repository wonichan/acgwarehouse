# 快速任务 14: 修复标签统计API 500错误

**创建时间:** 2026-03-19
**状态:** 待执行

## 问题描述
标签管理页面显示500错误：
```
Error loading tag statistics: Exception: Failed to get tag statistics: 500
```

## 根因分析
`GetTagStats` 方法中的SQL查询使用 `SUM(CASE ...)` 统计标签状态数量。当标签没有任何关联的图片时，`SUM` 函数返回 `NULL` 而不是 `0`，导致 `rows.Scan` 无法将NULL扫描到 `int64` 字段，从而报错。

## 修复方案
在SQL查询中使用 `COALESCE` 函数将 `NULL` 转为 `0`：

```sql
-- 修改前:
SUM(CASE WHEN review_state = 'confirmed' THEN 1 ELSE 0 END) as confirmed_count

-- 修改后:
COALESCE(SUM(CASE WHEN review_state = 'confirmed' THEN 1 ELSE 0 END), 0) as confirmed_count
```

## 任务
修复文件 `internal/repository/image_tag_repository.go` 第158-165行的SQL查询，为所有 `SUM` 表达式添加 `COALESCE`。
