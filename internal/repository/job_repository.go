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
	Update(job *domain.AsyncJob) error
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
		INSERT INTO async_jobs (type, status, payload, progress, error, created_at, started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, job.Type, job.Status, job.Payload, job.Progress, job.Error, job.CreatedAt, job.StartedAt, job.FinishedAt)
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
		SELECT id, type, status, payload, progress, error, created_at, started_at, finished_at
		FROM async_jobs WHERE id = ?
	`, id).Scan(&job.ID, &job.Type, &job.Status, &job.Payload, &job.Progress, &job.Error, &job.CreatedAt, &job.StartedAt, &job.FinishedAt)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (r *sqliteJobRepository) FindByStatus(status string) ([]domain.AsyncJob, error) {
	rows, err := r.db.Query(`
		SELECT id, type, status, payload, progress, error, created_at, started_at, finished_at
		FROM async_jobs WHERE status = ? ORDER BY id
	`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]domain.AsyncJob, 0)
	for rows.Next() {
		var job domain.AsyncJob
		if err := rows.Scan(&job.ID, &job.Type, &job.Status, &job.Payload, &job.Progress, &job.Error, &job.CreatedAt, &job.StartedAt, &job.FinishedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (r *sqliteJobRepository) Update(job *domain.AsyncJob) error {
	_, err := r.db.Exec(`
		UPDATE async_jobs
		SET status = ?, payload = ?, progress = ?, error = ?, started_at = ?, finished_at = ?
		WHERE id = ?
	`, job.Status, job.Payload, job.Progress, job.Error, job.StartedAt, job.FinishedAt, job.ID)
	return err
}
