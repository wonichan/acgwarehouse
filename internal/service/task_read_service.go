package service

import (
	"context"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

type TaskBatchReadFilter = repository.TaskBatchReadFilter
type TaskReadFilter = repository.TaskReadFilter

type TaskBatchSkipSummary struct {
	Total          int64 `json:"total"`
	Unchanged      int64 `json:"unchanged"`
	DuplicateTasks int64 `json:"duplicate_tasks"`
}

// TaskBatchFailureGroup represents a grouped failure reason within a batch.
// Each group aggregates tasks that failed for the same underlying reason,
// with a retry recommendation so operators can judge whether retry is appropriate.
type TaskBatchFailureGroup struct {
	ReasonKey        string `json:"reason_key"`
	ReasonLabel      string `json:"reason_label"`
	Count            int64  `json:"count"`
	RetryRecommended bool   `json:"retry_recommended"`
	RetryHint        string `json:"retry_hint"`
}

type TaskBatchReadModel struct {
	ID             int64                   `json:"id"`
	SourceType     string                  `json:"source_type"`
	SummaryLabel   string                  `json:"summary_label"`
	Status         string                  `json:"status"`
	TotalImages    int64                   `json:"total_images"`
	NewImages      int64                   `json:"new_images"`
	CreatedAt      time.Time               `json:"created_at"`
	FinishedAt     *time.Time              `json:"finished_at,omitempty"`
	SourceSummary  string                  `json:"source_summary"`
	SkipSummary    TaskBatchSkipSummary    `json:"skip_summary"`
	FailureSummary string                  `json:"failure_summary,omitempty"`
	FailureGroups  []TaskBatchFailureGroup `json:"failure_groups,omitempty"`
	StatusCounts   map[string]int64        `json:"status_counts"`
	TaskTypeCounts map[string]int64        `json:"task_type_counts"`
}

type TaskReadModel struct {
	ID               int64  `json:"id"`
	BatchID          int64  `json:"batch_id"`
	ImageID          int64  `json:"image_id"`
	ImagePath        string `json:"image_path"`
	ImageFilename    string `json:"image_filename"`
	TaskType         string `json:"task_type"`
	Status           string `json:"status"`
	SkipReason       string `json:"skip_reason,omitempty"`
	ErrorSummary     string `json:"error_summary,omitempty"`
	LatestAsyncJobID *int64 `json:"latest_async_job_id,omitempty"`
}

type TaskReadService struct {
	repo repository.TaskBatchReadRepository
}

func NewTaskReadService(repo repository.TaskBatchReadRepository) *TaskReadService {
	return &TaskReadService{repo: repo}
}

func (s *TaskReadService) ListBatches(ctx context.Context, filter repository.TaskBatchReadFilter) ([]TaskBatchReadModel, error) {
	records, err := s.repo.ListBatches(ctx, filter)
	if err != nil {
		return nil, err
	}
	result := make([]TaskBatchReadModel, 0, len(records))
	for _, record := range records {
		model := TaskBatchReadModel{
			ID:            record.ID,
			SourceType:    record.SourceType,
			SummaryLabel:  record.SummaryLabel,
			Status:        record.Status,
			TotalImages:   record.TotalImages,
			NewImages:     record.NewImages,
			CreatedAt:     record.CreatedAt,
			FinishedAt:    record.FinishedAt,
			SourceSummary: record.SourceSummary,
			SkipSummary: TaskBatchSkipSummary{
				Total:          record.SkippedImages,
				Unchanged:      record.SkippedUnchanged,
				DuplicateTasks: record.SkippedDuplicateTasks,
			},
			FailureSummary: record.LatestErrorSummary,
			StatusCounts:   record.StatusCounts,
			TaskTypeCounts: record.TaskTypeCounts,
		}
		// Build grouped failure summary from failed task error_summaries
		if record.StatusCounts != nil {
			if failedCount, hasFailures := record.StatusCounts["failed"]; hasFailures && failedCount > 0 {
				repoGroups, err := s.repo.LoadFailureGroups(ctx, record.ID)
				if err != nil {
					return nil, err
				}
				model.FailureGroups = classifyFailureGroups(repoGroups)
			}
		}
		result = append(result, model)
	}
	return result, nil
}

// retryableReasons defines error prefixes that indicate transient/retryable failures.
var retryableReasons = map[string]bool{
	"timeout":    true,
	"rate_limit": true,
	"network":    true,
	"connection": true,
}

// reasonLabels maps reason keys to human-readable labels.
var reasonLabels = map[string]string{
	"timeout":    "超时",
	"rate_limit": "速率限制",
	"network":    "网络错误",
	"connection": "连接失败",
	"auth":       "认证失败",
	"config":     "配置错误",
	"malformed":  "数据格式错误",
	"missing":    "文件缺失",
	"unknown":    "未知错误",
}

// classifyFailureGroups converts repository failure group records into service-level
// failure groups with retry recommendations and human-readable hints.
func classifyFailureGroups(repoGroups []repository.FailureGroupRecord) []TaskBatchFailureGroup {
	groups := make([]TaskBatchFailureGroup, 0, len(repoGroups))
	for _, rg := range repoGroups {
		key := rg.ReasonKey
		_, retryable := retryableReasons[key]

		label, ok := reasonLabels[key]
		if !ok {
			label = key
		}

		hint := "不建议重试，请检查配置或数据"
		if retryable {
			hint = "可安全重试，通常为临时性问题"
		}

		groups = append(groups, TaskBatchFailureGroup{
			ReasonKey:        key,
			ReasonLabel:      label,
			Count:            rg.Count,
			RetryRecommended: retryable,
			RetryHint:        hint,
		})
	}
	return groups
}

// ClassifyFailureGroups exports the classification logic for testing.
func ClassifyFailureGroups(repoGroups []repository.FailureGroupRecord) []TaskBatchFailureGroup {
	return classifyFailureGroups(repoGroups)
}

func (s *TaskReadService) ListTasks(ctx context.Context, filter repository.TaskReadFilter) ([]TaskReadModel, error) {
	records, err := s.repo.ListTasks(ctx, filter)
	if err != nil {
		return nil, err
	}
	result := make([]TaskReadModel, 0, len(records))
	for _, record := range records {
		result = append(result, TaskReadModel{
			ID:               record.ID,
			BatchID:          record.BatchID,
			ImageID:          record.ImageID,
			ImagePath:        record.ImagePath,
			ImageFilename:    record.ImageFilename,
			TaskType:         record.TaskType,
			Status:           record.Status,
			SkipReason:       record.SkipReason,
			ErrorSummary:     record.ErrorSummary,
			LatestAsyncJobID: record.LatestAsyncJobID,
		})
	}
	return result, nil
}
