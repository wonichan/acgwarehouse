# 快速任务 12: 修复Flutter标签管理页面报错 - 执行计划

**创建时间:** 2026-03-19
**状态:** 待执行

## 任务描述
修复Flutter标签管理页面加载时出现的404错误：
```
Error loading tag statistics: Exception: Failed to get tag statistics: 404
```

## 根因分析
- **后端API路由**: `/api/v1/tags/stats`
- **前端请求URL**: `/tags/statistics`
- **问题**: 路径拼写不一致导致404

## 修复方案
将前端请求路径从 `statistics` 改为 `stats`，与后端路由保持一致。

## 任务列表

### 任务1: 修复前端API路径
**文件**: `flutter_app/lib/services/tag_service.dart`
**操作**: 修改第178行，将路径从 `statistics` 改为 `stats`
**验证**: 检查修改后的代码编译无误
**完成标准**: 前端请求路径与后端路由一致

## 预期结果
标签管理页面能正常加载标签统计数据，不再出现404错误。
