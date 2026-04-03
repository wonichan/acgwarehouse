package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
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
	imageRepo       repository.ImageRepository
	taskPlatformSvc *TaskPlatformService
	jobLoader       ExistingJobLoader
	configProvider  func() *config.Config
}

// NewAIBackfillService creates a new backfill service.
func NewAIBackfillService(imageRepo repository.ImageRepository, taskPlatformSvc *TaskPlatformService, jobLoader ExistingJobLoader, configProvider func() *config.Config) *AIBackfillService {
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
	preview, err := s.PreviewBackfill(ctx, filter)
	if err != nil {
		return nil, err
	}

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
		return nil, fmt.Errorf("fetching backfill candidates: %w", err)
	}

	if len(candidates) == 0 {
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
		return nil, fmt.Errorf("planning backfill batch: %w", err)
	}

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
			return nil, fmt.Errorf("image not found for task %d", task.ID)
		}
		payload, err := json.Marshal(map[string]interface{}{
			"image_id": task.ImageID,
			"path":     ResolveAITagImagePath(matchImg),
			"prompt":   prompt,
		})
		if err != nil {
			return nil, fmt.Errorf("marshalling task payload: %w", err)
		}
		job, err := s.taskPlatformSvc.QueueTask(ctx, &task, domain.PlatformTaskTypeAITagGeneration, string(payload))
		if err != nil {
			return nil, fmt.Errorf("queueing task %d: %w", task.ID, err)
		}
		if s.jobLoader != nil && s.jobLoader.QueuedByType(domain.PlatformTaskTypeAITagGeneration) < ResolveAITagQueueLimit(s.currentConfig()) {
			s.jobLoader.LoadExistingJob(job)
		}
		queuedTasks = append(queuedTasks, task)
	}

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
