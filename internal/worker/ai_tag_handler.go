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

type TagGovernanceRegenerator interface {
	TagGovernanceMerger
	RemovePendingAITags(ctx context.Context, imageID int64) error
}

type AITagPresenceChecker interface {
	HasAITags(ctx context.Context, imageID int64) (bool, error)
}

// AITagConcurrencyLimiter AI 标签生成的并发控制器
// 全局变量，在服务启动时初始化
var AITagConcurrencyLimiter *ai.ConcurrencyLimiter

// DefaultTagPrompt 默认标签生成提示词
const DefaultTagPrompt = `请分析这张动漫/二次元风格图片，并输出 最多 8 个中文标签，绝对不能超过 8 个。
【核心原则】
1. 只输出“高置信度”标签，禁止猜测不确定角色名或IP名。
2. 如果无法确认具体角色或IP，必须使用更稳妥的降级标签，不要乱猜。
3. 标签要与画面中“清晰可见、主体突出”的内容强相关，避免宽泛、无关、牵强标签。
4. 优先选择“识别度高、检索价值高”的标签，避免同义重复。
5. 输出必须是中文标签，使用英文逗号分隔，不要空格，不要句子，不要解释。
【标签输出顺序（严格按顺序挑选）】
第1类：作品/IP标签（0或1个，能确认才输出）
- 先判断是否属于热门二次元IP，如：碧蓝航线、碧蓝档案、明日方舟、鸣潮、崩坏3、崩坏星穹铁道、原神、东方Project、FGO、少女前线等
- 只有在高度确定时，才能输出IP名
- 不确定时，这一类可以跳过
第2类：角色标签（1个）
- 能明确识别具体角色名时，输出角色名
- 不能明确识别时，必须从以下降级标签中选1个：
  动漫角色、游戏角色、原创角色
第3类：服饰标签（2-3个）
优先输出画面中最明显、最有识别度的服饰/穿搭标签，例如：
- 泳装、比基尼、女仆装、内衣、旗袍、和服、校服、礼服、兔女郎、洛丽塔、战斗服、制服、婚纱
- 黑丝、白丝、裤袜、过膝袜、吊带袜、连裤袜
- 外套、罩衫、披风、围裙、头饰、发饰、猫耳等
要求：
- 只选画面中明确可见的服饰
- 不要输出过于细碎、难以检索的小配件
- 不要重复表达相近含义
第4类：外貌/身体特征标签（1-2个）
只输出视觉上非常明显的特征，例如：
- 银发、白发、黑发、粉发、金发、双马尾、长发、短发
- 巨乳、贫乳（仅在非常明显时才可输出）
- 萝莉、御姐（仅在角色气质非常明确时才可输出）
要求：
- 优先发色、发型等稳定特征
- 身材类标签要非常保守，不明显就不要输出
第5类：主题/元素标签（1-2个）
根据画面主体内容选择最明显的主题标签，例如：
- 百合、兽耳、猫耳、机械、未来感、校园、战斗、武器
- 枪、刀、剑、弓、法杖
- 海边、泳池、卧室、舞台等
要求：
- 只输出对检索有帮助的主题
- 不要为了凑数输出空泛标签，如“好看”“可爱”“二次元”
【数量要求】
- 总标签数必须为 6-8 个
- 如果高置信标签不足 6 个，可以补充 1-2 个“清晰可见”的发色/发型/场景/元素标签
- 绝对不要为了凑数量而猜角色、猜IP、猜剧情关系
【禁止事项】
- 禁止输出解释、前缀、序号、换行
- 禁止使用中文逗号
- 禁止输出空格
- 禁止输出低置信度猜测标签
- 禁止输出意思重复的标签
- 禁止输出抽象评价词：好看、精致、可爱、性感、唯美等
【输出格式】
标签1,标签2,标签3,标签4,标签5,标签6
【正确示例】
碧蓝航线,爱宕,泳装,黑丝,长发,巨乳,海边
原创角色,女仆装,白丝,银发,长发,猫耳,室内
游戏角色,校服,黑丝,双马尾,粉发,校园,百合
【最后要求】
请直接输出最终标签结果，不要解释，不要分析过程。
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
func RegisterAITagHandler(manager *Manager, client ai.AIProvider, obsRepo repository.TagObservationRepository, governance TagGovernanceMerger, aiTagChecker AITagPresenceChecker) {
	manager.RegisterHandler("ai_tag_generation", NewAITagJobHandler(client, obsRepo, governance, aiTagChecker))
}

func NewAITagJobHandler(client ai.AIProvider, obsRepo repository.TagObservationRepository, governance TagGovernanceMerger, aiTagChecker AITagPresenceChecker) JobFunc {
	return func(ctx context.Context, id int64, payload string) error {
		return handleAITagGeneration(ctx, id, payload, client, obsRepo, governance, aiTagChecker)
	}
}

func NewAITagRegenerationJobHandler(client ai.AIProvider, obsRepo repository.TagObservationRepository, governance TagGovernanceRegenerator) JobFunc {
	return func(ctx context.Context, id int64, payload string) error {
		if governance == nil {
			return fmt.Errorf("regenerate tags: governance service is nil")
		}
		var p AITagPayload
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return fmt.Errorf("parse payload: %w", err)
		}
		if err := governance.RemovePendingAITags(ctx, p.ImageID); err != nil {
			return fmt.Errorf("remove pending ai tags: %w", err)
		}
		return handleAITagGeneration(ctx, id, payload, client, obsRepo, governance, nil)
	}
}

// handleAITagGeneration 处理 AI 标签生成任务
func handleAITagGeneration(ctx context.Context, id int64, payload string, client ai.AIProvider, obsRepo repository.TagObservationRepository, governance TagGovernanceMerger, aiTagChecker AITagPresenceChecker) error {
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

	if aiTagChecker != nil {
		hasAITags, err := aiTagChecker.HasAITags(ctx, p.ImageID)
		if err != nil {
			return fmt.Errorf("check existing ai tags: %w", err)
		}
		if hasAITags {
			log.Printf("AI 标签任务跳过: image_id=%d 已存在 AI 标签", p.ImageID)
			return nil
		}
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
