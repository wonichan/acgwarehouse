# Phase 16: Duplicate Detection Migration - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-04
**Phase:** 16-duplicate-detection-migration
**Areas discussed:** 计算迁移边界, Python ↔ Go 调用契约, 推荐保留策略增强, Python 不可用时的降级行为

---

## 计算迁移边界

### Q1: 计算链路的拆分点应该在哪里？

| Option | Description | Selected |
|--------|-------------|----------|
| 仅哈希计算迁移（推荐） | Python 只处理步骤 1+2+4（哈希计算和相似度比较），Go 保留分组逻辑、推荐选取和落库。Python 职责最单纯，但 O(n²) 比较仍留在 Go。 | |
| 全计算迁移 | Python 处理步骤 1+2+4+5：哈希计算 + 比较 + Union-Find 分组 + 推荐选取。Go 只做落库和 API 层。计算密集部分全部在 Python，但 Python 输出结构较复杂。 | ✓ |
| 哈希+分组迁移（不含推荐） | Python 处理 1+2+4（哈希 + 比较 + 分组），但推荐选取仍由 Go 做。因为推荐策略依赖图片元数据（分辨率、文件大小等），这些已在 Go 的数据库中。 | |

**User's choice:** 全计算迁移
**Notes:** 无

### Q2: 元数据如何传递给 Python？

| Option | Description | Selected |
|--------|-------------|----------|
| Go 向 Python 传元数据（推荐） | Go 在调用时把每张图片的元数据（分辨率、文件大小、格式等）一并发给 Python，Python 在分组后基于这些元数据计算推荐项。Python 仍不查数据库，但接受比纯计算输入稍微复杂的输入。 | ✓ |
| Python 自提取元数据 | Python 自己从图片文件中提取元数据（读图片尺寸、文件大小）。输入更简洁，但会重复 Go 已经做过的元数据提取工作。 | |

**User's choice:** Go 向 Python 传元数据
**Notes:** 无

### Q3: 迁移后 Go 侧的现有计算代码应该如何处理？

| Option | Description | Selected |
|--------|-------------|----------|
| 完全替换（删除 Go 计算） | 将 Go 现有的 hash_service.go + duplicate_service.go 中的计算逻辑移除，完全依赖 Python 侧车。如果 Python 不可用，重复检测直接失败。代码更干净，但完全依赖侧车。 | ✓ |
| 保留 Go 作为回退路径 | 保留 Go 现有哈希/分组代码作为后备，优先调用 Python，Python 不可用时回退到 Go 实现。更健壮，但保持两套代码的维护负担。 | |

**User's choice:** 完全替换（删除 Go 计算）
**Notes:** 无

### Q4: 哈希算法是否升级？

| Option | Description | Selected |
|--------|-------------|----------|
| 64-bit pHash（保持兼容） | 当前 Go 的 goimagehash 库计算的是 64-bit pHash。Python 也采用等价的 64-bit pHash 算法，确保与历史数据兼容。 | |
| 升级哈希算法 | 迁移时升级到更先进的算法（如 dHash、高精度 pHash），但会破坏与历史 pHash 数据的兼容性，需要全库重新计算。 | ✓ |

**User's choice:** 升级哈希算法
**Notes:** 选择高精度 pHash（hash_size=16，256-bit）

### Q5: 历史图片的旧 pHash 如何处理？

| Option | Description | Selected |
|--------|-------------|----------|
| 首次运行时全量重算 | 重复检测触发时全库重新计算所有图片的新 pHash。实现简单但大图库下首次运行耗时较长。 | ✓ |
| 后台迁移任务（推荐） | 迁移后添加一个一次性迁移任务，在后台批量重新计算所有图片的新 pHash。可以利用现有 v3.0 的任务平台来跟踪进度。 | |

**User's choice:** 首次运行时全量重算
**Notes:** 无

---

## Python ↔ Go 调用契约

### Q1: Go 调用 Python 的模式？

| Option | Description | Selected |
|--------|-------------|----------|
| 单次批量调用 | Go 发一次 HTTP POST 请求，把所有图片打包发给 Python。简单直接，但大图库时可能超时且没有进度反馈。 | |
| 异步任务模式（推荐） | Go 发一个启动请求，Python 开始后台处理并返回任务 ID。Go 定期轮询状态（进度百分比）。处理完成后 Go 获取结果。支持进度跟踪，不会超时，但实现稍复杂。 | ✓ |
| 分批调用 | Go 把图片分成多个小批次逐批发给 Python。每批返回该批的哈希结果，Go 汇总后自己做分组。进度跟踪在 Go 侧，且分组逻辑又回到 Go。 | |

**User's choice:** 异步任务模式
**Notes:** 无

### Q2: 图片数据如何传给 Python？

| Option | Description | Selected |
|--------|-------------|----------|
| 本地文件路径（推荐） | Go 向 Python 发送图片的本地文件系统绝对路径列表。Python 直接读取本地文件计算哈希。当前都是本地部署，最简单高效，无需传输文件内容。 | ✓ |
| 二进制传输 | 通过 HTTP 发送图片二进制数据。支持远程部署，但当前没有远程部署需求，带宽浪费巨大。 | |

**User's choice:** 本地文件路径
**Notes:** 无

---

## 推荐保留策略增强

### Q1: 推荐策略如何增强？

| Option | Description | Selected |
|--------|-------------|----------|
| 多维度评分（推荐） | 在现有分辨率优先基础上，增加文件大小、格式优先级（PNG > JPEG > WebP）、修改时间等维度。按加权综合评分排序，推荐得分最高的。同时返回推荐依据文案。 | ✓ |
| 仅补充依据文案 | 保持现有逻辑（仅按分辨率），但补充推荐依据文案。最小改动，但依据较单一。 | |
| 你来决定 | 具体的评分维度和权重由工程师确定。 | |

**User's choice:** 多维度评分
**Notes:** 无

### Q2: 推荐依据的返回格式？

| Option | Description | Selected |
|--------|-------------|----------|
| 结构化数据（推荐） | 推荐依据以结构化数据返回，如 `{"reasons": [{"factor": "分辨率", "value": "1920x1080", "weight": 0.5}, ...], "score": 85}`。前端可以根据结构化数据自由展示。 | ✓ |
| 纯文案字符串 | 返回可读的文案字符串，如 "推荐原因：分辨率最高(1920x1080)，文件最大(2.3MB)"。简单直接，但前端难以进一步处理。 | |

**User's choice:** 结构化数据
**Notes:** 无

---

## Python 不可用时的降级行为

### Q1: Python 不可用时重复检测如何响应？

| Option | Description | Selected |
|--------|-------------|----------|
| 明确报错 + 诊断提示（推荐） | 用户触发重复检测时，如果 Python 不可用，直接返回明确的错误状态，如"计算服务不可用，请检查 Python 侧车状态"。符合 ROADMAP 成功标准第 3 条。不需要维护两套代码。 | ✓ |
| 排队等待恢复 | 检测到 Python 不可用时，将检测任务排入等待队列，当 Python 恢复时自动执行。用户不会立即看到失败，但可能不知道任务在等待。 | |

**User's choice:** 明确报错 + 诊断提示
**Notes:** 无

### Q2: 错误检测时机？

| Option | Description | Selected |
|--------|-------------|----------|
| 前置检查（推荐） | 通过 Phase 15 已有的 sidecar runtime 状态检查，在用户触发检测前先检查 Python 是否就绪。未就绪则立即拒绝并告知原因。 | ✓ |
| 失败后报错 | 直接尝试调用 Python，失败后再报错。简单，但用户会等待超时后才看到错误。 | |

**User's choice:** 前置检查
**Notes:** 无

---

## the agent's Discretion

- 具体的 HTTP 端点路径和 JSON schema 设计
- 评分维度的具体权重分配
- 进度轮询频率和超时配置
- Python 侧新 pHash 全量重算的并发控制策略
- 256-bit pHash 在 SQLite 中的具体存储方案
- 异步任务的取消和清理机制

## Deferred Ideas

- Python 不可用时自动降级到 Go 本地后备计算路径 — COMP-05 / Phase 22
- 增量检测优化 — Phase 22 或后续迭代
- 前端重复检测结果 UI 重构 — Phase 17+
- 远程/云端 Python 侧车部署支持 — 超出当前形态
