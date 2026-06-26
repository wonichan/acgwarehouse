# 设计 · 用户与认证

> 依赖地基 §2/§9.0/§9.1。

## 8. 认证与权限（middleware）

- `Auth`：解析 `Authorization: Bearer <jwt>`，校验签名/过期，`ctx.Set(user_id/role)`，`ctx.Next(c)`；失败 `AbortWithStatusJSON(401)`。
- `RequireAdmin`：Auth 之后校验 role==admin，否则 403。
- 公开组（无 Auth）：图片查询/详情/搜索、公开收藏夹浏览、热榜、标签建议。
- 登录组：评分、收藏 CRUD、打标签。
- 管理组（Auth+Admin）：标签删除/更新、图片软删除/恢复。
