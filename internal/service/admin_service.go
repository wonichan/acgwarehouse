package service

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

// JobManagerControl defines the interface for controlling the job manager.
// This avoids an import cycle between service and worker packages.
type JobManagerControl interface {
	Pause()
	Resume()
	IsPaused() bool
	IsRunning() bool
	QueueSize() int
	GetWorkerCount() int
	AddJob(ctx context.Context, jobType, payload string) (int64, error)
}

// JobManagerStats holds runtime statistics about the job manager.
type JobManagerStats struct {
	QueueSize int
	IsRunning bool
	IsPaused  bool
}

// AdminService provides aggregated status and safe operations for the admin dashboard.
// It orchestrates health checks, task management, and library statistics.
type AdminService struct {
	cfg            *config.Config
	jobRepo        repository.JobRepository
	imageRepo      repository.ImageRepository
	tagRepo        repository.TagRepository
	collectionRepo repository.CollectionRepository
	jobManager     JobManagerControl
	taskReadSvc    *TaskReadService
	taskBatchRepo  repository.TaskBatchRepository
	taskRepo       repository.PlatformTaskRepository
}

// HealthStatus represents the health of the system.
type HealthStatus struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// ConfigSummary exposes non-sensitive configuration details.
type ConfigSummary struct {
	ServerHost string `json:"server_host"`
	ServerPort int    `json:"server_port"`
	Env        string `json:"env"`
	// Indicate if secrets are configured (without exposing them)
	HasAIKey         bool   `json:"has_ai_key"`
	HasCOSSecretKey  bool   `json:"has_cos_secret_key"`
	HasAdminPassword bool   `json:"-"`
	AdminUsername    string `json:"admin_username"`
}

// TaskSummary represents the task queue status.
type TaskSummary struct {
	Total    int64 `json:"total"`
	Ready    int64 `json:"ready"`
	Running  int64 `json:"running"`
	Finished int64 `json:"finished"`
	Failed   int64 `json:"failed"`
}

// LibraryStats represents the image library statistics.
type LibraryStats struct {
	TotalImages      int64 `json:"total_images"`
	TotalTags        int64 `json:"total_tags"`
	TotalCollections int64 `json:"total_collections"`
}

// Summary is the aggregated status response for the admin dashboard.
type Summary struct {
	Health  HealthStatus  `json:"health"`
	Config  ConfigSummary `json:"config"`
	Tasks   TaskSummary   `json:"tasks"`
	Library LibraryStats  `json:"library"`
}

// QueueOverview is the runtime queue state for admin monitoring.
type QueueOverview struct {
	IsRunning   bool `json:"is_running"`
	IsPaused    bool `json:"is_paused"`
	QueueSize   int  `json:"queue_size"`
	WorkerCount int  `json:"worker_count"`
}

// TaskPlatformOverview is the Phase 13 monitoring contract.
type TaskPlatformOverview struct {
	Health  HealthStatus     `json:"health"`
	Config  ConfigSummary    `json:"config"`
	Library LibraryStats     `json:"library"`
	Queue   QueueOverview    `json:"queue"`
	Batches map[string]int64 `json:"batches"`
	Tasks   map[string]int64 `json:"tasks"`
}

type RetryBatchResult struct {
	Batch        *domain.TaskBatch
	CreatedTasks []domain.PlatformTask
	RetryCount   int
}

const maxFailedRetryAttempts = 5

// NewAdminService creates a new AdminService.
func NewAdminService(
	cfg *config.Config,
	jobRepo repository.JobRepository,
	imageRepo repository.ImageRepository,
	tagRepo repository.TagRepository,
	collectionRepo repository.CollectionRepository,
	jobManager JobManagerControl,
	extras ...any,
) *AdminService {
	var taskReadSvc *TaskReadService
	var taskBatchRepo repository.TaskBatchRepository
	var taskRepo repository.PlatformTaskRepository
	for _, extra := range extras {
		switch v := extra.(type) {
		case *TaskReadService:
			taskReadSvc = v
		case repository.TaskBatchRepository:
			taskBatchRepo = v
		case repository.PlatformTaskRepository:
			taskRepo = v
		}
	}
	return &AdminService{
		cfg:            cfg,
		jobRepo:        jobRepo,
		imageRepo:      imageRepo,
		tagRepo:        tagRepo,
		collectionRepo: collectionRepo,
		jobManager:     jobManager,
		taskReadSvc:    taskReadSvc,
		taskBatchRepo:  taskBatchRepo,
		taskRepo:       taskRepo,
	}
}

// GetSummary returns an aggregated status summary for the admin dashboard.
// It includes health status, config summary (without secrets), task counts, and library stats.
func (s *AdminService) GetSummary(ctx context.Context) (*Summary, error) {
	summary := &Summary{
		Health: HealthStatus{
			Status:    "healthy",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
		Config: ConfigSummary{
			ServerHost:       s.cfg.Server.Host,
			ServerPort:       s.cfg.Server.Port,
			Env:              s.cfg.Server.Env,
			HasAIKey:         s.cfg.AI.APIKey != "",
			HasCOSSecretKey:  s.cfg.COS.SecretKey != "",
			HasAdminPassword: s.cfg.Admin.Password != "",
			AdminUsername:    s.cfg.Admin.Username,
		},
	}

	// Get task counts
	var err error
	summary.Tasks.Ready, err = s.jobRepo.CountByStatus("ready")
	if err != nil {
		log.Printf("[service] GetSummary failed: status=ready error=%v", err)
		summary.Tasks.Ready = 0
	}
	summary.Tasks.Running, err = s.jobRepo.CountByStatus("running")
	if err != nil {
		log.Printf("[service] GetSummary failed: status=running error=%v", err)
		summary.Tasks.Running = 0
	}
	summary.Tasks.Finished, err = s.jobRepo.CountByStatus("finished")
	if err != nil {
		log.Printf("[service] GetSummary failed: status=finished error=%v", err)
		summary.Tasks.Finished = 0
	}
	summary.Tasks.Failed, err = s.jobRepo.CountByStatus("failed")
	if err != nil {
		log.Printf("[service] GetSummary failed: status=failed error=%v", err)
		summary.Tasks.Failed = 0
	}
	summary.Tasks.Total = summary.Tasks.Ready + summary.Tasks.Running + summary.Tasks.Finished + summary.Tasks.Failed

	// Get library stats
	summary.Library.TotalImages, _ = s.imageRepo.Count()
	if tagCount, err := s.tagRepo.Count(ctx); err == nil {
		summary.Library.TotalTags = int64(tagCount)
	}
	summary.Library.TotalCollections, _ = s.collectionRepo.Count(ctx)

	return summary, nil
}

// GetTaskPlatformOverview returns queue runtime + platform batch/task overview for Phase 13.
func (s *AdminService) GetTaskPlatformOverview(ctx context.Context) (*TaskPlatformOverview, error) {
	summary, err := s.GetSummary(ctx)
	if err != nil {
		log.Printf("[service] GetTaskPlatformOverview failed: error=%v", err)
		return nil, err
	}

	overview := &TaskPlatformOverview{
		Health:  summary.Health,
		Config:  summary.Config,
		Library: summary.Library,
		Queue: QueueOverview{
			IsRunning:   s.jobManager.IsRunning(),
			IsPaused:    s.jobManager.IsPaused(),
			QueueSize:   s.jobManager.QueueSize(),
			WorkerCount: s.jobManager.GetWorkerCount(),
		},
		Batches: map[string]int64{},
		Tasks:   map[string]int64{},
	}

	if s.taskReadSvc == nil {
		return overview, nil
	}

	const pageSize = 200
	offset := 0
	for {
		batches, err := s.taskReadSvc.ListBatches(ctx, TaskBatchReadFilter{Limit: pageSize, Offset: offset})
		if err != nil {
			log.Printf("[service] GetTaskPlatformOverview failed: error=%v", err)
			return nil, err
		}
		if len(batches) == 0 {
			break
		}

		for _, batch := range batches {
			overview.Batches[batch.Status]++
			for status, count := range batch.StatusCounts {
				overview.Tasks[status] += count
			}
		}

		if len(batches) < pageSize {
			break
		}
		offset += len(batches)
	}

	return overview, nil
}

func (s *AdminService) ClearTaskQueue(ctx context.Context) (int, error) {
	log.Printf("[service] ClearTaskQueue started")
	if s.taskRepo == nil || s.taskBatchRepo == nil {
		err := fmt.Errorf("task control repositories not configured")
		log.Printf("[service] ClearTaskQueue failed: error=%v", err)
		return 0, err
	}
	tasks, err := s.taskRepo.List(ctx, repository.PlatformTaskListFilter{Limit: 1000})
	if err != nil {
		log.Printf("[service] ClearTaskQueue failed: error=%v", err)
		return 0, err
	}
	count, err := s.cancelTasks(ctx, tasks, false, true)
	if err != nil {
		log.Printf("[service] ClearTaskQueue failed: error=%v", err)
		return 0, err
	}
	log.Printf("[service] ClearTaskQueue completed: cancelled_count=%d", count)
	return count, nil
}

func (s *AdminService) CancelTaskBatch(ctx context.Context, batchID int64) (int, error) {
	log.Printf("[service] CancelTaskBatch started: batch_id=%d", batchID)
	if batchID <= 0 {
		err := fmt.Errorf("invalid batch_id")
		log.Printf("[service] CancelTaskBatch failed: error=%v", err)
		return 0, err
	}
	if s.taskRepo == nil || s.taskBatchRepo == nil {
		err := fmt.Errorf("task control repositories not configured")
		log.Printf("[service] CancelTaskBatch failed: error=%v", err)
		return 0, err
	}
	tasks, err := s.taskRepo.List(ctx, repository.PlatformTaskListFilter{BatchID: &batchID, Limit: 1000})
	if err != nil {
		log.Printf("[service] CancelTaskBatch failed: error=%v", err)
		return 0, err
	}
	count, err := s.cancelTasks(ctx, tasks, true, false)
	if err != nil {
		log.Printf("[service] CancelTaskBatch failed: error=%v", err)
		return 0, err
	}
	log.Printf("[service] CancelTaskBatch completed: cancelled_count=%d", count)
	return count, nil
}

func (s *AdminService) CancelTask(ctx context.Context, taskID int64) (int, error) {
	log.Printf("[service] CancelTask started: task_id=%d", taskID)
	if taskID <= 0 {
		err := fmt.Errorf("invalid task_id")
		log.Printf("[service] CancelTask failed: error=%v", err)
		return 0, err
	}
	if s.taskRepo == nil || s.taskBatchRepo == nil {
		err := fmt.Errorf("task control repositories not configured")
		log.Printf("[service] CancelTask failed: error=%v", err)
		return 0, err
	}
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		log.Printf("[service] CancelTask failed: error=%v", err)
		return 0, err
	}
	if task == nil {
		err := fmt.Errorf("task not found")
		log.Printf("[service] CancelTask failed: error=%v", err)
		return 0, err
	}
	count, err := s.cancelTasks(ctx, []domain.PlatformTask{*task}, false, false)
	if err != nil {
		log.Printf("[service] CancelTask failed: error=%v", err)
		return 0, err
	}
	log.Printf("[service] CancelTask completed: cancelled_count=%d", count)
	return count, nil
}

func (s *AdminService) cancelTasks(ctx context.Context, tasks []domain.PlatformTask, includeRunning bool, clearOnly bool) (int, error) {
	if len(tasks) == 0 {
		return 0, nil
	}
	count := 0
	affectedBatches := make(map[int64]struct{})
	now := time.Now()
	for _, task := range tasks {
		if task.Status != domain.PlatformTaskStatusPending && task.Status != domain.PlatformTaskStatusQueued {
			if !includeRunning || task.Status != domain.PlatformTaskStatusRunning {
				continue
			}
		}
		if clearOnly && task.Status == domain.PlatformTaskStatusRunning {
			continue
		}
		task.Status = domain.PlatformTaskStatusCancelled
		task.FinishedAt = &now
		if err := s.taskRepo.Update(ctx, &task); err != nil {
			return count, err
		}
		if task.LatestAsyncJobID != nil {
			jobs, err := s.jobRepo.FindByPlatformTaskID(task.ID)
			if err != nil {
				return count, err
			}
			for i := range jobs {
				if jobs[i].Status == "ready" {
					status := domain.PlatformTaskStatusCancelled
					if err := s.jobRepo.UpdateStatus(jobs[i].ID, status, nil); err != nil {
						return count, err
					}
				}
			}
		}
		affectedBatches[task.BatchID] = struct{}{}
		count++
	}
	for batchID := range affectedBatches {
		if _, err := s.taskBatchRepo.RefreshStatus(ctx, batchID); err != nil {
			return count, err
		}
	}
	return count, nil
}

// GetJobs returns recent jobs for the admin dashboard.
func (s *AdminService) GetJobs(ctx context.Context, limit int) ([]any, error) {
	jobs, err := s.jobRepo.FindRecent(limit)
	if err != nil {
		return nil, err
	}

	result := make([]any, len(jobs))
	for i, job := range jobs {
		result[i] = job
	}
	return result, nil
}

// GetTaskBatches returns admin-facing task batch read models.
func (s *AdminService) GetTaskBatches(ctx context.Context, filter TaskBatchReadFilter) ([]TaskBatchReadModel, error) {
	if s.taskReadSvc == nil {
		return []TaskBatchReadModel{}, nil
	}
	return s.taskReadSvc.ListBatches(ctx, filter)
}

// GetTasks returns admin-facing platform task details.
func (s *AdminService) GetTasks(ctx context.Context, filter TaskReadFilter) ([]TaskReadModel, error) {
	if s.taskReadSvc == nil {
		return []TaskReadModel{}, nil
	}
	return s.taskReadSvc.ListTasks(ctx, filter)
}

// TriggerScan creates a manual scan job.
func (s *AdminService) TriggerScan(ctx context.Context) (int64, error) {
	log.Printf("[service] TriggerScan started")
	// The job manager will handle the scan via registered handler
	jobID, err := s.jobManager.AddJob(ctx, "manual_scan", "{}")
	if err != nil {
		log.Printf("[service] TriggerScan failed: error=%v", err)
		return 0, err
	}
	log.Printf("[service] TriggerScan completed: job_id=%d", jobID)
	return jobID, nil
}

// RetryFailedJobs retries eligible failed tasks across failed and partial_failed batches.
func (s *AdminService) RetryFailedJobs(ctx context.Context) (int, error) {
	log.Printf("[service] RetryFailedJobs started")
	batches, err := s.listRetryableBatches(ctx)
	if err != nil {
		err := fmt.Errorf("task retry service not configured")
		log.Printf("[service] RetryFailedJobs failed: error=%v", err)
		return 0, err
	}

	count := 0
	for _, batch := range batches {
		result, err := s.RetryFailedBatchTasks(ctx, batch.ID)
		if err != nil {
			continue
		}
		count += result.RetryCount
	}
	log.Printf("[service] RetryFailedJobs completed: retried_count=%d", count)
	return count, nil
}

func (s *AdminService) RetryFailedBatchTasks(ctx context.Context, batchID int64) (*RetryBatchResult, error) {
	log.Printf("[service] RetryFailedBatchTasks started: batch_id=%d", batchID)
	if batchID <= 0 {
		err := fmt.Errorf("invalid batch_id")
		log.Printf("[service] RetryFailedBatchTasks failed: error=%v", err)
		return nil, err
	}
	if s.taskBatchRepo == nil || s.taskRepo == nil {
		err := fmt.Errorf("task control repositories not configured")
		log.Printf("[service] RetryFailedBatchTasks failed: error=%v", err)
		return nil, err
	}
	originalBatch, err := s.taskBatchRepo.FindByID(ctx, batchID)
	if err != nil {
		log.Printf("[service] RetryFailedBatchTasks failed: error=%v", err)
		return nil, err
	}
	if originalBatch == nil {
		err := fmt.Errorf("batch not found")
		log.Printf("[service] RetryFailedBatchTasks failed: error=%v", err)
		return nil, err
	}
	tasks, err := s.taskRepo.List(ctx, repository.PlatformTaskListFilter{BatchID: &batchID, Limit: 1000})
	if err != nil {
		log.Printf("[service] RetryFailedBatchTasks failed: error=%v", err)
		return nil, err
	}
	result, err := s.retryFailedPlatformTasks(ctx, originalBatch, tasks)
	if err != nil {
		log.Printf("[service] RetryFailedBatchTasks failed: error=%v", err)
		return nil, err
	}
	log.Printf("[service] RetryFailedBatchTasks completed: retried_count=%d", result.RetryCount)
	return result, nil
}

func (s *AdminService) RetryFailedTask(ctx context.Context, taskID int64) (*RetryBatchResult, error) {
	log.Printf("[service] RetryFailedTask started: task_id=%d", taskID)
	if taskID <= 0 {
		err := fmt.Errorf("invalid task_id")
		log.Printf("[service] RetryFailedTask failed: error=%v", err)
		return nil, err
	}
	if s.taskBatchRepo == nil || s.taskRepo == nil {
		err := fmt.Errorf("task control repositories not configured")
		log.Printf("[service] RetryFailedTask failed: error=%v", err)
		return nil, err
	}
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		log.Printf("[service] RetryFailedTask failed: error=%v", err)
		return nil, err
	}
	if task == nil {
		err := fmt.Errorf("task not found")
		log.Printf("[service] RetryFailedTask failed: error=%v", err)
		return nil, err
	}
	if task.Status != domain.PlatformTaskStatusFailed {
		err := fmt.Errorf("only failed tasks can be retried")
		log.Printf("[service] RetryFailedTask failed: error=%v", err)
		return nil, err
	}
	originalBatch, err := s.taskBatchRepo.FindByID(ctx, task.BatchID)
	if err != nil {
		log.Printf("[service] RetryFailedTask failed: error=%v", err)
		return nil, err
	}
	if originalBatch == nil {
		err := fmt.Errorf("batch not found")
		log.Printf("[service] RetryFailedTask failed: error=%v", err)
		return nil, err
	}
	result, err := s.retryFailedPlatformTasks(ctx, originalBatch, []domain.PlatformTask{*task})
	if err != nil {
		log.Printf("[service] RetryFailedTask failed: error=%v", err)
		return nil, err
	}
	log.Printf("[service] RetryFailedTask completed: retried_count=%d", result.RetryCount)
	return result, nil
}

func (s *AdminService) retryFailedPlatformTasks(ctx context.Context, originalBatch *domain.TaskBatch, tasks []domain.PlatformTask) (*RetryBatchResult, error) {
	failedTasks := make([]domain.PlatformTask, 0, len(tasks))
	for _, task := range tasks {
		if task.Status != domain.PlatformTaskStatusFailed {
			continue
		}
		canRetry, err := s.canRetryFailedTask(ctx, task)
		if err != nil {
			return nil, err
		}
		if canRetry {
			failedTasks = append(failedTasks, task)
		}
	}
	if len(failedTasks) == 0 {
		return nil, fmt.Errorf("no failed tasks available for retry: retry limit reached")
	}
	sort.Slice(failedTasks, func(i, j int) bool { return failedTasks[i].ID < failedTasks[j].ID })

	platformSvc := s.newTaskPlatformService()
	if platformSvc == nil {
		return nil, fmt.Errorf("task platform service not configured")
	}

	newBatch := &domain.TaskBatch{
		SourceType:   domain.TaskBatchSourceRetry,
		TriggerKey:   fmt.Sprintf("retry-batch-%d", originalBatch.ID),
		SummaryLabel: fmt.Sprintf("retry failed tasks for batch #%d", originalBatch.ID),
		Status:       domain.TaskBatchStatusPending,
		TotalImages:  int64(len(failedTasks)),
		CreatedAt:    time.Now(),
	}
	if err := s.taskBatchRepo.Create(ctx, newBatch); err != nil {
		return nil, err
	}
	if sources, err := s.taskBatchRepo.ListSources(ctx, originalBatch.ID); err == nil {
		seen := make(map[string]struct{}, len(sources))
		for _, source := range sources {
			if _, ok := seen[source.SourceRoot]; ok {
				continue
			}
			seen[source.SourceRoot] = struct{}{}
			if err := s.taskBatchRepo.AddSource(ctx, &domain.TaskBatchSource{BatchID: newBatch.ID, SourceRoot: source.SourceRoot, SourceLabel: source.SourceLabel}); err != nil {
				return nil, err
			}
		}
	}

	createdTasks := make([]domain.PlatformTask, 0, len(failedTasks))
	for _, failedTask := range failedTasks {
		payload, err := s.retryTaskPayload(failedTask)
		if err != nil {
			return nil, err
		}
		newTask := domain.PlatformTask{
			BatchID:         newBatch.ID,
			ImageID:         failedTask.ImageID,
			TaskType:        failedTask.TaskType,
			SourceType:      domain.TaskBatchSourceRetry,
			Status:          domain.PlatformTaskStatusPending,
			DedupeKey:       failedTask.DedupeKey,
			ImageVersionKey: failedTask.ImageVersionKey,
			CreatedAt:       time.Now(),
		}
		if err := s.taskRepo.Create(ctx, &newTask); err != nil {
			return nil, err
		}
		job, err := platformSvc.QueueTask(ctx, &newTask, failedTask.TaskType, payload)
		if err != nil {
			return nil, err
		}
		_ = job
		createdTasks = append(createdTasks, newTask)
	}

	refreshedBatch, err := s.taskBatchRepo.RefreshStatus(ctx, newBatch.ID)
	if err != nil {
		return nil, err
	}
	return &RetryBatchResult{Batch: refreshedBatch, CreatedTasks: createdTasks, RetryCount: len(createdTasks)}, nil
}

func (s *AdminService) retryTaskPayload(task domain.PlatformTask) (string, error) {
	if task.LatestAsyncJobID != nil {
		job, err := s.jobRepo.FindByID(*task.LatestAsyncJobID)
		if err == nil && job != nil && job.Payload != "" {
			return job.Payload, nil
		}
	}
	jobs, err := s.jobRepo.FindByPlatformTaskID(task.ID)
	if err != nil {
		return "", err
	}
	if len(jobs) == 0 {
		return "", fmt.Errorf("retry payload not found")
	}
	return jobs[len(jobs)-1].Payload, nil
}

func (s *AdminService) listRetryableBatches(ctx context.Context) ([]domain.TaskBatch, error) {
	if s.taskBatchRepo == nil {
		return nil, fmt.Errorf("task retry service not configured")
	}

	batches := make([]domain.TaskBatch, 0, 200)
	seen := make(map[int64]struct{})
	statuses := []string{domain.TaskBatchStatusFailed, domain.TaskBatchStatusPartialFailed}
	for _, status := range statuses {
		items, err := s.taskBatchRepo.List(ctx, repository.TaskBatchListFilter{Status: status, Limit: 200})
		if err != nil {
			return nil, err
		}
		for _, batch := range items {
			if _, ok := seen[batch.ID]; ok {
				continue
			}
			seen[batch.ID] = struct{}{}
			batches = append(batches, batch)
		}
	}
	return batches, nil
}

func (s *AdminService) canRetryFailedTask(ctx context.Context, task domain.PlatformTask) (bool, error) {
	if s.taskRepo == nil {
		return false, fmt.Errorf("task control repositories not configured")
	}

	history, err := s.taskRepo.ListByImageAndTypes(ctx, task.ImageID, []string{task.TaskType})
	if err != nil {
		return false, err
	}

	failedAttempts := 0
	for _, historicalTask := range history {
		if !sameRetryChain(task, historicalTask) {
			continue
		}
		if historicalTask.ID > task.ID {
			switch historicalTask.Status {
			case domain.PlatformTaskStatusPending,
				domain.PlatformTaskStatusQueued,
				domain.PlatformTaskStatusRunning,
				domain.PlatformTaskStatusCompleted:
				return false, nil
			}
		}
		if historicalTask.Status == domain.PlatformTaskStatusFailed {
			failedAttempts++
		}
	}

	return failedAttempts < maxFailedRetryAttempts, nil
}

func sameRetryChain(base, candidate domain.PlatformTask) bool {
	if base.ImageID != candidate.ImageID || base.TaskType != candidate.TaskType {
		return false
	}
	if base.DedupeKey != "" {
		return candidate.DedupeKey == base.DedupeKey
	}
	if base.ImageVersionKey != "" {
		return candidate.ImageVersionKey == base.ImageVersionKey
	}
	return true
}

func (s *AdminService) newTaskPlatformService() *TaskPlatformService {
	if s.taskBatchRepo == nil || s.taskRepo == nil || s.jobRepo == nil {
		return nil
	}
	return NewTaskPlatformService(s.taskBatchRepo, s.taskRepo, s.jobRepo)
}

// PauseBackgroundTasks pauses the job manager from processing new jobs.
func (s *AdminService) PauseBackgroundTasks(ctx context.Context) error {
	log.Printf("[service] PauseBackgroundTasks started")
	s.jobManager.Pause()
	return nil
}

// ResumeBackgroundTasks allows the job manager to continue processing jobs.
func (s *AdminService) ResumeBackgroundTasks(ctx context.Context) error {
	log.Printf("[service] ResumeBackgroundTasks started")
	s.jobManager.Resume()
	return nil
}

// IsBackgroundRunning returns true if background tasks are running (not paused).
func (s *AdminService) IsBackgroundRunning() bool {
	return !s.jobManager.IsPaused()
}
