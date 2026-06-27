# 集成验证 - 技术设计

## 测试策略

使用 shell 脚本 + curl 进行 HTTP 集成测试。每个测试用例：
1. 发送 HTTP 请求
2. 验证 HTTP 状态码
3. 验证响应 JSON 结构
4. 记录 PASS / FAIL 结果

## 测试执行架构

```
测试脚本 (tests/integration_test.sh)
├── 启动服务 (PORT=7070)
├── 等待服务就绪 (ping 轮询)
├── 按顺序执行测试组
│   ├── T01: 健康检查
│   ├── T02: 用户注册
│   ├── T03: 用户登录
│   ├── T04: 获取当前用户
│   ├── T05: 图片列表
│   ├── T06: 图片详情
│   ├── T07: 图片搜索
│   ├── T08: 标签列表
│   ├── T09: 标签建议
│   ├── T10: 标签 CRUD
│   ├── T11: 批量打标签
│   ├── T12: 图片评分
│   ├── T13: 收藏夹 CRUD
│   ├── T14: 收藏夹图片管理
│   ├── T15: 热榜查询
│   ├── T16: 管理员操作
│   └── T17: 权限控制
├── 汇总结果
└── 停止服务
```

## 测试数据策略

- 使用现有数据库中的数据
- 创建专用测试用户（testuser / testadmin）
- 测试用户创建后用于后续认证测试
- 管理员通过环境变量 ADMIN_USERNAME/ADMIN_PASSWORD 引导

## 服务启动配置

```bash
PORT=7070
ADMIN_USERNAME=testadmin
ADMIN_PASSWORD=admin123
JWT_SECRET=test-secret-key
SQLITE_PATH=data/acgwarehouse.db
BLEVE_PATH=data/bleve
```

## 结果记录

- 每个测试用例输出 PASS / FAIL
- 汇总统计：总数、通过、失败、跳过
- 失败用例记录到 `bugs.md`

## 关键验证点

1. **响应格式一致性**：所有成功响应包含 `data` 字段，失败响应包含 `code` + `message`
2. **认证拦截**：未认证请求访问受保护接口返回 401
3. **权限控制**：普通用户访问管理员接口返回 403
4. **参数校验**：非法参数返回 400
5. **分页结构**：列表接口返回 `total`、`page`、`size`、`list`
6. **业务逻辑**：评分范围、收藏夹可见性、标签操作等