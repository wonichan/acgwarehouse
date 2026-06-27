# 集成验证功能测试

## Goal

设计完整的集成验证功能用例，启动后端服务在 7070 端口，依次执行用例验证所有功能是否符合逻辑，是否有 bug。

## Background

### 已确认事实（从代码探索）

**后端架构**：
- Go + Hertz HTTP 框架
- SQLite 双连接池（读写分离）
- Bleve 全文搜索索引
- API 基础路径：`/api/v1`
- 默认端口：8080（可通过 `PORT` 环境变量修改）

**API 接口清单**：

1. **用户认证** (`/api/v1/users`)
   - `POST /register` - 用户注册
   - `POST /login` - 用户登录，返回 JWT token
   - `GET /me` - 获取当前用户信息（需认证）

2. **图片管理** (`/api/v1/images`, `/api/v1/search`)
   - `GET /images` - 图片列表（支持 filename、tag 过滤）
   - `GET /images/:id` - 图片详情（记录浏览事件）
   - `GET /search` - 全文搜索
   - `DELETE /images/:id` - 软删除（需管理员认证）
   - `POST /images/:id/restore` - 恢复（需管理员认证）

3. **标签管理** (`/api/v1/tags`)
   - `GET /tags` - 标签列表
   - `GET /tags/suggest` - 标签建议（参数：q, limit）
   - `POST /tags` - 创建标签（需认证）
   - `PUT /tags/:id` - 更新标签（需管理员认证）
   - `DELETE /tags/:id` - 删除标签（需管理员认证）
   - `POST /images/tags` - 批量打标签（需认证）
   - `DELETE /images/tags` - 批量取消标签（需认证）

4. **评分** (`/api/v1/images/:id/rating`)
   - `PUT /images/:id/rating` - 图片评分（需认证，分数 0-100）

5. **收藏夹** (`/api/v1/collections`)
   - `GET /collections` - 用户收藏夹列表（需认证）
   - `POST /collections` - 创建收藏夹（需认证）
   - `GET /collections/:id` - 收藏夹详情（公开可访问）
   - `PUT /collections/:id` - 更新收藏夹（需认证）
   - `DELETE /collections/:id` - 删除收藏夹（需认证）
   - `POST /collections/:id/items` - 添加图片到收藏夹（需认证）
   - `DELETE /collections/:id/items/:imageId` - 移除图片（需认证）

6. **热榜** (`/api/v1/rankings`)
   - `GET /rankings` - 热榜列表（参数：period=day/week/month, page, size）

7. **健康检查**
   - `GET /api/v1/ping` - 服务健康检查

**数据库**：
- 已存在数据库：`data/acgwarehouse.db`（约 2.3MB）
- 已存在搜索索引：`data/bleve/`

**用户角色**：
- `user` - 普通用户
- `admin` - 管理员（可通过 `ADMIN_USERNAME` / `ADMIN_PASSWORD` 引导）

**认证机制**：
- JWT token，有效期默认 168h（7天）
- Header: `Authorization: Bearer <token>`

**错误响应格式**：
```json
{
  "code": "<error_code>",
  "message": "<错误消息>",
  "details": "<详情>"
}
```

**成功响应格式**：
```json
{
  "data": <响应数据>
}
```

**分页参数**：
- `page` - 页码（默认 1）
- `size` - 每页数量（默认 20）
- `sort` - 排序字段
- `order` - 排序方向（asc/desc）

## Requirements

### R1: 启动后端服务
- 在端口 7070 启动服务
- 使用现有数据库和搜索索引
- 设置管理员账号用于测试

### R2: 用户认证测试用例
- 用户注册（正常、重复用户名、非法参数）
- 用户登录（正常、错误密码、不存在用户）
- 获取当前用户信息（认证、未认证）

### R3: 图片管理测试用例
- 图片列表查询（分页、过滤）
- 图片详情查询
- 全文搜索
- 软删除（管理员权限）
- 恢复图片（管理员权限）

### R4: 标签管理测试用例
- 标签列表查询
- 标签建议
- 创建标签（认证用户）
- 更新标签（管理员权限）
- 删除标签（管理员权限）
- 批量打标签
- 批量取消标签

### R5: 评分测试用例
- 图片评分（正常范围 0-100）
- 评分边界测试（0, 100）
- 评分非法值测试（负数、超范围）
- 未认证评分

### R6: 收藏夹测试用例
- 创建收藏夹（公开、私有）
- 用户收藏夹列表
- 收藏夹详情（公开、私有）
- 更新收藏夹
- 删除收藏夹
- 添加图片到收藏夹
- 移除图片
- 非所有者操作测试

### R7: 热榜测试用例
- 日榜查询
- 周榜查询
- 月榜查询
- 分页参数
- 非法周期参数

### R8: 健康检查
- ping 接口测试

## Acceptance Criteria

- [ ] 后端服务成功在 7070 端口启动
- [ ] 所有 API 接口可正常访问
- [ ] 正常用例返回预期结果
- [ ] 异常用例返回正确错误码和消息
- [ ] 认证机制工作正常（JWT token）
- [ ] 权限控制工作正常（普通用户 vs 管理员）
- [ ] 分页功能工作正常
- [ ] 发现的 bug 有详细记录（接口、输入、预期、实际）

## Out of Scope

- 前端 UI 测试
- 数据库初始化（使用现有数据）
- 性能压力测试
- 并发安全测试

## Open Questions

1. **测试数据准备**：是否需要创建专门的测试数据？还是使用现有数据库中的数据？
   - 推荐：使用现有数据，避免污染真实数据
   - 理由：现有数据库已包含约 2.3MB 数据，足够测试

2. **测试执行方式**：用脚本自动化执行还是手动逐个执行？
   - 推荐：编写 shell 脚本或 Go 测试代码自动化执行
   - 理由：自动化可重复执行，便于回归测试

3. **bug 报告格式**：发现 bug 后如何记录？
   - 推荐：记录到任务目录的 `bugs.md` 文件
   - 格式：接口路径、请求参数、预期结果、实际结果、错误消息