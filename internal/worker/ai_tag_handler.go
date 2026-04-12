package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/ai"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

const aiTagBatchSize = 4

const (
	aiTagBatchModeSingle = "single"
	aiTagBatchModeAuto   = "auto"
	aiTagBatchModeMulti  = "multi"
)

var errInvalidAITagOutput = errors.New("invalid ai tag output")

// AITagPayload AI 标签生成任务的 payload 结构
type AITagPayload struct {
	ImageID int64  `json:"image_id"`
	Path    string `json:"path"`
	Prompt  string `json:"prompt,omitempty"` // 用户自定义提示词，为空则使用默认提示词
}

// DefaultTagPrompt 默认标签生成提示词
const DefaultTagPrompt = `请分析这张二次元风格图片，并输出 最多 8 个中文标签。
【核心原则】
1. 只输出"高置信度"标签，禁止猜测不确定角色名或IP名。
2. 标签要与画面中"清晰可见、主体突出"的内容强相关，避免宽泛、无关、牵强标签。
3. 优先选择"识别度高、检索价值高"的标签，避免同义重复。
4. 输出必须是中文标签，使用英文逗号分隔，不要空格，不要句子，不要解释。
【标签输出顺序（严格按顺序挑选）】
第1类：作品/IP标签（0或1个，能确认才输出）
- 先判断是否属于热门二次元IP，如：碧蓝航线、碧蓝档案、明日方舟、鸣潮、崩坏3、崩坏星穹铁道、原神、东方Project、FGO、少女前线等
- 只有在高度确定时，才能输出IP名
- 不确定时，这一类可以跳过
第2类：角色标签（0或1个）
- 能明确识别具体角色名时，输出角色名
- 不能明确识别时，不输出角色标签，不要乱猜
第3类：服饰标签（1-3个）
优先输出画面中最明显、最有识别度的服饰/穿搭标签，例如：
- 泳装、比基尼、女仆装、内衣、旗袍、和服、校服、礼服、兔女郎、洛丽塔、战斗服、制服、婚纱
- 黑丝、白丝、裤袜、过膝袜、吊带袜、连裤袜、短袜、长腿袜、光腿
要求：
- 只选画面中明确可见的服饰
- 按"最显眼→次显眼"顺序
第4类：角色特征标签（1-3个）
仅挑选能一眼看出、高可信度的特征：
- 发色/发型：银发、金发、粉发、蓝发、双马尾、单马尾、长发、短发
- 胸围：A cup、B cup、C cup、D cup（仅在有明显且明确的视觉参考时）
第5类：场景/环境标签（0-1个）
- 只有在场景非常明确且容易混淆时才输出
- 例如：海滩、教室、温泉、雪地、泳池

【重要】
- 只输出 2-8 个标签
- 禁止输出解释、前缀、序号、换行
- 标签之间用英文逗号分隔
输出格式：标签1,标签2,标签3,...（最多8个标签，逗号分隔）
`

func GetDefaultTagPrompt() string {
	return DefaultTagPrompt
}

// AITagConcurrencyLimiter AI 标签生成的并发控制器
var aiTagConcurrencyLimiter atomic.Pointer[ai.ConcurrencyLimiter]

// InitAITagConcurrencyLimiter 初始化 AI 标签生成的并发控制器
func InitAITagConcurrencyLimiter(maxConcurrency int) {
	aiTagConcurrencyLimiter.Store(ai.NewConcurrencyLimiter(maxConcurrency))
	log.Printf("AI 标签生成并发限制已设置: %d", maxConcurrency)
}

// SetAITagConcurrencyLimiter 动态调整 AI 标签生成的并发限制
func SetAITagConcurrencyLimiter(maxConcurrency int) {
	if limiter := aiTagConcurrencyLimiter.Load(); limiter != nil {
		limiter.SetLimit(maxConcurrency)
		log.Printf("AI 标签生成并发限制已调整为: %d", maxConcurrency)
	}
}

// TagGovernanceMerger 标签合并接口
type TagGovernanceMerger interface {
	MergeTags(ctx context.Context, imageID int64, tags []string, observationID int64, confidence float64) error
}

// TagGovernanceRegenerator 标签重生成接口
type TagGovernanceRegenerator interface {
	TagGovernanceMerger
	RemovePendingAITags(ctx context.Context, imageID int64) error
}

// AITagPresenceChecker AI 标签存在性检查
type AITagPresenceChecker interface {
	HasAITags(ctx context.Context, imageID int64) (bool, error)
}

type aiTagBatchRepo interface {
	FindByID(id int64) (*domain.AsyncJob, error)
	FindAndClaimReadyJobs(ctx context.Context, jobType string, limit int) ([]domain.AsyncJob, error)
	Update(job *domain.AsyncJob) error
}

type aiTagBatchTaskRepo interface {
	FindByID(ctx context.Context, taskID int64) (*domain.PlatformTask, error)
}

type aiTagBatchPlatformSvc interface {
	MarkJobsCompleted(ctx context.Context, jobIDs []int64) error
	MarkJobsFailed(ctx context.Context, jobIDs []int64, errorSync string) error
}

// RegisterBatchAITagHandler 注册批量 AI 标签处理器
func RegisterBatchAITagHandler(manager *Manager, repo aiTagBatchRepo, client ai.AIProvider, obsRepo repository.TagObservationRepository, governance TagGovernanceMerger, platformSvc aiTagBatchPlatformSvc, taskRepo aiTagBatchTaskRepo, aiTagChecker AITagPresenceChecker, batchMode string) {
	manager.RegisterHandler("ai_tag_generation", NewBatchAITagJobHandler(repo, client, obsRepo, governance, platformSvc, taskRepo, aiTagChecker, batchMode))
}

// NewBatchAITagJobHandler 创建批量 AI 标签处理器

func NewBatchAITagJobHandler(_ aiTagBatchRepo, client ai.AIProvider, obsRepo repository.TagObservationRepository, governance TagGovernanceMerger, _ aiTagBatchPlatformSvc, _ aiTagBatchTaskRepo, aiTagChecker AITagPresenceChecker, batchMode string, coordinators ...*aiTagBatchCoordinator) JobFunc {
	coordinator := firstAITagBatchCoordinator(coordinators...)
	if coordinator == nil {
		coordinator = newAITagBatchCoordinator(aiTagBatchSize, defaultAITagBatchWaitWindow)
	}
	return func(ctx context.Context, id int64, payload string) error {
		return handleQueuedAITagGeneration(ctx, id, payload, client, obsRepo, governance, aiTagChecker, batchMode, coordinator)
	}
}

func firstAITagBatchCoordinator(coordinators ...*aiTagBatchCoordinator) *aiTagBatchCoordinator {
	for _, coordinator := range coordinators {
		if coordinator != nil {
			return coordinator
		}
	}
	return nil
}

func handleQueuedAITagGeneration(ctx context.Context, id int64, payload string, client ai.AIProvider, obsRepo repository.TagObservationRepository, governance TagGovernanceMerger, aiTagChecker AITagPresenceChecker, batchMode string, coordinator *aiTagBatchCoordinator) error {
	if governance == nil {
		return fmt.Errorf("batch merge: governance service is nil")
	}

	effectiveMode := normalizeAITagBatchMode(batchMode)
	if client == nil || client.Name() != "doubao" {
		effectiveMode = aiTagBatchModeAuto
	}

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
			return nil
		}
	}

	if effectiveMode == aiTagBatchModeSingle || coordinator == nil {
		return executeAITagBatch(ctx, []aiTagBatchItem{{JobID: id, Payload: p}}, client, obsRepo, governance, effectiveMode)[0]
	}

	return coordinator.Submit(ctx, aiTagBatchItem{JobID: id, Payload: p}, func(batchCtx context.Context, items []aiTagBatchItem) []error {
		return executeAITagBatch(batchCtx, items, client, obsRepo, governance, effectiveMode)
	})
}

func executeAITagBatch(ctx context.Context, items []aiTagBatchItem, client ai.AIProvider, obsRepo repository.TagObservationRepository, governance TagGovernanceMerger, batchMode string) []error {
	errs := make([]error, len(items))
	if len(items) == 0 {
		return errs
	}
	if governance == nil {
		for i := range errs {
			errs[i] = fmt.Errorf("batch merge: governance service is nil")
		}
		return errs
	}

	if limiter := aiTagConcurrencyLimiter.Load(); limiter != nil {
		release, err := limiter.Acquire(ctx)
		if err != nil {
			for i := range errs {
				errs[i] = fmt.Errorf("acquire concurrency slot: %w", err)
			}
			return errs
		}
		defer release()
	}

	log.Printf("AI 批量标签任务批次: job_ids=%v total=%d", batchItemJobIDs(items), len(items))
	requests := make([]ai.TagRequest, len(items))
	for i := range items {
		prompt := items[i].Payload.Prompt
		if prompt == "" {
			prompt = DefaultTagPrompt
		}
		requests[i] = ai.TagRequest{
			ImageID: items[i].Payload.ImageID,
			Path:    items[i].Payload.Path,
			Prompt:  prompt,
		}
	}

	var (
		groups     [][]string
		modelName  string
		confidence float64
	)
	if batchMode != aiTagBatchModeMulti && len(requests) == 1 {
		result, err := client.GenerateTags(ctx, requests[0].Path, requests[0].Prompt)
		if err != nil {
			for i := range errs {
				errs[i] = fmt.Errorf("generate tags: %w", err)
			}
			return errs
		}
		groups = [][]string{result.Tags}
		modelName = result.ModelName
		confidence = result.Confidence
	} else {
		batchResult, err := client.GenerateTagsBatch(ctx, requests)
		if err != nil {
			for i := range errs {
				errs[i] = fmt.Errorf("generate tags: %w", err)
			}
			return errs
		}
		groups = batchResult.Groups
		modelName = batchResult.ModelName
		confidence = batchResult.Confidence
	}

	for i := range items {
		if i >= len(groups) {
			errs[i] = fmt.Errorf("generate tags: missing batch result")
			continue
		}
		tags := groups[i]
		if err := validateGeneratedTags(tags); err != nil {
			errs[i] = fmt.Errorf("validate tags: %w", err)
			continue
		}
		obs := &domain.TagObservation{
			ImageID:      items[i].Payload.ImageID,
			RawText:      strings.Join(tags, ", "),
			Confidence:   confidence,
			EvidenceType: "ai_generated",
			Provider:     client.Name(),
			ModelName:    modelName,
			CreatedAt:    time.Now(),
		}
		if err := obsRepo.Save(ctx, obs); err != nil {
			errs[i] = fmt.Errorf("save observation: %w", err)
			continue
		}
		if err := governance.MergeTags(ctx, obs.ImageID, tags, obs.ID, obs.Confidence); err != nil {
			errs[i] = fmt.Errorf("merge tags: %w", err)
			continue
		}
		log.Printf("AI 批量标签单个任务完成: job_id=%d image_id=%d tag_count=%d", items[i].JobID, obs.ImageID, len(tags))
	}

	log.Printf("AI 批量标签全部完成: total=%d", len(items))
	return errs
}

func batchItemJobIDs(items []aiTagBatchItem) []int64 {
	ids := make([]int64, len(items))
	for i := range items {
		ids[i] = items[i].JobID
	}
	return ids
}

// NewAITagRegenerationJobHandler 创建 AI 标签重生成处理器
func NewAITagRegenerationJobHandler(client ai.AIProvider, obsRepo repository.TagObservationRepository, governance TagGovernanceRegenerator) JobFunc {
	return func(ctx context.Context, id int64, payload string) error {
		if governance == nil {
			return fmt.Errorf("regenerate tags: governance service is nil")
		}
		var p AITagPayload
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return fmt.Errorf("parse payload: %w", err)
		}
		log.Printf("AI 标签重生成任务开始清理旧标签: job_id=%d image_id=%d", id, p.ImageID)
		if err := governance.RemovePendingAITags(ctx, p.ImageID); err != nil {
			log.Printf("AI 标签重生成任务清理旧标签失败: job_id=%d image_id=%d error=%v", id, p.ImageID, err)
			return fmt.Errorf("remove pending ai tags: %w", err)
		}
		log.Printf("AI 标签重生成任务清理旧标签完成: job_id=%d image_id=%d", id, p.ImageID)
		return handleAITagGenerationWithPayload(ctx, id, p, client, obsRepo, governance, nil)
	}
}

func normalizeAITagBatchMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case aiTagBatchModeSingle, aiTagBatchModeAuto, aiTagBatchModeMulti:
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return aiTagBatchModeAuto
	}
}

func handleAITagGenerationWithPayload(ctx context.Context, id int64, p AITagPayload, client ai.AIProvider, obsRepo repository.TagObservationRepository, governance TagGovernanceMerger, _ AITagPresenceChecker) error {
	if governance == nil {
		return fmt.Errorf("merge tags: governance service is nil")
	}

	startedAt := time.Now()
	providerName := ""
	if client != nil {
		providerName = client.Name()
	}
	log.Printf("AI 标签任务开始: job_id=%d image_id=%d path=%s provider=%s custom_prompt=%t", id, p.ImageID, p.Path, providerName, p.Prompt != "")

	if limiter := aiTagConcurrencyLimiter.Load(); limiter != nil {
		release, err := limiter.Acquire(ctx)
		if err != nil {
			log.Printf("AI 标签任务获取并发槽位失败: job_id=%d image_id=%d error=%v", id, p.ImageID, err)
			return fmt.Errorf("acquire concurrency slot: %w", err)
		}
		defer release()
	}

	prompt := p.Prompt
	if prompt == "" {
		prompt = DefaultTagPrompt
	}
	log.Printf("AI 标签任务调用模型: job_id=%d image_id=%d provider=%s prompt_length=%d", id, p.ImageID, providerName, len(prompt))
	result, err := client.GenerateTags(ctx, p.Path, prompt)
	if err != nil {
		log.Printf("AI 标签任务调用模型失败: job_id=%d image_id=%d provider=%s error=%v", id, p.ImageID, providerName, err)
		return fmt.Errorf("generate tags: %w", err)
	}
	if err := validateGeneratedTags(result.Tags); err != nil {
		log.Printf("AI 标签任务结果校验失败: job_id=%d image_id=%d provider=%s raw_tag_count=%d error=%v", id, p.ImageID, providerName, len(result.Tags), err)
		return fmt.Errorf("validate tags: %w", err)
	}
	log.Printf("AI 标签任务生成完成: job_id=%d image_id=%d provider=%s model=%s tag_count=%d confidence=%.4f", id, p.ImageID, providerName, result.ModelName, len(result.Tags), result.Confidence)

	obs := &domain.TagObservation{
		ImageID:      p.ImageID,
		RawText:      strings.Join(result.Tags, ", "),
		Confidence:   result.Confidence,
		EvidenceType: "ai_generated",
		Provider:     client.Name(),
		ModelName:    result.ModelName,
		CreatedAt:    time.Now(),
	}

	if err := obsRepo.Save(ctx, obs); err != nil {
		log.Printf("AI 标签任务保存观测失败: job_id=%d image_id=%d provider=%s error=%v", id, p.ImageID, providerName, err)
		return fmt.Errorf("save observation: %w", err)
	}

	if err := governance.MergeTags(ctx, p.ImageID, result.Tags, obs.ID, result.Confidence); err != nil {
		log.Printf("AI 标签任务合并标签失败: job_id=%d image_id=%d observation_id=%d error=%v", id, p.ImageID, obs.ID, err)
		return fmt.Errorf("merge tags: %w", err)
	}
	log.Printf("AI 标签任务完成: job_id=%d image_id=%d observation_id=%d tag_count=%d duration=%s", id, p.ImageID, obs.ID, len(result.Tags), time.Since(startedAt))

	return nil
}

func validateGeneratedTags(tags []string) error {
	cleaned := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	if len(cleaned) == 0 {
		return fmt.Errorf("%w: empty tag list", errInvalidAITagOutput)
	}
	if len(cleaned) == 1 && looksLikeAIErrorMessage(cleaned[0]) {
		return fmt.Errorf("%w: %s", errInvalidAITagOutput, cleaned[0])
	}
	return nil
}

func looksLikeAIErrorMessage(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	phrases := []string{
		"无法分析",
		"未提供任何图像",
		"请上传",
		"请提供",
		"后重试",
		"unable to analyze",
		"cannot analyze",
		"no image",
		"please upload",
		"please provide",
	}
	for _, phrase := range phrases {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	return false
}
