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
- 表情/姿态：微笑、害羞、傲娇、战斗姿态
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
func NewBatchAITagJobHandler(repo aiTagBatchRepo, client ai.AIProvider, obsRepo repository.TagObservationRepository, governance TagGovernanceMerger, platformSvc aiTagBatchPlatformSvc, taskRepo aiTagBatchTaskRepo, aiTagChecker AITagPresenceChecker, batchMode string) JobFunc {
	return func(ctx context.Context, id int64, payload string) error {
		return handleBatchAITagGeneration(ctx, id, payload, repo, client, obsRepo, governance, platformSvc, taskRepo, aiTagChecker, batchMode)
	}
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

// handleBatchAITagGeneration 批量处理 AI 标签生成任务
// 当前触发 job + 额外抓取最多 (batchSize-1) 个 ready job → 一次 AI 调用
func handleBatchAITagGeneration(ctx context.Context, triggeringJobID int64, triggeringPayload string, repo aiTagBatchRepo, client ai.AIProvider, obsRepo repository.TagObservationRepository, governance TagGovernanceMerger, platformSvc aiTagBatchPlatformSvc, taskRepo aiTagBatchTaskRepo, aiTagChecker AITagPresenceChecker, batchMode string) error {
	if governance == nil {
		return fmt.Errorf("batch merge: governance service is nil")
	}
	effectiveMode := normalizeAITagBatchMode(batchMode)
	if client == nil || client.Name() != "doubao" {
		effectiveMode = aiTagBatchModeAuto
	}

	if limiter := aiTagConcurrencyLimiter.Load(); limiter != nil {
		release, err := limiter.Acquire(ctx)
		if err != nil {
			return fmt.Errorf("acquire concurrency slot: %w", err)
		}
		defer release()
	}

	// 构建当前触发 job
	var tp AITagPayload
	if err := json.Unmarshal([]byte(triggeringPayload), &tp); err != nil {
		return fmt.Errorf("parse triggering payload: %w", err)
	}

	triggeringJob := domain.AsyncJob{
		ID:      triggeringJobID,
		Payload: triggeringPayload,
	}
	if persistedJob, err := repo.FindByID(triggeringJobID); err == nil && persistedJob != nil {
		triggeringJob = *persistedJob
	}

	// 额外抓取最多 (aiTagBatchSize - 1) 个 ready job
	extraJobs := []domain.AsyncJob{}
	if effectiveMode != aiTagBatchModeSingle {
		var err error
		extraJobs, err = repo.FindAndClaimReadyJobs(ctx, "ai_tag_generation", aiTagBatchSize-1)
		if err != nil {
			log.Printf("AI 批量标签抓取额外任务失败: %v，仅处理当前 job", err)
		}
		extraJobs = isolateClaimedJobsByBatch(ctx, triggeringJob, extraJobs, repo, taskRepo)
	}

	allJobs := append([]domain.AsyncJob{triggeringJob}, extraJobs...)
	log.Printf("AI 批量标签任务批次: job_ids=%v total=%d", jobIDsList(allJobs), len(allJobs))

	requests := make([]ai.TagRequest, 0, len(allJobs))
	requestJobs := make([]domain.AsyncJob, 0, len(allJobs))
	completedIDs := make([]int64, 0, len(allJobs))
	skippedIDs := make([]int64, 0, len(allJobs))
	failedIDs := make([]int64, 0, len(allJobs))
	var triggeringErr error
	for _, job := range allJobs {
		var p AITagPayload
		if err := json.Unmarshal([]byte(job.Payload), &p); err != nil {
			log.Printf("AI 批量标签解析 payload 失败: job_id=%d error=%v，标记失败", job.ID, err)
			markJobFailed(&job, fmt.Sprintf("parse payload: %v", err))
			_ = repo.Update(&job)
			failedIDs = append(failedIDs, job.ID)
			continue
		}
		prompt := p.Prompt
		if prompt == "" {
			prompt = DefaultTagPrompt
		}
		if aiTagChecker != nil {
			hasAITags, err := aiTagChecker.HasAITags(ctx, p.ImageID)
			if err != nil {
				return fmt.Errorf("check existing ai tags: %w", err)
			}
			if hasAITags {
				markJobFinished(&job)
				_ = repo.Update(&job)
				skippedIDs = append(skippedIDs, job.ID)
				continue
			}
		}
		requests = append(requests, ai.TagRequest{
			ImageID: p.ImageID,
			Path:    p.Path,
			Prompt:  prompt,
		})
		requestJobs = append(requestJobs, job)
	}

	if len(requests) == 0 {
		if len(skippedIDs) > 0 {
			if platformSvc != nil {
				if err := platformSvc.MarkJobsCompleted(ctx, skippedIDs); err != nil {
					log.Printf("AI 批量标签同步平台任务完成状态失败: job_ids=%v error=%v", skippedIDs, err)
				}
			}
			log.Printf("AI 批量标签全部跳过: total=%d skipped=%d", len(allJobs), len(skippedIDs))
			if len(failedIDs) == 0 {
				return nil
			}
		}
		if len(failedIDs) > 0 && platformSvc != nil {
			if err := platformSvc.MarkJobsFailed(ctx, failedIDs, "payload parsing failed"); err != nil {
				log.Printf("AI 批量标签同步无效任务失败状态失败: job_ids=%v error=%v", failedIDs, err)
			}
		}
		if len(failedIDs) > 0 && len(skippedIDs) == 0 {
			return fmt.Errorf("all %d batch payloads invalid", len(allJobs))
		}
		return nil
	}

	if len(failedIDs) > 0 && platformSvc != nil {
		if err := platformSvc.MarkJobsFailed(ctx, failedIDs, "payload parsing failed"); err != nil {
			log.Printf("AI 批量标签同步无效任务失败状态失败: job_ids=%v error=%v", failedIDs, err)
		}
		failedIDs = nil
	}

	var batchResult *ai.BatchTagResult
	if effectiveMode != aiTagBatchModeMulti && len(requests) == 1 {
		if len(skippedIDs) > 0 && platformSvc != nil {
			if err := platformSvc.MarkJobsCompleted(ctx, skippedIDs); err != nil {
				log.Printf("AI 批量标签同步平台任务完成状态失败: job_ids=%v error=%v", skippedIDs, err)
			}
		}
		result, err := client.GenerateTags(ctx, requests[0].Path, requests[0].Prompt)
		if err != nil {
			return markAllBatchJobsFailed(ctx, requestJobs, repo, platformSvc, err)
		}
		batchResult = &ai.BatchTagResult{
			Groups:     [][]string{result.Tags},
			ModelName:  result.ModelName,
			Confidence: result.Confidence,
		}
	} else {
		if len(skippedIDs) > 0 && platformSvc != nil {
			if err := platformSvc.MarkJobsCompleted(ctx, skippedIDs); err != nil {
				log.Printf("AI 批量标签同步平台任务完成状态失败: job_ids=%v error=%v", skippedIDs, err)
			}
		}
		var err error
		batchResult, err = client.GenerateTagsBatch(ctx, requests)
		if err != nil {
			return markAllBatchJobsFailed(ctx, requestJobs, repo, platformSvc, err)
		}
	}

	for i := range requestJobs {
		if i >= len(batchResult.Groups) {
			break
		}
		tags := batchResult.Groups[i]
		if err := validateGeneratedTags(tags); err != nil {
			log.Printf("AI 批量标签结果校验失败: job_id=%d image_id=%d error=%v", requestJobs[i].ID, extractImageID(requestJobs[i].Payload), err)
			markJobFailed(&requestJobs[i], err.Error())
			_ = repo.Update(&requestJobs[i])
			failedIDs = append(failedIDs, requestJobs[i].ID)
			if requestJobs[i].ID == triggeringJobID {
				triggeringErr = fmt.Errorf("validate tags: %w", err)
			}
			continue
		}

		payload := extractPayload(requestJobs[i].Payload)
		obs := &domain.TagObservation{
			ImageID:      payload.ImageID,
			RawText:      strings.Join(tags, ", "),
			Confidence:   batchResult.Confidence,
			EvidenceType: "ai_generated",
			Provider:     client.Name(),
			ModelName:    batchResult.ModelName,
			CreatedAt:    time.Now(),
		}

		if err := obsRepo.Save(ctx, obs); err != nil {
			log.Printf("AI 批量标签保存观测失败: job_id=%d error=%v", requestJobs[i].ID, err)
			markJobFailed(&requestJobs[i], fmt.Sprintf("save observation: %v", err))
			_ = repo.Update(&requestJobs[i])
			failedIDs = append(failedIDs, requestJobs[i].ID)
			if requestJobs[i].ID == triggeringJobID {
				triggeringErr = fmt.Errorf("save observation: %w", err)
			}
			continue
		}

		if err := governance.MergeTags(ctx, obs.ImageID, tags, obs.ID, obs.Confidence); err != nil {
			log.Printf("AI 批量标签合并标签失败: job_id=%d image_id=%d error=%v", requestJobs[i].ID, obs.ImageID, err)
			markJobFailed(&requestJobs[i], fmt.Sprintf("merge tags: %v", err))
			_ = repo.Update(&requestJobs[i])
			failedIDs = append(failedIDs, requestJobs[i].ID)
			if requestJobs[i].ID == triggeringJobID {
				triggeringErr = fmt.Errorf("merge tags: %w", err)
			}
			continue
		}

		markJobFinished(&requestJobs[i])
		_ = repo.Update(&requestJobs[i])
		completedIDs = append(completedIDs, requestJobs[i].ID)
		log.Printf("AI 批量标签单个任务完成: job_id=%d image_id=%d tag_count=%d", requestJobs[i].ID, obs.ImageID, len(tags))
	}

	if len(completedIDs) > 0 && platformSvc != nil {
		if err := platformSvc.MarkJobsCompleted(ctx, completedIDs); err != nil {
			log.Printf("AI 批量标签同步平台任务完成状态失败: job_ids=%v error=%v", completedIDs, err)
		}
	}
	if len(failedIDs) > 0 && platformSvc != nil {
		if err := platformSvc.MarkJobsFailed(ctx, failedIDs, "tag processing failed"); err != nil {
			log.Printf("AI 批量标签同步平台任务失败状态失败: job_ids=%v error=%v", failedIDs, err)
		}
	}
	if triggeringErr != nil {
		return triggeringErr
	}

	log.Printf("AI 批量标签全部完成: total=%d success=%d failed=%d", len(completedIDs)+len(failedIDs), len(completedIDs), len(failedIDs))
	return nil
}

func isolateClaimedJobsByBatch(ctx context.Context, triggeringJob domain.AsyncJob, jobs []domain.AsyncJob, repo aiTagBatchRepo, taskRepo aiTagBatchTaskRepo) []domain.AsyncJob {
	if taskRepo == nil || triggeringJob.PlatformTaskID == nil {
		return jobs
	}
	triggerTask, err := taskRepo.FindByID(ctx, *triggeringJob.PlatformTaskID)
	if err != nil || triggerTask == nil {
		return jobs
	}
	filtered := make([]domain.AsyncJob, 0, len(jobs))
	for _, job := range jobs {
		if job.PlatformTaskID == nil {
			releaseClaimedJob(&job, repo)
			continue
		}
		task, err := taskRepo.FindByID(ctx, *job.PlatformTaskID)
		if err != nil || task == nil || task.BatchID != triggerTask.BatchID {
			releaseClaimedJob(&job, repo)
			continue
		}
		filtered = append(filtered, job)
	}
	return filtered
}

func releaseClaimedJob(job *domain.AsyncJob, repo aiTagBatchRepo) {
	if job == nil {
		return
	}
	job.Status = "ready"
	job.StartedAt = nil
	job.FinishedAt = nil
	job.Error = nil
	job.Progress = 0
	_ = repo.Update(job)
}

func containsJobID(jobIDs []int64, target int64) bool {
	for _, jobID := range jobIDs {
		if jobID == target {
			return true
		}
	}
	return false
}

func markAllBatchJobsFailed(ctx context.Context, jobs []domain.AsyncJob, repo aiTagBatchRepo, platformSvc aiTagBatchPlatformSvc, err error) error {
	errText := err.Error()
	jobIDs := make([]int64, len(jobs))
	for i := range jobs {
		markJobFailed(&jobs[i], errText)
		_ = repo.Update(&jobs[i])
		jobIDs[i] = jobs[i].ID
	}
	if platformSvc != nil {
		if syncErr := platformSvc.MarkJobsFailed(ctx, jobIDs, errText); syncErr != nil {
			log.Printf("AI 批量标签同步平台任务失败状态失败: job_ids=%v error=%v", jobIDs, syncErr)
		}
	}
	return fmt.Errorf("generate tags: %w", err)
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

func markJobFailed(job *domain.AsyncJob, errMsg string) {
	now := time.Now()
	job.Status = "failed"
	job.Error = &errMsg
	job.FinishedAt = &now
}

func markJobFinished(job *domain.AsyncJob) {
	now := time.Now()
	job.Status = "finished"
	job.Progress = 100
	job.FinishedAt = &now
}

func extractImageID(payload string) int64 {
	var p AITagPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return 0
	}
	return p.ImageID
}

func extractPayload(payload string) AITagPayload {
	var p AITagPayload
	_ = json.Unmarshal([]byte(payload), &p)
	return p
}

func jobIDsList(jobs []domain.AsyncJob) []int64 {
	ids := make([]int64, len(jobs))
	for i, j := range jobs {
		ids[i] = j.ID
	}
	return ids
}
