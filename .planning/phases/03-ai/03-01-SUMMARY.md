---
phase: 03-ai
plan: 01
subsystem: AI Service Layer
tags: [ai, provider, rate-limiting, async-task, tdd]
requires: [config, domain, worker]
provides: [ai-provider-interface, qwen-provider, doubao-provider, rate-limiter, ai-tag-handler]
affects: [tag-observation-repository, schema]
tech_stack:
  added:
    - golang.org/x/time/rate (token bucket)
  patterns:
    - Provider pattern (AIProvider interface)
    - Decorator pattern (RateLimitedClient)
    - Factory pattern (NewProvider)
key_files:
  created:
    - internal/ai/provider.go
    - internal/ai/qwen_provider.go
    - internal/ai/doubao_provider.go
    - internal/ai/rate_limiter.go
    - internal/ai/ai_test.go
    - internal/ai/rate_limiter_test.go
    - internal/worker/ai_tag_handler.go
    - internal/worker/ai_tag_handler_test.go
    - internal/repository/tag_observation_repository.go
  modified:
    - internal/config/config.go (RequestsPerMinute field)
    - internal/repository/schema.go (tag_observations table)
    - go.mod (golang.org/x/time dependency)
decisions:
  - Use OpenAI-compatible API format for both Qwen and Doubao
  - Token bucket rate limiting with burst=1 for strict control
  - Async task handler with observation persistence
  - Base64 encoding for local image files
metrics:
  duration: ~2 hours
  tasks_completed: 5
  tests_added: 25+
  files_created: 9
  files_modified: 3
---

# Phase 03 Plan 01: AI 服务基础层 Summary

## One-liner

实现可切换的 AI 提供商抽象层（千问 VL/豆包视觉）、Token Bucket 限流客户端和异步任务处理器，为后续标签生成和治理提供稳定可靠的 AI 服务基础设施。

## Changes

### Task 1: AI 提供商抽象层

创建 `internal/ai/` 包，定义核心接口和工厂函数：

- `TagResult` 结构体：Tags、Confidence、ModelName、RawResponse 字段
- `AIProvider` 接口：Name()、GenerateTags() 方法
- `NewProvider()` 工厂函数：根据配置返回 QwenProvider 或 DoubaoProvider

**Commit:** 2c36b2e

### Task 2: 千问 VL 提供商实现

实现千问 VL (Qwen-VL) 提供商：

- OpenAI Chat Completions API 兼容格式
- 支持本地文件 Base64 编码和远程 URL
- 错误处理：429 速率限制、500+ 服务不可用
- 标签解析：逗号分割、空白清理

**Commit:** 6e85014

### Task 3: 豆包视觉模型提供商实现

实现豆包视觉模型提供商（结构类似 QwenProvider）：

- 火山引擎端点 `https://ark.cn-beijing.volces.com/api/v3`
- OpenAI 兼容 API 格式
- 相同的错误处理和标签解析逻辑

**Commit:** 0779326

### Task 4: 限流客户端实现

使用 `golang.org/x/time/rate` 实现 Token Bucket 限流：

- `RateLimitedClient` 包装底层 AIProvider
- `requestsPerMinute` 配置项，默认 60 请求/分钟
- Burst=1 严格限制，不允许突发
- Context 取消支持

**Commit:** b18f485

### Task 5: AI 标签生成异步任务处理器

注册 `ai_tag_generation` 任务类型到 JobManager：

- `AITagPayload` 结构体（ImageID、Path）
- `RegisterAITagHandler()` 注册函数
- `TagObservationRepository` 接口和 SQLite 实现
- 默认提示词模板：生成 8-12 个描述性标签

**Commit:** 3d44cc4

## Deviations from Plan

None - plan executed exactly as written.

## Key Decisions

1. **OpenAI 兼容 API 格式**：千问和豆包都支持 OpenAI Chat Completions 格式，简化实现
2. **Token Bucket 限流**：使用标准库 `golang.org/x/time/rate`，burst=1 确保严格限制
3. **Base64 图片编码**：本地文件自动转换为 data URI 格式，支持两种 AI 服务
4. **观测记录持久化**：每次 AI 调用结果保存到 tag_observations 表

## Verification

### Tests

```
=== RUN   TestTagResultStructure
--- PASS
=== RUN   TestAIProviderInterface
--- PASS
=== RUN   TestNewProvider_Qwen
--- PASS
=== RUN   TestNewProvider_Doubao
--- PASS
=== RUN   TestQwenProvider_BuildRequest
--- PASS
=== RUN   TestQwenProvider_ParseResponse
--- PASS
=== RUN   TestQwenProvider_HandleErrors
--- PASS
=== RUN   TestDoubaoProvider_BuildRequest
--- PASS
=== RUN   TestDoubaoProvider_ParseResponse
--- PASS
=== RUN   TestDoubaoProvider_VolcanoEndpoint
--- PASS
=== RUN   TestRateLimitedClient_LimitsRequests
--- PASS (6.00s)
=== RUN   TestRateLimitedClient_ContinuousRequests
--- PASS (2.00s)
=== RUN   TestRateLimitedClient_ContextCancel
--- PASS
=== RUN   TestRegisterAITagHandler_Registration
--- PASS
=== RUN   TestAITagHandler_ParsesPayload
--- PASS
=== RUN   TestAITagHandler_SavesObservation
--- PASS
PASS
```

### Build

```bash
go build ./...
# Success - no errors
```

## Must-Haves Check

- [x] `internal/ai/provider.go` - AIProvider 接口定义 ✓
- [x] `internal/ai/qwen_provider.go` - QwenProvider 实现 ✓
- [x] `internal/ai/doubao_provider.go` - DoubaoProvider 实现 ✓
- [x] `internal/ai/rate_limiter.go` - RateLimitedClient 实现 ✓
- [x] `internal/worker/ai_tag_handler.go` - ai_tag_generation 任务注册 ✓

## Next Steps

- Plan 02: 标签数据层（Repository、归并服务）
- Plan 03: 标签管理 API 层
- Plan 04: Flutter 标签前端层

---

**Completed:** 2026-03-15
**Duration:** ~2 hours
**Commits:** 5