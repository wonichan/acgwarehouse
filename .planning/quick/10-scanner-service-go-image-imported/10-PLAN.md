# Quick Task 10: 添加去重检查避免重复创建 image_imported 任务

**Created:** 2026-03-19
**Status:** Planned

## 任务目标

在 `scanner_service.go:146-164` 添加去重检查，避免为已导入的图片重复创建 `image_imported` 任务。

## 问题描述

当前 `importFile` 方法每次扫描到图片时都会创建一个新的 `image_imported` 任务，即使该图片已经有待处理的相同类型任务。这可能导致：
1. 短时间内多次触发扫描会产生重复任务
2. 任务队列膨胀，浪费处理资源
3. 同一图片被多次处理缩略图

## 解决方案

添加一个 repository 方法 `FindByTypeAndStatus`，在创建任务前检查是否已存在针对该图片的 pending/ready 状态任务。

## 任务列表

### Task 1: 添加 JobRepository 查询方法

**Files:** `internal/repository/job_repository.go`

**Action:**
1. 在 JobRepository 接口中添加 `FindByTypeAndStatus(jobType string, status string) ([]domain.AsyncJob, error)` 方法
2. 在 sqliteJobRepository 中实现该方法

**Verify:**
- 接口定义正确
- 实现使用了正确的 SQL 查询

**Done:**
- [ ] 接口添加方法签名
- [ ] 实现查询方法

### Task 2: 修改 ScannerService 添加去重逻辑

**Files:** `internal/service/scanner_service.go`

**Action:**
在 `importFile` 方法中创建 `image_imported` 任务前，添加以下检查：
1. 查询所有 `image_imported` 类型的 `ready` 状态任务
2. 解析 payload 检查是否已存在相同 image_id 的任务
3. 如果存在，跳过创建任务

**Verify:**
- 代码能编译通过
- 去重逻辑正确检查 image_id
- 不存在时才创建新任务

**Done:**
- [ ] 添加去重检查逻辑
- [ ] 正确处理错误情况
- [ ] 代码通过编译

### Task 3: 验证修改

**Action:**
1. 运行 `go build ./...` 确保项目能编译
2. 检查是否有测试需要更新

**Verify:**
- `go build ./...` 成功
- 无编译错误

**Done:**
- [ ] 项目编译成功
