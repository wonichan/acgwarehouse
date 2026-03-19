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
	Prompt  string `json:"prompt,omitempty"` // 用户自定义提示词，为空则使用默认提示词
}

type TagGovernanceMerger interface {
	MergeTags(ctx context.Context, imageID int64, tags []string, observationID int64, confidence float64) error
}

// DefaultTagPrompt 默认标签生成提示词
const DefaultTagPrompt = `请分析这张动漫风格的图片，提取符合以下分类的中文标签5-10个，用英文逗号分隔输出：
**画面风格
**角色类型
**人物名字识别（如可辨识）
- 动漫角色
- 游戏角色
- 手游角色
- 虚拟主播/Vtuber
- 画师原创角色（OC）
**服装特征
**外貌特征
**场景氛围
**情绪表达
**视觉特点
参考示例输出格式：
插画,初音未来,JK制服,水手服,双马尾,蓝发,雨夜氛围感,少女插画,黑丝白丝,温柔,治愈系
原创角色示例：
插画,原创角色,厚涂,金发少女,和服,樱花,温柔,暖色调,全身像，绝对领域
游戏角色示例：
游戏角色,明日方舟,能天使,银发,兽耳,战斗服,冷淡表情，黑丝
以上提供的标签都是举例说明，不一定非得是这些内容。只输出标签，用逗号分隔，不要解释。`

// GetDefaultTagPrompt 返回默认的 AI 标签生成提示词
func GetDefaultTagPrompt() string {
	return DefaultTagPrompt
}

// RegisterAITagHandler 注册 AI 标签生成任务处理器
func RegisterAITagHandler(manager *Manager, client ai.AIProvider, obsRepo repository.TagObservationRepository, governance TagGovernanceMerger) {
	manager.RegisterHandler("ai_tag_generation", func(ctx context.Context, id int64, payload string) error {
		return handleAITagGeneration(ctx, id, payload, client, obsRepo, governance)
	})
}

// handleAITagGeneration 处理 AI 标签生成任务
func handleAITagGeneration(ctx context.Context, id int64, payload string, client ai.AIProvider, obsRepo repository.TagObservationRepository, governance TagGovernanceMerger) error {
	// 解析 payload
	var p AITagPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return fmt.Errorf("parse payload: %w", err)
	}

	// 调用 AI 服务生成标签
	prompt := p.Prompt
	if prompt == "" {
		prompt = DefaultTagPrompt
	}
	result, err := client.GenerateTags(ctx, p.Path, prompt)
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
	if governance == nil {
		return fmt.Errorf("merge tags: governance service is nil")
	}

	if err := governance.MergeTags(ctx, p.ImageID, result.Tags, obs.ID, result.Confidence); err != nil {
		return fmt.Errorf("merge tags: %w", err)
	}

	return nil
}
