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
**画面风格：** 日系二次元插画、厚涂、赛璐璐、平涂、水彩风、Q版、萌系、御姐系
**角色类型：** 少女、少年、萝莉、御姐、正太、成男、成女、双人、多人、百合、耽美
**人物名字识别（如可辨识）：**
- 动漫角色
- 游戏角色
- 手游角色
- 虚拟主播/Vtuber
- 画师原创角色（OC）
**服装特征：** JK制服、水手服、西装、和服、旗袍、连衣裙、女仆装、战斗服、休闲装、黑丝、白丝、过膝袜、绝对领域
**外貌特征：** 黑长直、双马尾、短发、银发、金发、异色瞳、眼镜、兽耳、尾巴、翅膀
**场景氛围：** 雨夜、晴天、黄昏、夜晚、樱花、海边、教室、卧室、街道、幻想场景、都市、古风
**情绪表达：** 温柔、治愈、忧郁、元气、傲娇、冷淡、开心、害羞、严肃、神秘
**视觉特点：** 光影、逆光、侧光、全身像、半身像、特写、仰视、俯视
参考示例输出格式：
日系二次元插画,初音未来,JK制服,水手服,双马尾,蓝发,雨夜氛围感,少女插画,黑丝白丝,温柔,治愈系,侧光,半身像
原创角色示例：
日系二次元插画,原创角色,厚涂,金发少女,和服,樱花,温柔,暖色调,全身像
游戏角色示例：
日系平涂,明日方舟,能天使,银发,兽耳,战斗服,冷淡表情,逆光,高质量
只输出标签，用逗号分隔，不要解释。`

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
