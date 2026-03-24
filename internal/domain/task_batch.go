package domain

import "time"

const (
	TaskBatchSourceImportScan   = "import_scan"
	TaskBatchSourceManualSingle = "manual_single"
	TaskBatchSourceManualBatch  = "manual_batch"
)

const (
	TaskBatchStatusPending       = "pending"
	TaskBatchStatusRunning       = "running"
	TaskBatchStatusCompleted     = "completed"
	TaskBatchStatusPartialFailed = "partial_failed"
	TaskBatchStatusFailed        = "failed"
	TaskBatchStatusCancelled     = "cancelled"
)

type TaskBatch struct {
	ID                    int64             `json:"id" db:"id"`
	SourceType            string            `json:"source_type" db:"source_type"`
	TriggerKey            string            `json:"trigger_key" db:"trigger_key"`
	SummaryLabel          string            `json:"summary_label" db:"summary_label"`
	Status                string            `json:"status" db:"status"`
	TotalImages           int64             `json:"total_images" db:"total_images"`
	NewImages             int64             `json:"new_images" db:"new_images"`
	SkippedImages         int64             `json:"skipped_images" db:"skipped_images"`
	SkippedUnchanged      int64             `json:"skipped_unchanged" db:"skipped_unchanged"`
	SkippedDuplicateTasks int64             `json:"skipped_duplicate_tasks" db:"skipped_duplicate_tasks"`
	LatestErrorSummary    *string           `json:"latest_error_summary" db:"latest_error_summary"`
	CreatedAt             time.Time         `json:"created_at" db:"created_at"`
	StartedAt             *time.Time        `json:"started_at" db:"started_at"`
	FinishedAt            *time.Time        `json:"finished_at" db:"finished_at"`
	Sources               []TaskBatchSource `json:"sources,omitempty" db:"-"`
}

type TaskBatchSource struct {
	ID          int64  `json:"id" db:"id"`
	BatchID     int64  `json:"batch_id" db:"batch_id"`
	SourceRoot  string `json:"source_root" db:"source_root"`
	SourceLabel string `json:"source_label" db:"source_label"`
}

func (TaskBatch) TableName() string {
	return "task_batches"
}
