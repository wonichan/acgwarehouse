# 错误信息

**错误详情:**
```
Error loading tag statistics: Exception: Failed to get tag statistics: 404
```

**问题分析:**
- 类型: HTTP 404 错误
- 位置: 标签管理页面加载标签统计信息时
- 原因: API 端点不存在或请求路径错误
- 可能的问题:
  1. 后端 API 路由未定义
  2. 前端请求 URL 路径错误
  3. 路由拼写错误或大小写问题
  4. API 版本不匹配

**需要检查:**
1. Flutter 前端代码中调用 tag statistics API 的位置
2. 后端 API 路由定义
3. 请求 URL 是否正确
