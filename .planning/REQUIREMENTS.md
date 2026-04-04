# Requirements: ACGWarehouse v4.0 Windows Photos 风格重构与计算层拆分

**Defined:** 2026-04-03
**Core Value:** 让用户能够高效地管理和检索二次元图片库，通过 AI 自动化减少手动整理的工作量，实现"存入即整理"的体验。

## v4.0 Requirements

### Desktop Shell（桌面主壳层）

- [ ] **DSK-01**: 用户可以通过桌面顶部工具栏访问搜索、导入和设置等核心操作
- [ ] **DSK-02**: 用户可以在桌面端使用方块网格模式浏览图库
- [ ] **DSK-03**: 用户可以在桌面端按标签筛选图库内容
- [ ] **DSK-04**: 管理员可以在桌面端管理标签（查看、编辑、整理）而不必切换到旧入口

### Viewer（查看器体验）

- [ ] **VIEW-01**: 用户可以双击图片在独立查看器窗口中打开图片，而不阻塞主图库
- [ ] **VIEW-02**: 用户可以在查看器底部通过胶片条快速切换同一组图片
- [ ] **VIEW-03**: 用户可以在查看器中进行缩放、拖拽和双击放大等基础查看操作
- [ ] **VIEW-04**: 用户可以在查看器中看到图片元信息（如文件名、分辨率、大小及相关标签）

### Compute Sidecar（Python 计算侧车）

- [x] **COMP-01**: 用户启动桌面应用时，系统会按 `Flutter → Go → Python` 的顺序完成本地服务拉起，并在 Go 与 Python 都就绪后才进入可用状态
- [x] **COMP-02**: 管理员启动桌面应用时，系统会自动拉起并监控 Python 计算侧车进程
- [x] **COMP-03**: 系统会把重复检测、图像哈希和相似度计算从现有 Go 直接计算迁移到 Python 侧车执行
- [x] **COMP-04**: 用户在重复检测结果中可以看到每组图片的推荐保留项与推荐依据
- [ ] **COMP-05**: 当 Python 侧车不可用时，系统仍能给出可诊断的失败状态与恢复提示，而不是静默失效
- [x] **COMP-06**: Flutter 端可以获取当前 Go 本地服务地址并在应用启动过程中完成连接，而不依赖写死的固定端口

### Operations（运营与诊断）

- [ ] **OPS-01**: 管理员可以在桌面端进入导入后任务监控入口查看批次与任务状态
- [ ] **OPS-02**: 管理员可以在桌面端查看 Python 侧车的运行状态、最近错误摘要与手动重启入口
- [ ] **OPS-03**: 系统可以在 Windows 环境下完成 Flutter + Go + Python 的打包与分发验证，使用户无需单独安装 Python 运行环境

### Performance（大图库性能）

- [ ] **PERF-01**: 用户在数万张图片规模的图库中仍可以流畅浏览方块网格，而不会因首屏加载导致长时间卡顿
- [ ] **PERF-02**: 用户在数万张图片规模的图库中执行标签筛选或切换视图时，可以在可接受时间内看到结果反馈
- [ ] **PERF-03**: 管理员触发重复检测任务时，前台浏览与后台任务监控不会因大批量计算而整体失去响应

## v5.0+ Future Requirements

### Desktop Enhancements（桌面增强）

- **DSEE-01**: 用户可以在桌面端使用瀑布流 / River 模式浏览图库
- **DSEE-02**: 用户可以通过文件夹树形导航按目录浏览图库
- **DSEE-03**: 用户可以在桌面端获得更完善的相似图片可视化对比体验

### Reliability & Expansion（稳定性与扩展）

- **SAFE-04**: 当 Python 侧车故障时，系统可以自动降级到可工作的后备计算路径
- **EXT-03**: 系统可以把更多图像分析任务迁移到统一计算侧车接口
- **PLAT-03**: 用户可以在 Linux 桌面端使用完整图片库能力
- **PLAT-04**: 用户可以在 iOS / macOS 客户端使用完整图片库能力

## Out of Scope

| Feature | Reason |
|---------|--------|
| 图片编辑（裁剪、滤镜、调色） | 本里程碑聚焦桌面浏览 / 查看与计算层拆分，避免范围蔓延 |
| 实时全库重复检测 | 数万张图片下成本与阻塞风险过高，本里程碑采用触发式批量检测 |
| Python 直接持久化数据库 | 违反 Go 主控 / Python 计算侧车的职责边界 |
| 硬编码固定端口的侧车通信 | Windows 环境端口冲突风险高，必须保留随机端口方案 |
| Linux / iOS / macOS 新客户端交付 | 本里程碑聚焦 Windows 主路径与 Python 侧车首期落地 |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| DSK-01 | Phase 17 | Pending |
| DSK-02 | Phase 17 | Pending |
| DSK-03 | Phase 17 | Pending |
| DSK-04 | Phase 19 | Pending |
| VIEW-01 | Phase 18 | Pending |
| VIEW-02 | Phase 18 | Pending |
| VIEW-03 | Phase 18 | Pending |
| VIEW-04 | Phase 18 | Pending |
| COMP-01 | Phase 15 | Complete |
| COMP-02 | Phase 15 | Complete |
| COMP-03 | Phase 16 | Complete |
| COMP-04 | Phase 16 | Complete |
| COMP-05 | Phase 22 | Pending |
| COMP-06 | Phase 15 | Complete |
| OPS-01 | Phase 20 | Pending |
| OPS-02 | Phase 20 | Pending |
| OPS-03 | Phase 21 | Pending |
| PERF-01 | Phase 22 | Pending |
| PERF-02 | Phase 22 | Pending |
| PERF-03 | Phase 22 | Pending |

**Coverage:**
- v4.0 requirements: 20 total
- Mapped to phases: 20 ✓
- Unmapped: 0 ✓

---
*Requirements defined: 2026-04-03*
*Last updated: 2026-04-03 after milestone v4.0 requirement drafting*
