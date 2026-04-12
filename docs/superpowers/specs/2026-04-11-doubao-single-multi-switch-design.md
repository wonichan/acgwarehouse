# Doubao Single-Image / Multi-Image Switch Design

## Goal

将当前 Doubao AI 打标流程中“1 张走单图、>1 张走多图”的隐式行为改为**显式配置驱动**，并同时控制两层策略：

- worker 是否聚合多个待处理任务
- provider 最终走单图请求还是多图请求

要求保留原有单次单图任务链路，并允许通过配置切换到重构后的单次多图任务链路；该优化只对 Doubao 生效，其他 provider 不调整。

## Problem Summary

当前实现已经具备多图能力，但行为边界仍然是硬编码的：

- `internal/ai/doubao_provider.go` 提供单图 `GenerateTags`
- `internal/ai/doubao_batch_provider.go` 提供多图 `GenerateTagsBatch`
- `internal/ai/fallback_doubao_provider.go` 已支持 Doubao 多模型失败切换
- `internal/worker/ai_tag_handler.go` 当前逻辑是：
  - worker 先抓取当前 job，再额外 claim 最多 `aiTagBatchSize-1` 个 ready job
  - 若最终请求数为 1，则调用 `GenerateTags`
  - 若最终请求数大于 1，则调用 `GenerateTagsBatch`

这意味着：

1. “是否聚合任务”与“是否调用多图接口”没有可配置开关
2. 原有单图路径虽然仍在，但无法被显式强制保留
3. 新多图路径虽然已存在，但无法被显式强制验证
4. Doubao 专属优化策略没有收敛到清晰配置语义中

## Approved Direction

采用 **Doubao 专属单一模式枚举配置** 的方案，同时控制 worker 聚合层与 API 调用层。

建议新增配置：

- `ai.doubao_batch_mode: single | auto | multi`

语义如下：

- `single`
  - worker 不额外 claim 其他 AI 打标 job
  - 每次只处理当前任务
  - 始终走单图 `GenerateTags`
- `auto`
  - 保留当前行为
  - worker 允许聚合多个 job
  - 最终只有 1 张时走 `GenerateTags`，大于 1 张时走 `GenerateTagsBatch`
- `multi`
  - worker 允许聚合多个 job
  - 无论最终拿到 1 张还是多张，都统一走 batch 路径
  - 目标是显式验证和运行多图链路，而不是继续回退到旧单图链路

## Scope

In scope:

- 为 Doubao 增加单图 / 多图模式配置
- 让该配置同时影响 worker 聚合行为与 provider 调用路径
- 保留现有 fallback model 机制
- 为新配置补齐 config、example config、测试覆盖
- 审查并沿用当前 large thumbnail 作为 AI 输入来源的策略

Out of scope:

- 修改 Qwen、Zhipu 等其他 provider 的行为
- 本次直接将 Doubao 从 Chat API 切换到 Responses API
- 本次直接调整 thumbnail large 的尺寸、质量、预缩放参数
- 重构 TagGovernance、Observation 持久化、平台任务同步逻辑
- 重新设计 worker 批大小策略（`aiTagBatchSize = 4` 本次保持不变）

## Codebase Evidence

- `internal/worker/ai_tag_handler.go`
  - `handleBatchAITagGeneration` 会 claim 额外 ready job 组成批次
  - 当 `len(requests) == 1` 时调用 `client.GenerateTags`
  - 当 `len(requests) > 1` 时调用 `client.GenerateTagsBatch`
- `internal/ai/doubao_provider.go`
  - `DoubaoProvider.GenerateTags` 走单图 OpenAI-compatible chat/completions 调用
- `internal/ai/doubao_batch_provider.go`
  - `DoubaoProvider.GenerateTagsBatch` 已实现多图请求构造、响应解析与编号输出格式
- `internal/ai/fallback_doubao_provider.go`
  - 对 `GenerateTags` 与 `GenerateTagsBatch` 都已实现逐模型失败切换
- `internal/ai/provider.go`
  - `NewProvider` 会按 `fallback_models` 组装单个或 fallback Doubao provider
- `internal/config/config.go`
  - 目前 `AIConfig` 只有 `Provider / Model / FallbackModels / MaxConcurrency / RequestsPerMinute` 等字段
  - 尚无 Doubao 单图 / 多图模式配置
- `deploy/config/config.example.yaml`
  - 已有 Doubao `model` 与 `fallback_models` 示例
  - 尚无本次模式配置示例
- `internal/service/ai_image_source.go`
  - AI 打标优先使用 `ThumbnailLargeUrl`，没有则退回原图路径

## External Evidence and Constraints

基于已调研的官方资料与代码现状，本次设计采用以下结论：

1. Doubao Seed 2.0 官方能力支持一次请求携带多张图片，因此仓库里已有的 `GenerateTagsBatch` 方向是成立的。
2. 官方公开资料足以支持“单次多图请求”的产品方向，但当前未取得可直接落地的明确像素/分辨率硬阈值，因此本次不在设计中硬改 large thumbnail 参数。
3. 官方有 Responses API 路径，但本次先不切换接口。先把现有 Chat API 上的单图 / 多图行为配置化，避免同时叠加两类变化导致定位困难。
4. 现有 fallback model 机制已经可覆盖单图与多图两条路径，重构时应保留这一稳定性能力。

## Design Principles

1. **显式优于隐式**：单图 / 多图行为必须由配置表达，而不是由批次数量隐式决定。
2. **最小范围变更**：本次聚焦 Doubao 调度与调用策略，不扩散到其他 provider 或缩略图生成参数。
3. **旧链路可强制保留**：必须能显式跑回原单图路径，便于回滚与对比验证。
4. **新链路可强制验证**：必须能显式强制多图链路，即使只有 1 张任务也可压测新路径。
5. **职责分层清晰**：config 决定策略，worker 决定任务组装与路径选择，provider 负责单图/多图能力实现。

## Architecture

### 1. Configuration Layer

在 `internal/config/config.go` 的 `AIConfig` 中新增 Doubao 专属模式字段，例如：

- `DoubaoBatchMode string `yaml:"doubao_batch_mode"``

默认值建议为：

- 当 `provider == doubao` 且未配置时，默认 `auto`

该字段仅在 `cfg.AI.Provider == "doubao"` 时生效；其他 provider 忽略此字段。

尽管 `AIConfig` 是通用结构，本次仍接受把 `DoubaoBatchMode` 放在其中，因为当前需求明确限定为 Doubao 专属优化，且仓库尚未形成统一的 provider-specific nested config 模式。本次不额外引入新的 provider 配置分层抽象，避免超出需求范围。

环境变量可同步支持一个简洁映射，例如：

- `AI_DOUBAO_BATCH_MODE=single|auto|multi`

### 2. Worker Layer

`internal/worker/ai_tag_handler.go` 中的 `handleBatchAITagGeneration` 从“永远先 claim 批次，再按请求数判断调用路径”改为“先解析模式，再决定是否 claim 与如何调用”：

- `single`
  - 不调用 `repo.FindAndClaimReadyJobs(...)`
  - 仅将 triggering job 转成单个 `TagRequest`
  - 调用 `client.GenerateTags(...)`
- `auto`
  - 维持当前 claim 行为
  - 最终 1 张走 `GenerateTags`
  - 最终 >1 张走 `GenerateTagsBatch`
- `multi`
  - 维持当前 claim 行为
  - 无论最终请求数是 1 还是多张，都统一调用 `GenerateTagsBatch`

模式值不应由 worker 直接反查全局 config；实现上应通过以下两种方式之一显式传入：

1. 在 handler 注册或构造时，将 `DoubaoBatchMode` 作为显式参数传给批量 AI tag handler
2. 或让 Doubao provider 暴露一个可查询的有效模式接口/能力，并由 worker 在初始化后读取

本 spec 更推荐 **方式 1：在 handler/worker 构造阶段显式注入模式值**，因为它比扩展通用 `AIProvider` 接口更局部，且不会让所有 provider 被迫实现一个 Doubao 专属能力。

这意味着 worker 层负责两件事：

1. 是否聚合 job
2. 进入 provider 的单图还是多图调用入口

### 3. Provider Layer

provider 层尽量保持“能力实现”角色，不承载 job 聚合策略：

- `DoubaoProvider.GenerateTags` 保持单图实现
- `DoubaoProvider.GenerateTagsBatch` 保持多图实现
- `FallbackDoubaoProvider` 保持对两条路径的失败切换

允许 `GenerateTagsBatch` 在 `multi` 模式下接收仅 1 个 request；此时不应偷偷回退到单图链路，否则会破坏“强制验证多图路径”的目标。

因此，本次设计要求：

- 将 `internal/ai/doubao_batch_provider.go` 中 `len(requests) == 1` 时直接调用 `GenerateTags` 的现状，改为由上层模式决定是否允许该回退
- 若模式要求强制 batch，则即便只有 1 张图，也走 batch request 构造与 batch response 解析

为此，provider 侧需要能拿到一个“是否允许单项 batch 回退到单图”的有效开关。该开关可以是：

- 注入到 `DoubaoProvider` 实例中的 provider 内部字段
- 或由 `GenerateTagsBatch` 增加内部 helper，根据构造参数决定是否绕回单图路径

不管采用哪种实现，`FallbackDoubaoProvider` 在切换到后备模型时也必须保持相同语义：

- `single` 与 `auto` 下允许单项 batch 回退到单图
- `multi` 下所有模型都必须坚持 batch 路径，不能某个 fallback client 又悄悄回退到单图

### 4. Input Source Strategy

本次继续使用现有 `ResolveAITagImagePath`：

- 优先 `ThumbnailLargeUrl`
- 否则回退原图路径

原因：

- 当前代码路径稳定，且 large thumbnail 已是 AI 输入默认来源
- 外部资料不足以支持一个明确的新像素阈值或新的 large thumbnail 参数
- 将输入尺寸策略与单图/多图切换拆开，有利于降低变更风险

## Behavior Matrix

| 模式 | claim 额外 job | 单任务时调用 | 多任务时调用 | 用途 |
|---|---|---|---|---|
| `single` | 否 | `GenerateTags` | 不会出现 | 保留旧链路 / 稳定回滚 |
| `auto` | 是 | `GenerateTags` | `GenerateTagsBatch` | 保持当前兼容行为 |
| `multi` | 是 | `GenerateTagsBatch` | `GenerateTagsBatch` | 强制验证和运行新链路 |

## Error Handling

- 多图与单图都继续沿用现有错误传播方式：provider 返回错误 → worker 统一 `markAllBatchJobsFailed` 或单任务失败。
- `FallbackDoubaoProvider` 继续按模型顺序重试，不因模式切换改变行为。
- 若配置值非法，应在 config 加载阶段或 provider 初始化阶段显式回退到默认值/报错，避免 worker 在运行时默默采用未知策略。
- `multi` 模式可能导致“只有 1 张图也走 batch prompt”的额外 token 开销；这是有意接受的设计成本，因为该模式的目标就是强制验证新多图链路，而不是最低成本运行。

## Testing Intent

至少新增或更新以下测试：

1. `AIConfig` 能解析 `doubao_batch_mode`
2. 未配置时 Doubao 默认模式为 `auto`
3. 非法模式值的处理符合预期（推荐：归一化到默认值或直接报配置错误，需实现时统一）
4. `single` 模式下 worker 不 claim 额外 job，且仅调用单图接口
5. `auto` 模式下：
   - 1 张调用单图接口
   - 多张调用多图接口
6. `multi` 模式下：
   - 1 张也调用 batch 接口
   - 多张继续调用 batch 接口
   - 单张 batch 响应（例如 `1: 标签1,标签2`）也能被正确解析为单组标签
7. fallback model 在单图与多图模式下都仍然生效
8. 现有 observation 保存、governance merge、平台任务完成/失败同步不回归

## Implementation Notes

- 本次推荐把“模式解析”抽成单独的小函数或枚举 helper，避免在 worker 中散落字符串比较。
- `aiTagBatchSize = 4` 暂不调整；如果未来要做更细的 Doubao 多图上限控制，应作为独立配置项新增，而不是混入本次模式配置。
- 如果后续要切 Responses API，建议新增一层 Doubao transport / request builder 抽象，而不是把接口切换逻辑直接塞进 worker。
- `multi` 模式的核心价值是**强制多图链路稳定运行**，因此不能在内部偷偷回退到 `GenerateTags`。
- 推荐在 config 默认值阶段把空值归一化为 `auto`，避免遗漏配置时出现不稳定的隐式行为。

## Follow-Up Work (Not in This Spec)

以下内容适合作为下一阶段独立优化：

1. 评估 Doubao Chat API → Responses API 的切换收益与改造成本
2. 基于更完整的官方输入限制资料，单独评估 large thumbnail 参数是否需要调整
3. 为 Doubao 多图模式补充更细粒度的批次大小配置，而不是固定使用 `4`
