package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

// AITagConcurrencyLimiter AI 标签生成的并发控制器
// 全局变量，在服务启动时初始化
var AITagConcurrencyLimiter *ai.ConcurrencyLimiter

// DefaultTagPrompt 默认标签生成提示词
const DefaultTagPrompt = `请分析这张动漫风格的图片，提取6-8个中文标签。
【标签选择流程】
第一步：识别游戏IP
- 检查是否属于：碧蓝航线、碧蓝档案、明日方舟、鸣潮、星穹铁道等热门游戏IP
- 如是，IP名作为第1个标签
第二步：识别角色
- 如能识别具体角色名，作为第2个标签
- 如不能，根据画风判断：动漫角色/游戏角色/原创角色
第三步：分析服饰标签
衣服着装， 泳装 ，女仆装 ， 内衣 ， 黑丝，白丝 等丝袜， 以及特殊配件如狐耳、兽耳、双马尾等
第四步：外貌特征（1-2个）
发色、发型、瞳色、兽耳等
第五步：主题标签（如有）
- 百合
- 校园
- 枪、刀、剑等武器
-
【输出格式】
- 英文逗号分隔，无空格
- 6-8个标签
- 只输出标签，不要解释
【示例】
碧蓝航线,爱宕,泳装,黑丝,狐耳,金发
原创角色,女仆,白丝,粉发,双马尾
百合,碧蓝档案,美少女,校园,互动
---`

// GetDefaultTagPrompt 返回默认的 AI 标签生成提示词
func GetDefaultTagPrompt() string {
	return DefaultTagPrompt
}

// InitAITagConcurrencyLimiter 初始化 AI 标签生成的并发控制器
func InitAITagConcurrencyLimiter(maxConcurrency int) {
	AITagConcurrencyLimiter = ai.NewConcurrencyLimiter(maxConcurrency)
	log.Printf("AI 标签生成并发限制已设置: %d", maxConcurrency)
}

// RegisterAITagHandler 注册 AI 标签生成任务处理器
func RegisterAITagHandler(manager *Manager, client ai.AIProvider, obsRepo repository.TagObservationRepository, governance TagGovernanceMerger) {
	manager.RegisterHandler("ai_tag_generation", func(ctx context.Context, id int64, payload string) error {
		return handleAITagGeneration(ctx, id, payload, client, obsRepo, governance)
	})
}

// handleAITagGeneration 处理 AI 标签生成任务
func handleAITagGeneration(ctx context.Context, id int64, payload string, client ai.AIProvider, obsRepo repository.TagObservationRepository, governance TagGovernanceMerger) error {
	// 如果设置了并发控制器，先获取槽位
	if AITagConcurrencyLimiter != nil {
		release, err := AITagConcurrencyLimiter.Acquire(ctx)
		if err != nil {
			return fmt.Errorf("acquire concurrency slot: %w", err)
		}
		defer release()
	}

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
