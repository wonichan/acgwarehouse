# 架构研究

**项目:** ACG 图库 v4.0 Windows Photos 风格重构与计算层拆分
**调研日期:** 2026-04-03
**置信度:** HIGH

---

## 标准架构

### 系统概览

ACG 图库 v4.0 采用三层架构，将现有 Go + Flutter 系统演进为 Flutter(UI) + Go(主控) + Python(计算) 的分层模型：

```
┌─────────────────────────────────────────────────────────────┐
│                     Flutter Desktop UI                        │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐        │
│  │ 图库浏览 │  │ 查看器   │  │ 标签管理 │  │ 导入中心 │        │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘        │
│       │            │            │            │              │
│       └────── HTTP/WebSocket (localhost) ──────┘            │
├───────┴────────────┴────────────┴────────────┴──────────────┤
│                      Go Backend 主控层                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ 数据库管理     │  │ 任务平台编排   │  │ 进程生命周期   │      │
│  │ SQLite/FTS5  │  │ 批次/队列控制  │  │ Python 拉起   │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                 │                 │               │
│         │                 │                 └─ HTTP ────────┤
├─────────┴─────────────────┴─────────────────┴───────────────┤
│                    Python 计算侧车层                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ 精确重复检测   │  │ 视觉相似检测   │  │ 重复组聚类   │      │
│  │ MD5/SHA1     │  │ pHash/dHash  │  │ 汉明距离分组  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

### 组件职责

| 组件           | 职责                                       | 实现方式                 |
| -------------- | ------------------------------------------ | ------------------------ |
| Flutter Desktop | 界面渲染、状态管理、用户交互               | Flutter + Riverpod       |
| Go Backend     | 业务逻辑、数据库、任务编排、进程管理       | Go + Gin + SQLite        |
| Python Sidecar | 图像哈希、重复检测、相似度计算             | Python + FastAPI + Pillow |

---

## 推荐项目结构

```
acgwarehouse/
├── flutter_app/              # Flutter 桌面 UI
│   ├── lib/
│   │   ├── screens/          # 页面组件
│   │   │   ├── gallery/      # 图库主页面（瀑布流/网格）
│   │   │   ├── viewer/       # 图片查看器（filmstrip 模式）
│   │   │   ├── tags/         # 标签管理页面
│   │   │   └── imports/      # 导入中心
│   │   ├── widgets/          # 共享组件
│   │   ├── providers/        # Riverpod 状态管理
│   │   └── services/         # HTTP/WebSocket 客户端
│   └── windows/              # Windows 平台特定配置
├── cmd/server/               # Go 主控服务入口
│   └── main.go               # 启动入口（拉起 Python）
├── internal/                 # Go 内部模块
│   ├── handler/              # HTTP handlers
│   ├── service/              # 业务逻辑层
│   │   ├── import_service.go # 导入编排
│   │   ├── tag_service.go    # 标签管理
│   │   ├── duplicate_service.go # 重复检测编排（调用 Python）
│   │   └── process_manager.go # Python 进程管理
│   ├── repository/           # 数据访问层
│   └── domain/               # 领域模型
├── services/                 # Python 计算服务（新增）
│   └── python-dedupe/        # 重复检测计算侧车
│       ├── app.py            # FastAPI 服务入口
│       ├── requirements.txt  # Python 依赖
│       ├── algorithms/       # 图像哈希算法
│       │   ├── exact_hash.py # MD5/SHA1 精确哈希
│       │   ├── perceptual_hash.py # pHash/dHash
│       │   └── clustering.py # 重复组聚类
│       └── models/           # 请求/响应模型
├── deploy/                   # 打包部署配置
│   ├── package.ps1           # Windows 打包脚本
│   ├── installer/            # Inno Setup 配置
│   └── config/               # 配置文件
└── test/                     # 测试
```

### 结构设计理由

- **`flutter_app/`**: 保持现有 Flutter 代码结构，仅重构 Windows 桌面页面组件
- **`services/python-dedupe/`**: Python 侧车作为独立服务模块，便于打包和版本管理
- **`internal/service/process_manager.go`**: 新增进程生命周期管理，隔离进程治理逻辑
- **`deploy/package.ps1`**: 集成打包脚本，合并 Flutter/Go/Python 到单一分发目录

---

## 架构模式

### 模式 1: 主控编排模式（Go 主控）

**定义:** Go 后端作为系统主控，负责业务逻辑、数据持久化、任务编排和 Python 进程生命周期管理。

**何时使用:** 当需要明确业务边界、避免职责混乱、保持系统可观测性时。

**权衡:** 
- **优点:** 职责清晰、调试容易、数据库访问集中、任务状态可控
- **缺点:** 需要实现进程管理逻辑、增加 Go 代码复杂度

**示例:**

```go
// internal/service/process_manager.go
type PythonProcessManager struct {
    cmd     *exec.Cmd
    port    int
    client  *http.Client
    healthTicker *time.Ticker
}

func (pm *PythonProcessManager) Start() error {
    // 1. 启动 Python 进程（随机端口）
    pm.cmd = exec.Command("dedupe_service.exe")
    pm.cmd.Stdout = os.Stdout
    
    // 2. 读取 Python 打印的端口号
    portReader := bufio.NewReader(pm.cmd.StdoutPipe())
    pm.cmd.Start()
    portLine, _ := portReader.ReadString('\n')
    pm.port = parsePort(portLine)
    
    // 3. 建立 HTTP 连接
    pm.client = &http.Client{
        BaseURL: fmt.Sprintf("http://127.0.0.1:%d", pm.port),
    }
    
    // 4. 启动心跳检测
    pm.startHealthCheck()
    return nil
}

func (pm *PythonProcessManager) Stop() error {
    // 优雅关闭：先关闭 HTTP 连接，再发送终止信号
    pm.healthTicker.Stop()
    pm.cmd.Process.Signal(syscall.SIGTERM)
    return pm.cmd.Wait()
}
```

---

### 模式 2: 计算侧车模式（Python 计算层）

**定义:** Python 服务作为无状态计算侧车，仅接收计算请求、执行图像算法、返回结果，不涉及业务状态持久化。

**何时使用:** 当需要隔离计算密集型任务、利用 Python 图像处理生态、避免 Go 重复实现算法时。

**权衡:**
- **优点:** 职责单一、可独立测试、利用成熟 Python 库、避免 Go 性能陷阱
- **缺点:** 需要进程管理、增加打包复杂度、有进程间通信开销

**示例:**

```python
# services/python-dedupe/app.py
from fastapi import FastAPI
import uvicorn
import sys
import imagehash
from PIL import Image

app = FastAPI()

@app.post("/compute_hash")
async def compute_hash(request: HashRequest):
    # 仅计算哈希，不持久化
    img = Image.open(request.image_path)
    
    if request.hash_type == "exact":
        hash_value = compute_md5(request.image_path)
    elif request.hash_type == "perceptual":
        hash_value = str(imagehash.phash(img, hash_size=16))
    
    return {"hash": hash_value, "hash_type": request.hash_type}

if __name__ == "__main__":
    # 随机端口 + 打印到 stdout（供 Go 读取）
    port = find_random_port()
    print(f"PORT:{port}", flush=True)
    
    # --noconsole 时需重定向 stdout/stderr
    if sys.stdout is None:
        sys.stdout = open(os.devnull, "w")
    
    uvicorn.run(app, host="127.0.0.1", port=port)
```

---

### 模式 3: 三阶段优雅关闭模式

**定义:** Flutter → Go → Python 的链式关闭流程，确保资源释放顺序正确。

**何时使用:** Windows 桌面应用需避免僵尸进程、确保数据库连接正常关闭、防止文件损坏时。

**权衡:**
- **优点:** 资源释放有序、避免僵尸进程、可恢复性强
- **缺点:** 关闭耗时略长、需实现超时兜底

**示例:**

```dart
// flutter_app/lib/services/shutdown_service.dart
Future<void> gracefulShutdown() async {
  // Flutter → Go: 通知关闭
  await http.post('http://localhost:8080/shutdown');
  
  // Go 会自动清理 Python，等待 Go 退出
  await Future.delayed(Duration(seconds: 2));
  
  // Flutter 退出
  exit(0);
}
```

```go
// internal/handler/shutdown.go
func (h *Handler) Shutdown(c *gin.Context) {
    // Go → Python: 优雅关闭
    h.pythonManager.Stop()
    
    // 关闭数据库连接
    h.db.Close()
    
    // 停止接收新请求（已通过 context 传递）
    c.JSON(200, gin.H{"status": "shutting_down"})
}
```

---

## 数据流

### 请求流

```
[用户操作: 点击"开始重复检测"]
    ↓ HTTP POST /duplicates/scan
[Flutter] 发起检测请求
    ↓
[Go Handler] 接收请求，创建批次
    ↓
[Go Service] 查询待检测图片列表
    ↓ HTTP POST http://127.0.0.1:{python_port}/detect_duplicates
[Python Sidecar] 计算文件哈希 + pHash
    ↓
[Python] 聚类相似图片组（汉明距离分组）
    ↓ HTTP Response
[Go Service] 写入 duplicate_groups / duplicate_candidates
    ↓ WebSocket 推送进度
[Flutter] 展示重复组确认界面
```

### 状态管理

```
[SQLite 数据库]
    ↓ (查询)
[Go Repository] ←→ [Go Service] ←→ [Python Sidecar]
    ↓ (写入结果)
[SQLite 更新: duplicate_groups]
    ↓ (WebSocket 推送)
[Flutter Provider 更新状态]
```

### 关键数据流

1. **图片导入流:** Flutter 发起导入 → Go 扫描文件系统 → Go 写入 SQLite → Go 分发缩略图任务 → Go 调用 AI 标签 → WebSocket 推送进度 → Flutter 更新 UI

2. **重复检测流:** Flutter 发起检测 → Go 获取图片列表 → Go 调用 Python 计算 → Python 返回重复组 → Go 写入数据库 → WebSocket 推送 → Flutter 展示确认界面

3. **标签筛选流:** Flutter 选择筛选条件 → Go 解析筛选表达式 → Go 查询 SQLite + FTS → Go 返回分页结果 → Flutter 渲染筛选结果

---

## Windows 打包与分发

### 打包策略

| 组件           | 打包工具      | 输出格式            | 部署位置               |
| -------------- | ------------- | ------------------- | ---------------------- |
| Flutter Desktop | flutter build windows | Release 目录         | Release/flutter_app/   |
| Go Backend     | go build      | acg_backend.exe     | Release/               |
| Python Sidecar | PyInstaller --onefile | dedupe_service.exe  | Release/               |

### 打包流程

```powershell
# deploy/package.ps1

# 1. Flutter 打包
cd flutter_app
flutter build windows --release
$flutterDir = "build/windows/x64/runner/Release"

# 2. Go 打包
cd cmd/server
go build -o acg_backend.exe .
Move-Item acg_backend.exe $flutterDir

# 3. Python 打包
cd services/python-dedupe
pyinstaller --onefile --name dedupe_service --noconsole app.py
Move-Item dist/dedupe_service.exe $flutterDir

# 4. 创建安装包（可选）
Inno Setup Compiler deploy/installer/setup.iss
```

### 启动顺序

```
[Flutter App 启动]
    ↓
[Flutter] 拉起 Go: Process.start('acg_backend.exe')
    ↓
[Go] 拉起 Python: exec.Command('dedupe_service.exe').Start()
    ↓
[Python] 启动 FastAPI（随机端口）→ 打印 "PORT:{port}" 到 stdout
    ↓
[Go] 读取 Python 端口，建立 HTTP 连接
    ↓
[Go] 启动 HTTP 服务（固定端口 8080）
    ↓
[Flutter] 连接 localhost:8080，应用就绪
```

---

## 集成点

### 外部服务

| 服务           | 集成模式               | 注意事项                       |
| -------------- | ---------------------- | ------------------------------ |
| 千问/豆包 AI API | Go 通过 HTTP 调用      | 保持现有集成路径不变           |
| Python Sidecar | Go 启动 + HTTP localhost | 随机端口、stdout 端口传递、心跳检测 |
| SQLite 数据库   | Go 直接访问            | 保持现有数据库路径             |

### 内部边界

| 边界                      | 通信方式       | 注意事项                       |
| ------------------------- | -------------- | ------------------------------ |
| Flutter ↔ Go Backend      | HTTP + WebSocket | Flutter 不直接访问数据库或文件 |
| Go Backend ↔ Python Sidecar | HTTP (localhost) | Python 只接收计算请求，不持久化 |

### Flutter 禁止项

- ❌ 直接文件系统操作（必须通过 Go API）
- ❌ 直接数据库读写（必须通过 Go API）
- ❌ 直接调用 AI 模型（必须通过 Go 编排）
- ❌ 直接运行图像算法（必须通过 Go → Python 调用）

### Python 禁止项

- ❌ 业务状态持久化（只返回计算结果）
- ❌ 直接操作数据库（由 Go 写入）
- ❌ 标签管理逻辑（由 Go 处理）
- ❌ 文件系统操作（只读计算）

---

## 新增 vs 修改组件

### 新增组件（v4.0）

| 组件                       | 职责                           | 位置                       |
| -------------------------- | ------------------------------ | -------------------------- |
| Python Sidecar 进程管理    | 启动、健康检查、优雅关闭       | `internal/service/process_manager.go` |
| Python 重复检测服务        | 精确哈希、感知哈希、聚类       | `services/python-dedupe/`   |
| 重复检测编排逻辑           | 调用 Python、结果落库         | `internal/service/duplicate_service.go` |
| Windows Photos 风格 UI     | NavigationView、filmstrip      | `flutter_app/lib/screens/`  |
| Windows 打包脚本           | 合并多进程到单一分发          | `deploy/package.ps1`        |

### 修改组件（v4.0）

| 组件                       | 修改内容                       | 理由                       |
| -------------------------- | ------------------------------ | -------------------------- |
| Go duplicate_service.go    | 从直接计算改为调用 Python      | 职责拆分、利用 Python 库   |
| Flutter 图库页面           | NavigationView 布局、filmstrip | Windows Photos 风格重构    |
| Go main.go                 | 新增 Python 进程启动逻辑       | 主控编排职责               |
| SQLite 数据库              | 新增 duplicate_groups 表       | 存储重复检测结果           |

### 保持不变组件

| 组件                       | 保持不变原因                   |
| -------------------------- | ------------------------------ |
| Go import_service.go       | 导入逻辑已稳定，无需重构       |
| Go tag_service.go          | 标签管理职责清晰，无需拆分     |
| Go database/repository     | 数据访问层职责单一，无需改动   |
| SQLite 主数据库            | 现有表结构稳定，仅新增重复检测表 |

---

## 构建顺序推荐

### Phase 1: Python 侧车基础（基础设施）

**目标:** 建立 Python 进程生命周期管理和基础通信框架。

**组件:**
- `internal/service/process_manager.go`（进程管理）
- `services/python-dedupe/app.py`（FastAPI 入口）
- Go → Python 基础 HTTP 通信（健康检查）

**验证:** Go 能成功启动 Python、读取端口、建立连接、优雅关闭。

---

### Phase 2: 重复检测计算迁移（核心计算）

**目标:** 将重复检测逻辑从 Go 迁移到 Python，验证计算侧车模式。

**组件:**
- `services/python-dedupe/algorithms/exact_hash.py`（精确哈希）
- `services/python-dedupe/algorithms/perceptual_hash.py`（感知哈希）
- `services/python-dedupe/algorithms/clustering.py`（聚类）
- `internal/service/duplicate_service.go`（编排逻辑）

**验证:** Python 能计算哈希、聚类重复组，Go 能写入数据库。

---

### Phase 3: Flutter 桌面重构（用户体验）

**目标:** 重构 Windows 桌面 UI 为 Windows Photos 风格，集成新的重复检测流程。

**组件:**
- `flutter_app/lib/screens/gallery/`（NavigationView 布局）
- `flutter_app/lib/screens/viewer/`（filmstrip 查看器）
- `flutter_app/lib/screens/duplicates/`（重复组确认界面）
- WebSocket 进度推送集成

**验证:** 用户可通过新界面发起重复检测、查看进度、确认保留策略。

---

### Phase 4: Windows 打包集成（部署）

**目标:** 集成 Flutter/Go/Python 打包流程，生成单一分发目录。

**组件:**
- `deploy/package.ps1`（打包脚本）
- `deploy/installer/setup.iss`（Inno Setup 配置）
- `services/python-dedupe/requirements.txt`（依赖管理）

**验证:** 用户无需安装 Python/Go，解压即用，进程启动顺序正确。

---

### Phase 5: 优雅关闭与错误恢复（稳定性）

**目标:** 补齐进程生命周期治理、心跳检测、错误恢复策略。

**组件:**
- `internal/service/process_manager.go`（心跳检测、自动重启）
- `flutter_app/lib/services/shutdown_service.dart`（优雅关闭）
- Go Python 调用超时与重试逻辑

**验证:** 进程异常退出时能自动恢复，关闭时无僵尸进程。

---

## 反模式

### 反模式 1: Python 直接操作数据库

**错误做法:** Python 侧车直接写入 SQLite，管理任务状态。

**为何错误:** 破坏职责边界，Python 应保持无状态，Go 主控应统一管理数据。

**正确做法:** Python 仅返回计算结果（哈希值、重复组列表），由 Go 写入数据库。

---

### 反模式 2: Flutter 直接调用 Python

**错误做法:** Flutter 通过 HTTP 直接调用 Python Sidecar API。

**为何错误:** 绕过 Go 主控，失去任务编排、状态管理、错误恢复能力。

**正确做法:** Flutter → Go → Python，所有请求通过 Go 编排。

---

### 反模式 3: 硬编码端口

**错误做法:** Python 固定监听端口 5000，Go 固定监听端口 8080。

**为何错误:** 端口冲突风险，多实例运行困难，无法适配用户环境。

**正确做法:** Python 随机端口（`127.0.0.1:0`），通过 stdout 传递实际端口，Go 动态连接。

---

### 反模式 4: 忽略 Windows 打包生命周期

**错误做法:** 假设用户已安装 Python，或假设 Flutter 启动顺序自动正确。

**为何错误:** Windows 用户环境多样，进程启动顺序错误会导致应用无法运行。

**正确做法:** 使用 PyInstaller 打包 Python 为独立 exe，Flutter 拉起 Go，Go 拉起 Python。

---

## 职责边界总结

| 层级     | 保持职责                             | 新增职责                     | 拆出职责           |
| -------- | ------------------------------------ | ---------------------------- | ------------------ |
| Flutter  | UI 渲染、状态管理                    | Windows Photos 风格重构      | 无                 |
| Go       | 数据库、导入、标签管理、任务编排     | Python 进程生命周期管理      | 重复检测计算       |
| Python   | 无（新增层）                         | 图像哈希、重复检测、聚类     | 无（纯计算层）     |

---

## 数据来源

- **PyInstaller 文档** (HIGH confidence) — https://github.com/pyinstaller/pyinstaller
- **ImageHash 文档** (HIGH confidence) — https://github.com/johannesbuchner/imagehash  
- **Windows 照片程序研究** (HIGH confidence) — 内部设计文档 `Windows11-Photos-App-ACG-Gallery-Research.md`
- **技术方案文档** (HIGH confidence) — 内部架构文档 `ACG-Gallery-Go-Python-Flutter-Technical-Plan.md`
- **Go 优雅关闭模式** (HIGH confidence) — https://medium.com/@rafalroppel/graceful-shutdown-in-go-explained-signals-contexts-and-the-correct-shutdown-sequence-f24fd9ef8fac
- **Windows 打包最佳实践** (HIGH confidence) — https://ahmedsyntax.com/pyinstaller-onefile/

---

## 总结

v4.0 架构的核心是**职责拆分而非重写**：保持 Go 主控编排能力，将计算密集型任务迁移到 Python 侧车，重构 Flutter 桌面 UI 为 Windows Photos 风格，并通过 Windows 打包集成确保用户无需配置环境即可使用。

**关键决策:**
1. Go 继续负责数据库、任务编排、进程管理
2. Python 仅负责图像哈希计算，不涉及业务状态
3. Flutter 不直接访问文件系统或数据库
4. Windows 打包采用 PyInstaller + Go build + Flutter build 的合并模式

**构建顺序:** 先建立 Python 进程管理基础设施 → 迁移重复检测计算 → 重构 Flutter UI → 集成打包 → 补齐错误恢复策略。

---

*架构研究用于: ACG 图库 v4.0 Windows Photos 风格重构与计算层拆分*
*调研日期: 2026-04-03*
