package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/ai"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

// AITagPayload AI 标签生成任务的 payload 结构
type AITagPayload struct {
	ImageID int64  `json:"image_id"`
	Path    string `json:"path"`
}

// DefaultTagPrompt 默认标签生成提示词
const DefaultTagPrompt = `分析这张图片，生成8-12个描述性标签。标签应涵盖：
- 人物特征（如：girl, blue hair, long hair）
- 场景环境（如：outdoors, indoors, night）
- 服装装饰（如：school uniform, dress）
- 动作姿态（如：standing, sitting）
- 艺术风格（如：anime, illustration, digital art）

请直接用逗号分隔输出标签，不要其他解释。`

// RegisterAITagHandler 注册 AI 标签生成任务处理器
func RegisterAITagHandler(manager *Manager, client ai.AIProvider, obsRepo repository.TagObservationRepository) {
	manager.RegisterHandler("ai_tag_generation", func(ctx context.Context, id int64, payload string) error {
		return handleAITagGeneration(ctx, id, payload, client, obsRepo)
	})
}

// handleAITagGeneration 处理 AI 标签生成任务
func handleAITagGeneration(ctx context.Context, id int64, payload string, client ai.AIProvider, obsRepo repository.TagObservationRepository) error {
	// 解析 payload
	var p AITagPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return fmt.Errorf("parse payload: %w", err)
	}

	// 调用 AI 服务生成标签
	result, err := client.GenerateTags(ctx, p.Path, DefaultTagPrompt)
	if err != nil {
		return fmt.Errorf("generate tags: %w", err)
	}

	// 保存观测记录
	obs := &domain.TagObservation{
		ImageID:       p.ImageID,
		RawText:       strings.Join(result.Tags, ", "),
		Confidence:    result.Confidence,
		EvidenceType:  "ai_generated",
		Provider:      client.Name(),
		ModelName:     result.ModelName,
		PromptVersion: "v1",
		CreatedAt:     time.Now(),
	}

	if err := obsRepo.Save(ctx, obs); err != nil {
		return fmt.Errorf("save observation: %w", err)
	}

	return nil
}
