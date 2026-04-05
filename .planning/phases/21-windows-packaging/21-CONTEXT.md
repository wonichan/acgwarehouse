# Phase 21: Windows Packaging - Context

**Gathered:** 2026-04-05
**Status:** Ready for planning

<domain>
## Phase Boundary

在 Windows 环境下把 Flutter 桌面端、Go 主控服务和 Python 计算侧车打包为单一可分发交付物，让用户在**无需单独安装 Python** 的前提下即可解压、启动并使用完整功能。该阶段聚焦分发形态、启动链路、目录布局与失败诊断，不包含性能优化、自动降级计算回退或新的产品能力。

</domain>

<decisions>
## Implementation Decisions

### 交付与分发形态
- **D-01:** Phase 21 采用**绿色免安装**作为主交付体验，而不是标准安装器。
- **D-02:** 分发物锁定为**单个 ZIP 包**；用户下载后通过解压获得完整可运行目录。
- **D-03:** 当前阶段目标架构锁定为**Windows x64**，不在本阶段同时交付 ARM64。

### Python 运行时与启动链路
- **D-04:** Python 运行时采用**随包内嵌**策略，保证离线可用并满足“无需单独安装 Python”的要求。
- **D-05:** 用户只看到**一个统一启动入口**；该入口负责触发既定的 `Flutter → Go → Python` 启动顺序。
- **D-06:** 延续前序阶段约束：`Go` 继续作为唯一主控，`Python` 继续保持纯计算层职责，不直接承担业务主控或持久化职责。

### 目录与文件布局
- **D-07:** 绿色包采用**完全便携目录布局**：程序文件、配置、数据库、日志、运行时文件都保留在解压目录内，而不是拆分到用户目录。
- **D-08:** 日志、runtime manifest、诊断文件统一放在**包内固定子目录**，保证用户可以直接定位、携带与打包反馈。

### 升级与覆盖策略
- **D-09:** 后续版本升级允许用户**原地覆盖旧目录**完成更新，而不是要求每次解压到新目录。
- **D-10:** 因允许原地覆盖，后续 planning / implementation 必须把“旧文件残留、文件占用、运行时兼容和用户数据不被误覆盖”视为显式验证项。

### 失败诊断体验
- **D-11:** 启动失败时产品必须提供**明确错误页或错误对话框**，而不是只写后台日志。
- **D-12:** 错误反馈必须至少区分是 `Go` 启动失败、`Python` sidecar 启动失败，还是整体启动链路异常，并明确提示日志文件位置。
- **D-13:** sidecar 异常仍延续既有原则：必须**可诊断、不可静默失效**。

### the agent's Discretion
- 绿色包采用哪种具体打包工具链（例如脚本打包目录、Installer-free bundle、或 Python 侧具体封装方案）
- 统一启动入口的具体实现形式（launcher、主程序引导、或现有 Flutter runner 协调方式）
- ZIP 内固定子目录的具体命名（如 `logs/`、`runtime/`、`data/`）
- 原地覆盖升级时的文档提示、文件占用检测和回滚提示细节

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase Scope & Requirements
- `.planning/ROADMAP.md` — Phase 21 的目标、依赖关系与 3 条 success criteria
- `.planning/REQUIREMENTS.md` — `OPS-03` 的正式定义，以及 Windows 下无需单独安装 Python 的打包验证要求
- `.planning/PROJECT.md` — v4.0 里程碑约束、Go 主控 / Python 计算侧车职责边界、Windows 单机可打包/可诊断/可回退目标
- `.planning/STATE.md` — 当前阶段位置、Phase 21 风险提示与现有项目状态

### Prior Phase Decisions
- `.planning/phases/15-compute-sidecar-infrastructure/15-CONTEXT.md` — `Go` 唯一主控、`Flutter → Go → Python` 启动协同、runtime manifest 地址发现、sidecar 诊断边界
- `.planning/phases/16-duplicate-detection-migration/16-CONTEXT.md` — Python 仅承担计算层、不可用时明确失败、重复检测依赖 sidecar 的正式落地结果
- `.planning/phases/20-operations-monitoring/20-CONTEXT.md` — sidecar 诊断、错误摘要和重启入口已进入桌面产品路径，为打包后诊断体验提供产品基线

### Research & Technical Baseline
- `.planning/research/SUMMARY.md` — Windows 打包建议、PyInstaller 风险、端口冲突与多进程打包陷阱汇总
- `.planning/research/ARCHITECTURE.md` — 三层架构职责边界与 Windows 打包策略建议
- `.planning/research/PITFALLS.md` — 僵尸进程、打包启动失败、端口冲突、错误归因等高风险陷阱
- `ACG-Gallery-Go-Python-Flutter-Technical-Plan.md` — Go 管理 Python 进程、HTTP localhost 通信与多进程桌面打包技术基线

### Existing Implementation Entry Points
- `internal/app/bootstrap.go` — 当前 sidecar 启动、探活与命令装配入口，后续需要适配打包后的可执行路径
- `internal/sidecar/runtime.go` — sidecar 生命周期状态机、启动超时与关闭语义
- `docs/deployment.md` — 现有单机部署文档基线，可为 Windows 打包验证文档提供迁移参考
- `Makefile` — 当前 Go 构建入口，可扩展为 Windows packaging 构建脚本入口
- `flutter_app/pubspec.yaml` — Flutter Windows 桌面依赖与构建基线

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/app/bootstrap.go`：当前已集中承载 sidecar `CommandFactory`、探活逻辑与服务初始化，适合作为打包后路径解析和启动协调接入点
- `internal/sidecar/runtime.go`：已有 `not_started → starting → ready → degraded → stopped` 状态机，可直接承接打包后启动验证与故障诊断语义
- `flutter_app/windows/runner/`：Flutter Windows runner 已存在，说明桌面产物构建链已具备基础宿主
- `Makefile`：已有 Go 构建目标，可扩展为统一打包脚本入口

### Established Patterns
- 现有产品已经锁定 `Flutter → Go → Python` 启动顺序，打包阶段不能推翻该编排关系
- `Go` 已形成 sidecar 唯一主控模式，说明打包后也应继续由 `Go` 负责拉起与治理 Python，而不是让 Flutter 直接治理 sidecar
- sidecar 异常已被定义为需要显式诊断，说明打包交付不能只追求“能跑”，还必须保留失败可见性
- 当前仓库仍以单机运行路径为主，因此 Windows packaging 应优先复用本地目录与本地 HTTP 通信模式，而不是引入远程部署假设

### Integration Points
- `internal/app/bootstrap.go:49` 的 sidecar runtime 初始化需要适配打包后 Python 可执行文件/运行时位置
- `internal/app/bootstrap.go:57` 的 `CommandFactory` 当前直接使用 `python services/python-sidecar/main.py`，后续必须切换到打包产物可执行路径或内嵌运行时入口
- Flutter 桌面启动入口需要与打包后的统一 launcher / 主入口协调，确保 Flutter 连接到由 Go 暴露的真实运行地址
- 打包产物需要为日志、manifest、诊断文件建立固定目录，供桌面错误页和后续 Phase 20 诊断能力复用

</code_context>

<specifics>
## Specific Ideas

- 本阶段明确偏向“**真正可搬走**”的便携包心智：用户下载 ZIP、解压到任意目录、从单一入口启动
- 虽然采用绿色包，但仍要求失败时有产品级错误提示，而不是把排障成本全部转嫁给日志
- 允许原地覆盖升级，意味着实现上必须认真处理旧版本残留文件和数据兼容，而不是把升级约束交给用户自己猜

</specifics>

<deferred>
## Deferred Ideas

- Windows ARM64 同时交付 — 暂不纳入 Phase 21 首期范围
- 独立诊断器/单独诊断启动器 — 当前先以错误页 + 日志位置为主，不扩大为独立工具能力
- Python 不可用时自动降级到 Go 后备计算路径 — 属于 `COMP-05` / Phase 22 范围
- 自动更新机制 — 不在本阶段分发验证范围内

</deferred>

---

*Phase: 21-windows-packaging*
*Context gathered: 2026-04-05*
