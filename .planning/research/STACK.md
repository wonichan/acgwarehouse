# 技术栈研究

**领域:** Windows 桌面图库应用 + Python 计算侧车
**里程碑:** v4.0 Windows Photos 风格重构与计算层拆分
**调研日期:** 2026-04-03
**置信度:** HIGH

## 执行摘要

v4.0里程碑的核心是**双线并行重构**：一是Flutter桌面端UI重构为Windows Photos风格体验，二是引入Python计算侧车承接图像哈希与重复检测任务。技术栈新增聚焦于**Python HTTP服务框架**、**Flutter Fluent材质增强**、**Go进程生命周期管理**、**Windows打包工具链**。所有新增项都围绕"保持单机部署路径可运行"与"计算层无状态边界"两大约束。

## 推荐技术栈

### 核心技术

| 技术 | 版本 | 用途 | 推荐原因 |
|------|------|------|----------|
| **Go** | 1.25+ | 主控层：业务编排、进程管理、数据库、API | 已有32k行代码基础，高性能适合文件操作，纯Go SQLite驱动避免CGO，适合Windows打包 |
| **Python** | 3.11+ | 计算层：图像哈希、重复检测、相似度计算 | Python 生态的图像算法库成熟（Pillow, imagehash），快速原型开发，适合计算密集型任务 |
| **Flutter** | 3.9.2+ | UI层：桌面界面渲染、状态管理 | 已有Flutter代码基础，fluent_ui提供原生Windows体验，跨端复用能力保留 |
| **SQLite** | ncruces/go-sqlite3 | 本地数据库：图片索引、标签、批次 | 纯Go驱动无CGO依赖，单机部署简化，已有完整schema和业务层 |
| **FastAPI** | 0.115+ | Python HTTP 服务框架 | 现代异步框架，自动OpenAPI文档，lifespan事件支持启动/关闭生命周期，性能优于Flask |
| **uvicorn** | 0.24+ | ASGI 服务器 | FastAPI官方推荐，生产级性能，支持单进程部署（桌面侧车场景） |
| **PyInstaller** | 6.0+ | Python 打包工具 | 将Python服务打包为单exe文件，用户无需安装Python环境，Windows桌面分发标准方案 |

### Flutter 桌面增强库

| 库 | 版本 | 用途 | 使用场景 |
|-----|------|------|----------|
| **fluent_ui** | ^4.9.1 (已有) | Windows Fluent Design UI组件 | 已在项目中使用，提供NavigationView侧边导航、主题系统、Windows原生控件样式 |
| **window_manager** | ^0.5.1 | 窗口生命周期管理 | 已有v0.4.3，建议升级。支持窗口事件监听（关闭、聚焦）、防止关闭确认对话框、进程间协调 |
| **flutter_acrylic** | ^1.1.0 | Windows 11 Mica/Acrylic材质效果 | 新增。实现透明背景、Mica材质（融合壁纸色）、Acrylic磨砂玻璃效果，符合Windows 11设计规范 |
| **go_router** | ^12.0.0 | 路由管理 | 新增。桌面端多页面导航（图库→查看器→设置），支持深层链接，配合NavigationView使用 |
| **flutter_riverpod** | ^2.4.0 | 状态管理 | 新增或替代现有provider。推荐用于复杂桌面应用状态（图库筛选、任务进度、批次监控） |
| **photo_view** | ^0.14.0 | 图片查看器手势交互 | 新增。支持缩放、拖拽、双击放大，用于filmstrip式图片查看器 |
| **desktop_drop** | ^0.4.0 | 拖拽文件导入 | 新增。支持拖拽文件夹/图片到应用窗口触发导入 |

### Python 计算层依赖

| 库 | 版本 | 用途 | 使用场景 |
|-----|------|------|----------|
| **Pillow** | 10.0+ | 图像加载与基础处理 | 所有图像操作的基础，打开图片文件、缩略图生成、格式转换 |
| **imagehash** | 4.3+ | 感知哈希算法库 | pHash（感知哈希）、dHash（差异哈希）、average_hash，用于相似图片检测，汉明距离计算 |
| **numpy** | 1.24+ | 数值计算 | 图像矩阵操作、哈希向量计算、聚类算法辅助 |
| **fastapi** | 0.115+ | HTTP服务框架 | 提供REST API给Go调用，自动生成文档，支持健康检查、任务状态接口 |
| **uvicorn** | 0.24+ | ASGI服务器 | 运行FastAPI应用，单进程模式适合桌面侧车，随机端口避免冲突 |

### Go 主控层新增需求

| 库/模式 | 用途 | 实现方式 |
|---------|------|----------|
| **os/exec** | 启动Python进程 | `exec.Command("dedupe_service.exe")`，读取stdout获取随机端口 |
| **context + signal** | 优雅关闭 | 监听SIGTERM（Linux）/实现HTTP shutdown endpoint（Windows），超时后强制Kill |
| **HTTP client** | 调用Python服务 | 标准net/http包，localhost通信，超时控制，重试策略 |
| **心跳检测** | Python进程健康监控 | 定时HTTP ping `/health`，超时自动重启，失败日志记录 |

### Windows 打包工具链

| 工具 | 用途 | 配置要点 |
|------|------|----------|
| **PyInstaller** | Python → exe | `pyinstaller --onefile --name dedupe_service app.py`，单文件分发，包含所有依赖 |
| **go build** | Go → exe | `go build -o acg_backend.exe .`，纯Go编译无CGO依赖，Windows兼容 |
| **flutter build windows** | Flutter → Windows app | `flutter build windows --release`，生成build/windows/x64/runner/Release/ |
| **合并打包** | 最终分发包 | 复制三exe到同一目录，可选zip或NSIS安装器 |

## 安装与配置

### Python 计算侧车

```bash
# 创建独立虚拟环境
cd services/python-dedupe
python -m venv venv
source venv/bin/activate  # Linux/Mac
venv\Scripts\activate     # Windows

# 安装依赖
pip install fastapi==0.115.0
pip install uvicorn==0.24.0
pip install Pillow==10.0.0
pip install imagehash==4.3.1
pip install numpy==1.24.0

# 打包为exe（Windows）
pip install pyinstaller==6.0.0
pyinstaller --onefile --name dedupe_service app.py
# 输出: dist/dedupe_service.exe
```

### Flutter 桌面依赖

```yaml
# flutter_app/pubspec.yaml 新增依赖
dependencies:
  # 已有（保留）
  fluent_ui: ^4.9.1
  window_manager: ^0.5.1  # 升级版本
  provider: ^6.1.1        # 保留现有provider，后续可选迁移到riverpod
  
  # 新增
  flutter_acrylic: ^1.1.0  # Mica/Acrylic材质
  go_router: ^12.0.0       # 路由管理
  photo_view: ^0.14.0      # 图片查看手势
  desktop_drop: ^0.4.0     # 拖拽导入
  web_socket_channel: ^2.4.0  # WebSocket实时进度（已有http，新增ws）
  
  # 可选（状态管理重构）
  flutter_riverpod: ^2.4.0  # 如需更强大的状态管理
  freezed_annotation: ^2.4.0  # 配合riverpod不可变状态
```

```bash
cd flutter_app
flutter pub get
```

### Go 主控层

```bash
# 已有依赖（无需新增外部库，使用标准库）
# go.mod 保持现有：
# - gin: HTTP框架
# - ncruces/go-sqlite3: SQLite驱动
# - goimagehash: 已有感知哈希（Go实现），Python侧车将替代部分职责

# 新增：进程管理模式（标准库）
import (
    "os/exec"
    "context"
    "os/signal"
    "syscall"
    "net/http"
    "time"
)
```

## 考虑过的替代方案

| 推荐方案 | 替代方案 | 替代方案适用场景 |
|----------|----------|------------------|
| **HTTP (localhost)** | gRPC | gRPC适合高性能服务间通信，但桌面侧车场景localhost环回开销可忽略，HTTP调试简单、跨语言通用 |
| **HTTP (localhost)** | FFI/CGO | FFI理论上性能最高，但Go调用Python需处理GIL、交叉编译痛苦、崩溃风险高，HTTP稳定性更好 |
| **FastAPI** | Flask | Flask同步框架，FastAPI异步+自动文档+类型检查，现代项目首选 |
| **FastAPI** | 自建socket | 不需要HTTP框架的极简场景，但FastAPI提供健康检查、错误处理、文档，生产更稳健 |
| **PyInstaller --onefile** | Nuitka | Nuitka编译为真正的机器码，性能更好，但PyInstaller生态成熟、配置简单、打包速度快 |
| **PyInstaller --onefile** | Docker容器 | 服务器部署场景，但Windows桌面分发要求exe，用户不安装Docker |
| **flutter_acrylic** | bitsdojo_window | bitsdojo_window也支持透明窗口，但flutter_acrylic专门针对Fluent材质，与fluent_ui集成更好 |
| **SQLite** | PostgreSQL | PostgreSQL适合高并发生产环境，但本里程碑仍聚焦单机部署，SQLite简化运维 |
| **Go进程管理** | 外部supervisor | supervisor适合服务器多服务管理，但桌面应用需自包含，Go主控进程更自然 |

## 不应使用的技术

| 避免使用 | 原因 | 推荐替代 |
|----------|------|----------|
| **CGO调用Python** | 交叉编译痛苦（Windows/Linux/Mac三平台需各自编译），Python GIL导致崩溃，调试困难 | HTTP localhost通信，简单可靠 |
| **gRPC** | 需要proto文件生成代码，增加构建复杂度，localhost场景性能收益微乎其微 | HTTP REST API，调试友好 |
| **Docker for Windows分发** | 要求用户安装Docker Desktop，非技术用户门槛高，不符合"解压即用"桌面应用理念 | PyInstaller打包exe + Flutter/Go原生编译 |
| **硬编码端口（8080/5000）** | 端口冲突风险高，用户环境可能已有服务占用 | 随机端口（127.0.0.1:0），stdout传递端口信息 |
| **Python持久化数据库** | 违反"计算层无状态"边界，职责混乱，增加调试难度 | Python只做计算，Go负责所有持久化 |
| **Python直接文件操作** | 权限与安全边界不清，进程隔离失效 | Python只读计算，Go负责文件移动/删除 |
| **Windows任务计划器启动Python** | 需要管理员权限配置，不符合绿色软件理念 | Flutter启动Go → Go启动Python，自包含进程树 |
| **PostgreSQL迁移** | 本里程碑明确约束"SQLite主路径"，迁移是破坏性重构 | 保持SQLite，未来里程碑再评估 |
| **Flask同步框架** | FastAPI已是Python现代Web框架标准，异步性能更好，自动文档 | FastAPI + uvicorn异步ASGI |
| **混合状态管理（provider+riverpod）** | 两个状态管理框架共存增加复杂度，迁移成本高 | 统一使用一种（保留provider或迁移到riverpod） |

## 按里程碑阶段的技术栈变化

### v4.0 之前（已有）

```
Go主控（18k行）
├── Gin HTTP API
├── SQLite (ncruces纯Go驱动)
├── 图片扫描服务
├── AI标签编排（调用云端API）
├── 重复检测（goimagehash，Go实现）
└── 导入后任务平台

Flutter前端（13k行）
├── fluent_ui 侧边导航
├── provider 状态管理
├── cached_network_image 图片加载
└── window_manager 窗口控制
```

### v4.0 新增（本里程碑）

```
新增Python计算侧车（~1k行）
├── FastAPI HTTP服务
│   ├── /health 健康检查
│   ├── /hash 计算pHash/dHash
│   └── /detect 批量重复检测
├── imagehash感知哈希算法
├── Pillow图像加载
├── PyInstaller打包为exe

Go主控新增
├── 进程管理模块
│   ├── 启动Python进程（exec.Command）
│   ├── 读取随机端口（stdout）
│   ├── 心跳检测（定时HTTP ping）
│   └── 优雅关闭（Windows HTTP endpoint / Linux SIGTERM）
├── 重复检测编排重构
│   ├── 调用Python服务替代goimagehash
│   ├── 批次管理保留在Go
│   └── 结果写入SQLite保留在Go

Flutter桌面增强
├── flutter_acrylic Mica材质
├── NavigationView重构（Windows Photos风格）
├── go_router路由系统
├── photo_view图片查看手势
├── desktop_drop拖拽导入
├── WebSocket实时进度推送
└── Filmstrip组件（底部缩略图条）
```

### v4.0 之后（未来里程碑可能）

```
Python侧车扩展
├── 图像质量评估（模糊检测）
├── 图像内容分析（颜色分布、主体检测）
├── 本地AI模型推理（可选）
└── 更多计算任务外移

Go主控演进
├── 计算任务统一接口（ComputeTask抽象）
├── 多计算服务编排（Python + 可能的其他侧车）
└── 进程池管理（如需并行计算）

Flutter跨端复用
├── Android端复用相同Go backend
├── Web端复用相同API
└── 桌面特有体验保留（Windows Fluent）
```

## 版本兼容性

| 包组合 | 兼容性 | 注意事项 |
|--------|--------|----------|
| **FastAPI 0.115 + uvicorn 0.24** | ✅ 兼容 | uvicorn支持FastAPI最新lifespan事件 |
| **Pillow 10.0 + imagehash 4.3** | ✅ 兼容 | imagehash依赖Pillow，版本匹配良好 |
| **numpy 1.24 + Pillow 10.0** | ✅ 兼容 | Pillow内部使用numpy数组，无冲突 |
| **fluent_ui 4.9 + flutter_acrylic 1.1** | ✅ 兼容 | flutter_acrylic提供材质，fluent_ui提供组件，互不干扰 |
| **window_manager 0.5 + fluent_ui 4.9** | ✅ 兼容 | 升级到0.5版本修复事件监听稳定性问题 |
| **go_router 12.0 + fluent_ui NavigationView** | ✅ 兼容 | go_router管理路由状态，NavigationView展示导航UI |
| **Flutter 3.9.2 + 所有新依赖** | ✅ 兼容 | 新依赖支持Flutter 3.x |
| **PyInstaller 6.0 + Python 3.11** | ✅ 兼容 | PyInstaller 6.0支持Python 3.11，打包稳定 |
| **Go 1.25 + Python 3.11进程通信** | ✅ 兼容 | HTTP通信跨语言无版本依赖 |
| **SQLite ncruces/go-sqlite3 + Go 1.25** | ✅ 兼容 | 纯Go驱动，Go版本升级无影响 |

## 集成要点

### Go ↔ Python 进程生命周期

**启动流程：**
```
Flutter App 启动
    ↓
Flutter 拉起 Go: exec.Command("acg_backend.exe")
    ↓
Go 启动 Python: exec.Command("dedupe_service.exe")
    ↓
Python 打印端口到 stdout: "PORT:12345"
    ↓
Go 读取 stdout，解析端口
    ↓
Go 建立HTTP连接: http://127.0.0.1:12345
    ↓
Go 启动自身API服务: localhost:8080
    ↓
Flutter 连接 localhost:8080
    ↓
系统就绪
```

**优雅关闭流程：**
```
用户关闭 Flutter 窗口
    ↓
Flutter 发送 POST /shutdown 到 Go
    ↓
Go 关闭 Python HTTP连接池
    ↓
Go 发送 POST /shutdown 到 Python (Windows无SIGTERM)
    ↓
Python 关闭资源，退出进程
    ↓
Go 关闭 SQLite 连接
    ↓
Go 退出进程
    ↓
Flutter 完全退出
```

**Windows特定处理：**
- Windows没有SIGTERM信号，必须通过HTTP endpoint `/shutdown`通知Python退出
- Go使用`cmd.Process.Kill()`作为强制终止（超时后）
- Python实现lifespan shutdown事件清理资源

### Flutter ↔ Go 实时通信

**HTTP API（已有）：**
- 图片CRUD、标签管理、导入任务
- RESTful设计，JSON响应

**WebSocket（新增）：**
- 导入进度实时推送（批次百分比、当前图片）
- Python计算任务状态（检测进度、失败通知）
- 避免前端轮询，提升响应体验

### Python计算侧车边界

**Python职责（明确）：**
- ✅ 计算图像哈希（pHash, dHash, SHA256）
- ✅ 批量重复检测与聚类
- ✅ 相似度计算（汉明距离）
- ✅ 保留建议算法（分辨率/清晰度评估）
- ✅ 返回计算结果（JSON），不做持久化

**Python禁止（明确）：**
- ❌ 直接写入SQLite
- ❌ 文件系统操作（移动/删除文件）
- ❌ 标签管理逻辑
- ❌ 业务状态持久化
- ❌ 调用AI标签服务（Go负责）

**Go编排职责：**
- ✅ 获取待检测图片列表
- ✅ 批次任务调度
- ✅ 调用Python计算
- ✅ 结果写入SQLite
- ✅ 文件移动执行（去重操作）
- ✅ WebSocket通知前端

## 现有代码约束

### Go已实现能力（避免重复）

| 已有实现 | v4.0处理方式 |
|----------|--------------|
| `goimagehash` 感知哈希 | **迁移到Python侧车**，Go改为调用Python服务，保留fallback逻辑（Python不可用时） |
| 图片扫描服务 | **保留在Go**，Python不参与文件扫描 |
| AI标签编排 | **保留在Go**，Python不调用云端API |
| 导入后任务平台 | **保留在Go**，批次模型不变，Python只承担计算任务节点 |
| SQLite schema | **保留不变**，新增duplicate_groups/candidates表（已有） |
| Gin HTTP API | **保留扩展**，新增Python进程管理endpoints |

### Flutter已实现组件（复用）

| 已有组件 | v4.0处理方式 |
|----------|--------------|
| fluent_ui NavigationView | **重构布局**，调整为Windows Photos风格（侧边栏项目、顶部工具栏） |
| provider状态管理 | **保留或逐步迁移到riverpod**，避免混合框架 |
| 图片网格视图 | **重构为瀑布流+网格双模式**，参考Windows Photos River/Square模式 |
| cached_network_image | **保留**，用于缩略图加载 |
| window_manager | **升级版本并新增事件监听**，处理关闭确认 |

## 数据流与职责划分

```
┌─────────────┐
│ Flutter UI  │ HTTP + WebSocket
│  (展示层)   │ ←───────────────→
└─────────────┘
                  │
                  ↓
            ┌──────────────┐
            │   Go 主控    │ HTTP (localhost)
            │  (编排层)    │ ←───────────────→
            └──────────────┘
                  │                │
                  ↓                ↓
            ┌──────────┐    ┌─────────────┐
            │  SQLite  │    │ Python侧车  │
            │ (持久化) │    │  (计算层)   │
            └──────────┘    └─────────────┘
```

**数据流示例（重复检测）：**
1. Flutter: 用户点击"开始检测" → POST `/duplicates/scan`
2. Go: 获取待检测图片路径列表
3. Go: POST `http://127.0.0.1:12345/detect` → Python
4. Python: 计算哈希，聚类相似组 → 返回JSON
5. Go: 写入duplicate_groups表
6. Go: WebSocket推送进度 → Flutter实时更新
7. Flutter: 显示重复组确认界面
8. Flutter: 用户选择保留策略 → POST `/duplicates/apply`
9. Go: 执行文件移动（不调用Python）
10. Go: 更新images状态 → 完成

## 研究来源

- **Context7 /fastapi/fastapi** — FastAPI lifespan事件、启动/关闭生命周期、健康检查实现
- **Context7 /johannesbuchner/imagehash** — pHash/dHash算法、汉明距离计算、相似检测实现
- **Context7 /bdlukaa/fluent_ui** — NavigationView组件、Mica材质类、Windows Fluent Design实现
- **Context7 /leanflutter/window_manager** — 窗口事件监听、关闭确认对话框、生命周期管理
- **WebSearch PyInstaller打包** — Python打包exe单文件分发、FastAPI服务打包实践
- **WebSearch Go进程管理** — Windows优雅关闭、os/exec模式、signal handling
- **WebSearch flutter_acrylic** — Mica/Acrylic材质实现、Windows 11设计系统集成
- **官方文档 Pillow** — 图像处理基础库（Python生态标准）
- **官方文档 uvicorn** — ASGI服务器配置、单进程生产部署
- **本地规划文档** — `.planning/PROJECT.md`, `Windows11-Photos-App-ACG-Gallery-Research.md`, `ACG-Gallery-Go-Python-Flutter-Technical-Plan.md` (项目边界与架构决策)

---
*技术栈研究：ACG图库v4.0 Windows桌面重构与Python计算侧车*
*调研日期：2026-04-03*
*置信度：HIGH（基于Context7官方文档 + 项目规划文档 + 已有代码基础）*
