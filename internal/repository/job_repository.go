package repository

import (
	"database/sql"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type JobRepository interface {
	Save(job *domain.AsyncJob) error
	FindByID(id int64) (*domain.AsyncJob, error)
	FindByStatus(status string) ([]domain.AsyncJob, error)
	FindByType(jobType string) ([]domain.AsyncJob, error)
	FindByTypeAndStatus(jobType string, status string) ([]domain.AsyncJob, error)
	Update(job *domain.AsyncJob) error
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

// ResetRunningToReady resets all running jobs to ready status.
// This is used to recover jobs that were in running state when the server crashed.
func (r *sqliteJobRepository) ResetRunningToReady() (int64, error) {
	result, err := r.db.Exec(`UPDATE async_jobs SET status = 'ready', started_at = NULL WHERE status = 'running'`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
