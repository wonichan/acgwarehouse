package service

import (
	"context"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/config"
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

// NewAdminService creates a new AdminService.
func NewAdminService(
	cfg *config.Config,
	jobRepo repository.JobRepository,
	imageRepo repository.ImageRepository,
	tagRepo repository.TagRepository,
	collectionRepo repository.CollectionRepository,
	jobManager JobManagerControl,
	taskReadSvcOpt ...*TaskReadService,
) *AdminService {
	var taskReadSvc *TaskReadService
	if len(taskReadSvcOpt) > 0 {
		taskReadSvc = taskReadSvcOpt[0]
	}
	return &AdminService{
		cfg:            cfg,
		jobRepo:        jobRepo,
		imageRepo:      imageRepo,
		tagRepo:        tagRepo,
		collectionRepo: collectionRepo,
		jobManager:     jobManager,
		taskReadSvc:    taskReadSvc,
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
		summary.Tasks.Ready = 0
	}
	summary.Tasks.Running, err = s.jobRepo.CountByStatus("running")
	if err != nil {
		summary.Tasks.Running = 0
	}
	summary.Tasks.Finished, err = s.jobRepo.CountByStatus("finished")
	if err != nil {
		summary.Tasks.Finished = 0
	}
	summary.Tasks.Failed, err = s.jobRepo.CountByStatus("failed")
	if err != nil {
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

// GetJobs returns recent jobs for the admin dashboard.
func (s *AdminService) GetJobs(ctx context.Context, limit int) ([]interface{}, error) {
	jobs, err := s.jobRepo.FindRecent(limit)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(jobs))
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
	// The job manager will handle the scan via registered handler
	return s.jobManager.AddJob(ctx, "manual_scan", "{}")
}

// RetryFailedJobs resets all failed jobs to ready status for reprocessing.
// Returns the number of jobs retried.
func (s *AdminService) RetryFailedJobs(ctx context.Context) (int, error) {
	failedJobs, err := s.jobRepo.FindFailed()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, job := range failedJobs {
		// Reset status to ready and clear error
		if err := s.jobRepo.UpdateStatus(job.ID, "ready", nil); err != nil {
			continue
		}
		count++
	}

	return count, nil
}

// PauseBackgroundTasks pauses the job manager from processing new jobs.
func (s *AdminService) PauseBackgroundTasks(ctx context.Context) error {
	s.jobManager.Pause()
	return nil
}

// ResumeBackgroundTasks allows the job manager to continue processing jobs.
func (s *AdminService) ResumeBackgroundTasks(ctx context.Context) error {
	s.jobManager.Resume()
	return nil
}

// IsBackgroundRunning returns true if background tasks are running (not paused).
func (s *AdminService) IsBackgroundRunning() bool {
	return !s.jobManager.IsPaused()
}
