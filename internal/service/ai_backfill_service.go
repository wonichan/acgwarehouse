package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/logger"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

// BackfillPreviewResult holds the preview counts for a backfill operation.
type BackfillPreviewResult struct {
	HitCount              int64 `json:"hit_count"`
	EnqueueableCount      int64 `json:"enqueueable_count"`
	SkippedWithAITag      int64 `json:"skipped_with_ai_tag"`
	SkippedWithActiveTask int64 `json:"skipped_with_active_task"`
	SkippedTotal          int64 `json:"skipped_total"`
}

// BackfillExecuteResult holds the result of executing a backfill operation.
type BackfillExecuteResult struct {
	Success           bool                  `json:"success"`
	BatchID           int64                 `json:"batch_id,omitempty"`
	CreatedTasks      int                   `json:"created_tasks"`
	SkippedTotal      int64                 `json:"skipped_total"`
	SkippedWithAITag  int64                 `json:"skipped_with_ai_tag"`
	SkippedWithActive int64                 `json:"skipped_with_active_task"`
	NoOpReason        string                `json:"no_op_reason,omitempty"`
	Batch             *domain.TaskBatch     `json:"batch,omitempty"`
	CreatedTaskList   []domain.PlatformTask `json:"created_task_list,omitempty"`
}

type ExistingJobLoader interface {
	LoadExistingJob(job *domain.AsyncJob) bool
	QueuedByType(jobType string) int
}

// AIBackfillService orchestrates filtered backfill preview and execution.
type AIBackfillService struct {
	imageRepo       repository.BackfillImageQuery
	taskPlatformSvc *TaskPlatformService
	jobLoader       ExistingJobLoader
	configProvider  func() *config.Config
}

// NewAIBackfillService creates a new backfill service.
func NewAIBackfillService(imageRepo repository.BackfillImageQuery, taskPlatformSvc *TaskPlatformService, jobLoader ExistingJobLoader, configProvider func() *config.Config) *AIBackfillService {
	return &AIBackfillService{
		imageRepo:       imageRepo,
		taskPlatformSvc: taskPlatformSvc,
		jobLoader:       jobLoader,
		configProvider:  configProvider,
	}
}

// PreviewBackfill evaluates the current filtered result and returns classification counts.
// It rejects unfiltered requests per D-02.
func (s *AIBackfillService) PreviewBackfill(ctx context.Context, filter repository.BackfillCandidateFilter) (*BackfillPreviewResult, error) {
	if !IsFilterNarrowed(filter) {
		return nil, fmt.Errorf("backfill requires at least one narrowing filter (tag_ids or has_tags); unfiltered full-library scans are not allowed per D-02")
	}

	hitCount, err := s.imageRepo.CountBackfillHitCount(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("counting filtered images: %w", err)
	}

	enqueueableCount, err := s.imageRepo.CountBackfillCandidates(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("counting eligible candidates: %w", err)
	}

	skippedWithAITag, err := s.imageRepo.CountBackfillSkippedWithAITag(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("counting skipped-with-ai-tag: %w", err)
	}

	skippedWithActiveTask, err := s.imageRepo.CountBackfillSkippedWithActiveTask(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("counting skipped-with-active-task: %w", err)
	}

	skippedTotal := skippedWithAITag + skippedWithActiveTask

	return &BackfillPreviewResult{
		HitCount:              hitCount,
		EnqueueableCount:      enqueueableCount,
		SkippedWithAITag:      skippedWithAITag,
		SkippedWithActiveTask: skippedWithActiveTask,
		SkippedTotal:          skippedTotal,
	}, nil
}

// ExecuteBackfill creates a manual_batch for eligible candidates and returns structured results.
// Returns an explicit no-op result when zero tasks are created per D-13.
func (s *AIBackfillService) ExecuteBackfill(ctx context.Context, filter repository.BackfillCandidateFilter, prompt string) (*BackfillExecuteResult, error) {
	logger.Infof("AI 标签补跑开始: has_tag_ids=%t has_has_tags=%t custom_prompt=%t", len(filter.TagIDs) > 0, filter.HasTags != nil, prompt != "")
	preview, err := s.PreviewBackfill(ctx, filter)
	if err != nil {
		logger.Errorf("AI 标签补跑预览失败: error=%v", err)
		return nil, err
	}
	logger.Infof("AI 标签补跑预览完成: hit_count=%d enqueueable_count=%d skipped_with_ai_tag=%d skipped_with_active_task=%d", preview.HitCount, preview.EnqueueableCount, preview.SkippedWithAITag, preview.SkippedWithActiveTask)

	// D-13: Explicit no-op when no eligible candidates
	if preview.EnqueueableCount == 0 {
		reason := "当前筛选结果中没有可补跑的图片"
		if preview.SkippedWithAITag > 0 && preview.SkippedWithActiveTask > 0 {
			reason = fmt.Sprintf("当前筛选结果中没有可补跑的图片（%d 张已有 AI 标签， %d 张已有在途任务）", preview.SkippedWithAITag, preview.SkippedWithActiveTask)
		} else if preview.SkippedWithAITag > 0 {
			reason = fmt.Sprintf("当前筛选结果中没有可补跑的图片（%d 张已有 AI 标签）", preview.SkippedWithAITag)
		} else if preview.SkippedWithActiveTask > 0 {
			reason = fmt.Sprintf("当前筛选结果中没有可补跑的图片（%d 张已有在途任务）", preview.SkippedWithActiveTask)
		}
		return &BackfillExecuteResult{
			Success:           false,
			CreatedTasks:      0,
			SkippedTotal:      preview.SkippedTotal,
			SkippedWithAITag:  preview.SkippedWithAITag,
			SkippedWithActive: preview.SkippedWithActiveTask,
			NoOpReason:        reason,
		}, nil
	}

	// Fetch eligible candidate images
	candidates, err := s.imageRepo.FindBackfillCandidates(ctx, filter)
	if err != nil {
		logger.Errorf("AI 标签补跑拉取候选图片失败: error=%v", err)
		return nil, fmt.Errorf("fetching backfill candidates: %w", err)
	}
	logger.Infof("AI 标签补跑候选图片已加载: candidate_count=%d", len(candidates))
	thumbnailBaseURL := ResolveRelativeThumbnailBaseURL(s.currentConfig())

	if len(candidates) == 0 {
		logger.Infof("AI 标签补跑无候选图片: enqueueable_count=%d", preview.EnqueueableCount)
		return &BackfillExecuteResult{
			Success:           false,
			CreatedTasks:      0,
			SkippedTotal:      preview.SkippedTotal,
			SkippedWithAITag:  preview.SkippedWithAITag,
			SkippedWithActive: preview.SkippedWithActiveTask,
			NoOpReason:        "当前筛选结果中没有可补跑的图片",
		}, nil
	}

	// Build plan items for TaskPlatformService
	images := make([]*domain.Image, len(candidates))
	for i := range candidates {
		images[i] = &candidates[i]
	}

	items := make([]TaskPlatformPlanItem, 0, len(images))
	roots := make([]string, 0, len(images))
	for _, img := range images {
		items = append(items, TaskPlatformPlanItem{
			ImageID:          img.ID,
			ImageVersionKey:  BuildImageVersionKey(img),
			SourceDescriptor: img.Path,
		})
		roots = append(roots, img.SourceRoot)
	}

	plan, err := s.taskPlatformSvc.PlanBatch(ctx, TaskPlatformPlanRequest{
		SourceType:   domain.TaskBatchSourceManualBatch,
		SummaryLabel: BuildTaskBatchSummaryLabel(domain.TaskBatchSourceManualBatch, roots, len(items)),
		SourceRoots:  roots,
		TaskTypes:    []string{domain.PlatformTaskTypeAITagGeneration},
		Items:        items,
	})
	if err != nil {
		logger.Errorf("AI 标签补跑批次规划失败: item_count=%d error=%v", len(items), err)
		return nil, fmt.Errorf("planning backfill batch: %w", err)
	}
	logger.Infof("AI 标签补跑批次规划完成: batch_id=%d created_tasks=%d", plan.Batch.ID, len(plan.CreatedTasks))

	// Queue each created task with AI tag payload
	queuedTasks := make([]domain.PlatformTask, 0, len(plan.CreatedTasks))
	for i := range plan.CreatedTasks {
		task := plan.CreatedTasks[i]
		// Find the matching image for this task
		var matchImg *domain.Image
		for idx := range candidates {
			if candidates[idx].ID == task.ImageID {
				matchImg = &candidates[idx]
				break
			}
		}
		if matchImg == nil {
			logger.Infof("AI 标签补跑任务缺少图片映射: task_id=%d image_id=%d", task.ID, task.ImageID)
			return nil, fmt.Errorf("image not found for task %d", task.ID)
		}
		payload, err := json.Marshal(map[string]interface{}{
			"image_id": task.ImageID,
			"path":     ResolveAITagImagePath(matchImg, thumbnailBaseURL),
			"prompt":   prompt,
		})
		if err != nil {
			logger.Errorf("AI 标签补跑任务 payload 序列化失败: task_id=%d image_id=%d error=%v", task.ID, task.ImageID, err)
			return nil, fmt.Errorf("marshalling task payload: %w", err)
		}
		job, err := s.taskPlatformSvc.QueueTask(ctx, &task, domain.PlatformTaskTypeAITagGeneration, string(payload))
		if err != nil {
			logger.Errorf("AI 标签补跑任务入队失败: batch_id=%d task_id=%d image_id=%d error=%v", plan.Batch.ID, task.ID, task.ImageID, err)
			return nil, fmt.Errorf("queueing task %d: %w", task.ID, err)
		}
		logger.Infof("AI 标签补跑任务已入队: batch_id=%d task_id=%d image_id=%d job_id=%d", plan.Batch.ID, task.ID, task.ImageID, job.ID)
		if s.jobLoader != nil && s.jobLoader.QueuedByType(domain.PlatformTaskTypeAITagGeneration) < ResolveAITagQueueLimit(s.currentConfig()) {
			s.jobLoader.LoadExistingJob(job)
		}
		queuedTasks = append(queuedTasks, task)
	}
	logger.Infof("AI 标签补跑完成: batch_id=%d created_tasks=%d skipped_total=%d", plan.Batch.ID, len(queuedTasks), preview.SkippedTotal)

	return &BackfillExecuteResult{
		Success:           true,
		BatchID:           plan.Batch.ID,
		CreatedTasks:      len(queuedTasks),
		SkippedTotal:      preview.SkippedTotal,
		SkippedWithAITag:  preview.SkippedWithAITag,
		SkippedWithActive: preview.SkippedWithActiveTask,
		Batch:             plan.Batch,
		CreatedTaskList:   queuedTasks,
	}, nil
}

// IsFilterNarrowed returns true if the filter has at least one narrowing criterion.
// Per D-02, unfiltered requests must be rejected.
func IsFilterNarrowed(filter repository.BackfillCandidateFilter) bool {
	if len(filter.TagIDs) > 0 {
		return true
	}
	if filter.HasTags != nil {
		return true
	}
	return false
}

func (s *AIBackfillService) currentConfig() *config.Config {
	if s == nil || s.configProvider == nil {
		return nil
	}
	return s.configProvider()
}
