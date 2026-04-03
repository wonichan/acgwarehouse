# v4.0 里程碑研究综合总结

**项目:** ACG 图库 v4.0 Windows Photos 风格重构与计算层拆分
**领域:** Windows 桌面图库应用 + Python 计算侧车（棕地重构）
**调研日期:** 2026-04-03
**置信度:** HIGH（整体）

---

## 执行摘要

v4.0里程碑的核心是**双线并行重构**：一是将现有Flutter桌面端重构为Windows Photos风格体验（侧边导航、瀑布流、独立查看器），二是引入Python计算侧车承接图像哈希与重复检测任务，形成Flutter(UI) + Go(主控) + Python(计算)的三层架构。这是典型的棕地重构场景，需在32k行现有Go代码与13k行Flutter代码基础上演进，而非从零构建。

**推荐的技术路径：** Go继续承担数据库管理、任务编排、进程生命周期控制；Python作为无状态计算侧车，仅处理图像哈希算法，不涉及业务状态持久化；Flutter负责UI渲染与状态管理，不直接访问文件系统或数据库。进程间通信采用HTTP localhost方式，避免gRPC/CGO的复杂度。Windows打包采用PyInstaller将Python打包为exe，配合Go编译与Flutter build windows，合并为单一分发目录。

**最关键风险：** 进程生命周期失控（僵尸进程）、端口冲突（硬编码8080）、Python侧车故障时无降级路径、SQLite并发阻塞（未启用WAL）、UI重构用户体验断裂（无Feature Flag渐进推出）。预防策略集中在Phase 1（架构边界建立）与Phase 2（Python侧车落地），需实现随机端口分配、健康检查机制、降级路径、WAL模式启用、Feature Flag基础设施。

---

## 核心发现

### 推荐技术栈（来自 STACK.md）

技术栈新增聚焦于**Python HTTP服务框架**、**Flutter Fluent材质增强**、**Go进程生命周期管理**、**Windows打包工具链**，所有新增项围绕"保持单机部署路径可运行"与"计算层无状态边界"两大约束。

**核心技术（必需）：**
- **Go 1.25+** — 主控层（业务编排、进程管理、数据库） — 已有32k行代码基础，纯Go SQLite驱动避免CGO，适合Windows打包
- **Python 3.11+** — 计算层（图像哈希、重复检测） — Python生态图像算法库成熟（Pillow, imagehash），适合计算密集型任务
- **Flutter 3.9.2+** — UI层（桌面界面渲染） — 已有Flutter代码基础，fluent_ui提供原生Windows体验
- **FastAPI 0.115+** — Python HTTP服务框架 — 现代异步框架，自动OpenAPI文档，lifespan事件支持启动/关闭生命周期
- **PyInstaller 6.0+** — Python打包工具 — 将Python服务打包为单exe文件，用户无需安装Python环境
- **SQLite ncruces/go-sqlite3** — 本地数据库 — 纯Go驱动无CGO依赖，单机部署简化，已有完整schema

**Flutter桌面增强库（新增）：**
- **flutter_acrylic ^1.1.0** — Windows 11 Mica/Acrylic材质效果 — 新增，实现透明背景、融合壁纸色
- **go_router ^12.0.0** — 路由管理 — 新增，桌面端多页面导航（图库→查看器→设置）
- **flutter_riverpod ^2.4.0** — 状态管理 — 新增，复杂桌面应用状态（图库筛选、任务进度）
- **photo_view ^0.14.0** — 图片查看器手势交互 — 新增，缩放、拖拽、双击放大
- **desktop_drop ^0.4.0** — 拖拽文件导入 — 新增，拖拽文件夹/图片触发导入
- **window_manager ^0.5.1** — 窗口生命周期管理 — 升级版本，支持窗口事件监听、关闭确认

**Python计算层依赖（必需）：**
- **Pillow 10.0+** — 图像加载与基础处理 — 所有图像操作的基础
- **imagehash 4.3+** — 感知哈希算法库 — pHash、dHash、汉明距离计算
- **numpy 1.24+** — 数值计算 — 图像矩阵操作、哈希向量计算

**关键技术决策：**
- HTTP localhost通信（非gRPC/CGO） — 简化跨语言集成，调试友好
- 随机端口分配（127.0.0.1:0） — 避免端口冲突，stdout传递端口信息
- PyInstaller --onefile打包 — 单文件分发，Windows桌面标准方案
- SQLite WAL模式启用 — 支持并发读写，避免阻塞

---

### 特性期望与优先级（来自 FEATURES.md）

v4.0为棕地重构，需明确依赖现有系统行为（图片扫描、AI标签、搜索、收藏夹、导入任务平台），并聚焦高风险重构路径。

**必需特性（Table Stakes，用户默认期望具备）：**
- **侧边导航栏** — Windows Photos标准范式，可收起设计（展开240px/收起48px），支持收藏、文件夹、导入任务监控入口
- **顶部工具栏** — 固定高度52px，包含搜索栏、导入按钮、设置入口，核心交互入口
- **瀑布流视图** — Windows Photos River模式，保留原始宽高比，需适配虚拟滚动与缩略图缓存
- **图片缩略图虚拟滚动** — 大图库（10k+图片）流畅浏览必需，优先加载可见区域
- **搜索栏（文件名/标签）** — 快速定位目标图片，现有Go+SQLite FTS5已支持，需重构前端集成到顶部工具栏
- **图片导入流程** — 现有Go扫描服务已支持，需重构前端导入入口（拖拽导入、文件夹选择）
- **Python计算侧车进程管理** — Go负责启动、健康检查、优雅关闭Python进程，心跳检测、端口冲突避免、僵尸进程清理

**竞争优势特性（Differentiators，非必需但有价值）：**
- **智能重复检测（精确+感知哈希）** — 解决"相同图片不同格式/分辨率"痛点，Python侧车承接SHA256精确重复 + pHash/dHash感知哈希 + Union-Find聚类分组
- **相似图片分组与推荐保留** — 从"检测重复"升级到"推荐最佳版本"，Python计算层提供质量评估信号（锐度、曝光、EXIF完整度）
- **侧边导航"导入任务监控"入口** — 将后台运营工具收敛到桌面主体验，v3.0任务平台前端集成
- **桌面窗口打包为单文件分发** — 用户无需安装Python/Go，解压即用，降低部署门槛

**应避免特性（Anti-Features）：**
- **实时全库重复检测** — O(n²)计算量巨大，阻塞UI数小时 → 触发式分批检测，WebSocket推送进度
- **Python直接持久化到数据库** — 违反架构边界，职责混乱 → Go编排落库，Python只返回计算结果
- **硬编码Python服务端口** — 端口冲突风险高 → 随机端口+stdout传递
- **图片编辑功能** — v4.0聚焦浏览/查看/计算重构 → 延后到v5.0+

**MVP必需（Launch With，v4.0验证目标）：**
- 侧边导航栏基础框架、顶部工具栏、瀑布流视图、Python计算侧车进程管理、智能重复检测（精确+感知哈希）、侧边导航"导入任务监控"入口、Go↔Python HTTP通信契约、桌面窗口打包基础验证

**验证后添加（v4.x迭代）：**
- 独立查看器窗口（P2）、底部胶片条（P2）、相似图片推荐保留（P2）、Python侧车故障诊断（P2）、侧边栏文件夹树形视图（P2）、方块网格视图模式（P3）

**高风险重构路径：**
1. **Python侧车迁移重复检测** — 需完整实现进程管理 + HTTP契约 + 批次编排 + 结果落库
2. **Flutter桌面UX重构** — 需重写NavigationView、顶部工具栏、图库布局，保持与Go API兼容

---

### 架构方案（来自 ARCHITECTURE.md）

v4.0架构的核心是**职责拆分而非重写**：保持Go主控编排能力，将计算密集型任务迁移到Python侧车，重构Flutter桌面UI为Windows Photos风格。

**三层架构系统概览：**
```
Flutter Desktop UI（展示层）
  └── HTTP + WebSocket (localhost) ↕
Go Backend 主控层（编排层）
  ├── SQLite/FTS5（数据库管理）
  ├── 任务平台编排（批次/队列控制）
  ├── 进程生命周期（Python拉起）
  └── HTTP (localhost) ↕
Python Sidecar 计算层（计算层）
  ├── 精确重复检测（MD5/SHA1）
  ├── 视觉相似检测（pHash/dHash）
  └── 重复组聚类（汉明距离分组）
```

**组件职责边界：**
- **Flutter Desktop** — 界面渲染、状态管理、用户交互（禁止直接文件系统操作、数据库读写、AI模型调用）
- **Go Backend** — 业务逻辑、数据库、任务编排、进程管理（新增：Python进程生命周期管理）
- **Python Sidecar** — 图像哈希、重复检测、相似度计算（禁止业务状态持久化、数据库操作、文件系统操作）

**推荐项目结构（新增）：**
- `services/python-dedupe/` — Python计算侧车（新增），包含FastAPI入口、算法模块、依赖管理
- `internal/service/process_manager.go` — Python进程管理（新增），启动、健康检查、优雅关闭
- `internal/service/duplicate_service.go` — 重复检测编排（修改），从直接计算改为调用Python
- `flutter_app/lib/screens/` — Windows Photos风格UI（重构），NavigationView、filmstrip查看器
- `deploy/package.ps1` — Windows打包脚本（新增），合并多进程到单一分发目录

**核心架构模式：**
1. **主控编排模式（Go主控）** — Go作为系统主控，负责业务逻辑、数据持久化、任务编排和Python进程生命周期管理
2. **计算侧车模式（Python计算层）** — Python作为无状态计算侧车，仅接收计算请求、执行图像算法、返回结果
3. **三阶段优雅关闭模式** — Flutter → Go → Python链式关闭流程，确保资源释放顺序正确（Windows无SIGTERM，需HTTP /shutdown endpoint）

**Windows打包与分发策略：**
- Flutter Desktop: `flutter build windows --release` → Release/flutter_app/
- Go Backend: `go build -o acg_backend.exe .` → Release/
- Python Sidecar: `pyinstaller --onefile --noconsole app.py` → dedupe_service.exe → Release/
- 启动顺序：Flutter启动 → 拉起Go exe → Go拉起Python exe → Go读取Python端口 → Go启动HTTP服务 → Flutter连接Go API

**数据流示例（重复检测）：**
1. Flutter: 用户点击"开始检测" → POST `/duplicates/scan`
2. Go: 获取待检测图片路径列表
3. Go: POST `http://127.0.0.1:{python_port}/detect` → Python
4. Python: 计算哈希，聚类相似组 → 返回JSON
5. Go: 写入duplicate_groups表
6. Go: WebSocket推送进度 → Flutter实时更新
7. Flutter: 显示重复组确认界面

---

### 关键陷阱（来自 PITFALLS.md）

棕地重构场景下的9个关键陷阱，预防策略集中在Phase 1与Phase 2。

**陷阱1：僵尸进程与生命周期失控（CRITICAL）**
- **问题：** Go启动Python后未正确清理，导致Python进程残留、端口占用、重启失败
- **根源：** Windows无SIGTERM信号，cmd.Start()后未调用cmd.Wait()，缺少进程监听
- **预防：** 必须调用cmd.Wait()防止僵尸进程；Windows优雅关闭先HTTP /shutdown，后Process.Kill()；实现心跳检测
- **预防阶段：** Phase 1（架构边界建立），Phase 2（Python侧车落地）

**陷阱2：PyInstaller打包陷阱与启动失败（CRITICAL）**
- **问题：** 用户下载后无法启动，缺少依赖DLL、ImportError、--noconsole模式下stdout=None导致写入失败
- **根源：** 依赖收集不完整、Windows路径差异、控制台模式问题、启动顺序阻塞
- **预防：** 修复标准流缺失（sys.stdout=open(os.devnull)）；正确资源路径处理（sys._MEIPASS）；端口打印必须flush=True；先用--onedir调试，再用--onefile
- **预防阶段：** Phase 3（Windows打包与分发），Phase 4（诊断与回退）

**陷阱3：端口冲突与启动协调失败（CRITICAL）**
- **问题：** 应用启动"端口已被占用"错误，启动顺序错乱，Python端口未传递给Go
- **根源：** 端口硬编码（8080/5000常见），启动顺序依赖未等待，无启动确认机制，无超时与重试
- **预防：** Python使用随机端口（--port=0），stdout打印端口供Go读取；Go等待Python打印端口（最多10秒）；Flutter通过环境变量获知Go端口
- **预防阶段：** Phase 1（架构边界建立），Phase 2（Python侧车落地）

**陷阱4：侧车故障时的降级策略缺失（CRITICAL）**
- **问题：** Python侧车崩溃时，整个应用无法使用，重复检测功能完全不可用
- **根源：** 硬依赖设计未设计降级路径，无故障隔离，缺少健康检查
- **预防：** 实现双路径设计（Python侧车 + Go fallback）；定期健康检查（每30秒）；自动恢复机制
- **预防阶段：** Phase 2（Python侧车落地），Phase 4（诊断与回退）

**陷阱5：SQLite并发访问数据一致性问题（HIGH）**
- **问题：** 多进程同时访问导致"database is locked"错误，写入阻塞UI
- **根源：** SQLite并发限制、缺少连接池、WAL模式未启用、事务边界不清
- **预防：** 启用WAL模式（PRAGMA journal_mode=WAL）；设置busy_timeout=5000；连接池配置（MaxOpenConns=5）；长事务快速提交
- **预防阶段：** Phase 1（架构边界建立），Phase 2（Python侧车落地）

**陷阱6：UI重构期间的用户体验断裂（HIGH）**
- **问题：** 新旧界面混用，导航逻辑不一致，瀑布流性能下降，无渐进式切换
- **根源：** 一次性替换无Feature Flag，缺少用户测试，瀑布流虚拟化未实现
- **预防：** Feature Flag双界面支持；用户设置迁移保留旧设置；渐进式推出（10% → 50% → 100%）；性能监控
- **预防阶段：** Phase 1（Feature Flag基础设施），Phase 5（桌面UI重构）

**陷阱7：数据库Schema迁移回滚失败（HIGH）**
- **问题：** Python侧车引入新表后迁移失败，数据库处于不一致状态，无法恢复旧版本
- **根源：** 缺少schema_version版本表，事务边界错误，无回滚脚本，破坏性变更
- **预防：** 创建schema_migrations版本表；每个迁移独立事务；记录回滚脚本；迁移后验证数据完整性
- **预防阶段：** Phase 2（Python侧车落地），Phase 4（诊断与回退）

**陷阱8：多进程错误归属与诊断困难（MEDIUM）**
- **问题：** 用户遇到错误无法定位问题来源，日志分散在三个进程难以关联
- **根源：** 错误信息丢失，缺少request ID跨进程传递，日志格式不一致
- **预防：** 每个请求分配ID，跨进程传递；Python返回详细错误结构（error + stack_trace）；统一日志格式
- **预防阶段：** Phase 2（错误传递协议），Phase 4（日志聚合+用户诊断工具）

**陷阱9：范围蔓延与里程碑边界失控（CRITICAL）**
- **问题：** "既然重构UI，不如顺便加编辑功能"、"Python既然能做哈希，不如也做AI标签"
- **根源：** 缺少Out of Scope清单，功能诱惑，缺少Phase优先级
- **预防：** ROADMAP.md明确Out of Scope；Phase内范围控制；遇到新需求记录到deferred.md，绝不当前Phase追加
- **预防阶段：** 里程碑规划阶段，每个Phase严格执行范围边界

**技术债模式（绝不可接受）：**
- 端口硬编码、Python崩溃后无降级、cmd.Start()不调用Wait()、PyInstaller首次打包用--onefile、SQLite未启用WAL、UI重构无Feature Flag、Schema迁移无回滚脚本、快捷径实现"顺便做X"

---

## 路线图规划启示

基于研究发现的Phase结构建议与顺序理由。

### Phase 1: Python侧车基础设施（架构边界建立）

**理由：** 进程生命周期管理是所有后续工作的基石。陷阱1（僵尸进程）、陷阱3（端口冲突）、陷阱5（SQLite并发）、陷阱6（Feature Flag基础设施）必须在Phase 1预防，否则后续Phase无法稳定推进。

**交付内容：**
- Python进程启动、健康检查、优雅关闭完整生命周期
- Go ↔ Python HTTP通信契约（健康检查endpoint）
- SQLite WAL模式启用、连接池配置
- Feature Flag基础设施（为Phase 5 UI重构准备）

**处理特性：** Python计算侧车进程管理（Table Stakes必需）

**预防陷阱：** 僵尸进程与生命周期失控、端口冲突与启动协调失败、SQLite并发问题、范围蔓延失控（建立范围边界）

**验证方式：** 关闭应用后任务管理器无残留进程；多实例可同时启动；并发读写无阻塞；Out of Scope清单明确

**置信度：** HIGH（Go官方文档明确要求cmd.Wait()，Windows进程管理有明确模式）

---

### Phase 2: 重复检测计算迁移（核心计算能力）

**理由：** Python侧车的首个计算能力验证。Phase 1基础设施验证后，Phase 2迁移重复检测逻辑，测试计算侧车模式、降级路径、错误传递协议。陷阱4（降级策略）、陷阱7（Schema迁移）、陷阱8（错误诊断）需在此Phase预防。

**交付内容：**
- Python精确哈希计算（SHA256/MD5）
- Python感知哈希计算（pHash/dHash）
- Python重复组聚类（Union-Find算法）
- Go编排逻辑（调用Python、结果落库）
- 健康检查 + 双路径降级设计
- Schema迁移安全流程（版本管理表、回滚脚本）

**处理特性：** 智能重复检测（精确+感知哈希）（Differentiators竞争优势）、Go↔Python HTTP通信契约（MVP必需）

**预防陷阱：** 侧车故障降级策略缺失、数据库Schema迁移回滚失败、多进程错误诊断困难

**使用技术栈：** Python Pillow + imagehash + numpy；Go process_manager.go + duplicate_service.go

**实现架构组件：** Python计算层（services/python-dedupe/）、Go编排层

**验证方式：** Python崩溃后功能降级可用；迁移失败可回滚；request ID可关联日志

**置信度：** HIGH（Sidecar模式标准要求降级路径，imagehash库文档完整）

---

### Phase 3: Flutter桌面UX重构骨架（用户体验起点）

**理由：** Phase 1+2验证架构稳定后，Phase 3重构Flutter桌面UI为Windows Photos风格骨架。侧边导航、顶部工具栏、瀑布流视图是用户可见的核心体验。陷阱6（UI体验断裂）需在此Phase预防（Feature Flag已在Phase 1准备）。

**交付内容：**
- 侧边导航栏基础框架（NavigationView可收起设计）
- 顶部工具栏（搜索栏+导入按钮+设置入口）
- 瀑布流视图（虚拟滚动优化、缩略图缓存策略）
- Feature Flag双界面支持（新旧界面可切换）
- WebSocket实时进度推送

**处理特性：** 侧边导航栏（Table Stakes必需）、顶部工具栏（Table Stakes必需）、瀑布流视图（Table Stakes必需）

**预防陷阱：** UI重构用户体验断裂（Feature Flag渐进推出，性能监控）

**使用技术栈：** Flutter fluent_ui + flutter_acrylic + go_router + flutter_riverpod + photo_view

**实现架构组件：** Flutter Desktop UI层

**验证方式：** Feature Flag可切换，瀑布流滚动性能通过（FPS监控）

**置信度：** MEDIUM（Flutter Windows桌面重构为高风险路径，需Phase级验证后提升信心）

---

### Phase 4: Windows打包与分发集成（部署门槛降低）

**理由：** Phase 1-3功能验证后，Phase 4集成Flutter/Go/Python打包流程，生成单一分发目录。陷阱2（PyInstaller打包）需在此Phase预防。

**交付内容：**
- PyInstaller打包Python为dedupe_service.exe
- Go编译为acg_backend.exe
- Flutter build windows生成Release目录
- 合并打包脚本（deploy/package.ps1）
- 启动顺序编排验证（Flutter → Go → Python）
- 多机器测试（无Python环境机器）

**处理特性：** 桌面窗口打包基础验证（Differentiators竞争优势）

**预防陷阱：** PyInstaller打包陷阱与启动失败（依赖收集完整、--noconsole处理、多机器测试）

**使用技术栈：** PyInstaller --onefile + go build + flutter build windows

**实现架构组件：** Windows打包流程

**验证方式：** 用户无需安装Python/Go，解压即用；多机器测试通过

**置信度：** HIGH（PyInstaller官方文档明确列出陷阱，打包流程有明确模式）

---

### Phase 5: 导入任务监控与侧车诊断（运营收敛到桌面）

**理由：** Phase 2降级路径验证后，Phase 5暴露健康状态与错误日志，将v3.0任务平台收敛到桌面主体验。陷阱8（错误诊断）剩余部分在此Phase补齐。

**交付内容：**
- 侧边导航"导入任务监控"入口（批次列表、任务进度、失败摘要）
- Python侧车故障诊断UI（进程状态、错误日志、手动重启）
- Go错误日志暴露API（/system/python-status）
- 用户诊断工具（日志导出功能）

**处理特性：** 侧边导航"导入任务监控"入口（Differentiators竞争优势）、Python侧车故障诊断（Differentiators竞争优势）

**预防陷阱：** 多进程错误诊断困难（日志聚合、用户诊断工具）

**使用技术栈：** Flutter Riverpod状态管理；Go WebSocket推送；Python错误结构传递

**实现架构组件：** 导入任务监控页面、系统状态页面

**验证方式：** request ID可关联日志，错误详情可见，诊断信息可导出

**置信度：** MEDIUM（前端监控页面需与现有v3.0任务平台API集成，需验证兼容性）

---

### Phase 6: 独立查看器与胶片条（Windows Photos核心体验增强）

**理由：** Phase 3桌面UX骨架验证后，Phase 6引入独立非模态查看器窗口与底部胶片条。这是Windows Photos的核心体验，但需window_manager多窗口支持，技术复杂度高。

**交付内容：**
- 独立查看器窗口（非模态、可多窗口对比）
- 底部胶片条（Filmstrip快速切换同文件夹图片）
- 状态同步（Go WebSocket推送当前文件夹图片列表）
- 窗口状态持久化（大小、位置记忆）

**处理特性：** 独立查看器窗口（Table Stakes，P2延后）、底部胶片条（Table Stakes，P2延后）

**预防陷阱：** 无新增陷阱预防（依赖Phase 3 Feature Flag与性能监控基础设施）

**使用技术栈：** Flutter window_manager + photo_view；Go WebSocket状态推送

**实现架构组件：** 查看器窗口组件、Filmstrip组件

**验证方式：** 非模态多窗口可同时打开；胶片条可快速切换图片

**置信度：** MEDIUM（Flutter多窗口管理为桌面特有能力，需验证window_manager稳定性）

---

### Phase顺序理由

**依赖关系驱动的顺序：**
1. **Phase 1优先：** 进程生命周期管理是所有后续工作的前提。僵尸进程、端口冲突、SQLite并发、Feature Flag基础设施必须最先建立。
2. **Phase 2次之：** Python侧车的首个计算能力验证，需Phase 1基础设施稳定后才能迁移重复检测逻辑。降级路径、Schema迁移、错误传递在此Phase预防。
3. **Phase 3第三：** Flutter桌面UX骨架重构，需架构稳定后才能重构用户界面。Feature Flag已在Phase 1准备，避免一次性替换导致体验断裂。
4. **Phase 4第四：** Windows打包集成，需功能验证后才能打包分发。多机器测试确保用户无需配置环境。
5. **Phase 5第五：** 导入任务监控与诊断，需降级路径验证后才能暴露健康状态。日志聚合需Phase 2错误传递协议基础。
6. **Phase 6可选：** 独立查看器与胶片条，依赖Phase 3桌面UX骨架完成。技术复杂度高，可根据工期选择性交付。

**架构模式驱动的分组：**
- **Phase 1-2：架构基础设施组** — 建立进程管理、通信契约、数据库配置、降级路径
- **Phase 3：用户可见体验组** — 桌面UX骨架重构，Feature Flag渐进推出
- **Phase 4：部署集成组** — Windows打包、启动顺序验证
- **Phase 5：运营与诊断组** — 任务监控、错误日志暴露、用户诊断工具
- **Phase 6：增强体验组** — 独立查看器、胶片条（可选）

**陷阱预防分布：**
- **Phase 1：** 陷阱1、陷阱3、陷阱5、陷阱9（范围边界）
- **Phase 2：** 陷阱4、陷阱7、陷阱8
- **Phase 3：** 陷阱6
- **Phase 4：** 陷阱2
- **Phase 5：** 陷阱8剩余部分

---

### 研究标记

**需要Phase级深入研究的阶段：**
- **Phase 3（Flutter桌面UX重构）：** 高风险重构路径，Flutter Windows桌面特有能力（window_manager多窗口、fluent_acrylic Mica材质）需深入研究实现细节与稳定性。桌面UX行业标准为MEDIUM置信度，需Phase级验证后提升信心。
- **Phase 6（独立查看器与胶片条）：** Flutter多窗口管理为桌面特有能力，window_manager稳定性需验证。状态同步机制（Go WebSocket推送当前文件夹图片列表）需研究实现模式。

**可跳过研究阶段的标准模式：**
- **Phase 1（Python侧车基础设施）：** Go进程管理、SQLite WAL配置、Feature Flag基础设施均为HIGH置信度，官方文档与标准模式完整。
- **Phase 2（重复检测计算迁移）：** Python imagehash库、Union-Find聚类算法、Schema迁移安全流程均为HIGH置信度，官方文档与最佳实践完整。
- **Phase 4（Windows打包）：** PyInstaller打包流程、Go编译、Flutter build windows均为HIGH置信度，官方文档明确列出陷阱与预防策略。

---

## 置信度评估

| 领域 | 置信度 | 说明 |
|------|--------|------|
| **技术栈** | HIGH | Context7官方文档（FastAPI lifespan、imagehash算法、fluent_ui组件、window_manager事件）+ 项目规划文档 + 已有代码基础 |
| **特性** | MEDIUM | Windows Photos程序调研文档（HIGH）+ 技术方案文档（HIGH）+ 竞品调研（MEDIUM，Tidy应用）+ 行业标准（MEDIUM）。Flutter桌面UX重构为高风险路径，需Phase级验证后提升信心。 |
| **架构** | HIGH | PyInstaller文档（HIGH）+ ImageHash文档（HIGH）+ Windows照片程序研究（HIGH）+ 技术方案文档（HIGH）+ Go优雅关闭模式（HIGH）+ Windows打包最佳实践（HIGH） |
| **陷阱** | HIGH | Go官方文档 + PyInstaller官方文档 + Stormkit博客 + SQLite Forum + TechDebt.guru + Dapr Docs + SRE School（均为权威来源） |

**整体置信度：HIGH**

理由：技术栈与架构研究基于Context7官方文档与项目规划文档，陷阱研究基于Go官方文档、PyInstaller官方文档、权威博客与标准模式，均为高质量来源。特性研究中Flutter桌面UX重构为高风险路径（MEDIUM置信度），但通过Phase 3验证可提升信心。

---

### 需在规划/实施阶段解决的缺口

**特性研究缺口（需Phase级验证）：**
- **Flutter Windows桌面UX重构稳定性：** window_manager多窗口支持、fluent_acrylic Mica材质渲染、瀑布流虚拟滚动性能需在Phase 3实际验证。建议Phase 3引入性能监控（FPS、滚动延迟），验证10k+图片库流畅浏览。
- **独立查看器窗口与胶片条状态同步：** Go WebSocket推送当前文件夹图片列表的实现模式需在Phase 6研究。建议Phase 6调用`/gsd-research-phase`深入研究Flutter多窗口管理最佳实践。

**架构集成缺口（需实施阶段验证）：**
- **v3.0任务平台API兼容性：** Phase 5导入任务监控需与现有Go任务平台API（batch_service.go + async_jobs worker）集成，需验证前端监控页面与API兼容性。建议Phase 5规划阶段验证API契约。
- **Python进程启动超时与重试策略：** Phase 1需确定具体超时阈值（10秒？30秒？）与重试次数。建议参考Go进程管理最佳实践（Graceful Shutdown in Go），结合桌面应用启动体验确定阈值。

**陷阱预防缺口（需Phase级验证）：**
- **降级路径实际表现：** Phase 2双路径设计（Python侧车 + Go fallback）需验证实际切换延迟与用户体验影响。建议Phase 2引入降级通知UI，用户可见"重复检测降级运行"状态。

---

## 来源汇总

### 高置信度来源（官方文档）

**技术栈研究（STACK.md）：**
- **Context7 /fastapi/fastapi** — FastAPI lifespan事件、启动/关闭生命周期、健康检查实现
- **Context7 /johannesbuchner/imagehash** — pHash/dHash算法、汉明距离计算、相似检测实现
- **Context7 /bdlukaa/fluent_ui** — NavigationView组件、Mica材质类、Windows Fluent Design实现
- **Context7 /leanflutter/window_manager** — 窗口事件监听、关闭确认对话框、生命周期管理
- **官方文档 Pillow** — 图像处理基础库（Python生态标准）
- **官方文档 uvicorn** — ASGI服务器配置、单进程生产部署
- **本地规划文档** — `.planning/PROJECT.md`, `Windows11-Photos-App-ACG-Gallery-Research.md`, `ACG-Gallery-Go-Python-Flutter-Technical-Plan.md`（项目边界与架构决策）

**架构研究（ARCHITECTURE.md）：**
- **PyInstaller 文档** — https://github.com/pyinstaller/pyinstaller（打包陷阱、--noconsole处理、依赖收集）
- **ImageHash 文档** — https://github.com/johannesbuchner/imagehash（pHash/dHash算法）
- **Windows 照片程序研究** — 内部设计文档 `Windows11-Photos-App-ACG-Gallery-Research.md`（导航、布局、查看器、胶片条）
- **技术方案文档** — 内部架构文档 `ACG-Gallery-Go-Python-Flutter-Technical-Plan.md`（Go/Python/Flutter职责边界、通信方式）
- **Go 优雅关闭模式** — https://medium.com/@rafalroppel/graceful-shutdown-in-go-explained-signals-contexts-and-the-correct-shutdown-sequence-f24fd9ef8fac（Windows信号处理差异、cmd.Wait()必要性）
- **Windows 打包最佳实践** — https://ahmedsyntax.com/pyinstaller-onefile/（打包流程、多机器测试）

**陷阱研究（PITFALLS.md）：**
- **Go 官方文档** — `Wait4`系统调用、进程生命周期管理（僵尸进程预防）
- **PyInstaller 官方文档** — "Common Issues and Pitfalls"章节（--noconsole stdout问题、依赖收集）
- **PyInstaller Wiki** — 调试打包问题的最佳实践（--onedir调试、UPX排除）
- **Flutter Windows 文档** — 生命周期管理、外部窗口处理
- **Stormkit 博客** — Hunting Zombie Processes in Go and Docker（僵尸进程案例）
- **TheLinuxCode** — SQLite Transactions in Practice（WAL模式、busy_timeout）
- **SQLite Forum** — SQLite Versioning & Migration Strategies（Schema迁移安全流程）
- **TechDebt.guru** — Feature Flags for Safe Refactoring（Feature Flag渐进推出）
- **Dapr Docs** — Sidecar health checks（健康检查端点设计）
- **SRE School** — Sidecar graceful degradation patterns（降级路径设计）

### 中置信度来源（行业标准与竞品）

**特性研究（FEATURES.md）：**
- **竞品特性调研（Tidy 应用）** — https://tidy.gallery/blog/ai-duplicate-photo-detection/（重复检测与推荐保留最佳实践，Union-Find聚类）
- **行业标准搜索（进程管理/任务队列）** — WebSearch结果（BullMQ、Lambda+SQS），后台任务编排模式参考
- **桌面 UX 行业标准** — WebSearch结果（Windows 11 Gallery、导航最佳实践），桌面图库UX模式参考

### 低置信度来源（需Phase级验证）

**个人经验（未验证，需Phase具体验证）：**
- **多进程调试经验模式** — request ID跨进程传递（Phase 8验证）
- **Windows 端口冲突常见性** — 随机端口必要性（Phase 3验证）
- **瀑布流性能问题普遍性** — 虚拟滚动必要性（Phase 3验证）

---

## 总结

v4.0里程碑研究综合置信度为HIGH，技术栈、架构、陷阱研究均基于官方文档与权威来源。特性研究置信度为MEDIUM，Flutter桌面UX重构为高风险路径需Phase 3验证后提升信心。

**最强结论：**
1. **技术路径明确：** Flutter(UI) + Go(主控) + Python(计算)三层架构，HTTP localhost通信，PyInstaller打包，SQLite WAL模式
2. **架构核心：** 职责拆分而非重写，Go主控编排，Python无状态计算，Flutter不直接访问文件系统/数据库
3. **MVP必需集：** 侧边导航栏、顶部工具栏、瀑布流视图、Python侧车进程管理、智能重复检测、导入任务监控、Windows打包基础验证

**最关键约束：**
1. **进程生命周期治理：** 必须调用cmd.Wait()防止僵尸进程，Windows无SIGTERM需HTTP /shutdown endpoint
2. **端口冲突避免：** 随机端口分配（127.0.0.1:0），stdout传递端口信息
3. **降级路径必需：** Python侧车崩溃时Go fallback路径必须存在，健康检查定期探测
4. **范围边界明确：** Out of Scope清单（图片编辑、Python做AI标签、macOS/Linux支持），防止范围蔓延

**需求定义应捕获：**
1. **进程生命周期管理完整流程：** 启动（随机端口+stdout传递）、健康检查（定时ping）、优雅关闭（HTTP /shutdown + Process.Kill()）
2. **降级路径设计：** 双路径检查（Python侧车 + Go fallback），降级通知UI，自动恢复机制
3. **Feature Flag基础设施：** 新旧界面可切换，渐进式推出（10% → 50% → 100%），性能监控
4. **Schema迁移安全流程：** schema_migrations版本表，每个迁移独立事务，回滚脚本必需
5. **错误传递协议：** request ID跨进程传递，Python返回详细错误结构（error + stack_trace），统一日志格式
6. **Windows打包验证：** 多机器测试（无Python环境机器），依赖收集完整，--noconsole标准流修复

**研究完成：** 2026-04-03
**状态：** 准备进入需求定义与路线图规划阶段

---

*v4.0里程碑研究综合总结*
*调研日期：2026-04-03*
*整体置信度：HIGH*
