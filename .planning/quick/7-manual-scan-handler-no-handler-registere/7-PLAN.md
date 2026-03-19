# Quick Task 7: 创建并注册 manual_scan handler

**Mode:** quick  
**Directory:** .planning/quick/7-manual-scan-handler-no-handler-registere  
**Description:** 创建并注册 manual_scan handler，修复后台扫描功能"no handler registered"错误

## 问题分析

当用户在管理后台点击"触发扫描"按钮时，系统创建了一个类型为 `"manual_scan"` 的异步任务，但 JobManager 中没有注册对应的 handler，导致报错 "no handler registered"。

## 任务清单

### Task 1: 创建 scan handler 文件

**文件:** `internal/worker/scan_handler.go`

**操作:**
创建一个新的 handler 文件，实现 manual_scan 任务的处理逻辑。参考 thumbnail_handler.go 的模式，该 handler 需要：
1. 定义 ScanHandler 结构体
2. 实现 Handle 方法，调用 ScannerService.Scan 执行扫描
3. 从配置中读取扫描根目录 (cfg.Storage.ScanRoots)

**验证:**
- 文件创建成功
- 代码可以编译通过
- handler 正确处理 manual_scan 任务

**完成条件:**
- [ ] scan_handler.go 文件已创建
- [ ] 代码符合项目编码规范
- [ ] 无编译错误

---

### Task 2: 在 main.go 中注册 handler

**文件:** `cmd/server/main.go`

**操作:**
在 main 函数中添加 manual_scan handler 的注册，位置在 thumbnail handler 注册之后（第 132 行之后）：

1. 创建 metadataSvc
2. 创建 scannerSvc
3. 创建 scanHandler
4. 注册 handler: `jobManager.RegisterHandler("manual_scan", scanHandler.Handle)`

**验证:**
- main.go 可以编译通过
- handler 正确注册到 jobManager

**完成条件:**
- [ ] main.go 已修改
- [ ] 无编译错误
- [ ] 服务器启动时正确注册 manual_scan handler

---

## 关键引用

**参考文件:**
- `internal/worker/thumbnail_handler.go` - handler 实现模式
- `internal/service/scanner_service.go` - ScannerService API
- `cmd/server/main.go` - handler 注册位置

**配置引用:**
- `cfg.Storage.ScanRoots` - 扫描根目录配置

## 预期结果

用户点击管理后台"触发扫描"按钮后：
1. 创建 manual_scan 类型的异步任务
2. JobManager 找到对应的 handler 并执行
3. 扫描指定目录下的图片并导入数据库
4. 任务状态更新为 finished
