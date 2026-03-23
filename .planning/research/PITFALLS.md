# Pitfalls Research: ACGWarehouse v3.0 导入后任务平台化

**Domain:** 导入后任务平台风险
**Milestone:** v3.0 导入后任务平台化
**Researched:** 2026-03-23
**Confidence:** HIGH

## Critical Pitfalls

### 1. 只有单图任务，没有批次视角
- **What goes wrong:** 后台能看到任务，但无法回答“这次导入现在处理到哪里了”
- **Why it matters:** 用户导入 10k+ 图片时，需要的是批次级可见性
- **Prevention:** 先建立导入批次和导入后处理批次模型

### 2. 自动入队没有过滤规则
- **What goes wrong:** 已有 AI 标签图片被反复重新入队，成本和重复处理快速放大
- **Prevention:** 默认仅无 AI 标签图片自动入队，并保留幂等 / 去重规则

### 3. 单图失败阻塞整批任务
- **What goes wrong:** 一张坏图、一次 API 失败或一次超时导致整个批次停住
- **Prevention:** 单图失败隔离、失败计数、可重试、批次继续推进

### 4. 后台只能看，不能救场
- **What goes wrong:** 用户知道队列卡住了，但无法暂停、继续、取消、清空或重试
- **Prevention:** 监控和控制必须一起设计，不要把操作能力留到后续

### 5. 导入尚未稳定就启动后处理
- **What goes wrong:** 文件仍在写入、导入尚未结算时就创建后处理任务，导致漏图或脏任务
- **Prevention:** 先定义导入完成 / 可调度时机，再触发导入后任务平台

### 6. 旧异步入口与新平台并存
- **What goes wrong:** 同一类任务同时存在旧触发路径和新平台路径，状态和控制互相打架
- **Prevention:** 尽早收敛任务创建入口，把旧路径迁移为平台入口的包装层

## Watch List for Planning

- 现有后台页面是否已经有可复用的任务 / 统计组件
- 现有 AI 标签触发逻辑在哪里被调用，是否存在多个入口
- 当前异步任务持久化能力是否足够支撑重启恢复和重试

## Sources

- PhotoPrism Import Safety Docs
- Chevereto Bulk Importer
- LibrePhotos Job System
- Immich Jobs / Workers

---
*Pitfalls research for milestone v3.0*
