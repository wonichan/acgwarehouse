package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/d1client"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type d1TaskBatchRepository struct {
	client *d1client.Client
}

func NewD1TaskBatchRepository(client *d1client.Client) TaskBatchRepository {
	return &d1TaskBatchRepository{client: client}
}

func (r *d1TaskBatchRepository) Create(ctx context.Context, batch *domain.TaskBatch) error {
	if batch.CreatedAt.IsZero() {
		batch.CreatedAt = time.Now()
	}
	id, err := r.client.ExecReturningID(ctx, `
		INSERT INTO task_batches (
			source_type, trigger_key, summary_label, status,
			total_images, new_images, skipped_images, skipped_unchanged,
			skipped_duplicate_tasks, latest_error_summary, created_at, started_at, finished_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, batch.SourceType, batch.TriggerKey, batch.SummaryLabel, batch.Status,
		batch.TotalImages, batch.NewImages, batch.SkippedImages, batch.SkippedUnchanged,
		batch.SkippedDuplicateTasks, batch.LatestErrorSummary, batch.CreatedAt, batch.StartedAt, batch.FinishedAt)
	if err != nil {
		return err
	}
	batch.ID = id
	return nil
}

func (r *d1TaskBatchRepository) AddSource(ctx context.Context, source *domain.TaskBatchSource) error {
	id, err := r.client.ExecReturningID(ctx, `
		INSERT INTO task_batch_sources (batch_id, source_root, source_label)
		VALUES (?, ?, ?)
	`, source.BatchID, source.SourceRoot, source.SourceLabel)
	if err != nil {
		return err
	}
	source.ID = id
	return nil
}

func (r *d1TaskBatchRepository) FindByID(ctx context.Context, batchID int64) (*domain.TaskBatch, error) {
	batch, err := r.queryOne(ctx, `
		SELECT id, source_type, trigger_key, summary_label, status,
			total_images, new_images, skipped_images, skipped_unchanged,
			skipped_duplicate_tasks, latest_error_summary, created_at, started_at, finished_at
		FROM task_batches WHERE id = ?
	`, batchID)
	if err != nil {
		return nil, err
	}
	sources, err := r.ListSources(ctx, batchID)
	if err != nil {
		return nil, err
	}
	batch.Sources = sources
	return batch, nil
}

func (r *d1TaskBatchRepository) List(ctx context.Context, filter TaskBatchListFilter) ([]domain.TaskBatch, error) {
	clauses := make([]string, 0, 2)
	args := make([]any, 0, 4)
	if filter.SourceType != "" {
		clauses = append(clauses, "source_type = ?")
		args = append(args, filter.SourceType)
	}
	if filter.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	query := `
		SELECT id, source_type, trigger_key, summary_label, status,
			total_images, new_images, skipped_images, skipped_unchanged,
			skipped_duplicate_tasks, latest_error_summary, created_at, started_at, finished_at
		FROM task_batches`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	query += " ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, max(filter.Offset, 0))

	rows, err := r.client.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return mapTaskBatchesFromD1(rows)
}

func (r *d1TaskBatchRepository) ListSources(ctx context.Context, batchID int64) ([]domain.TaskBatchSource, error) {
	rows, err := r.client.Query(ctx, `
		SELECT id, batch_id, source_root, source_label
		FROM task_batch_sources WHERE batch_id = ? ORDER BY id
	`, batchID)
	if err != nil {
		return nil, err
	}
	sources := make([]domain.TaskBatchSource, 0, len(rows))
	for _, row := range rows {
		id, err := toInt64(row["id"])
		if err != nil {
			return nil, err
		}
		sourceBatchID, err := toInt64(row["batch_id"])
		if err != nil {
			return nil, err
		}
		sources = append(sources, domain.TaskBatchSource{
			ID:          id,
			BatchID:     sourceBatchID,
			SourceRoot:  toStringDefault(row["source_root"], ""),
			SourceLabel: toStringDefault(row["source_label"], ""),
		})
	}
	return sources, nil
}

func (r *d1TaskBatchRepository) ListAggregates(ctx context.Context, batchID int64) ([]TaskBatchAggregate, error) {
	rows, err := r.client.Query(ctx, `
		SELECT batch_id, task_type, status, COUNT(*) AS count, MAX(COALESCE(finished_at, started_at, queued_at, created_at)) AS latest_task_at
		FROM platform_tasks
		WHERE batch_id = ?
		GROUP BY batch_id, task_type, status
		ORDER BY task_type, status
	`, batchID)
	if err != nil {
		return nil, err
	}
	aggregates := make([]TaskBatchAggregate, 0, len(rows))
	for _, row := range rows {
		rowBatchID, err := toInt64(row["batch_id"])
		if err != nil {
			return nil, err
		}
		count, err := toInt64(row["count"])
		if err != nil {
			return nil, err
		}
		aggregate := TaskBatchAggregate{
			BatchID:  rowBatchID,
			TaskType: toStringDefault(row["task_type"], ""),
			Status:   toStringDefault(row["status"], ""),
			Count:    count,
		}
		if latest := toStringDefault(row["latest_task_at"], ""); latest != "" {
			aggregate.LatestTaskAt = &latest
		}
		aggregates = append(aggregates, aggregate)
	}
	return aggregates, nil
}

func (r *d1TaskBatchRepository) Update(ctx context.Context, batch *domain.TaskBatch) error {
	return r.client.Exec(ctx, `
		UPDATE task_batches
		SET source_type = ?, trigger_key = ?, summary_label = ?, status = ?,
			total_images = ?, new_images = ?, skipped_images = ?, skipped_unchanged = ?,
			skipped_duplicate_tasks = ?, latest_error_summary = ?, started_at = ?, finished_at = ?
		WHERE id = ?
	`, batch.SourceType, batch.TriggerKey, batch.SummaryLabel, batch.Status,
		batch.TotalImages, batch.NewImages, batch.SkippedImages, batch.SkippedUnchanged,
		batch.SkippedDuplicateTasks, batch.LatestErrorSummary, batch.StartedAt, batch.FinishedAt, batch.ID)
}

func (r *d1TaskBatchRepository) RefreshStatus(ctx context.Context, batchID int64) (*domain.TaskBatch, error) {
	batch, err := r.FindByID(ctx, batchID)
	if err != nil {
		return nil, err
	}

	rows, err := r.client.Query(ctx, `SELECT status, error_summary FROM platform_tasks WHERE batch_id = ?`, batchID)
	if err != nil {
		return nil, err
	}

	var total, pending, queued, running, failed, cancelled, skipped int64
	var latestFailure *string
	for _, row := range rows {
		total++
		status := toStringDefault(row["status"], "")
		switch status {
		case domain.PlatformTaskStatusPending:
			pending++
		case domain.PlatformTaskStatusQueued:
			queued++
		case domain.PlatformTaskStatusRunning:
			running++
		case domain.PlatformTaskStatusFailed:
			failed++
			if latestFailure == nil {
				if value := toStringDefault(row["error_summary"], ""); value != "" {
					latestFailure = &value
				}
			}
		case domain.PlatformTaskStatusCancelled:
			cancelled++
		case domain.PlatformTaskStatusSkipped:
			skipped++
		}
	}

	now := time.Now()
	switch {
	case total == 0:
		batch.Status = domain.TaskBatchStatusCompleted
		batch.FinishedAt = &now
	case pending > 0 || queued > 0 || running > 0:
		batch.Status = domain.TaskBatchStatusRunning
		if batch.StartedAt == nil {
			batch.StartedAt = &now
		}
		batch.FinishedAt = nil
	case failed == total:
		batch.Status = domain.TaskBatchStatusFailed
		batch.FinishedAt = &now
	case cancelled == total:
		batch.Status = domain.TaskBatchStatusCancelled
		batch.FinishedAt = &now
	case skipped == total:
		batch.Status = domain.TaskBatchStatusCompleted
		batch.FinishedAt = &now
	case failed > 0:
		batch.Status = domain.TaskBatchStatusPartialFailed
		batch.FinishedAt = &now
	default:
		batch.Status = domain.TaskBatchStatusCompleted
		batch.FinishedAt = &now
	}
	batch.LatestErrorSummary = latestFailure
	if err := r.Update(ctx, batch); err != nil {
		return nil, err
	}
	return batch, nil
}

func (r *d1TaskBatchRepository) queryOne(ctx context.Context, query string, args ...any) (*domain.TaskBatch, error) {
	row, err := r.client.QueryOne(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, sql.ErrNoRows
	}
	return mapTaskBatchFromD1(row)
}

func mapTaskBatchesFromD1(rows []map[string]any) ([]domain.TaskBatch, error) {
	batches := make([]domain.TaskBatch, 0, len(rows))
	for _, row := range rows {
		batch, err := mapTaskBatchFromD1(row)
		if err != nil {
			return nil, err
		}
		batches = append(batches, *batch)
	}
	return batches, nil
}

func mapTaskBatchFromD1(row map[string]any) (*domain.TaskBatch, error) {
	id, err := toInt64(row["id"])
	if err != nil {
		return nil, err
	}
	createdAt, _ := toTime(row["created_at"])
	var latestError *string
	if value := toStringDefault(row["latest_error_summary"], ""); value != "" {
		latestError = &value
	}
	var startedAt, finishedAt *time.Time
	if t, err := toTime(row["started_at"]); err == nil && !t.IsZero() {
		startedAt = &t
	}
	if t, err := toTime(row["finished_at"]); err == nil && !t.IsZero() {
		finishedAt = &t
	}
	totalImages, _ := toInt64(row["total_images"])
	newImages, _ := toInt64(row["new_images"])
	skippedImages, _ := toInt64(row["skipped_images"])
	skippedUnchanged, _ := toInt64(row["skipped_unchanged"])
	skippedDuplicateTasks, _ := toInt64(row["skipped_duplicate_tasks"])
	return &domain.TaskBatch{
		ID:                    id,
		SourceType:            toStringDefault(row["source_type"], ""),
		TriggerKey:            toStringDefault(row["trigger_key"], ""),
		SummaryLabel:          toStringDefault(row["summary_label"], ""),
		Status:                toStringDefault(row["status"], ""),
		TotalImages:           totalImages,
		NewImages:             newImages,
		SkippedImages:         skippedImages,
		SkippedUnchanged:      skippedUnchanged,
		SkippedDuplicateTasks: skippedDuplicateTasks,
		LatestErrorSummary:    latestError,
		CreatedAt:             createdAt,
		StartedAt:             startedAt,
		FinishedAt:            finishedAt,
	}, nil
}
