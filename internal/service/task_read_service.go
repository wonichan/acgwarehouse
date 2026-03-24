package service

import (
	"context"

	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

type TaskBatchReadFilter = repository.TaskBatchReadFilter
type TaskReadFilter = repository.TaskReadFilter

type TaskBatchSkipSummary struct {
	Total          int64 `json:"total"`
	Unchanged      int64 `json:"unchanged"`
	DuplicateTasks int64 `json:"duplicate_tasks"`
}

type TaskBatchReadModel struct {
	ID             int64                `json:"id"`
	SourceType     string               `json:"source_type"`
	SummaryLabel   string               `json:"summary_label"`
	Status         string               `json:"status"`
	TotalImages    int64                `json:"total_images"`
	NewImages      int64                `json:"new_images"`
	SourceSummary  string               `json:"source_summary"`
	SkipSummary    TaskBatchSkipSummary `json:"skip_summary"`
	FailureSummary string               `json:"failure_summary,omitempty"`
	StatusCounts   map[string]int64     `json:"status_counts"`
	TaskTypeCounts map[string]int64     `json:"task_type_counts"`
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
		result = append(result, TaskBatchReadModel{
			ID:            record.ID,
			SourceType:    record.SourceType,
			SummaryLabel:  record.SummaryLabel,
			Status:        record.Status,
			TotalImages:   record.TotalImages,
			NewImages:     record.NewImages,
			SourceSummary: record.SourceSummary,
			SkipSummary: TaskBatchSkipSummary{
				Total:          record.SkippedImages,
				Unchanged:      record.SkippedUnchanged,
				DuplicateTasks: record.SkippedDuplicateTasks,
			},
			FailureSummary: record.LatestErrorSummary,
			StatusCounts:   record.StatusCounts,
			TaskTypeCounts: record.TaskTypeCounts,
		})
	}
	return result, nil
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
