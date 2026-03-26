# Roadmap: ACGWarehouse

## Milestones

- ✅ **v1.0 MVP** — Phases 1-6 (shipped 2026-03-19)
  - 详见: `.planning/milestones/v1.0-ROADMAP.md`
- ✅ **v2.0 UI/UX 重构** — Phases 7-10 (shipped 2026-03-22)
  - 详见: `.planning/milestones/v2.0-ROADMAP.md`
- 📋 **v3.0 导入后任务平台化** — Phases 11-14 (planned)

## Completed Phases

<details>
<summary>✅ v1.0 MVP (Phases 1-6) — SHIPPED 2026-03-19</summary>

- [x] Phase 1: 基础架构、图片扫描与标签基础层
- [x] Phase 2: 缩略图、基础浏览与 AI 复核界面底座
- [x] Phase 3: AI 开放标签与治理
- [x] Phase 4: 重复检测与搜索
- [x] Phase 5: 收藏夹与批量操作
- [x] Phase 6: 优化与部署

</details>

<details>
<summary>✅ v2.0 UI/UX 重构与多端适配 (Phases 7-10) — SHIPPED 2026-03-22</summary>

- [x] Phase 7: 架构基础层
- [x] Phase 8: Windows 桌面端 UI
- [x] Phase 9: Android 移动端 UI
- [x] Phase 10: 主题统一与优化

</details>

---

### 📋 v3.0 导入后任务平台化 (Planned)

**Milestone Goal:** 将导入后的 AI 标签等后台处理任务统一纳入可观测、可控制、可恢复的平台，解决海量导入后仍需逐张人工触发的问题。

#### Phase 11: 任务平台基础与批次模型
**Goal**: 建立统一的导入批次、后处理任务与状态流转模型，为所有导入后任务提供同一平台入口。
**Depends on**: Phase 10 (v2.0 complete)
**Requirements**: PIPE-01, PIPE-03, SAFE-03
**Success Criteria** (what must be TRUE):
1. 用户导入图片后，系统会生成一个可追踪的导入后处理批次。
2. 导入后任务在统一状态机中流转，而不是分散在各个独立入口里手动触发。
3. 同一批未变更图片不会因为重复触发而无限重复入队。
**Plans**: 4 plans

Plans:
- [x] 11-01: 定义导入批次、任务与任务状态模型
- [x] 11-02: 建立任务持久化与状态流转规则
- [x] 11-03: 接入统一调度入口与分发骨架
- [x] 11-04: 提供后台查询所需的批次 / 任务读模型

#### Phase 12: 导入后任务接入与自动调度
**Goal**: 将 AI 标签与导入后处理链路接入统一平台，实现导入完成后的自动入队与条件过滤。
**Depends on**: Phase 11
**Requirements**: AIQ-01, AIQ-02
**Success Criteria** (what must be TRUE):
1. 用户导入图片后，符合条件的图片会自动加入 AI 打标签队列。
2. 默认只有没有 AI 标签的图片才会被自动加入 AI 打标签队列。
3. 导入后的处理不再依赖逐图人工触发。
**Plans**: 4 plans

Plans:
- [ ] 12-01: 数据模型变更与查询基础（image_tag.source 字段 + FindImagesWithoutAITags）
- [ ] 12-02: 定时扫描服务实现（AITagAutoScheduler + 配置项）
- [ ] 12-03: 集成调度服务到应用启动流程
- [ ] 12-04: 端到端验证与真实批次测试

#### Phase 13: 后台监控与队列控制
**Goal**: 在后台管理页面提供按批次监控、队列状态查看和核心控制动作。
**Depends on**: Phase 12
**Requirements**: PIPE-02, OPS-01, OPS-02, OPS-03, OPS-04, OPS-05, OPS-06
**Success Criteria** (what must be TRUE):
1. 管理员可以按批次查看队列中的待处理、执行中、成功、失败、已取消状态。
2. 管理员可以暂停、继续、取消、清空队列，而不需要逐图干预。
3. 管理员可以对失败任务执行重试操作。
**Plans**: 4 plans

Plans:
- [ ] 13-01: 设计后台任务平台页的批次与状态视图
- [ ] 13-02: 提供队列统计与任务明细接口
- [ ] 13-03: 实现暂停 / 继续 / 取消 / 清空控制动作
- [ ] 13-04: 实现失败任务重试与操作反馈

#### Phase 14: 补跑恢复与运营收尾
**Goal**: 完成“未打标签图片”批量补入队、失败隔离与恢复体验，补齐运营闭环。
**Depends on**: Phase 13
**Requirements**: AIQ-03, SAFE-01, SAFE-02
**Success Criteria** (what must be TRUE):
1. 管理员可以批量把未打过 AI 标签的图片加入处理队列。
2. 单个图片任务失败不会阻塞同批次其它图片继续处理。
3. 管理员可以看到失败状态与失败原因摘要，并据此进行恢复操作。
**Plans**: 3 plans

Plans:
- [ ] 14-01: 实现“未打标签图片”批量补入队入口
- [ ] 14-02: 强化单图失败隔离与批次继续执行逻辑
- [ ] 14-03: 收敛失败可见性、恢复反馈与里程碑验收

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10 → 11 → 12 → 13 → 14

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. 基础架构、图片扫描与标签基础层 | v1.0 | complete | Complete | 2026-03-19 |
| 2. 缩略图、基础浏览与 AI 复核界面底座 | v1.0 | complete | Complete | 2026-03-19 |
| 3. AI 开放标签与治理 | v1.0 | complete | Complete | 2026-03-19 |
| 4. 重复检测与搜索 | v1.0 | complete | Complete | 2026-03-19 |
| 5. 收藏夹与批量操作 | v1.0 | complete | Complete | 2026-03-19 |
| 6. 优化与部署 | v1.0 | complete | Complete | 2026-03-19 |
| 7. 架构基础层 | v2.0 | complete | Complete | 2026-03-22 |
| 8. Windows 桌面端 UI | v2.0 | complete | Complete | 2026-03-22 |
| 9. Android 移动端 UI | v2.0 | complete | Complete | 2026-03-22 |
| 10. 主题统一与优化 | v2.0 | complete | Complete | 2026-03-22 |
| 11. 任务平台基础与批次模型 | v3.0 | complete | Complete | 2026-03-24 |
| 12. 导入后任务接入与自动调度 | v3.0 | 0/4 | Not started | - |
| 13. 后台监控与队列控制 | v3.0 | 0/4 | Not started | - |
| 14. 补跑恢复与运营收尾 | v3.0 | 0/3 | Not started | - |
