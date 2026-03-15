# Phase 3: AI 开放标签与治理 - Research

**研究时间：** 2026-03-15
**研究目标：** 如何实现 Phase 3 的 AI 标签生成与治理能力

---

## Executive Summary

Phase 3 需要集成千问 VL 和豆包多模态 AI 为图片生成开放描述标签，并建立完整的标签治理能力。研究发现：

1. **API 兼容性**：千问和豆包均支持 OpenAI 兼容的 Chat Completions API，可抽象为统一接口
2. **限流策略**：使用 token bucket 算法配合 Go 的 `golang.org/x/time/rate` 包实现客户端限流
3. **异步架构**：复用 Phase 1 的 `JobManager` 基础设施，注册新的 `ai_tag_generation` 任务类型
4. **标签治理**：基于现有 schema 实现观测→归并→确认的完整流程

---

## 1. AI 服务集成研究

### 1.1 千问 VL (Qwen-VL) API

**官方 API 端点：**
- 国内：`https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions`
- 国际：通过 Alibaba Cloud Model Studio

**请求格式（OpenAI 兼容）：**
```json
{
  "model": "qwen-vl-max",
  "messages": [
    {
      "role": "user",
      "content": [
        {"type": "text", "text": "请识别这张图片中的元素，输出简洁的标签列表"},
        {"type": "image_url", "image_url": {"url": "https://example.com/image.jpg"}}
      ]
    }
  ],
  "max_tokens": 512,
  "temperature": 0.3
}
```

**关键参数：**
- `max_tokens`: 建议 512，足以返回 10-20 个标签
- `temperature`: 建议 0.3-0.5，降低随机性
- `top_p`: 建议 0.8

**图片输入方式：**
1. URL 方式：`{"type": "image_url", "image_url": {"url": "https://..."}}`
2. Base64 方式：`{"type": "image_url", "image_url": {"url": "data:image/jpeg;base64,..."}}`

**响应格式：**
```json
{
  "choices": [{
    "message": {
      "role": "assistant",
      "content": "蓝天, 白云, 动漫女孩, 长发, 校服, 微笑, 户外, 树木, 阳光, 温暖"
    }
  }]
}
```

### 1.2 豆包视觉模型 API

**官方 API 端点：**
- 火山引擎：`https://ark.cn-beijing.volces.com/api/v3/chat/completions`

**请求格式（OpenAI 兼容）：**
```json
{
  "model": "doubao-vision-pro-32k",
  "messages": [
    {
      "role": "user", 
      "content": [
        {"type": "text", "text": "请识别图片中的元素，输出标签"},
        {"type": "image_url", "image_url": {"url": "data:image/jpeg;base64,..."}}
      ]
    }
  ]
}
```

**模型选择：**
- `doubao-vision-pro`: 高性能，适合复杂场景
- `doubao-vision-lite`: 性价比高，适合批量处理
- `doubao-vision-mini`: 快速响应，适合实时场景

### 1.3 提供商抽象层设计

```go
// internal/ai/provider.go
type TagResult struct {
    Tags       []string
    Confidence float64
    ModelName  string
    RawResponse string
}

type AIProvider interface {
    Name() string
    GenerateTags(ctx context.Context, imageURL string, prompt string) (*TagResult, error)
}

// 千问实现
type QwenProvider struct {
    apiKey  string
    model   string
    baseURL string
}

// 豆包实现
type DoubaoProvider struct {
    apiKey    string
    model     string
    endpoint  string
}
```

**提供商工厂：**
```go
func NewProvider(cfg *config.AIConfig) (AIProvider, error) {
    switch cfg.Provider {
    case "qwen":
        return &QwenProvider{apiKey: cfg.APIKey, model: cfg.Model}, nil
    case "doubao":
        return &DoubaoProvider{apiKey: cfg.APIKey, model: cfg.Model}, nil
    default:
        return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
    }
}
```

---

## 2. 限流与异步处理研究

### 2.1 客户端限流实现

**推荐方案：Token Bucket 算法**

使用 Go 标准扩展库 `golang.org/x/time/rate`：

```go
import "golang.org/x/time/rate"

type RateLimitedClient struct {
    provider  AIProvider
    limiter   *rate.Limiter
}

func NewRateLimitedClient(provider AIProvider, requestsPerMinute int) *RateLimitedClient {
    // 每秒允许的请求数 = requestsPerMinute / 60
    rps := float64(requestsPerMinute) / 60.0
    return &RateLimitedClient{
        provider: provider,
        limiter:  rate.NewLimiter(rate.Limit(rps), 1), // burst=1，严格限制
    }
}

func (c *RateLimitedClient) GenerateTags(ctx context.Context, imageURL, prompt string) (*TagResult, error) {
    if err := c.limiter.Wait(ctx); err != nil {
        return nil, fmt.Errorf("rate limit wait: %w", err)
    }
    return c.provider.GenerateTags(ctx, imageURL, prompt)
}
```

### 2.2 异步任务处理架构

**复用 Phase 1 的 JobManager：**

```go
// 注册 AI 标签生成任务
jobManager.RegisterHandler("ai_tag_generation", func(ctx context.Context, id int64, payload string) error {
    var req struct {
        ImageID int64  `json:"image_id"`
        Path    string `json:"path"`
    }
    if err := json.Unmarshal([]byte(payload), &req); err != nil {
        return err
    }
    
    // 1. 调用 AI 服务
    result, err := aiClient.GenerateTags(ctx, req.Path, prompt)
    if err != nil {
        return err
    }
    
    // 2. 保存观测记录
    for _, tag := range result.Tags {
        obs := &domain.TagObservation{
            ImageID:       req.ImageID,
            RawText:       tag,
            Confidence:    result.Confidence,
            Provider:      provider.Name(),
            ModelName:     result.ModelName,
            PromptVersion: "v1",
        }
        obsRepo.Save(obs)
    }
    
    // 3. 触发标签归并
    tagGovernanceService.MergeTags(ctx, req.ImageID, result.Tags)
    
    return nil
})
```

### 2.3 重试策略

**指数退避重试：**
```go
func (s *AITagService) processWithRetry(ctx context.Context, imageID int64, path string) error {
    maxRetries := 3
    baseDelay := time.Second
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        err := s.process(ctx, imageID, path)
        if err == nil {
            return nil
        }
        
        // 检查是否为可重试错误
        if !isRetryable(err) {
            return err
        }
        
        delay := baseDelay * time.Duration(1<<attempt) // 1s, 2s, 4s
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(delay):
            continue
        }
    }
    return fmt.Errorf("max retries exceeded")
}

func isRetryable(err error) bool {
    // 速率限制、网络超时、服务不可用可重试
    if errors.Is(err, context.DeadlineExceeded) {
        return true
    }
    // 解析 429 状态码
    var apiErr *APIError
    if errors.As(err, &apiErr) {
        return apiErr.StatusCode == 429 || apiErr.StatusCode >= 500
    }
    return false
}
```

---

## 3. 标签治理研究

### 3.1 标签归并策略

**精确匹配归并流程：**
```
AI 返回标签 → 遍历每个标签
  ↓
精确匹配现有标签 (tag.preferred_label == tag_text)
  ↓ 是
关联到现有标签，创建 image_tags 记录
  ↓ 否
创建新标签 (review_state='pending')，创建 image_tags 记录
```

**代码实现：**
```go
func (s *TagGovernanceService) MergeTags(ctx context.Context, imageID int64, tags []string) error {
    for _, tagText := range tags {
        // 精确匹配
        existingTag, err := s.tagRepo.FindByLabel(ctx, tagText)
        if err == nil {
            // 关联到现有标签
            s.createImageTag(ctx, imageID, existingTag.ID, observationID)
        } else {
            // 创建新标签
            newTag := &domain.Tag{
                PreferredLabel: tagText,
                Slug:           slugify(tagText),
                ReviewState:    "pending",
                TrustScore:     0.5,
            }
            s.tagRepo.Save(ctx, newTag)
            s.createImageTag(ctx, imageID, newTag.ID, observationID)
        }
    }
    return nil
}
```

### 3.2 别名管理

**别名搜索策略：**
```go
// 搜索时自动匹配别名
func (s *TagService) SearchTags(ctx context.Context, query string) ([]*domain.Tag, error) {
    // 1. 精确匹配标签
    tags, _ := s.tagRepo.FindByLabelLike(ctx, query)
    
    // 2. 通过别名匹配
    normalizedQuery := normalize(query)
    aliases, _ := s.aliasRepo.FindByNormalizedLabel(ctx, normalizedQuery)
    
    // 合并结果
    for _, alias := range aliases {
        tag, _ := s.tagRepo.FindByID(ctx, alias.TagID)
        tags = append(tags, tag)
    }
    
    return deduplicate(tags), nil
}
```

### 3.3 复核状态流转

```
AI 新建标签 → pending (待确认)
    ↓ 用户确认
confirmed (已确认)
    ↓ 用户拒绝
rejected (已拒绝)
```

**状态更新 API：**
```go
// POST /api/v1/image-tags/:image_id/:tag_id/review
func (h *TagHandler) ReviewTag(c *gin.Context) {
    imageID := c.Param("image_id")
    tagID := c.Param("tag_id")
    
    var req struct {
        Action string `json:"action"` // confirm, reject
    }
    
    // 更新 image_tags.review_state
    h.tagService.UpdateReviewState(ctx, imageID, tagID, req.Action)
}
```

---

## 4. 提示词设计研究

### 4.1 标签生成提示词模板

```
你是一个专业的图片标签生成助手。请分析图片内容，输出简洁的标签列表。

要求：
1. 输出 8-12 个标签，用逗号分隔
2. 标签应为短词或短语，不使用长句
3. 涵盖：人物、服装、动作、场景、氛围、风格等方面
4. 按重要性排序，重要的标签放在前面
5. 使用中文输出

示例输出格式：
蓝天, 白云, 动漫女孩, 长发, 校服, 微笑, 户外, 阳光
```

### 4.2 版本管理

```go
const (
    PromptV1 = `...` // 初始版本
    PromptV2 = `...` // 优化版本
)

var CurrentPromptVersion = PromptV1

func GetPrompt() (string, string) {
    return CurrentPromptVersion, "v1"
}
```

---

## 5. Flutter 前端研究

### 5.1 标签筛选组件

**侧边栏抽屉实现：**
```dart
class TagFilterDrawer extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Drawer(
      child: Column(
        children: [
          // 搜索框
          TextField(
            decoration: InputDecoration(hintText: '搜索标签...'),
            onChanged: (query) => _searchTags(query),
          ),
          // 标签列表
          Expanded(
            child: Consumer<TagProvider>(
              builder: (context, provider, _) {
                return ListView.builder(
                  itemCount: provider.tags.length,
                  itemBuilder: (context, index) {
                    final tag = provider.tags[index];
                    return CheckboxListTile(
                      title: Text(tag.preferredLabel),
                      subtitle: Text('${tag.usageCount} 张图片'),
                      value: provider.selectedTags.contains(tag.id),
                      onChanged: (selected) => _toggleTag(tag.id),
                    );
                  },
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}
```

### 5.2 标签确认界面

**图片详情页标签区域：**
```dart
class ImageTagsSection extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        // 已确认标签
        _buildTagChips(confirmedTags, Colors.green),
        SizedBox(height: 16),
        // 待确认标签
        Text('待确认标签', style: Theme.of(context).textTheme.subtitle1),
        _buildPendingTags(pendingTags),
      ],
    );
  }
  
  Widget _buildPendingTags(List<Tag> tags) {
    return Wrap(
      children: tags.map((tag) => Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Chip(label: Text(tag.preferredLabel)),
          IconButton(icon: Icon(Icons.check), onPressed: () => _confirm(tag)),
          IconButton(icon: Icon(Icons.close), onPressed: () => _reject(tag)),
        ],
      )).toList(),
    );
  }
}
```

---

## 6. Repository 层设计

### 6.1 需要新增的 Repository

```go
// TagRepository
type TagRepository interface {
    Save(ctx context.Context, tag *domain.Tag) error
    FindByID(ctx context.Context, id int64) (*domain.Tag, error)
    FindByLabel(ctx context.Context, label string) (*domain.Tag, error)
    FindByLabelLike(ctx context.Context, query string) ([]*domain.Tag, error)
    FindAll(ctx context.Context, limit, offset int) ([]*domain.Tag, error)
    UpdateReviewState(ctx context.Context, id int64, state string) error
    IncrementUsageCount(ctx context.Context, id int64) error
}

// TagObservationRepository
type TagObservationRepository interface {
    Save(ctx context.Context, obs *domain.TagObservation) error
    FindByImageID(ctx context.Context, imageID int64) ([]*domain.TagObservation, error)
}

// TagAliasRepository
type TagAliasRepository interface {
    Save(ctx context.Context, alias *domain.TagAlias) error
    FindByTagID(ctx context.Context, tagID int64) ([]*domain.TagAlias, error)
    FindByNormalizedLabel(ctx context.Context, normalized string) (*domain.TagAlias, error)
}

// ImageTagRepository
type ImageTagRepository interface {
    Save(ctx context.Context, imageTag *domain.ImageTag) error
    FindByImageID(ctx context.Context, imageID int64) ([]*domain.ImageTag, error)
    FindByTagID(ctx context.Context, tagID int64) ([]*domain.ImageTag, error)
    UpdateReviewState(ctx context.Context, imageID, tagID int64, state string) error
    Delete(ctx context.Context, imageID, tagID int64) error
}
```

### 6.2 ImageTag Domain 模型

```go
// internal/domain/image_tag.go
package domain

import "time"

type ImageTag struct {
    ImageID            int64     `json:"image_id"`
    TagID              int64     `json:"tag_id"`
    SourceObservationID *int64   `json:"source_observation_id"`
    Confidence         float64   `json:"confidence"`
    ReviewState        string    `json:"review_state"`
}
```

---

## 7. API 端点设计

### 7.1 标签管理 API

| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/api/v1/tags` | 获取标签列表（支持搜索、分页） |
| POST | `/api/v1/tags` | 创建新标签 |
| PUT | `/api/v1/tags/:id` | 更新标签 |
| DELETE | `/api/v1/tags/:id` | 删除标签 |
| GET | `/api/v1/tags/:id/aliases` | 获取标签别名 |
| POST | `/api/v1/tags/:id/aliases` | 添加标签别名 |
| DELETE | `/api/v1/tags/:id/aliases/:alias_id` | 删除别名 |

### 7.2 图片标签关联 API

| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/api/v1/images/:id/tags` | 获取图片的所有标签 |
| POST | `/api/v1/images/:id/tags` | 为图片添加标签 |
| DELETE | `/api/v1/images/:id/tags/:tag_id` | 移除图片标签 |
| POST | `/api/v1/images/:id/tags/:tag_id/review` | 确认/拒绝标签 |
| POST | `/api/v1/images/:id/tags/batch-review` | 批量确认标签 |

### 7.3 AI 任务 API

| 方法 | 端点 | 描述 |
|------|------|------|
| POST | `/api/v1/images/:id/ai-tags` | 触发 AI 标签生成 |
| GET | `/api/v1/images/:id/ai-tags/status` | 获取 AI 任务状态 |

---

## 8. Validation Architecture

### 8.1 AI 服务集成测试

```go
func TestQwenProvider_GenerateTags(t *testing.T) {
    // 使用 mock server 模拟 API
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"choices":[{"message":{"content":"蓝天,白云,动漫女孩"}}]}`))
    }))
    defer server.Close()
    
    provider := &QwenProvider{baseURL: server.URL}
    result, err := provider.GenerateTags(context.Background(), "http://example.com/image.jpg", "test")
    
    assert.NoError(t, err)
    assert.Equal(t, []string{"蓝天", "白云", "动漫女孩"}, result.Tags)
}
```

### 8.2 限流测试

```go
func TestRateLimitedClient_RateLimit(t *testing.T) {
    provider := &MockProvider{}
    client := NewRateLimitedClient(provider, 60) // 60/min = 1/sec
    
    start := time.Now()
    for i := 0; i < 3; i++ {
        client.GenerateTags(context.Background(), "", "")
    }
    elapsed := time.Since(start)
    
    // 3 requests at 1/sec should take ~2 seconds
    assert.GreaterOrEqual(t, elapsed.Seconds(), 2.0)
}
```

### 8.3 标签归并测试

```go
func TestTagGovernanceService_MergeTags(t *testing.T) {
    // 准备测试数据
    existingTag := &domain.Tag{ID: 1, PreferredLabel: "蓝天"}
    
    // 测试精确匹配
    err := service.MergeTags(ctx, imageID, []string{"蓝天", "白云"})
    
    // 验证
    assert.NoError(t, err)
    // "蓝天" 应关联到现有标签
    // "白云" 应创建新标签
}
```

### 8.4 API 端点测试

```go
func TestTagHandler_GetTags(t *testing.T) {
    router := gin.Default()
    handler := &TagHandler{service: mockService}
    handler.RegisterRoutes(router)
    
    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/api/v1/tags?search=蓝", nil)
    router.ServeHTTP(w, req)
    
    assert.Equal(t, 200, w.Code)
}
```

---

## 9. 标准库与依赖

### 9.1 需要添加的 Go 依赖

```
golang.org/x/time/rate  // Token bucket 限流
```

### 9.2 已有依赖（复用）

- `github.com/gin-gonic/gin` - HTTP 框架
- `gopkg.in/yaml.v3` - 配置解析
- `ncruces/go-sqlite3` - SQLite 驱动

---

## 10. 风险与规避

| 风险 | 规避策略 |
|------|----------|
| AI API 速率限制 | Token bucket 限流 + 异步队列 |
| 开放标签漂移 | 精确匹配归并 + 复核状态 |
| AI 幻觉标签 | 低置信度过滤 + 用户确认 |
| 提供商切换 | 抽象层设计 + 配置驱动 |
| 大批量处理失败 | 重试机制 + 错误记录 |

---

## 11. 实现优先级

### Wave 1 (基础层)
1. AI 提供商抽象层
2. 限流客户端
3. 标签 Repository 层
4. 异步任务注册

### Wave 2 (服务层)
1. 标签归并服务
2. 标签管理 API
3. 图片标签关联 API

### Wave 3 (前端层)
1. Flutter 标签筛选组件
2. 标签确认/合并界面
3. 标签管理页面

---

*研究完成时间：2026-03-15*