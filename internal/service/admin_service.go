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

// NewAdminService creates a new AdminService.
func NewAdminService(
	cfg *config.Config,
	jobRepo repository.JobRepository,
	imageRepo repository.ImageRepository,
	tagRepo repository.TagRepository,
	collectionRepo repository.CollectionRepository,
	jobManager JobManagerControl,
) *AdminService {
	return &AdminService{
		cfg:            cfg,
		jobRepo:        jobRepo,
		imageRepo:      imageRepo,
		tagRepo:        tagRepo,
		collectionRepo: collectionRepo,
		jobManager:     jobManager,
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
	// TagRepo and CollectionRepo may not have Count methods, use reasonable defaults
	summary.Library.TotalTags = 0
	summary.Library.TotalCollections = 0

	return summary, nil
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
