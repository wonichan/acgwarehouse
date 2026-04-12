package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type JobRepository interface {
	Save(job *domain.AsyncJob) error
	FindByID(id int64) (*domain.AsyncJob, error)
	FindByPlatformTaskID(platformTaskID int64) ([]domain.AsyncJob, error)
	FindByStatus(status string) ([]domain.AsyncJob, error)
	FindByType(jobType string) ([]domain.AsyncJob, error)
	FindByTypeAndStatus(jobType string, status string) ([]domain.AsyncJob, error)
	Update(job *domain.AsyncJob) error
	// FindAndClaimReadyJobs atomically finds up to limit ready jobs of the given type
	// and transitions them to 'running', returning the claimed jobs.
	FindAndClaimReadyJobs(ctx context.Context, jobType string, limit int) ([]domain.AsyncJob, error)
	// Admin dashboard support methods
	FindRecent(limit int) ([]domain.AsyncJob, error)
	FindFailed() ([]domain.AsyncJob, error)
	UpdateStatus(id int64, status string, errorMsg *string) error
	CountByStatus(status string) (int64, error)
	// ResetRunningToReady resets all running jobs to ready status (for recovery after crash)
	ResetRunningToReady() (int64, error)
}

type sqliteJobRepository struct {
	db *sql.DB
}

func NewJobRepository(db *sql.DB) JobRepository {
	return &sqliteJobRepository{db: db}
}

func (r *sqliteJobRepository) Save(job *domain.AsyncJob) error {
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	result, err := r.db.Exec(`
		INSERT INTO async_jobs (platform_task_id, type, status, payload, progress, error, created_at, started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, job.PlatformTaskID, job.Type, job.Status, job.Payload, job.Progress, job.Error, job.CreatedAt, job.StartedAt, job.FinishedAt)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	job.ID = id
	return nil
}

func (r *sqliteJobRepository) FindByID(id int64) (*domain.AsyncJob, error) {
	job := &domain.AsyncJob{}
	err := r.db.QueryRow(`
		SELECT id, platform_task_id, type, status, payload, progress, error, created_at, started_at, finished_at
		FROM async_jobs WHERE id = ?
	`, id).Scan(&job.ID, &job.PlatformTaskID, &job.Type, &job.Status, &job.Payload, &job.Progress, &job.Error, &job.CreatedAt, &job.StartedAt, &job.FinishedAt)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (r *sqliteJobRepository) FindByStatus(status string) ([]domain.AsyncJob, error) {
	return r.findMany(`
		SELECT id, platform_task_id, type, status, payload, progress, error, created_at, started_at, finished_at
		FROM async_jobs WHERE status = ? ORDER BY id
	`, status)
}

func (r *sqliteJobRepository) FindByPlatformTaskID(platformTaskID int64) ([]domain.AsyncJob, error) {
	return r.findMany(`
		SELECT id, platform_task_id, type, status, payload, progress, error, created_at, started_at, finished_at
		FROM async_jobs WHERE platform_task_id = ? ORDER BY id
	`, platformTaskID)
}

func (r *sqliteJobRepository) FindByType(jobType string) ([]domain.AsyncJob, error) {
	return r.findMany(`
		SELECT id, platform_task_id, type, status, payload, progress, error, created_at, started_at, finished_at
		FROM async_jobs WHERE type = ? ORDER BY id DESC
	`, jobType)
}

func (r *sqliteJobRepository) FindByTypeAndStatus(jobType string, status string) ([]domain.AsyncJob, error) {
	return r.findMany(`
		SELECT id, platform_task_id, type, status, payload, progress, error, created_at, started_at, finished_at
		FROM async_jobs WHERE type = ? AND status = ? ORDER BY id DESC
	`, jobType, status)
}

func (r *sqliteJobRepository) findMany(query string, args ...any) ([]domain.AsyncJob, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]domain.AsyncJob, 0)
	for rows.Next() {
		var job domain.AsyncJob
		if err := rows.Scan(&job.ID, &job.PlatformTaskID, &job.Type, &job.Status, &job.Payload, &job.Progress, &job.Error, &job.CreatedAt, &job.StartedAt, &job.FinishedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (r *sqliteJobRepository) Update(job *domain.AsyncJob) error {
	_, err := r.db.Exec(`
		UPDATE async_jobs
		SET platform_task_id = ?, status = ?, payload = ?, progress = ?, error = ?, started_at = ?, finished_at = ?
		WHERE id = ?
	`, job.PlatformTaskID, job.Status, job.Payload, job.Progress, job.Error, job.StartedAt, job.FinishedAt, job.ID)
	return err
}

// FindRecent returns the most recent jobs, ordered by created_at descending.
func (r *sqliteJobRepository) FindRecent(limit int) ([]domain.AsyncJob, error) {
	return r.findMany(`
		SELECT id, platform_task_id, type, status, payload, progress, error, created_at, started_at, finished_at
		FROM async_jobs ORDER BY created_at DESC LIMIT ?
	`, limit)
}

// FindFailed returns all jobs with status 'failed'.
func (r *sqliteJobRepository) FindFailed() ([]domain.AsyncJob, error) {
	return r.findMany(`
		SELECT id, platform_task_id, type, status, payload, progress, error, created_at, started_at, finished_at
		FROM async_jobs WHERE status = 'failed' ORDER BY created_at DESC
	`)
}

// UpdateStatus updates the status and optionally clears the error message of a job.
func (r *sqliteJobRepository) UpdateStatus(id int64, status string, errorMsg *string) error {
	_, err := r.db.Exec(`
		UPDATE async_jobs SET status = ?, error = ? WHERE id = ?
	`, status, errorMsg, id)
	return err
}

// CountByStatus returns the count of jobs with the given status.
func (r *sqliteJobRepository) CountByStatus(status string) (int64, error) {
	var count int64
	err := r.db.QueryRow(`SELECT COUNT(*) FROM async_jobs WHERE status = ?`, status).Scan(&count)
	return count, err
}

// FindAndClaimReadyJobs atomically finds up to limit ready jobs and transitions them to running.
func (r *sqliteJobRepository) FindAndClaimReadyJobs(ctx context.Context, jobType string, limit int) ([]domain.AsyncJob, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT id FROM async_jobs WHERE type = ? AND status = 'ready' ORDER BY id ASC LIMIT ?
	`, jobType, limit)
	if err != nil {
		return nil, err
	}

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()

	if len(ids) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	now := time.Now()
	_, err = tx.ExecContext(ctx, `
		UPDATE async_jobs SET status = 'running', started_at = ?
		WHERE type = ? AND status = 'ready' AND id IN (`+strings.Join(placeholders, ",")+`)
	`, append([]any{now, jobType}, args...)...)
	if err != nil {
		return nil, err
	}

	jobs, err := r.findManyTx(tx, `
		SELECT id, platform_task_id, type, status, payload, progress, error, created_at, started_at, finished_at
		FROM async_jobs WHERE type = ? AND status = 'running' AND id IN (`+strings.Join(placeholders, ",")+`) ORDER BY id ASC
	`, append([]any{jobType}, args...)...)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return jobs, nil
}

func (r *sqliteJobRepository) findManyTx(tx *sql.Tx, query string, args ...any) ([]domain.AsyncJob, error) {
	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]domain.AsyncJob, 0)
	for rows.Next() {
		var job domain.AsyncJob
		if err := rows.Scan(&job.ID, &job.PlatformTaskID, &job.Type, &job.Status, &job.Payload, &job.Progress, &job.Error, &job.CreatedAt, &job.StartedAt, &job.FinishedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (r *sqliteJobRepository) ResetRunningToReady() (int64, error) {
	result, err := r.db.Exec(`UPDATE async_jobs SET status = 'ready', started_at = NULL WHERE status = 'running'`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
