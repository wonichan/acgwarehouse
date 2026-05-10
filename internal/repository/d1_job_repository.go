package repository

import (
	"context"
	"strings"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/d1client"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type d1JobRepository struct {
	client *d1client.Client
}

func NewD1JobRepository(client *d1client.Client) JobRepository {
	return &d1JobRepository{client: client}
}

func (r *d1JobRepository) Save(job *domain.AsyncJob) error {
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	id, err := r.client.ExecReturningID(context.Background(), `
		INSERT INTO async_jobs (platform_task_id, type, status, payload, progress, error, created_at, started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, job.PlatformTaskID, job.Type, job.Status, job.Payload, job.Progress, job.Error, job.CreatedAt, job.StartedAt, job.FinishedAt)
	if err != nil {
		return err
	}
	job.ID = id
	return nil
}

func (r *d1JobRepository) FindByID(id int64) (*domain.AsyncJob, error) {
	row, err := r.client.QueryOne(context.Background(), `
		SELECT id, platform_task_id, type, status, payload, progress, error, created_at, started_at, finished_at
		FROM async_jobs WHERE id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return mapAsyncJobFromD1(row)
}

func (r *d1JobRepository) FindByPlatformTaskID(platformTaskID int64) ([]domain.AsyncJob, error) {
	rows, err := r.client.Query(context.Background(), `
		SELECT id, platform_task_id, type, status, payload, progress, error, created_at, started_at, finished_at
		FROM async_jobs WHERE platform_task_id = ? ORDER BY id
	`, platformTaskID)
	if err != nil {
		return nil, err
	}
	return mapAsyncJobsFromD1(rows)
}

func (r *d1JobRepository) FindByStatus(status string) ([]domain.AsyncJob, error) {
	rows, err := r.client.Query(context.Background(), `
		SELECT id, platform_task_id, type, status, payload, progress, error, created_at, started_at, finished_at
		FROM async_jobs WHERE status = ? ORDER BY id
	`, status)
	if err != nil {
		return nil, err
	}
	return mapAsyncJobsFromD1(rows)
}

func (r *d1JobRepository) FindByType(jobType string) ([]domain.AsyncJob, error) {
	rows, err := r.client.Query(context.Background(), `
		SELECT id, platform_task_id, type, status, payload, progress, error, created_at, started_at, finished_at
		FROM async_jobs WHERE type = ? ORDER BY id DESC
	`, jobType)
	if err != nil {
		return nil, err
	}
	return mapAsyncJobsFromD1(rows)
}

func (r *d1JobRepository) FindByTypeAndStatus(jobType string, status string) ([]domain.AsyncJob, error) {
	rows, err := r.client.Query(context.Background(), `
		SELECT id, platform_task_id, type, status, payload, progress, error, created_at, started_at, finished_at
		FROM async_jobs WHERE type = ? AND status = ? ORDER BY id DESC
	`, jobType, status)
	if err != nil {
		return nil, err
	}
	return mapAsyncJobsFromD1(rows)
}

func (r *d1JobRepository) Update(job *domain.AsyncJob) error {
	return r.client.Exec(context.Background(), `
		UPDATE async_jobs
		SET platform_task_id = ?, status = ?, payload = ?, progress = ?, error = ?, started_at = ?, finished_at = ?
		WHERE id = ?
	`, job.PlatformTaskID, job.Status, job.Payload, job.Progress, job.Error, job.StartedAt, job.FinishedAt, job.ID)
}

func (r *d1JobRepository) FindAndClaimReadyJobs(ctx context.Context, jobType string, limit int) ([]domain.AsyncJob, error) {
	if limit <= 0 {
		return []domain.AsyncJob{}, nil
	}
	rows, err := r.client.Query(ctx, `
		SELECT id FROM async_jobs WHERE type = ? AND status = 'ready' ORDER BY id ASC LIMIT ?
	`, jobType, int64(limit))
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []domain.AsyncJob{}, nil
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		id, err := toInt64(row["id"])
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, 2+len(ids))
	args = append(args, time.Now(), jobType)
	for _, id := range ids {
		args = append(args, id)
	}
	if err := r.client.Exec(ctx, `
		UPDATE async_jobs SET status = 'running', started_at = ?
		WHERE type = ? AND status = 'ready' AND id IN (`+placeholders+`)
	`, args...); err != nil {
		return nil, err
	}

	queryArgs := make([]any, 0, 1+len(ids))
	queryArgs = append(queryArgs, jobType)
	for _, id := range ids {
		queryArgs = append(queryArgs, id)
	}
	claimed, err := r.client.Query(ctx, `
		SELECT id, platform_task_id, type, status, payload, progress, error, created_at, started_at, finished_at
		FROM async_jobs WHERE type = ? AND status = 'running' AND id IN (`+placeholders+`) ORDER BY id ASC
	`, queryArgs...)
	if err != nil {
		return nil, err
	}
	return mapAsyncJobsFromD1(claimed)
}

func (r *d1JobRepository) FindRecent(limit int) ([]domain.AsyncJob, error) {
	rows, err := r.client.Query(context.Background(), `
		SELECT id, platform_task_id, type, status, payload, progress, error, created_at, started_at, finished_at
		FROM async_jobs ORDER BY created_at DESC LIMIT ?
	`, int64(limit))
	if err != nil {
		return nil, err
	}
	return mapAsyncJobsFromD1(rows)
}

func (r *d1JobRepository) FindFailed() ([]domain.AsyncJob, error) {
	rows, err := r.client.Query(context.Background(), `
		SELECT id, platform_task_id, type, status, payload, progress, error, created_at, started_at, finished_at
		FROM async_jobs WHERE status = 'failed' ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	return mapAsyncJobsFromD1(rows)
}

func (r *d1JobRepository) UpdateStatus(id int64, status string, errorMsg *string) error {
	return r.client.Exec(context.Background(), `UPDATE async_jobs SET status = ?, error = ? WHERE id = ?`, status, errorMsg, id)
}

func (r *d1JobRepository) CountByStatus(status string) (int64, error) {
	return r.client.QueryCount(context.Background(), `SELECT COUNT(*) as cnt FROM async_jobs WHERE status = ?`, status)
}

func (r *d1JobRepository) ResetRunningToReady() (int64, error) {
	count, err := r.client.QueryCount(context.Background(), `SELECT COUNT(*) AS cnt FROM async_jobs WHERE status = 'running'`)
	if err != nil {
		return 0, err
	}
	if err := r.client.Exec(context.Background(), `UPDATE async_jobs SET status = 'ready', started_at = NULL WHERE status = 'running'`); err != nil {
		return 0, err
	}
	return count, nil
}

func mapAsyncJobFromD1(row map[string]any) (*domain.AsyncJob, error) {
	id, err := toInt64(row["id"])
	if err != nil {
		return nil, err
	}
	var platformTaskID *int64
	if ptid, err := toInt64(row["platform_task_id"]); err == nil && ptid != 0 {
		platformTaskID = &ptid
	}
	createdAt, _ := toTime(row["created_at"])
	var startedAt, finishedAt *time.Time
	if sa, err := toTime(row["started_at"]); err == nil && !sa.IsZero() {
		startedAt = &sa
	}
	if fa, err := toTime(row["finished_at"]); err == nil && !fa.IsZero() {
		finishedAt = &fa
	}
	var errorStr *string
	if e, ok := row["error"]; ok && e != nil {
		if s, ok := e.(string); ok && s != "" {
			errorStr = &s
		}
	}
	return &domain.AsyncJob{
		ID:             id,
		PlatformTaskID: platformTaskID,
		Type:           toStringDefault(row["type"], ""),
		Status:         toStringDefault(row["status"], ""),
		Payload:        toStringDefault(row["payload"], ""),
		Progress:       mustFloat64(row["progress"]),
		Error:          errorStr,
		CreatedAt:      createdAt,
		StartedAt:      startedAt,
		FinishedAt:     finishedAt,
	}, nil
}

func mustFloat64(v any) float64 {
	f, _ := toFloat64(v)
	return f
}

func mapAsyncJobsFromD1(rows []map[string]any) ([]domain.AsyncJob, error) {
	jobs := make([]domain.AsyncJob, 0, len(rows))
	for _, row := range rows {
		job, err := mapAsyncJobFromD1(row)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *job)
	}
	return jobs, nil
}
