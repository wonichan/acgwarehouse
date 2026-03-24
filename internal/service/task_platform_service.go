package service

import (
	"context"
	"fmt"
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
	for _, item := range req.Items {
		createdForImage := false
		for _, taskType := range uniqueNonEmptyStrings(req.TaskTypes) {
			dedupeKey := buildPlatformTaskDedupeKey(item.ImageVersionKey, taskType)
			existing, err := s.taskRepo.FindActiveByDedupeKey(ctx, dedupeKey)
			if err != nil {
				return nil, err
			}
			if existing != nil {
				batch.SkippedDuplicateTasks++
				if existing.Status == domain.PlatformTaskStatusCompleted {
					batch.SkippedUnchanged++
				}
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

func buildPlatformTaskDedupeKey(imageVersionKey, taskType string) string {
	if imageVersionKey == "" {
		return taskType
	}
	return fmt.Sprintf("%s:%s", imageVersionKey, taskType)
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
