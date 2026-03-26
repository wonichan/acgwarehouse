# Requirements: ACGWarehouse v3.0 导入后任务平台化

**Defined:** 2026-03-23
**Core Value:** 让用户能够高效地管理和检索二次元图片库，通过 AI 自动化减少手动整理的工作量，实现"存入即整理"的体验。

## v3.0 Requirements

### Pipeline Platform（任务平台）

- [x] **PIPE-01**: 用户导入图片后，系统会为该次导入创建一个可追踪的后处理批次
- [ ] **PIPE-02**: 管理员可以按导入批次查看导入后任务，而不是逐张人工触发
- [x] **PIPE-03**: 系统会把 AI 标签等导入后任务纳入统一任务平台和生命周期管理

### AI Tag Automation（AI 标签自动化）

- [x] **AIQ-01**: 用户导入图片后，系统会自动把符合条件的图片加入 AI 打标签队列
- [x] **AIQ-02**: 默认只有没有 AI 标签的图片会自动进入 AI 打标签队列
- [ ] **AIQ-03**: 管理员可以批量把未打过 AI 标签的图片加入队列处理

### Operations（后台监控与控制）

- [ ] **OPS-01**: 管理员可以查看任务队列的待处理、执行中、成功、失败、已取消数量
- [ ] **OPS-02**: 管理员可以暂停后台任务队列
- [ ] **OPS-03**: 管理员可以继续已暂停的后台任务队列
- [ ] **OPS-04**: 管理员可以重试失败任务
- [ ] **OPS-05**: 管理员可以取消执行中或待处理任务
- [ ] **OPS-06**: 管理员可以清空尚未执行的待处理任务

### Reliability & Recovery（稳定性与恢复）

- [ ] **SAFE-01**: 单个图片任务失败不会阻塞同批次其它图片继续处理
- [ ] **SAFE-02**: 管理员可以看到任务失败状态与失败原因摘要，便于重试和排查
- [x] **SAFE-03**: 同一批未变更图片不会因重复触发而被无限重复入队

## Future Requirements

### Platform Expansion（平台扩展）

- **PLAT-01**: 用户可以在 iOS / macOS 客户端使用完整图片库能力
- **PLAT-02**: 用户可以在 Linux 桌面端使用完整图片库能力

### Advanced Scheduling（高级调度）

- **TASK-01**: 管理员可以为不同任务类型配置优先级与并发策略
- **TASK-02**: 管理员可以手动重新生成已有 AI 标签图片的 AI 结果

### Extensibility（扩展能力）

- **EXT-01**: 系统支持注册新的导入后任务类型，而不只限于 AI 标签
- **EXT-02**: 系统支持插件接入新的后台任务处理器

## Out of Scope

| Feature | Reason |
|---------|--------|
| 分布式多机器 worker 编排 | 当前主路径仍为单机 Docker Compose，先验证单机任务平台模型 |
| 第三方任务插件市场 | 平台边界尚未稳定，过早开放扩展会放大维护成本 |
| 任意已有 AI 标签图片自动重复入队 | 会显著增加成本与重复处理风险，当前只覆盖“无 AI 标签”场景 |
| 复杂优先级抢占与资源配额系统 | 先完成基础队列控制与可观测性，再评估高级调度 |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| PIPE-01 | Phase 11 | Complete |
| PIPE-03 | Phase 11 | Complete |
| SAFE-03 | Phase 11 | Complete |
| AIQ-01 | Phase 12 | Complete |
| AIQ-02 | Phase 12 | Complete |
| PIPE-02 | Phase 13 | Pending |
| OPS-01 | Phase 13 | Pending |
| OPS-02 | Phase 13 | Pending |
| OPS-03 | Phase 13 | Pending |
| OPS-04 | Phase 13 | Pending |
| OPS-05 | Phase 13 | Pending |
| OPS-06 | Phase 13 | Pending |
| AIQ-03 | Phase 14 | Pending |
| SAFE-01 | Phase 14 | Pending |
| SAFE-02 | Phase 14 | Pending |

**Coverage:**
- v3.0 requirements: 15 total
- Mapped to phases: 15
- Unmapped: 0 ✓

---
*Requirements defined: 2026-03-23*
*Last updated: 2026-03-26 after phase 12-03 execution*
