# Quick Task 16: AI 标签提示词自定义支持

**完成时间:** 2026-03-20

## 实现概述

实现了用户自定义 AI 标签生成提示词的功能，用户可以在 Flutter 图片详情页输入自定义提示词，系统会使用自定义提示词替代默认提示词进行 AI 标签生成。

## 变更文件

### 后端 (Go)

| 文件 | 变更 |
|------|------|
| `internal/worker/ai_tag_handler.go` | `AITagPayload` 添加 `Prompt` 字段，`handleAITagGeneration` 支持自定义提示词，添加 `GetDefaultTagPrompt()` 函数 |
| `internal/handler/ai_tag_handler.go` | `TriggerAITags` 和 `BatchTriggerAITags` 支持 `prompt` 参数，添加 `GetDefaultPrompt` API |
| `internal/handler/routes.go` | 注册 `GET /api/v1/ai-tags/default-prompt` 端点 |

### Flutter

| 文件 | 变更 |
|------|------|
| `flutter_app/lib/config/api_config.dart` | 添加 `defaultAIPrompt` 端点 |
| `flutter_app/lib/services/tag_service.dart` | `triggerAITags` 支持 `prompt` 参数，添加 `getDefaultAIPrompt()` 方法 |
| `flutter_app/lib/providers/tag_provider.dart` | `triggerAITags` 支持 `prompt` 参数，添加 `getDefaultAIPrompt()` 方法 |
| `flutter_app/lib/screens/image_detail_screen.dart` | 添加自定义提示词开关、富文本输入框、默认提示词加载 |

## API 变更

### 新增端点

- `GET /api/v1/ai-tags/default-prompt` - 获取默认 AI 标签生成提示词

### 修改端点

- `POST /api/v1/images/:id/ai-tags` - 新增可选 `prompt` 字段
- `POST /api/v1/images/batch-ai-tags` - 新增可选 `prompt` 字段

## 功能特性

1. **默认提示词展示**: 进入图片详情页时自动加载并展示默认提示词
2. **自定义提示词开关**: 用户可以通过开关切换是否使用自定义提示词
3. **富文本输入框**: 提供多行文本输入框，支持编辑自定义提示词
4. **恢复默认**: 一键恢复默认提示词
5. **批量支持**: 批量触发 AI 标签时也支持自定义提示词

## 测试验证

- [x] Go 后端编译通过
- [x] Flutter 代码分析通过