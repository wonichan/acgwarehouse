package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/d1client"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type d1PlatformTaskRepository struct {
	client *d1client.Client
}

func NewD1PlatformTaskRepository(client *d1client.Client) PlatformTaskRepository {
	return &d1PlatformTaskRepository{client: client}
}

func (r *d1PlatformTaskRepository) Create(ctx context.Context, task *domain.PlatformTask) error {
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	id, err := r.client.ExecReturningID(ctx, `
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
	task.ID = id
	return nil
}

func (r *d1PlatformTaskRepository) ImageExists(ctx context.Context, imageID int64) (bool, error) {
	count, err := r.client.QueryCount(ctx, `SELECT COUNT(*) AS cnt FROM images WHERE id = ?`, imageID)
	return count > 0, err
}

func (r *d1PlatformTaskRepository) FindByID(ctx context.Context, taskID int64) (*domain.PlatformTask, error) {
	return r.queryOne(ctx, `
		SELECT id, batch_id, image_id, task_type, source_type, status,
			dedupe_key, image_version_key, latest_async_job_id, skip_reason,
			error_summary, created_at, queued_at, started_at, finished_at
		FROM platform_tasks WHERE id = ?
	`, taskID)
}

func (r *d1PlatformTaskRepository) List(ctx context.Context, filter PlatformTaskListFilter) ([]domain.PlatformTask, error) {
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

	rows, err := r.client.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return mapPlatformTasksFromD1(rows)
}

func (r *d1PlatformTaskRepository) FindByDedupeKey(ctx context.Context, dedupeKey string) (*domain.PlatformTask, error) {
	return r.queryOne(ctx, `
		SELECT id, batch_id, image_id, task_type, source_type, status,
			dedupe_key, image_version_key, latest_async_job_id, skip_reason,
			error_summary, created_at, queued_at, started_at, finished_at
		FROM platform_tasks WHERE dedupe_key = ? ORDER BY id DESC LIMIT 1
	`, dedupeKey)
}

func (r *d1PlatformTaskRepository) FindActiveByDedupeKey(ctx context.Context, dedupeKey string) (*domain.PlatformTask, error) {
	return r.queryOne(ctx, `
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
}

func (r *d1PlatformTaskRepository) ListByImageAndTypes(ctx context.Context, imageID int64, taskTypes []string) ([]domain.PlatformTask, error) {
	if len(taskTypes) == 0 {
		return []domain.PlatformTask{}, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(taskTypes)), ",")
	args := []any{imageID}
	for _, taskType := range taskTypes {
		args = append(args, taskType)
	}
	rows, err := r.client.Query(ctx, fmt.Sprintf(`
		SELECT id, batch_id, image_id, task_type, source_type, status,
			dedupe_key, image_version_key, latest_async_job_id, skip_reason,
			error_summary, created_at, queued_at, started_at, finished_at
		FROM platform_tasks WHERE image_id = ? AND task_type IN (%s)
		ORDER BY id
	`, placeholders), args...)
	if err != nil {
		return nil, err
	}
	return mapPlatformTasksFromD1(rows)
}

func (r *d1PlatformTaskRepository) Update(ctx context.Context, task *domain.PlatformTask) error {
	return r.client.Exec(ctx, `
		UPDATE platform_tasks
		SET batch_id = ?, image_id = ?, task_type = ?, source_type = ?, status = ?,
			dedupe_key = ?, image_version_key = ?, latest_async_job_id = ?,
			skip_reason = ?, error_summary = ?, queued_at = ?, started_at = ?, finished_at = ?
		WHERE id = ?
	`, task.BatchID, task.ImageID, task.TaskType, task.SourceType, task.Status,
		task.DedupeKey, task.ImageVersionKey, task.LatestAsyncJobID,
		task.SkipReason, task.ErrorSummary, task.QueuedAt, task.StartedAt, task.FinishedAt, task.ID)
}

func (r *d1PlatformTaskRepository) SetLatestAsyncJob(ctx context.Context, taskID int64, asyncJobID *int64) error {
	var queuedAt any
	if asyncJobID != nil {
		queuedAt = time.Now()
	}
	return r.client.Exec(ctx, `
		UPDATE platform_tasks
		SET latest_async_job_id = ?,
			queued_at = CASE WHEN ? IS NULL THEN queued_at ELSE ? END,
			status = CASE WHEN ? IS NULL OR status <> ? THEN status ELSE ? END
		WHERE id = ?
	`, asyncJobID, asyncJobID, queuedAt, asyncJobID, domain.PlatformTaskStatusPending, domain.PlatformTaskStatusQueued, taskID)
}

func (r *d1PlatformTaskRepository) queryOne(ctx context.Context, query string, args ...any) (*domain.PlatformTask, error) {
	row, err := r.client.QueryOne(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, sql.ErrNoRows
	}
	return mapPlatformTaskFromD1(row)
}

func mapPlatformTasksFromD1(rows []map[string]any) ([]domain.PlatformTask, error) {
	tasks := make([]domain.PlatformTask, 0, len(rows))
	for _, row := range rows {
		task, err := mapPlatformTaskFromD1(row)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *task)
	}
	return tasks, nil
}

func mapPlatformTaskFromD1(row map[string]any) (*domain.PlatformTask, error) {
	id, err := toInt64(row["id"])
	if err != nil {
		return nil, err
	}
	batchID, err := toInt64(row["batch_id"])
	if err != nil {
		return nil, err
	}
	imageID, err := toInt64(row["image_id"])
	if err != nil {
		return nil, err
	}
	createdAt, _ := toTime(row["created_at"])
	var latestAsyncJobID *int64
	if id, err := toInt64(row["latest_async_job_id"]); err == nil && id != 0 {
		latestAsyncJobID = &id
	}
	var skipReason *string
	if value := toStringDefault(row["skip_reason"], ""); value != "" {
		skipReason = &value
	}
	var errorSummary *string
	if value := toStringDefault(row["error_summary"], ""); value != "" {
		errorSummary = &value
	}
	var queuedAt, startedAt, finishedAt *time.Time
	if t, err := toTime(row["queued_at"]); err == nil && !t.IsZero() {
		queuedAt = &t
	}
	if t, err := toTime(row["started_at"]); err == nil && !t.IsZero() {
		startedAt = &t
	}
	if t, err := toTime(row["finished_at"]); err == nil && !t.IsZero() {
		finishedAt = &t
	}
	return &domain.PlatformTask{
		ID:               id,
		BatchID:          batchID,
		ImageID:          imageID,
		TaskType:         toStringDefault(row["task_type"], ""),
		SourceType:       toStringDefault(row["source_type"], ""),
		Status:           toStringDefault(row["status"], ""),
		DedupeKey:        toStringDefault(row["dedupe_key"], ""),
		ImageVersionKey:  toStringDefault(row["image_version_key"], ""),
		LatestAsyncJobID: latestAsyncJobID,
		SkipReason:       skipReason,
		ErrorSummary:     errorSummary,
		CreatedAt:        createdAt,
		QueuedAt:         queuedAt,
		StartedAt:        startedAt,
		FinishedAt:       finishedAt,
	}, nil
}
