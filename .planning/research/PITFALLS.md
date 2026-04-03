# 领域陷阱研究

**项目:** ACG 图库 v4.0 Windows Photos 风格重构 + Python 计算侧车
**类型:** 棕地集成（现有 Go + Flutter + SQLite）
**调研时间:** 2026-04-03
**置信度:** HIGH

---

## 一、关键陷阱

### 陷阱 1: 僵尸进程与生命周期失控

**问题现象:**
Go 主控进程启动 Python 侧车后，在关闭时未能正确清理 Python 子进程，导致：
- Python 进程变成僵尸进程（zombie）持续占用系统资源
- Windows 任务管理器中残留无法关闭的进程
- 重启应用时因端口占用导致启动失败
- 进程表逐渐填满，最终导致系统无法启动新进程

**根源分析:**
1. **Windows vs Unix 信号差异** — Go 的 `syscall.SIGTERM` 在 Windows 上不会传递给子进程，需要使用 `exec.Command` 的 `Process.Kill()` 方法
2. **缺少进程监听** — Go 启动 Python 后没有调用 `cmd.Wait()`，导致子进程退出状态无法被回收
3. **关闭顺序混乱** — Flutter 关闭时未等待 Go 完成清理，Go 未等待 Python 完成响应
4. **无心跳检测** — Python 进程崩溃后 Go 无感知，继续发送请求直到超时

**预防策略:**
```go
// 正确的生命周期管理
func managePythonSidecar() {
    cmd := exec.Command("python_service.exe")
    
    // 1. 启动时建立管道通信
    stdout, _ := cmd.StdoutPipe()
    cmd.Start()
    
    // 2. 监听进程退出（防止僵尸）
    go func() {
        err := cmd.Wait() // 必须调用，否则僵尸
        log.Printf("Python exited: %v", err)
    }()
    
    // 3. Windows 优雅关闭：先 HTTP /shutdown，再 Kill
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    http.Post("http://localhost:port/shutdown", ...)
    select {
    case <-shutdownCtx.Done():
        cmd.Process.Kill() // Windows 强制终止
    }
}
```

**警告信号:**
- ❌ `cmd.Start()` 后没有 `cmd.Wait()` 调用 → 必定产生僵尸进程
- ❌ Windows 关闭流程中使用 `SIGTERM` → 无效，需要 `Process.Kill()`
- ❌ Python 端口硬编码（如 `5000`） → 重启时端口冲突风险
- ❌ 无健康检查端点（`/health`） → Python 崩溃后 Go 无法感知

**应预防的阶段:**
- Phase 1（架构边界建立） — 实现完整的生命周期管理
- Phase 2（Python 侧车落地） — 添加心跳检测与自动重启机制

**置信度:** HIGH（Go 官方文档明确要求 `cmd.Wait()`，Windows 进程管理有明确模式）

**来源:**
- Go 官方文档：`Wait4` 系统调用必须用于回收子进程状态
- Stormkit 博客：僵尸进程在 Docker/Go 环境中的案例
- Graceful Shutdown in Go：Windows 信号处理差异

---

### 陷阱 2: PyInstaller 打包陷阱与启动失败

**问题现象:**
用户下载安装包后无法启动，或启动后立即崩溃：
- 缺少依赖 DLL 导致启动失败（如 `vcruntime140.dll`）
- `ImportError: No module named X` 在打包后出现
- `--noconsole` 模式下 `sys.stdout = None` 导致写入失败
- 路径问题：打包后无法找到资源文件（图片、配置）
- 随机端口分配失败，启动时打印端口号无法被 Go 读取

**根源分析:**
1. **依赖收集不完整** — PyInstaller 静态分析无法检测动态导入（如 `importlib.import_module`）
2. **Windows 路径差异** — 打包后的临时路径（`sys._MEIPASS`）与开发环境不同
3. **控制台模式问题** — `--noconsole` 会禁用所有标准流，代码中 `print()` 或 `sys.stdout.write()` 会抛出 `AttributeError`
4. **启动顺序阻塞** — Go 在 Python 未完全启动前就开始发送请求，导致连接失败
5. **调试困难** — 用户无法看到错误日志，无法反馈具体问题

**预防策略:**
```python
# Python 服务必须处理无控制台模式
import sys, os

# 1. 修复标准流缺失
if sys.stdout is None:
    sys.stdout = open(os.devnull, 'w')
if sys.stderr is None:
    sys.stderr = open(os.devnull, 'w')

# 2. 正确的资源路径处理
def get_resource_path(relative_path):
    if hasattr(sys, '_MEIPASS'):
        return os.path.join(sys._MEIPASS, relative_path)
    return os.path.join(os.path.abspath('.'), relative_path)

# 3. 启动时打印端口（Go 必须能读取）
port = find_available_port()
print(f"LISTENING_PORT:{port}", flush=True)  # flush 是关键
sys.stdout.flush()
```

```bash
# 打包流程（调试优先）
# 1. 先用 --onedir 模式调试，再用 --onefile
pyinstaller --onedir --debug=all app.py

# 2. 确保所有依赖被收集
pyinstaller --hidden-import=PIL --hidden-import=imagehash app.py

# 3. 添加 UPX 排除（某些 DLL 不兼容 UPX）
pyinstaller --noupx app.py
```

**警告信号:**
- ❌ 打包后测试只在开发者机器运行 → 依赖了本地环境，未打包完整
- ❌ `--onefile` 模式作为首次打包 → 调试困难，无法检查依赖
- ❌ Python 代码中有 `print()` 但未处理 `--noconsole` → 启动崩溃
- ❌ 端口硬编码 → 多实例启动冲突
- ❌ 缺少 `flush=True` → Go 无法及时读取端口

**应预防的阶段:**
- Phase 3（Windows 打包与分发） — 完整的打包流程与多机器测试
- Phase 4（诊断与回退） — 崩溃日志收集与错误反馈机制

**置信度:** HIGH（PyInstaller 官方文档明确列出 `--noconsole` 和依赖收集陷阱）

**来源:**
- PyInstaller 官方文档："Common Issues and Pitfalls" 章节
- PyInstaller Recipe：`subprocess_args()` 处理 Windows 控制台问题
- PyInstaller Wiki：调试打包问题的最佳实践

---

### 陷阱 3: 端口冲突与启动协调失败

**问题现象:**
应用启动时出现"端口已被占用"错误，或启动顺序错乱：
- 用户电脑上已有其他服务占用 8080/5000 端口
- 多个 ACG 图库实例无法同时运行
- Go 在 Python 未完成启动前就开始发送请求
- Flutter 在 Go 未监听前就尝试连接
- Python 端口未传递给 Go，导致 Go 无法连接

**根源分析:**
1. **硬编码端口** — 技术方案中提到的 `localhost:8080` 和 `localhost:5000` 是常见端口，容易被其他应用占用
2. **启动顺序依赖** — Flutter → Go → Python 的三层启动链，每层都依赖前层完成
3. **无启动确认机制** — Python 打印端口后 Go 未等待就立即连接
4. **无超时与重试** — Go 连接 Python 失败后立即报错，而不是等待重试
5. **无进程间通信契约** — Python 如何通知 Go 已就绪？Go 如何通知 Flutter？

**预防策略:**
```go
// Go 使用随机端口 + 启动确认
func startPythonSidecar() (int, error) {
    // 1. Python 使用 --port=0（随机端口）
    cmd := exec.Command("python_service.exe", "--port=0")
    stdout, _ := cmd.StdoutPipe()
    scanner := bufio.NewScanner(stdout)
    
    cmd.Start()
    
    // 2. 等待 Python 打印端口（最多 10 秒）
    timeout := time.After(10 * time.Second)
    for {
        select {
        case <-timeout:
            cmd.Process.Kill()
            return 0, errors.New("Python startup timeout")
        default:
            if scanner.Scan() {
                line := scanner.Text()
                if strings.HasPrefix(line, "LISTENING_PORT:") {
                    port := parsePort(line)
                    return port, nil
                }
            }
        }
    }
}

// 3. Go 监听随机端口
listener, _ := net.Listen("tcp", "127.0.0.1:0")
goPort := listener.Addr().(*net.TCPAddr).Port

// 4. Flutter 通过环境变量或配置文件获知 Go 端口
os.Setenv("ACG_BACKEND_PORT", strconv.Itoa(goPort))
```

**警告信号:**
- ❌ 任何端口硬编码（包括技术方案中的 8080） → 冲突风险极高
- ❌ Flutter 启动后立即连接 Go → 未等待 Go 就绪
- ❌ Python 未打印端口或未 flush → Go 无法读取
- ❌ 缺少启动超时机制 → 用户无法区分"正在启动" vs "启动失败"
- ❌ 无连接重试逻辑 → 瞬态失败导致整体失败

**应预防的阶段:**
- Phase 1（架构边界建立） — 确立随机端口 + 启动确认契约
- Phase 2（Python 侧车落地） — 实现完整的启动等待与重试机制

**置信度:** HIGH（Windows 端口冲突是常见问题，随机端口是标准解决方案）

**来源:**
- Windows 端口管理案例："Taming the Digital Gatekeepers"
- Go 官方文档：`net.Listen("tcp", "127.0.0.1:0")` 随机端口分配
- Microsoft msquic issue：多端口监听的 Windows 特殊性

---

### 陷阱 4: 侧车故障时的降级策略缺失

**问题现象:**
Python 侧车崩溃或启动失败时，整个应用无法使用：
- 重复检测功能完全不可用，用户无法访问图库
- Go 任务队列因 Python 无响应而阻塞
- 错误信息不清晰："无法连接服务"无具体说明
- 无回退机制，用户只能重启应用或等待
- 崩溃后无自动恢复，需要手动重启

**根源分析:**
1. **硬依赖设计** — 重复检测功能未设计降级路径（如回退到 Go 内置哈希）
2. **无故障隔离** — Python 失败导致整个任务平台阻塞
3. **缺少健康检查** — Go 无法及时感知 Python 状态
4. **无降级通知** — 用户不知道哪些功能可用，哪些降级了
5. **缺少自动恢复** — Python 崩溃后需要手动重启应用

**预防策略:**
```go
// Go 实现优雅降级
type DuplicateChecker interface {
    Check(images []Image) ([]DuplicateGroup, error)
}

// 双路径设计
type HybridChecker struct {
    pythonChecker *PythonSidecarChecker
    fallbackChecker *GoHashChecker // 降级路径
    healthy bool
}

func (h *HybridChecker) Check(images []Image) ([]DuplicateGroup, error) {
    // 1. 先尝试 Python
    if h.healthy {
        result, err := h.pythonChecker.Check(images)
        if err == nil {
            return result, nil
        }
        // 2. 失败后降级
        h.healthy = false
        log.Warn("Python sidecar failed, falling back to Go hash")
    }
    
    // 3. 降级路径
    return h.fallbackChecker.Check(images)
}

// 4. 定期健康检查 + 自动恢复
func (h *HybridChecker) healthMonitor() {
    ticker := time.NewTicker(30 * time.Second)
    for range ticker.C {
        if h.pythonChecker.IsHealthy() {
            h.healthy = true
            log.Info("Python sidecar recovered")
        }
    }
}
```

**警告信号:**
- ❌ 重复检测完全依赖 Python → Python 崩溃 = 功能完全不可用
- ❌ Go 中无备选哈希实现 → 无法降级
- ❌ 缺少 `/health` 端点 → 无法监控 Python 状态
- ❌ 无降级通知 UI → 用户不知道功能受限
- ❌ 无自动恢复机制 → 需要用户手动重启

**应预防的阶段:**
- Phase 2（Python 侧车落地） — 实现健康检查 + 双路径降级
- Phase 4（诊断与回退） — 崩溃恢复 + 降级通知 UI

**置信度:** HIGH（Sidecar 模式标准要求降级路径）

**来源:**
- Dapr Sidecar health：健康检查端点设计
- SRE School：Sidecar graceful degradation patterns
- Zylos Research：AI agent systems graceful degradation

---

### 陷阱 5: SQLite 并发访问数据一致性问题

**问题现象:**
在 UI 重构期间，多个进程同时访问 SQLite 导致数据问题：
- Go 任务平台与 Flutter UI 同时读写导致锁冲突
- "database is locked" 错误频繁出现
- 数据写入顺序错乱，标签更新丢失
- 长时间写入阻塞 UI 读取
- WAL 模式配置不一致导致并发失败

**根源分析:**
1. **SQLite 并发限制** — 默认模式下，写入会阻塞所有读取
2. **缺少连接池** — 每个请求新建连接，导致锁竞争
3. **WAL 模式未启用** — 需要显式启用 WAL 才能支持并发读写
4. **事务边界不清** — 长事务阻塞其他操作
5. **无超时机制** — 锁等待无限期，导致应用卡死

**预防策略:**
```go
// SQLite 并发最佳实践
func setupDatabase() *sql.DB {
    db, _ := sql.Open("sqlite3", "file:acg.db?cache=shared&mode=rwc")
    
    // 1. 启用 WAL 模式（关键）
    db.Exec("PRAGMA journal_mode=WAL;")
    
    // 2. 设置合理的锁超时（5 秒）
    db.Exec("PRAGMA busy_timeout=5000;")
    
    // 3. 连接池配置
    db.SetMaxOpenConns(5)  // 限制连接数
    db.SetMaxIdleConns(5)
    
    // 4. 长时间写入使用独立事务
    func batchUpdate(images []Image) {
        tx, _ := db.Begin()
        for _, img := range images {
            tx.Exec("UPDATE images SET status = ?", img.Status)
        }
        tx.Commit() // 快速提交，减少阻塞
    }
    
    return db
}
```

**警告信号:**
- ❌ 未启用 WAL 模式 → 写入必定阻塞读取
- ❌ 无 `busy_timeout` 配置 → 锁等待无限期
- ❌ 连接数无限制 → 锁竞争加剧
- ❌ 长事务未拆分 → 阻塞时间过长
- ❌ 不同模块使用不同连接 → 无法共享锁状态

**应预防的阶段:**
- Phase 1（架构边界建立） — SQLite 并发配置标准化
- Phase 2（Python 侧车落地） — 确保写入不会阻塞 UI

**置信度:** HIGH（SQLite 并发是桌面应用经典陷阱）

**来源:**
- TheLinuxCode：SQLite Transactions in Practice
- ChatML Blog：SQLite Concurrency in Go Desktop IDE
- SQLite Forum：SQLite Versioning & Migration Strategies

---

### 陷阱 6: UI 重构期间的用户体验断裂

**问题现象:**
Windows Photos 风格重构时，用户遇到体验问题：
- 新旧界面混用，导航逻辑不一致
- 用户习惯的操作路径消失（如收藏夹位置变化）
- 瀑布流视图性能下降，滚动卡顿
- 查看器窗口大小不记忆用户设置
- 无渐进式切换，用户突然面对全新界面

**根源分析:**
1. **一次性替换** — 没有使用 Feature Flag，新旧界面无法并存
2. **缺少用户测试** — 重构未经小范围验证，直接全量发布
3. **性能问题忽略** — 瀑布流虚拟化未实现，大量图片渲染卡顿
4. **用户习惯未尊重** — 强制改变导航路径，无过渡期
5. **缺少反馈渠道** — 用户无法报告界面问题

**预防策略:**
```dart
// Feature Flag 双界面支持
class GalleryView extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final useNewUI = FeatureFlags.isPhotosStyleEnabled();
    
    return useNewUI 
      ? PhotosStyleGallery()  // Windows Photos 风格
      : LegacyGallery();      // 旧界面（降级路径）
  }
}

// 用户设置迁移
class ViewSettings {
  static Future<void> migrateUserPreferences() async {
    // 保留旧设置：窗口大小、缩放级别、筛选偏好
    final oldSettings = await loadLegacySettings();
    await saveNewSettings(oldSettings);
  }
}

// 渐进式推出（Phase 内验证）
void rolloutPhotosUI() {
  // 1. 内部测试阶段（10% 用户）
  FeatureFlags.setPhotosUIPercentage(10);
  
  // 2. 监控反馈与性能
  Analytics.trackPerformance('gallery_scroll_fps');
  
  // 3. 逐步扩大（50% → 100%）
}
```

**警告信号:**
- ❌ 无 Feature Flag 机制 → 无法回退或渐进推出
- ❌ 瀑布流未虚拟化 → 大图库性能必卡顿
- ❌ 用户偏好未迁移 → 窗口大小、筛选设置丢失
- ❌ 无性能监控 → 卡顿问题无法量化
- ❌ 缺少用户反馈入口 → 无法收集体验问题

**应预防的阶段:**
- Phase 1（架构边界建立） — Feature Flag 基础设施
- Phase 5（桌面 UI 重构） — 渐进式重构 + 性能验证

**置信度:** HIGH（Feature Flag 是 UI 重构标准模式）

**来源:**
- TechDebt.guru：Feature Flags for Safe Refactoring
- Modernization Intel：Feature Flags in Legacy Modernization
- Code With Seb：Progressive Rollouts in UI

---

### 陷阱 7: 数据库 Schema 迁移回滚失败

**问题现象:**
Python 侧车引入新数据表（如 `image_hashes`）后，迁移失败：
- 新表创建失败但旧数据已修改
- 迁移中途崩溃，数据库处于不一致状态
- 回滚脚本缺失或无效，无法恢复旧版本
- 用户数据库损坏，无法打开应用
- 无版本标记，无法判断迁移进度

**根源分析:**
1. **缺少迁移版本管理** — 无 `schema_version` 表记录迁移进度
2. **事务边界错误** — 多个 DDL 操作未在同一事务中
3. **无回滚脚本** — 只考虑成功路径，忽略失败恢复
4. **破坏性变更** — 直接删除旧表，无过渡期
5. **缺少数据验证** — 迁移后未验证数据完整性

**预防策略:**
```go
// Schema 迁移安全流程
func migrateSchema(db *sql.DB) error {
    // 1. 创建版本管理表（一次性）
    db.Exec(`
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version INTEGER PRIMARY KEY,
            applied_at TIMESTAMP,
            description TEXT
        )
    `)
    
    // 2. 检查当前版本
    var currentVersion int
    db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&currentVersion)
    
    // 3. 每个迁移独立事务
    migrations := []Migration{
        {Version: 1, SQL: "CREATE TABLE image_hashes (...)", Rollback: "DROP TABLE image_hashes"},
        {Version: 2, SQL: "ALTER TABLE images ADD COLUMN hash_status", Rollback: "ALTER TABLE images DROP COLUMN hash_status"},
    }
    
    for _, m := range migrations {
        if m.Version <= currentVersion {
            continue // 已完成
        }
        
        // 4. 事务执行 + 回滚准备
        tx, _ := db.Begin()
        _, err := tx.Exec(m.SQL)
        if err != nil {
            tx.Rollback()
            return fmt.Errorf("Migration %d failed: %v (rollback available)", m.Version, err)
        }
        
        // 5. 记录迁移完成
        tx.Exec("INSERT INTO schema_migrations VALUES (?, ?, ?)", m.Version, time.Now(), m.Description)
        tx.Commit()
    }
    
    return nil
}
```

**警告信号:**
- ❌ 无 `schema_migrations` 版本表 → 无法判断迁移进度
- ❌ 迁移未在事务中执行 → 失败后数据库不一致
- ❌ 无回滚脚本 → 无法恢复
- ❌ 直接删除旧表 → 破坏性变更无回退
- ❌ 迁移后未验证数据 → 潜在损坏未发现

**应预防的阶段:**
- Phase 2（Python 侧车落地） — 新表引入的迁移策略
- Phase 4（诊断与回退） — 迁移失败恢复机制

**置信度:** HIGH（Schema migration 是数据库高风险操作）

**来源:**
- SQLite Forum：SQLite Versioning & Migration Strategies
- Viprasol：Database Schema Migration strategies
- James Ross Jr：Zero-Downtime Database Migrations

---

### 陷阱 8: 多进程错误归属与诊断困难

**问题现象:**
用户遇到错误时，无法定位问题来源：
- "操作失败"无具体说明，不知道是 Go、Python 还是 Flutter 问题
- Python 异常未传递给 Go，Go 只看到超时
- 日志分散在三个进程，难以关联分析
- 无错误 ID 或请求追踪，无法复现问题
- 用户反馈无诊断信息（如日志片段）

**根源分析:**
1. **错误信息丢失** — Python 异常栈未传递给 Go，Go 只记录超时
2. **缺少请求追踪** — 无 request ID 跨进程传递
3. **日志格式不一致** — 三个进程日志格式不同，无法聚合分析
4. **无错误归档** — 错误发生后无详细记录，无法事后分析
5. **用户无诊断工具** — 用户无法导出日志供开发者分析

**预防策略:**
```go
// Go-Python 通信带错误详情
func callPythonService(reqID string, payload []byte) (*Response, error) {
    // 1. 每个请求分配 ID
    reqID := generateRequestID()
    
    // 2. HTTP 调用带超时 + 错误详情
    ctx, cancel := context.WithTimeout(30*time.Second)
    defer cancel()
    
    resp, err := client.Post(ctx, "http://localhost:port/check", 
        jsonPayload(map[string]interface{}{
            "request_id": reqID,
            "data": payload,
        }))
    
    // 3. Python 必须返回错误详情
    if err != nil {
        // 解析 Python 返回的错误结构
        var pythonErr struct {
            Error   string `json:"error"`
            Stack   string `json:"stack_trace"`
            Type    string `json:"error_type"`
        }
        json.Unmarshal(resp.Body, &pythonErr)
        
        // 4. Go 日志记录完整错误链
        log.Printf("[%s] Python error: %s\nStack: %s", reqID, pythonErr.Error, pythonErr.Stack)
        
        return nil, fmt.Errorf("Python failed: %s (request_id=%s)", pythonErr.Error, reqID)
    }
    
    return resp, nil
}

// 5. 统一日志格式（所有进程）
// Go: [2026-04-03 10:00:00] [req_123] [INFO] Python called
// Python: [2026-04-03 10:00:01] [req_123] [ERROR] hash computation failed
// Flutter: [2026-04-03 10:00:02] [req_123] [WARN] user action timeout
```

**警告信号:**
- ❌ Python 错误只返回 `500 Internal Error` → 无详细错误信息
- ❌ 无 request ID → 无法关联三个进程的日志
- ❌ 日志格式不一致 → 无法聚合分析
- ❌ 无错误归档机制 → 错误发生后无详细记录
- ❌ 用户无日志导出功能 → 无法反馈诊断信息

**应预防的阶段:**
- Phase 2（Python 侧车落地） — 错误传递协议
- Phase 4（诊断与回退） — 日志聚合 + 用户诊断工具

**置信度:** HIGH（多进程调试是分布式系统经典难题）

**来源:**
- Oneuptime：Dapr Service Invocation Error Handling
- 多进程调试通用模式（个人经验）

---

### 陷阱 9: 范围蔓延与里程碑边界失控

**问题现象:**
v4.0 里程碑执行期间，范围不断扩大：
- "既然重构了 UI，不如顺便加个编辑功能"
- "Python 侧车既然能做哈希，不如也做 AI 标签"
- "Windows 打包既然做了，不如也支持 macOS"
- 里程碑延期，Phase 数量不断增加
- 技术债累积，后续里程碑更难规划

**根源分析:**
1. **缺少范围边界文档** — 无明确的 "Out of Scope" 清单
2. **功能诱惑** — 重构过程暴露新需求，难以克制
3. **缺少 Phase 优先级** — 所有想法都想在当前里程碑完成
4. **无技术债管理** — 快捷径引入新功能，长期成本未评估
5. **缺少里程碑审计** — 范围变化未记录，无法追溯

**预防策略:**
```markdown
# ROADMAP.md 明确边界
## v4.0 Out of Scope（明确拒绝）
- ❌ 图片编辑功能 → 属于 v5.0，当前专注浏览与查看
- ❌ Python 承担 AI 标签 → Go 继续负责，Python 只做哈希
- ❌ macOS / Linux 支持 → 先完成 Windows 主路径
- ❌ 云同步 → 单机部署仍是当前主目标

# Phase 内范围控制
## Phase 3: Python 侧车落地
**必须做:**
- 精确哈希计算（MD5/SHA256）
- 视觉相似检测（pHash）
- Go-Python 生命周期管理

**绝不做:**
- ❌ AI 标签计算（留待 Phase 6+）
- ❌ 图像质量评估（留待 v5.0）
- ❌ 自动裁剪建议（留待 v5.0）

**遇到新需求的处理:**
1. 记录到 `deferred.md`
2. 评估技术债成本
3. 分配到后续 Phase 或里程碑
4. 绝不在当前 Phase 追加
```

**警告信号:**
- ❌ "顺便做 X" 思路出现 → 范围蔓延信号
- ❌ Phase 数量不断增加 → 边界失控
- ❌ 新需求无 deferred 记录 → 需求管理混乱
- ❌ 缺少 Out of Scope 清单 → 无边界约束
- ❌ 快捷径实现新功能 → 技术债累积

**应预防的阶段:**
- 里程碑规划阶段 — 明确 Out of Scope 清单
- 每个 Phase — 严格范围边界 + deferred 记录

**置信度:** HIGH（范围蔓延是项目管理经典陷阱）

**来源:**
- Brownfield vs. Greenfield：范围控制的重要性
- To Rewrite or Not to Rewrite：克制重构冲动

---

## 二、技术债模式

看似合理的捷径，但会带来长期问题。

| 捷径 | 立即收益 | 长期成本 | 何时可接受 |
|------|----------|----------|-----------|
| 端口硬编码（8080） | 启动逻辑简单 | 端口冲突、多实例无法运行 | **绝不可接受** |
| Python 崩溃后无降级 | 代码量减少 | 功能完全不可用，用户体验差 | **绝不可接受** |
| cmd.Start() 不调用 Wait() | 代码简洁 | 僵尸进程累积，系统崩溃 | **绝不可接受** |
| PyInstaller 首次打包用 --onefile | 分发简单 | 调试困难，无法检查依赖 | **绝不可接受** |
| SQLite 未启用 WAL | 配置简单 | 并发阻塞，UI 卡顿 | **绝不可接受** |
| UI 重构无 Feature Flag | 开发快速 | 无法回退，用户体验断裂 | **绝不可接受** |
| Schema 迁移无回滚脚本 | 开发快速 | 数据损坏无法恢复 | **绝不可接受** |
| Python 错误只返回 500 | 实现简单 | 无法诊断问题根源 | 仅在初期原型可接受，Phase 2 必修复 |
| 无 request ID 跨进程传递 | 实现简单 | 无法关联日志分析问题 | 仅在初期原型可接受，Phase 4 必修复 |
| 快捷径实现"顺便做 X" | 功能增加 | 技术债累积，后续重构困难 | **绝不可接受** |

---

## 三、集成陷阱

连接外部服务时的常见错误（此处为 Go-Python 集成）。

| 集成点 | 常见错误 | 正确做法 |
|--------|----------|----------|
| Go 启动 Python | `cmd.Start()` 后不调用 `cmd.Wait()` | 必须调用 `Wait()` 防止僵尸进程 |
| Python 端口传递 | Python 打印端口但未 `flush=True` | 必须立即 flush，Go 才能读取 |
| Go 连接 Python | 端口硬编码（如 5000） | 使用随机端口（`--port=0`） |
| Python 崩溃处理 | Go 只看到超时，无降级 | 实现健康检查 + 降级路径 |
| Python 错误传递 | 只返回 HTTP 500 状态码 | 返回详细错误结构（error + stack_trace） |
| 关闭流程 | Windows 使用 SIGTERM（无效） | 先 HTTP /shutdown，后 Process.Kill() |
| 请求追踪 | 无 request ID | 每个请求分配 ID，跨进程传递 |
| 健康检查 | 无 `/health` 端点 | 实现 `/healthz`，定期探测 |

---

## 四、性能陷阱

小规模工作正常，规模增长后失败的模式。

| 陷阱 | 症状 | 预防 | 失败阈值 |
|------|------|------|----------|
| 瀑布流无虚拟化 | 滚动卡顿，内存暴涨 | 实现虚拟滚动，只渲染可见项 | 100+ 图片必失败 |
| SQLite 连接无限制 | "database is locked" 频繁 | 连接池（MaxOpenConns=5） | 10+ 并发必失败 |
| Python 请求无超时 | 任务队列阻塞 | context.WithTimeout(30s) | 长任务必阻塞 |
| 图片缩略图全加载 | 启动慢，内存暴涨 | 按需加载 + 缓存 | 1k+ 图片必失败 |
| 全量哈希计算启动时 | 启动等待数分钟 | 后台增量计算 | 10k+ 图片必失败 |
| 无并发写入队列 | 写入阻塞 UI | 独立写入协程 + 队列 | 100+ 写入必阻塞 |

---

## 五、安全陷阱

本领域特有的安全问题（桌面应用 + 本地数据）。

| 错误 | 风险 | 预防 |
|------|------|------|
| Python 服务监听公网 IP | 外部可访问本地服务 | **只监听 127.0.0.1** |
| 数据库文件无权限限制 | 其他应用可读写数据 | Windows 设置文件权限（仅当前用户） |
| 日志包含文件完整路径 | 暴露用户目录结构 | 脱敏路径（显示相对路径） |
| Python 进程继承所有权限 | 安全隔离失效 | 使用受限权限启动 |
| 配置文件明文存储 API Key | 泄露云端服务密钥 | 加密存储或使用系统密钥管理 |

---

## 六、UX 陷阱

用户界面常见错误。

| 陷阱 | 用户影响 | 更好做法 |
|------|----------|----------|
| UI 重构无渐进推出 | 用户突然面对陌生界面 | Feature Flag 渐进切换 |
| 无降级功能通知 | 用户不知道哪些功能可用 | 显示状态栏："重复检测降级运行" |
| 启动无进度提示 | 用户以为应用崩溃 | 显示"正在启动服务..." + 进度 |
| 瀑布流滚动卡顿 | 用户体验极差，放弃使用 | 虚拟滚动 + 图片懒加载 |
| 查看器窗口大小不记忆 | 每次打开都重置大小 | 持久化窗口状态 |
| 导航路径强制改变 | 用户习惯操作消失 | 保留旧路径 + 新路径并存期 |
| 无错误详情展示 | "操作失败"无具体说明 | 显示错误类型 + 建议操作 |

---

## 七、"看似完成实则未完成" 检查清单

功能表面上完成，但缺少关键部分。

- [ ] **Python 生命周期管理:** 往往缺少 `cmd.Wait()` 调用 → 验证关闭后无僵尸进程残留
- [ ] **端口随机分配:** 往往硬编码端口 → 验证多实例可同时启动
- [ ] **Python 错误传递:** 往今只返回 HTTP 500 → 验证错误包含详细栈信息
- [ ] **SQLite 并发配置:** 往今默认配置未启用 WAL → 验证并发读写无阻塞
- [ ] **降级路径:** 往今 Python 崩溃后功能完全不可用 → 验证 Go fallback 路径存在
- [ ] **Feature Flag UI:** 往今一次性替换界面 → 验证新旧界面可切换
- [ ] **Schema 迁移回滚:** 往今无回滚脚本 → 验证迁移失败可恢复
- [ ] **健康检查端点:** 往今无 `/health` → 验证定期探测可感知 Python 状态
- [ ] **请求追踪:** 往今无 request ID → 验证三个进程日志可关联
- [ ] **用户诊断工具:** 往今用户无法导出日志 → 验证诊断信息可导出

---

## 八、恢复策略

预防失败后的恢复方法。

| 陷阱 | 恢复成本 | 恢复步骤 |
|------|----------|----------|
| 僵尸进程残留 | LOW | 任务管理器强制关闭 + 重启应用 |
| 端口冲突 | LOW | 关闭占用端口的应用 + 重启 |
| Python 崩溃无降级 | MEDIUM | 降级路径实现 + 自动恢复机制（需开发） |
| SQLite 数据损坏 | HIGH | 从备份恢复 + 迁移脚本修复（可能丢失数据） |
| Schema 迁移失败 | HIGH | 回滚脚本执行 + 数据库恢复（需准备脚本） |
| UI 重构用户流失 | HIGH | Feature Flag 回退旧界面 + 渐进推出重规划 |
| 性能卡顿用户放弃 | HIGH | 性能优化实现（虚拟化、懒加载） |

---

## 九、陷阱 → Phase 映射

Roadmap 各阶段应预防的陷阱。

| 陷阱 | 预防阶段 | 验证方式 |
|------|----------|----------|
| 僵尸进程与生命周期失控 | Phase 1（架构边界） | 关闭应用后任务管理器无残留进程 |
| 端口冲突与启动协调失败 | Phase 1（架构边界） | 多实例可同时启动，随机端口分配 |
| PyInstaller 打包陷阱 | Phase 3（Windows 打包） | 多机器测试（无 Python 环境机器） |
| 侧车故障降级策略缺失 | Phase 2（Python 侧车） | Python 崩溃后功能降级可用 |
| SQLite 并发问题 | Phase 1（架构边界） | 并发读写无阻塞，WAL 模式启用 |
| UI 重构体验断裂 | Phase 5（桌面 UI） | Feature Flag 可切换，性能监控通过 |
| Schema 迁移回滚失败 | Phase 2（Python 侧车） | 迁移失败可回滚，数据完整 |
| 多进程错误诊断困难 | Phase 4（诊断与回退） | request ID 可关联日志，错误详情可见 |
| 范围蔓延失控 | 里程碑规划阶段 | Out of Scope 清单明确，deferred 记录完整 |

---

## 十、来源

**高置信度来源（官方文档）:**
- Go 官方文档：`Wait4` 系统调用、进程生命周期管理
- PyInstaller 官方文档："Common Issues and Pitfalls" 章节
- PyInstaller Wiki：调试打包问题的最佳实践
- Flutter Windows 文档：生命周期管理、外部窗口处理

**高置信度来源（权威博客）:**
- Stormkit：Hunting Zombie Processes in Go and Docker
- TheLinuxCode：SQLite Transactions in Practice
- SQLite Forum：SQLite Versioning & Migration Strategies
- TechDebt.guru：Feature Flags for Safe Refactoring

**高置信度来源（标准模式）:**
- Dapr Docs：Sidecar health checks
- SRE School：Sidecar graceful degradation patterns
- Oneuptime：Health Probes for Sidecar Containers

**中置信度来源（经验总结）:**
- Brownfield vs. Greenfield：范围控制的重要性
- ChatML Blog：SQLite Concurrency in Go Desktop IDE
- Modernization Intel：Feature Flags in Legacy Modernization

**个人经验（未验证，需 Phase 具体验证）:**
- 多进程调试的经验模式（request ID 跨进程传递）
- Windows 端口冲突的常见性
- 瀑布流性能问题的普遍性

---

*陷阱研究针对：ACG 图库 v4.0 棕地集成*
*调研时间：2026-04-03*
*置信度：HIGH（官方文档 + 权威博客 + 标准模式）*
