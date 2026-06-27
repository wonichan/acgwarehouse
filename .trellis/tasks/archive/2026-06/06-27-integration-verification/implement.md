# 集成验证 - 执行计划

## 文件结构

```
.c. trellis/tasks/06-27-integration-verification/
├── prd.md
├── design.md
├── implement.md
└── bugs.md                       # 问题记录（执行时创建）
```

## 执行步骤

### Step 1: 创建测试脚本目录

```bash
mkdir -p /opt/acgwarehouse/tests
```

### Step 2: 创建测试脚本

文件：`tests/integration_test.sh`

功能：
- 启动后端服务（PORT=7070）
- 等待服务就绪
- 执行所有测试组
- 生成测试报告

### Step 3: 执行测试

```bash
cd /opt/acgwarehouse
./tests/integration_test.sh
```

### Step 4: 分析测试结果

- 检查每个测试用例的 PASS/FAIL
- 失败用例分析原因
- 记录到 `bugs.md`

### Step 5: 修复发现的问题（如有）

- 修复 bug
- 重复执行测试验证

## 测试脚本编写规划

### 脚本结构

```bash
#!/bin/bash
set -e

# 配置
PORT=7070
BASE_URL="http://localhost:${PORT}/api/v1"
TEST_USER=""
TEST_TOKEN=""
ADMIN_TOKEN=""

# 测试统计
TOTAL=0
PASS=0
FAIL=0

# 辅助函数
send_request() { ... }
check_status() { ... }
check_json() { ... }

# 测试用例函数
test_health_check() { ... }
test_user_register() { ... }
test_user_login() { ... }
# ... 更多测试
```

### 测试用例明细

| ID | 名称 | 接口 | 方法 | 关键验证 |
|----|------|------|------|----------|
| T01 | 健康检查 | /ping | GET | 200 OK, {message: "pong"} |
| T02 | 用户注册 - 正常 | /users/register | POST | 201, 返回用户信息 |
| T03 | 用户注册 - 重复用户名 | /users/register | POST | 400, 用户名已存在 |
| T04 | 用户注册 - 非法参数 | /users/register | POST | 400, 用户名密码不符合规则 |
| T05 | 用户登录 - 正常 | /users/login | POST | 200, 返回 token |
| T06 | 用户登录 - 错误密码 | /users/login | POST | 401, 凭据错误 |
| T07 | 获取当前用户 | /users/me | GET | 200 认证通过，400 未认证 |
| T08 | 图片列表 - 无过滤 | /images | GET | 200, 分页结构正确 |
| T09 | 图片列表 - filename 过滤 | /images?filename=test | GET | 200, 返回匹配项 |
| T10 | 图片详情 | /images/:id | GET | 200, 图片完整信息 |
| T11 | 图片搜索 | /search?q=关键词 | GET | 200, 返回搜索结果 |
| T12 | 标签列表 | /tags | GET | 200, 标签列表 |
| T13 | 标签建议 | /tags/suggest?q=测试&limit=10 | GET | 200, 建议标签 |
| T14 | 创建标签 | /tags | POST (认证) | 201, 新标签返回 |
| T15 | 更新标签 | /tags/:id | PUT (管理员) | 200 |
| T16 | 删除标签 | /tags/:id | DELETE (管理员) | 200 |
| T17 | 批量打标签 | /images/tags | POST (认证) | 200 |
| T18 | 批量取消标签 | /images/tags | DELETE (认证) | 200 |
| T19 | 图片评分 - 正常 | /images/:id/rating | PUT (认证) | 200 |
| T20 | 图片评分 - 超范围 | /images/:id/rating | PUT (认证) | 400, 参数错误 |
| T21 | 收藏夹列表 | /collections | GET (认证) | 200 |
| T22 | 创建收藏夹 - 公开 | /collections | POST (认证) | 201 |
| T23 | 创建收藏夹 - 私有 | /collections | POST (认证) | 201 |
| T24 | 收藏夹详情 | /collections/:id | GET | 200 |
| T25 | 更新收藏夹 | /collections/:id | PUT (认证) | 200 |
| T26 | 删除收藏夹 | /collections/:id | DELETE (认证) | 200 |
| T27 | 添加图片到收藏夹 | /collections/:id/items | POST (认证) | 200 |
| T28 | 移除收藏夹图片 | /collections/:id/items/:imageId | DELETE (认证) | 200 |
| T29 | 热榜 - 日榜 | /rankings?period=day | GET | 200 |
| T30 | 热榜 - 周榜 | /rankings?period=week | GET | 200 |
| T31 | 热榜 - 月榜 | /rankings?period=month | GET | 200 |
| T32 | 热榜 - 非法周期 | /rankings?period=invalid | GET | 400 |
| T33 | 软删除图片（管理员） | /images/:id | DELETE (管理员) | 200 |
| T34 | 恢复图片（管理员） | /images/:id/restore | POST (管理员) | 200 |
| T35 | 未认证访问受保护接口 | /images/:id | GET | 401 |
| T36 | 普通用户访问管理接口 | /tags/:id | PUT | 403 |

## 验证命令执行

### 启动服务

```bash
cd /opt/acgwarehouse
PORT=7070 ./cmd/web/main.go &
BACKEND_PID=$!
echo "Backend started with PID: $BACKEND_PID"
```

### 服务就绪检查

```bash
for i in {1..30}; do
  if curl -s http://localhost:7070/api/v1/ping > /dev/null; then
    echo "Service ready"
    break
  fi
  sleep 1
done
```

### 测试统计

```bash
echo "==================== 测试总结 ===================="
echo "总数: $TOTAL"
echo "通过: $PASS"
echo "失败: $FAIL"
echo "==============================================="
```

## 风险点

1. **端口占用**：7070 端口被占用时测试失败
   - 解决：检查端口使用情况

2. **数据库锁定**：并发测试可能导致 SQLite 锁定
   - 解决：串行执行测试用例

3. **测试数据污染**：重复测试可能创建重复数据
   - 解决：测试前清理测试用户

4. **COS 配置**：COS 相关接口可能需要真实配置
   - 解决：仅测试不涉及图片上传的接口

## 当前已知问题

无

## Rollback Plan

如测试失败：
1. 记录失败用例详情到 `bugs.md`
2. 分析失败原因
3. 修复问题（如需要）
4. 重复执行测试验证

## 后续跟进

- 如发现严重 bug：创建单独任务修复
- 待所有测试通过：标记任务完成
- 代码质量改善：运行单测套件验证