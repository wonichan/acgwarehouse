package service

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/logger"
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
	SourceType                string
	TriggerKey                string
	SummaryLabel              string
	SourceRoots               []string
	TaskTypes                 []string
	IgnoreHistoricalCompleted bool
	Items                     []TaskPlatformPlanItem
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
	logger.Infof("[service] PlanBatch started: source_type=%s summary_label=%s total_images=%d", req.SourceType, req.SummaryLabel, len(req.Items))
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
		logger.Errorf("[service] PlanBatch failed: error=%v", err)
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
	existingByDedupeKey, err := s.buildExistingTaskLookup(ctx, req.Items, taskTypes, req.IgnoreHistoricalCompleted)
	if err != nil {
		logger.Errorf("[service] PlanBatch failed: error=%v", err)
		return nil, err
	}
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
		exists, err := s.taskRepo.ImageExists(ctx, item.ImageID)
		if err != nil {
			logger.Errorf("[service] PlanBatch image existence check failed: image_id=%d error=%v", item.ImageID, err)
			return nil, err
		}
		if !exists {
			batch.SkippedImages++
			logger.Warnf("[service] PlanBatch skipped missing image: image_id=%d source_type=%s task_types=%d", item.ImageID, req.SourceType, len(taskTypes))
			continue
		}
		createdForImage := false
		for _, taskType := range taskTypes {
			dedupeKey := buildPlatformTaskDedupeKey(item.ImageVersionKey, taskType)
			existing := existingByDedupeKey[dedupeKey]
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
				logger.Errorf("[service] PlanBatch failed: image_id=%d task_type=%s error=%v", item.ImageID, taskType, err)
				return nil, err
			}
			createdTasks = append(createdTasks, task)
			taskCopy := task
			existingByDedupeKey[dedupeKey] = &taskCopy
			createdImages[item.ImageID] = struct{}{}
			createdForImage = true
		}
		if !createdForImage {
			batch.SkippedImages++
		}
	}

	batch.NewImages = int64(len(createdImages))
	if err := s.batchRepo.Update(ctx, batch); err != nil {
		logger.Errorf("[service] PlanBatch failed: error=%v", err)
		return nil, err
	}

	refreshed, err := s.batchRepo.RefreshStatus(ctx, batch.ID)
	if err != nil {
		logger.Errorf("[service] PlanBatch failed: error=%v", err)
		return nil, err
	}
	result := &TaskPlatformPlanResult{Batch: refreshed, CreatedTasks: createdTasks}
	logger.Infof("[service] PlanBatch completed: batch_id=%d created_tasks=%d", result.Batch.ID, len(result.CreatedTasks))
	return result, nil
}

func (s *TaskPlatformService) buildExistingTaskLookup(
	ctx context.Context,
	items []TaskPlatformPlanItem,
	taskTypes []string,
	ignoreHistoricalCompleted bool,
) (map[string]*domain.PlatformTask, error) {
	wantedKeys := make(map[string]struct{})
	for _, item := range items {
		if item.SkipPlanning {
			continue
		}
		for _, taskType := range taskTypes {
			wantedKeys[buildPlatformTaskDedupeKey(item.ImageVersionKey, taskType)] = struct{}{}
		}
	}

	lookup := make(map[string]*domain.PlatformTask, len(wantedKeys))
	if len(wantedKeys) == 0 {
		return lookup, nil
	}

	for _, status := range []string{
		domain.PlatformTaskStatusRunning,
		domain.PlatformTaskStatusQueued,
		domain.PlatformTaskStatusPending,
		domain.PlatformTaskStatusCompleted,
	} {
		tasks, err := s.taskRepo.List(ctx, repository.PlatformTaskListFilter{Status: status, Limit: 1000000, Offset: 0})
		if err != nil {
			return nil, err
		}
		for i := range tasks {
			task := tasks[i]
			if _, ok := wantedKeys[task.DedupeKey]; !ok {
				continue
			}
			if ignoreHistoricalCompleted && task.Status == domain.PlatformTaskStatusCompleted {
				continue
			}
			current := lookup[task.DedupeKey]
			if current == nil || platformTaskStatusPriority(task.Status) < platformTaskStatusPriority(current.Status) {
				taskCopy := task
				lookup[task.DedupeKey] = &taskCopy
			}
		}
	}

	return lookup, nil
}

func platformTaskStatusPriority(status string) int {
	switch status {
	case domain.PlatformTaskStatusRunning:
		return 0
	case domain.PlatformTaskStatusQueued:
		return 1
	case domain.PlatformTaskStatusPending:
		return 2
	case domain.PlatformTaskStatusCompleted:
		return 3
	default:
		return 4
	}
}

func (s *TaskPlatformService) RefreshBatchStatus(ctx context.Context, batchID int64) (*domain.TaskBatch, error) {
	logger.Infof("[service] RefreshBatchStatus started: batch_id=%d", batchID)
	batch, err := s.batchRepo.RefreshStatus(ctx, batchID)
	if err != nil {
		logger.Errorf("[service] RefreshBatchStatus failed: batch_id=%d error=%v", batchID, err)
		return nil, err
	}
	return batch, nil
}

func (s *TaskPlatformService) QueueTask(ctx context.Context, task *domain.PlatformTask, jobType, payload string) (*domain.AsyncJob, error) {
	taskType := ""
	if task != nil {
		taskType = task.TaskType
	}
	logger.Infof("[service] QueueTask started: task_type=%s job_type=%s", taskType, jobType)
	if task == nil {
		err := fmt.Errorf("platform task is required")
		logger.Errorf("[service] QueueTask failed: task_type=%s job_type=%s error=%v", taskType, jobType, err)
		return nil, err
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
		logger.Errorf("[service] QueueTask failed: error=%v", err)
		return nil, err
	}
	if err := s.taskRepo.SetLatestAsyncJob(ctx, task.ID, &job.ID); err != nil {
		logger.Errorf("[service] QueueTask failed: error=%v", err)
		// Mark the orphaned job as failed so refillLoop won't pick it up as a ghost task.
		errMsg := fmt.Sprintf("orphaned: SetLatestAsyncJob failed: %v", err)
		if cleanupErr := s.jobRepo.UpdateStatus(job.ID, "failed", &errMsg); cleanupErr != nil {
			logger.Errorf("[service] QueueTask cleanup failed: job_id=%d error=%v", job.ID, cleanupErr)
		}
		return nil, err
	}
	queuedTask, err := s.taskRepo.FindByID(ctx, task.ID)
	if err != nil {
		logger.Errorf("[service] QueueTask failed: error=%v", err)
		return nil, err
	}
	*task = *queuedTask
	if _, err := s.batchRepo.RefreshStatus(ctx, task.BatchID); err != nil {
		logger.Errorf("[service] QueueTask failed: error=%v", err)
		return nil, err
	}
	logger.Infof("[service] QueueTask completed: job_id=%d", job.ID)
	return job, nil
}

func (s *TaskPlatformService) MarkJobRunning(ctx context.Context, jobID int64) error {
	logger.Infof("[service] MarkJobRunning started: job_id=%d", jobID)
	if err := s.updateTaskStatusForJob(ctx, jobID, domain.PlatformTaskStatusRunning, nil); err != nil {
		logger.Errorf("[service] MarkJobRunning failed: job_id=%d error=%v", jobID, err)
		return err
	}
	logger.Infof("[service] MarkJobRunning completed: job_id=%d", jobID)
	return nil
}

func (s *TaskPlatformService) MarkJobCompleted(ctx context.Context, jobID int64) error {
	logger.Infof("[service] MarkJobCompleted started: job_id=%d", jobID)
	if err := s.updateTaskStatusForJob(ctx, jobID, domain.PlatformTaskStatusCompleted, nil); err != nil {
		logger.Errorf("[service] MarkJobCompleted failed: job_id=%d error=%v", jobID, err)
		return err
	}
	logger.Infof("[service] MarkJobCompleted completed: job_id=%d", jobID)
	return nil
}

func (s *TaskPlatformService) MarkJobFailed(ctx context.Context, jobID int64, errorSummary string) error {
	logger.Errorf("[service] MarkJobFailed started: job_id=%d error=%s", jobID, errorSummary)
	if err := s.updateTaskStatusForJob(ctx, jobID, domain.PlatformTaskStatusFailed, &errorSummary); err != nil {
		logger.Errorf("[service] MarkJobFailed failed: job_id=%d error=%v", jobID, err)
		return err
	}
	logger.Infof("[service] MarkJobFailed completed: job_id=%d", jobID)
	return nil
}

func (s *TaskPlatformService) MarkJobsCompleted(ctx context.Context, jobIDs []int64) error {
	for _, jobID := range jobIDs {
		if err := s.MarkJobCompleted(ctx, jobID); err != nil {
			return err
		}
	}
	return nil
}

func (s *TaskPlatformService) MarkJobsFailed(ctx context.Context, jobIDs []int64, errorSummary string) error {
	for _, jobID := range jobIDs {
		if err := s.MarkJobFailed(ctx, jobID, errorSummary); err != nil {
			return err
		}
	}
	return nil
}

func (s *TaskPlatformService) updateTaskStatusForJob(ctx context.Context, jobID int64, status string, errorSummary *string) error {
	job, err := s.jobRepo.FindByID(jobID)
	if err != nil {
		logger.Errorf("[service] updateTaskStatusForJob failed: job_id=%d status=%s error=%v", jobID, status, err)
		return err
	}
	if job.PlatformTaskID == nil {
		return nil
	}
	task, err := s.taskRepo.FindByID(ctx, *job.PlatformTaskID)
	if err != nil {
		logger.Errorf("[service] updateTaskStatusForJob failed: job_id=%d task_id=%d status=%s error=%v", jobID, *job.PlatformTaskID, status, err)
		return err
	}
	if task.Status == domain.PlatformTaskStatusCancelled {
		return nil
	}
	now := time.Now()
	switch status {
	case domain.PlatformTaskStatusRunning:
		task.Status = domain.PlatformTaskStatusRunning
		task.StartedAt = &now
		task.FinishedAt = nil
		task.ErrorSummary = nil
	case domain.PlatformTaskStatusCompleted:
		task.Status = domain.PlatformTaskStatusCompleted
		if task.StartedAt == nil {
			task.StartedAt = &now
		}
		task.FinishedAt = &now
		task.ErrorSummary = nil
	case domain.PlatformTaskStatusFailed:
		task.Status = domain.PlatformTaskStatusFailed
		if task.StartedAt == nil {
			task.StartedAt = &now
		}
		task.FinishedAt = &now
		task.ErrorSummary = errorSummary
	default:
		task.Status = status
	}
	if err := s.taskRepo.Update(ctx, task); err != nil {
		logger.Errorf("[service] updateTaskStatusForJob failed: job_id=%d task_id=%d status=%s error=%v", jobID, task.ID, status, err)
		return err
	}
	_, err = s.batchRepo.RefreshStatus(ctx, task.BatchID)
	if err != nil {
		logger.Errorf("[service] updateTaskStatusForJob failed: job_id=%d batch_id=%d status=%s error=%v", jobID, task.BatchID, status, err)
	}
	return err
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
