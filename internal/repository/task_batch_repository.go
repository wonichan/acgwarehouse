package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

var ErrTaskPlatformRepositoryNotImplemented = errors.New("task platform repository not implemented")

type TaskBatchListFilter struct {
	SourceType string
	Status     string
	Limit      int
	Offset     int
}

type TaskBatchAggregate struct {
	BatchID      int64
	TaskType     string
	Status       string
	Count        int64
	LatestTaskAt *string
}

type TaskBatchRepository interface {
	Create(ctx context.Context, batch *domain.TaskBatch) error
	AddSource(ctx context.Context, source *domain.TaskBatchSource) error
	FindByID(ctx context.Context, batchID int64) (*domain.TaskBatch, error)
	List(ctx context.Context, filter TaskBatchListFilter) ([]domain.TaskBatch, error)
	ListSources(ctx context.Context, batchID int64) ([]domain.TaskBatchSource, error)
	ListAggregates(ctx context.Context, batchID int64) ([]TaskBatchAggregate, error)
	Update(ctx context.Context, batch *domain.TaskBatch) error
	RefreshStatus(ctx context.Context, batchID int64) (*domain.TaskBatch, error)
}

type sqliteTaskBatchRepository struct {
	db *sql.DB
}

func NewTaskBatchRepository(db *sql.DB) TaskBatchRepository {
	return &sqliteTaskBatchRepository{db: db}
}

func (r *sqliteTaskBatchRepository) Create(ctx context.Context, batch *domain.TaskBatch) error {
	if batch.CreatedAt.IsZero() {
		batch.CreatedAt = time.Now()
	}
	result, err := r.db.ExecContext(ctx, `
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
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	batch.ID = id
	return nil
}

func (r *sqliteTaskBatchRepository) AddSource(ctx context.Context, source *domain.TaskBatchSource) error {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO task_batch_sources (batch_id, source_root, source_label)
		VALUES (?, ?, ?)
	`, source.BatchID, source.SourceRoot, source.SourceLabel)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	source.ID = id
	return nil
}

func (r *sqliteTaskBatchRepository) FindByID(ctx context.Context, batchID int64) (*domain.TaskBatch, error) {
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

func (r *sqliteTaskBatchRepository) List(ctx context.Context, filter TaskBatchListFilter) ([]domain.TaskBatch, error) {
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

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	batches := make([]domain.TaskBatch, 0)
	for rows.Next() {
		batch, err := scanTaskBatch(rows)
		if err != nil {
			return nil, err
		}
		batches = append(batches, *batch)
	}
	return batches, rows.Err()
}

func (r *sqliteTaskBatchRepository) ListSources(ctx context.Context, batchID int64) ([]domain.TaskBatchSource, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, batch_id, source_root, source_label
		FROM task_batch_sources WHERE batch_id = ? ORDER BY id
	`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sources := make([]domain.TaskBatchSource, 0)
	for rows.Next() {
		var source domain.TaskBatchSource
		if err := rows.Scan(&source.ID, &source.BatchID, &source.SourceRoot, &source.SourceLabel); err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, rows.Err()
}

func (r *sqliteTaskBatchRepository) ListAggregates(ctx context.Context, batchID int64) ([]TaskBatchAggregate, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT batch_id, task_type, status, COUNT(*), MAX(COALESCE(finished_at, started_at, queued_at, created_at))
		FROM platform_tasks
		WHERE batch_id = ?
		GROUP BY batch_id, task_type, status
		ORDER BY task_type, status
	`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	aggregates := make([]TaskBatchAggregate, 0)
	for rows.Next() {
		var aggregate TaskBatchAggregate
		var latestAt sql.NullString
		if err := rows.Scan(&aggregate.BatchID, &aggregate.TaskType, &aggregate.Status, &aggregate.Count, &latestAt); err != nil {
			return nil, err
		}
		if latestAt.Valid {
			aggregate.LatestTaskAt = &latestAt.String
		}
		aggregates = append(aggregates, aggregate)
	}
	return aggregates, rows.Err()
}

func (r *sqliteTaskBatchRepository) Update(ctx context.Context, batch *domain.TaskBatch) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE task_batches
		SET source_type = ?, trigger_key = ?, summary_label = ?, status = ?,
			total_images = ?, new_images = ?, skipped_images = ?, skipped_unchanged = ?,
			skipped_duplicate_tasks = ?, latest_error_summary = ?, started_at = ?, finished_at = ?
		WHERE id = ?
	`, batch.SourceType, batch.TriggerKey, batch.SummaryLabel, batch.Status,
		batch.TotalImages, batch.NewImages, batch.SkippedImages, batch.SkippedUnchanged,
		batch.SkippedDuplicateTasks, batch.LatestErrorSummary, batch.StartedAt, batch.FinishedAt, batch.ID)
	return err
}

func (r *sqliteTaskBatchRepository) RefreshStatus(ctx context.Context, batchID int64) (*domain.TaskBatch, error) {
	batch, err := r.FindByID(ctx, batchID)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `SELECT status, error_summary FROM platform_tasks WHERE batch_id = ?`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var total, pending, queued, running, completed, failed, cancelled, skipped int64
	var latestFailure *string
	for rows.Next() {
		var status string
		var errorSummary sql.NullString
		if err := rows.Scan(&status, &errorSummary); err != nil {
			return nil, err
		}
		total++
		switch status {
		case domain.PlatformTaskStatusPending:
			pending++
		case domain.PlatformTaskStatusQueued:
			queued++
		case domain.PlatformTaskStatusRunning:
			running++
		case domain.PlatformTaskStatusCompleted:
			completed++
		case domain.PlatformTaskStatusFailed:
			failed++
			if errorSummary.Valid && latestFailure == nil {
				latestFailure = &errorSummary.String
			}
		case domain.PlatformTaskStatusCancelled:
			cancelled++
		case domain.PlatformTaskStatusSkipped:
			skipped++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
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

func (r *sqliteTaskBatchRepository) queryOne(ctx context.Context, query string, args ...any) (*domain.TaskBatch, error) {
	row := r.db.QueryRowContext(ctx, query, args...)
	return scanTaskBatch(row)
}

type taskBatchScanner interface {
	Scan(dest ...any) error
}

func scanTaskBatch(scanner taskBatchScanner) (*domain.TaskBatch, error) {
	batch := &domain.TaskBatch{}
	var latestError sql.NullString
	var startedAt, finishedAt sql.NullTime
	if err := scanner.Scan(
		&batch.ID,
		&batch.SourceType,
		&batch.TriggerKey,
		&batch.SummaryLabel,
		&batch.Status,
		&batch.TotalImages,
		&batch.NewImages,
		&batch.SkippedImages,
		&batch.SkippedUnchanged,
		&batch.SkippedDuplicateTasks,
		&latestError,
		&batch.CreatedAt,
		&startedAt,
		&finishedAt,
	); err != nil {
		return nil, err
	}
	if latestError.Valid {
		batch.LatestErrorSummary = &latestError.String
	}
	if startedAt.Valid {
		batch.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		batch.FinishedAt = &finishedAt.Time
	}
	return batch, nil
}

func max(v, floor int) int {
	if v < floor {
		return floor
	}
	return v
}
