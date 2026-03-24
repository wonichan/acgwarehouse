package service

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

type TaskPlatformPlanItem struct {
	ImageID          int64
	ImageVersionKey  string
	SourceDescriptor string
	SkipPlanning     bool
	SkipReason       string
}

type TaskPlatformPlanRequest struct {
	SourceType   string
	TriggerKey   string
	SummaryLabel string
	SourceRoots  []string
	TaskTypes    []string
	Items        []TaskPlatformPlanItem
}

type TaskPlatformPlanResult struct {
	Batch        *domain.TaskBatch
	CreatedTasks []domain.PlatformTask
}

type TaskPlatformService struct {
	batchRepo repository.TaskBatchRepository
	taskRepo  repository.PlatformTaskRepository
	jobRepo   repository.JobRepository
}

func NewTaskPlatformService(
	batchRepo repository.TaskBatchRepository,
	taskRepo repository.PlatformTaskRepository,
	jobRepo repository.JobRepository,
) *TaskPlatformService {
	return &TaskPlatformService{
		batchRepo: batchRepo,
		taskRepo:  taskRepo,
		jobRepo:   jobRepo,
	}
}

func (s *TaskPlatformService) PlanBatch(ctx context.Context, req TaskPlatformPlanRequest) (*TaskPlatformPlanResult, error) {
	now := time.Now()
	batch := &domain.TaskBatch{
		SourceType:   req.SourceType,
		TriggerKey:   req.TriggerKey,
		SummaryLabel: req.SummaryLabel,
		Status:       domain.TaskBatchStatusPending,
		TotalImages:  int64(len(req.Items)),
		CreatedAt:    now,
	}
	if err := s.batchRepo.Create(ctx, batch); err != nil {
		return nil, err
	}

	for _, sourceRoot := range uniqueNonEmptyStrings(req.SourceRoots) {
		if err := s.batchRepo.AddSource(ctx, &domain.TaskBatchSource{
			BatchID:     batch.ID,
			SourceRoot:  sourceRoot,
			SourceLabel: sourceRoot,
		}); err != nil {
			return nil, err
		}
	}

	createdTasks := make([]domain.PlatformTask, 0)
	createdImages := make(map[int64]struct{})
	taskTypes := uniqueNonEmptyStrings(req.TaskTypes)
	for _, item := range req.Items {
		if item.SkipPlanning {
			batch.SkippedImages++
			if item.SkipReason == domain.PlatformTaskSkipReasonUnchanged {
				batch.SkippedUnchanged++
			}
			if len(taskTypes) > 0 {
				batch.SkippedDuplicateTasks += int64(len(taskTypes))
			}
			continue
		}
		createdForImage := false
		for _, taskType := range taskTypes {
			dedupeKey := buildPlatformTaskDedupeKey(item.ImageVersionKey, taskType)
			existing, err := s.taskRepo.FindActiveByDedupeKey(ctx, dedupeKey)
			if err != nil {
				return nil, err
			}
			if existing != nil {
				batch.SkippedDuplicateTasks++
				batch.SkippedUnchanged++
				continue
			}

			task := domain.PlatformTask{
				BatchID:         batch.ID,
				ImageID:         item.ImageID,
				TaskType:        taskType,
				SourceType:      req.SourceType,
				Status:          domain.PlatformTaskStatusPending,
				DedupeKey:       dedupeKey,
				ImageVersionKey: item.ImageVersionKey,
				CreatedAt:       now,
			}
			if err := s.taskRepo.Create(ctx, &task); err != nil {
				return nil, err
			}
			createdTasks = append(createdTasks, task)
			createdImages[item.ImageID] = struct{}{}
			createdForImage = true
		}
		if !createdForImage {
			batch.SkippedImages++
		}
	}

	batch.NewImages = int64(len(createdImages))
	if err := s.batchRepo.Update(ctx, batch); err != nil {
		return nil, err
	}

	refreshed, err := s.batchRepo.RefreshStatus(ctx, batch.ID)
	if err != nil {
		return nil, err
	}
	return &TaskPlatformPlanResult{Batch: refreshed, CreatedTasks: createdTasks}, nil
}

func (s *TaskPlatformService) RefreshBatchStatus(ctx context.Context, batchID int64) (*domain.TaskBatch, error) {
	return s.batchRepo.RefreshStatus(ctx, batchID)
}

func (s *TaskPlatformService) QueueTask(ctx context.Context, task *domain.PlatformTask, jobType, payload string) (*domain.AsyncJob, error) {
	if task == nil {
		return nil, fmt.Errorf("platform task is required")
	}
	job := &domain.AsyncJob{
		PlatformTaskID: &task.ID,
		Type:           jobType,
		Status:         "ready",
		Payload:        payload,
		Progress:       0,
		CreatedAt:      time.Now(),
	}
	if err := s.jobRepo.Save(job); err != nil {
		return nil, err
	}
	if err := s.taskRepo.SetLatestAsyncJob(ctx, task.ID, &job.ID); err != nil {
		return nil, err
	}
	queuedTask, err := s.taskRepo.FindByID(ctx, task.ID)
	if err != nil {
		return nil, err
	}
	*task = *queuedTask
	if _, err := s.batchRepo.RefreshStatus(ctx, task.BatchID); err != nil {
		return nil, err
	}
	return job, nil
}

func buildPlatformTaskDedupeKey(imageVersionKey, taskType string) string {
	if imageVersionKey == "" {
		return taskType
	}
	return fmt.Sprintf("%s:%s", imageVersionKey, taskType)
}

func BuildImageVersionKey(image *domain.Image) string {
	if image == nil {
		return ""
	}
	return fmt.Sprintf("image:%d:size:%d:phash:%d", image.ID, image.FileSize, image.PHash)
}

func BuildTaskBatchSummaryLabel(sourceType string, sourceRoots []string, totalImages int) string {
	roots := uniqueNonEmptyStrings(sourceRoots)
	if len(roots) == 0 {
		return fmt.Sprintf("%s batch (%d images)", strings.TrimSpace(sourceType), totalImages)
	}
	labels := make([]string, 0, len(roots))
	for _, root := range roots {
		base := filepath.Base(filepath.Clean(root))
		if base == "." || base == string(filepath.Separator) || strings.TrimSpace(base) == "" {
			base = root
		}
		labels = append(labels, base)
	}
	return fmt.Sprintf("%s batch [%s] (%d images)", strings.TrimSpace(sourceType), strings.Join(labels, ", "), totalImages)
}

func uniqueNonEmptyStrings(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
