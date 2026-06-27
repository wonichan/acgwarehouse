# 执行计划：后端服务压测与并发检查

## 前置条件

- [ ] wrk 已安装（`wrk --version`）
- [ ] Go 环境可用（`go version`）
- [ ] 数据库备份已创建（`cp data/acgwarehouse.db data/acgwarehouse.db.backup`）

## 执行步骤

### Phase 1: 环境准备

#### 1.1 安装 wrk（如未安装）
```bash
# 检查 wrk 是否已安装
wrk --version || {
  # Ubuntu/Debian
  sudo apt-get install -y wrk
}
```

#### 1.2 备份数据库
```bash
cp data/acgwarehouse.db data/acgwarehouse.db.backup
```

#### 1.3 创建压测目录
```bash
mkdir -p tests/load
```

### Phase 2: 测试数据准备

#### 2.1 编写数据准备脚本
创建 `tests/load/setup_data.go`：
- 创建测试用户（loadtest_user1, loadtest_user2, loadtest_admin）
- 为每个用户生成随机评分（每个用户评分 50-100 张图片）
- 为每个用户创建收藏夹并添加图片

#### 2.2 执行数据准备
```bash
cd /opt/acgwarehouse
go run tests/load/setup_data.go
```

#### 2.3 验证数据准备
```bash
# 检查测试用户是否创建成功
sqlite3 data/acgwarehouse.db "SELECT username FROM user WHERE username LIKE 'loadtest_%';"
```

### Phase 3: 压测脚本编写

#### 3.1 创建 wrk Lua 脚本

**场景 S1: 健康检查** - `tests/load/s1_ping.lua`
**场景 S2: 图片列表** - `tests/load/s2_images.lua`
**场景 S3: 图片详情** - `tests/load/s3_image_detail.lua`（ViewBuffer 并发测试）
**场景 S4: 图片搜索** - `tests/load/s4_search.lua`（Bleve 并发测试）
**场景 S5: 用户评分** - `tests/load/s5_rating.lua`（写事务并发测试）
**场景 S6: 收藏操作** - `tests/load/s6_collection.lua`
**场景 S7: 热榜查询** - `tests/load/s7_rankings.lua`
**场景 S8: 混合场景** - `tests/load/s8_mixed.lua`

#### 3.2 创建压测执行脚本
创建 `tests/load/run_all.sh`：
- 依次执行所有场景
- 收集 wrk 输出
- 生成汇总报告

### Phase 4: 后端服务启动

#### 4.1 启动后端服务
```bash
cd /opt/acgwarehouse
export PORT=7070
export LOG_LEVEL=info
go run ./cmd/web &
BACKEND_PID=$!
echo $BACKEND_PID > tests/load/backend.pid
```

#### 4.2 等待服务就绪
```bash
for i in {1..30}; do
  if curl -s http://localhost:7070/api/v1/ping > /dev/null 2>&1; then
    echo "Service ready"
    break
  fi
  sleep 1
done
```

### Phase 5: 压测执行

#### 5.1 执行压测（按并发等级递增）

**L1: 低负载测试**
```bash
wrk -t2 -c10 -d10s -s tests/load/s8_mixed.lua http://localhost:7070/api/v1
```

**L2: 中负载测试**
```bash
wrk -t4 -c50 -d15s -s tests/load/s8_mixed.lua http://localhost:7070/api/v1
```

**L3: 高负载测试**
```bash
wrk -t4 -c100 -d15s -s tests/load/s8_mixed.lua http://localhost:7070/api/v1
```

**L4: 极限负载测试**
```bash
wrk -t4 -c200 -d15s -s tests/load/s8_mixed.lua http://localhost:7070/api/v1
```

#### 5.2 资源监控（并行执行）
```bash
# 在另一个终端执行
tests/load/monitor.sh > tests/load/metrics.log
```

#### 5.3 专项并发测试

**ViewBuffer 阻塞测试**
```bash
wrk -t4 -c200 -d30s -s tests/load/s3_image_detail.lua http://localhost:7070/api/v1
```

**评分一致性测试**
```bash
wrk -t4 -c100 -d30s -s tests/load/s5_rating.lua http://localhost:7070/api/v1
# 压测后检查评分一致性
go run tests/load/check_rating_consistency.go
```

**Bleve 并发测试**
```bash
wrk -t4 -c200 -d30s -s tests/load/s4_search.lua http://localhost:7070/api/v1
```

### Phase 6: 结果收集与分析

#### 6.1 收集后端日志
```bash
# 如果日志输出到文件
cp /var/log/acgwarehouse.log tests/load/backend.log
```

#### 6.2 提取错误日志
```bash
grep -E "(error|Error|ERROR|panic|Panic|PANIC)" tests/load/backend.log > tests/load/errors.log
```

#### 6.3 分析并发问题
- 检查 SQLite 错误：`grep "database is locked" tests/load/errors.log`
- 检查 ViewBuffer 错误：`grep "flush view events" tests/load/errors.log`
- 检查 Bleve 错误：`grep -E "(bleve|index)" tests/load/errors.log`
- 检查事务错误：`grep "transaction" tests/load/errors.log`

#### 6.4 生成报告
创建 `tests/load/generate_report.go`：
- 解析 wrk 输出
- 汇总 QPS、延迟、错误率
- 生成 Markdown 报告

### Phase 7: 数据清理

#### 7.1 停止后端服务
```bash
kill $(cat tests/load/backend.pid) 2>/dev/null || true
```

#### 7.2 执行数据清理
```bash
go run tests/load/cleanup_data.go
```

#### 7.3 验证清理结果
```bash
sqlite3 data/acgwarehouse.db "SELECT COUNT(*) FROM user WHERE username LIKE 'loadtest_%';"
# 应返回 0
```

#### 7.4 恢复数据库（如有异常）
```bash
# 仅在数据异常时执行
cp data/acgwarehouse.db.backup data/acgwarehouse.db
```

### Phase 8: 报告输出

#### 8.1 生成压测报告
输出到 `.trellis/tasks/06-27-backend-load-test/report.md`

#### 8.2 生成并发问题清单
输出到 `.trellis/tasks/06-27-backend-load-test/concurrency_issues.md`

## 验证命令

### 压测脚本语法检查
```bash
luac -p tests/load/*.lua
```

### 后端服务健康检查
```bash
curl -s http://localhost:7070/api/v1/ping | jq .
```

### 数据一致性检查
```bash
go run tests/load/check_rating_consistency.go
```

## 回滚点

| 步骤 | 回滚操作 |
|------|----------|
| 2.2 数据准备失败 | 无需回滚，脚本未执行 |
| 5.x 压测导致崩溃 | 停止 wrk，重启服务 |
| 6.x 发现数据损坏 | 恢复数据库备份 |
| 7.2 清理失败 | 手动删除 `loadtest_%` 数据 |

## 预估时间

| 阶段 | 时间 |
|------|------|
| Phase 1: 环境准备 | 3 min |
| Phase 2: 数据准备 | 5 min |
| Phase 3: 脚本编写 | 10 min |
| Phase 4: 服务启动 | 1 min |
| Phase 5: 压测执行 | 8 min |
| Phase 6: 结果分析 | 3 min |
| Phase 7: 数据清理 | 2 min |
| Phase 8: 报告输出 | 2 min |
| **总计** | **~25 min** |

## 注意事项

1. **资源限制**：在 4c4g 机器上模拟 2c2g，wrk 线程数不超过 4
2. **数据隔离**：所有测试数据使用 `loadtest_` 前缀，便于清理
3. **日志监控**：压测期间实时观察错误日志
4. **渐进式压测**：从低负载开始，逐步增加，避免直接高负载导致崩溃
5. **备份优先**：压测前必须备份数据库
