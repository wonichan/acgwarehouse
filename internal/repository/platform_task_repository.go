package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type PlatformTaskListFilter struct {
	BatchID  *int64
	ImageID  *int64
	TaskType string
	Status   string
	Limit    int
	Offset   int
}

type PlatformTaskDedupeLookup struct {
	TaskType        string
	ImageVersionKey string
	DedupeKey       string
	Statuses        []string
}

type PlatformTaskRepository interface {
	Create(ctx context.Context, task *domain.PlatformTask) error
	FindByID(ctx context.Context, taskID int64) (*domain.PlatformTask, error)
	List(ctx context.Context, filter PlatformTaskListFilter) ([]domain.PlatformTask, error)
	FindByDedupeKey(ctx context.Context, dedupeKey string) (*domain.PlatformTask, error)
	FindActiveByDedupeKey(ctx context.Context, dedupeKey string) (*domain.PlatformTask, error)
	ListByImageAndTypes(ctx context.Context, imageID int64, taskTypes []string) ([]domain.PlatformTask, error)
	Update(ctx context.Context, task *domain.PlatformTask) error
	SetLatestAsyncJob(ctx context.Context, taskID int64, asyncJobID *int64) error
}

type sqlitePlatformTaskRepository struct {
	db *sql.DB
}

func NewPlatformTaskRepository(db *sql.DB) PlatformTaskRepository {
	return &sqlitePlatformTaskRepository{db: db}
}

func (r *sqlitePlatformTaskRepository) Create(ctx context.Context, task *domain.PlatformTask) error {
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO platform_tasks (
			batch_id, image_id, task_type, source_type, status,
			dedupe_key, image_version_key, latest_async_job_id,
			skip_reason, error_summary, created_at, queued_at, started_at, finished_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, task.BatchID, task.ImageID, task.TaskType, task.SourceType, task.Status,
		task.DedupeKey, task.ImageVersionKey, task.LatestAsyncJobID,
		task.SkipReason, task.ErrorSummary, task.CreatedAt, task.QueuedAt, task.StartedAt, task.FinishedAt)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	task.ID = id
	return nil
}

func (r *sqlitePlatformTaskRepository) FindByID(ctx context.Context, taskID int64) (*domain.PlatformTask, error) {
	return r.queryOne(ctx, `
		SELECT id, batch_id, image_id, task_type, source_type, status,
			dedupe_key, image_version_key, latest_async_job_id, skip_reason,
			error_summary, created_at, queued_at, started_at, finished_at
		FROM platform_tasks WHERE id = ?
	`, taskID)
}

func (r *sqlitePlatformTaskRepository) List(ctx context.Context, filter PlatformTaskListFilter) ([]domain.PlatformTask, error) {
	clauses := make([]string, 0, 4)
	args := make([]any, 0, 6)
	if filter.BatchID != nil {
		clauses = append(clauses, "batch_id = ?")
		args = append(args, *filter.BatchID)
	}
	if filter.ImageID != nil {
		clauses = append(clauses, "image_id = ?")
		args = append(args, *filter.ImageID)
	}
	if filter.TaskType != "" {
		clauses = append(clauses, "task_type = ?")
		args = append(args, filter.TaskType)
	}
	if filter.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	query := `
		SELECT id, batch_id, image_id, task_type, source_type, status,
			dedupe_key, image_version_key, latest_async_job_id, skip_reason,
			error_summary, created_at, queued_at, started_at, finished_at
		FROM platform_tasks`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	query += " ORDER BY id LIMIT ? OFFSET ?"
	args = append(args, limit, max(filter.Offset, 0))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]domain.PlatformTask, 0)
	for rows.Next() {
		task, err := scanPlatformTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *task)
	}
	return tasks, rows.Err()
}

func (r *sqlitePlatformTaskRepository) FindByDedupeKey(ctx context.Context, dedupeKey string) (*domain.PlatformTask, error) {
	return r.queryOne(ctx, `
		SELECT id, batch_id, image_id, task_type, source_type, status,
			dedupe_key, image_version_key, latest_async_job_id, skip_reason,
			error_summary, created_at, queued_at, started_at, finished_at
		FROM platform_tasks WHERE dedupe_key = ? ORDER BY id DESC LIMIT 1
	`, dedupeKey)
}

func (r *sqlitePlatformTaskRepository) ListByImageAndTypes(ctx context.Context, imageID int64, taskTypes []string) ([]domain.PlatformTask, error) {
	if len(taskTypes) == 0 {
		return []domain.PlatformTask{}, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(taskTypes)), ",")
	args := []any{imageID}
	for _, taskType := range taskTypes {
		args = append(args, taskType)
	}
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, batch_id, image_id, task_type, source_type, status,
			dedupe_key, image_version_key, latest_async_job_id, skip_reason,
			error_summary, created_at, queued_at, started_at, finished_at
		FROM platform_tasks WHERE image_id = ? AND task_type IN (%s)
		ORDER BY id
	`, placeholders), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]domain.PlatformTask, 0)
	for rows.Next() {
		task, err := scanPlatformTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *task)
	}
	return tasks, rows.Err()
}

func (r *sqlitePlatformTaskRepository) FindActiveByDedupeKey(ctx context.Context, dedupeKey string) (*domain.PlatformTask, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, batch_id, image_id, task_type, source_type, status,
			dedupe_key, image_version_key, latest_async_job_id, skip_reason,
			error_summary, created_at, queued_at, started_at, finished_at
		FROM platform_tasks
		WHERE dedupe_key = ? AND status IN (?, ?, ?, ?)
		ORDER BY CASE status
			WHEN ? THEN 0
			WHEN ? THEN 1
			WHEN ? THEN 2
			WHEN ? THEN 3
			ELSE 4 END, id DESC
		LIMIT 1
	`, dedupeKey,
		domain.PlatformTaskStatusRunning,
		domain.PlatformTaskStatusQueued,
		domain.PlatformTaskStatusPending,
		domain.PlatformTaskStatusCompleted,
		domain.PlatformTaskStatusRunning,
		domain.PlatformTaskStatusQueued,
		domain.PlatformTaskStatusPending,
		domain.PlatformTaskStatusCompleted,
	)
	task, err := scanPlatformTask(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return task, nil
}

func (r *sqlitePlatformTaskRepository) Update(ctx context.Context, task *domain.PlatformTask) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE platform_tasks
		SET batch_id = ?, image_id = ?, task_type = ?, source_type = ?, status = ?,
			dedupe_key = ?, image_version_key = ?, latest_async_job_id = ?,
			skip_reason = ?, error_summary = ?, queued_at = ?, started_at = ?, finished_at = ?
		WHERE id = ?
	`, task.BatchID, task.ImageID, task.TaskType, task.SourceType, task.Status,
		task.DedupeKey, task.ImageVersionKey, task.LatestAsyncJobID,
		task.SkipReason, task.ErrorSummary, task.QueuedAt, task.StartedAt, task.FinishedAt, task.ID)
	return err
}

func (r *sqlitePlatformTaskRepository) SetLatestAsyncJob(ctx context.Context, taskID int64, asyncJobID *int64) error {
	queuedAt := sql.NullTime{}
	if asyncJobID != nil {
		queuedAt = sql.NullTime{Time: time.Now(), Valid: true}
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE platform_tasks
		SET latest_async_job_id = ?,
			queued_at = CASE WHEN ? IS NULL THEN queued_at ELSE ? END,
			status = CASE WHEN ? IS NULL OR status <> ? THEN status ELSE ? END
		WHERE id = ?
	`, asyncJobID, asyncJobID, queuedAt, asyncJobID, domain.PlatformTaskStatusPending, domain.PlatformTaskStatusQueued, taskID)
	return err
}

func (r *sqlitePlatformTaskRepository) queryOne(ctx context.Context, query string, args ...any) (*domain.PlatformTask, error) {
	row := r.db.QueryRowContext(ctx, query, args...)
	return scanPlatformTask(row)
}

type platformTaskScanner interface {
	Scan(dest ...any) error
}

func scanPlatformTask(scanner platformTaskScanner) (*domain.PlatformTask, error) {
	task := &domain.PlatformTask{}
	var latestAsyncJobID sql.NullInt64
	var skipReason, errorSummary sql.NullString
	var queuedAt, startedAt, finishedAt sql.NullTime
	if err := scanner.Scan(
		&task.ID,
		&task.BatchID,
		&task.ImageID,
		&task.TaskType,
		&task.SourceType,
		&task.Status,
		&task.DedupeKey,
		&task.ImageVersionKey,
		&latestAsyncJobID,
		&skipReason,
		&errorSummary,
		&task.CreatedAt,
		&queuedAt,
		&startedAt,
		&finishedAt,
	); err != nil {
		return nil, err
	}
	if latestAsyncJobID.Valid {
		task.LatestAsyncJobID = &latestAsyncJobID.Int64
	}
	if skipReason.Valid {
		task.SkipReason = &skipReason.String
	}
	if errorSummary.Valid {
		task.ErrorSummary = &errorSummary.String
	}
	if queuedAt.Valid {
		task.QueuedAt = &queuedAt.Time
	}
	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		task.FinishedAt = &finishedAt.Time
	}
	return task, nil
}
