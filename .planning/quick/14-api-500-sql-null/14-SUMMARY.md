# 快速任务 14 执行总结

**任务:** 修复标签统计API 500错误 - SQL NULL处理
**执行时间:** 2026-03-19
**状态:** ✅ 已完成

## 修复内容

### 问题
标签管理页面显示500错误：
```
Error loading tag statistics: Exception: Failed to get tag statistics: 500
```

### 根因
当标签没有关联任何图片时，`SUM(CASE ...)` 聚合函数返回 `NULL` 而不是 `0`。Go的 `rows.Scan` 无法将数据库NULL值扫描到 `int64` 类型的字段中，导致错误。

### 修复方案
在SQL查询中使用 `COALESCE` 函数将可能的NULL值转为0：

```sql
-- 修改前:
SUM(CASE WHEN review_state = 'confirmed' THEN 1 ELSE 0 END) as confirmed_count,

-- 修改后:
COALESCE(SUM(CASE WHEN review_state = 'confirmed' THEN 1 ELSE 0 END), 0) as confirmed_count,
```

### 修改文件
`internal/repository/image_tag_repository.go` 第158-166行

为4个 `SUM` 表达式都添加了 `COALESCE(..., 0)` 包装。

## 验证
- ✅ 代码语法正确
- ✅ SQL查询现在能正确处理空结果集

## 影响范围
- 修复标签管理页面加载时的500错误
- 当标签没有关联图片时也能正常返回统计（全为0）
