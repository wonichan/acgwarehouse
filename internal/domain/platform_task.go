package domain

import "time"

const (
	PlatformTaskStatusPending   = "pending"
	PlatformTaskStatusQueued    = "queued"
	PlatformTaskStatusRunning   = "running"
	PlatformTaskStatusCompleted = "completed"
	PlatformTaskStatusFailed    = "failed"
	PlatformTaskStatusCancelled = "cancelled"
	PlatformTaskStatusSkipped   = "skipped"
)

const (
	PlatformTaskTypeImageImported     = "image_imported"
	PlatformTaskTypeThumbnailGenerate = "thumbnail_generate"
	PlatformTaskTypeAITagGeneration   = "ai_tag_generation"
)

const (
	PlatformTaskSkipReasonUnchanged           = "unchanged"
	PlatformTaskSkipReasonAlreadyCompleted    = "already_completed"
	PlatformTaskSkipReasonAlreadyRunning      = "already_running"
	PlatformTaskSkipReasonMissingPrerequisite = "missing_prerequisite"
)

type PlatformTask struct {
	ID               int64      `json:"id" db:"id"`
	BatchID          int64      `json:"batch_id" db:"batch_id"`
	ImageID          int64      `json:"image_id" db:"image_id"`
	TaskType         string     `json:"task_type" db:"task_type"`
	SourceType       string     `json:"source_type" db:"source_type"`
	Status           string     `json:"status" db:"status"`
	DedupeKey        string     `json:"dedupe_key" db:"dedupe_key"`
	ImageVersionKey  string     `json:"image_version_key" db:"image_version_key"`
	LatestAsyncJobID *int64     `json:"latest_async_job_id" db:"latest_async_job_id"`
	SkipReason       *string    `json:"skip_reason" db:"skip_reason"`
	ErrorSummary     *string    `json:"error_summary" db:"error_summary"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	QueuedAt         *time.Time `json:"queued_at" db:"queued_at"`
	StartedAt        *time.Time `json:"started_at" db:"started_at"`
	FinishedAt       *time.Time `json:"finished_at" db:"finished_at"`
}

func (PlatformTask) TableName() string {
	return "platform_tasks"
}
