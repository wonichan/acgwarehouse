package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

func newTestJobDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	// Create table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS async_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			platform_task_id INTEGER,
			type TEXT NOT NULL,
			status TEXT DEFAULT 'ready',
			payload TEXT,
			progress REAL DEFAULT 0.0,
			error TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			started_at TIMESTAMP,
			finished_at TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_async_jobs_status ON async_jobs(status);
		CREATE INDEX IF NOT EXISTS idx_async_jobs_type ON async_jobs(type);
		CREATE INDEX IF NOT EXISTS idx_async_jobs_platform_task_id ON async_jobs(platform_task_id);
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	return db
}

func TestJobRepository_Save(t *testing.T) {
	db := newTestJobDB(t)
	defer db.Close()

	repo := NewJobRepository(db)
	job := &domain.AsyncJob{
		Type:      "test_job",
		Status:    "ready",
		Payload:   `{"key": "value"}`,
		CreatedAt: time.Now(),
	}

	err := repo.Save(job)
	if err != nil {
		t.Fatalf("Failed to save job: %v", err)
	}

	if job.ID == 0 {
		t.Error("Expected job ID to be set after save")
	}
}

func TestJobRepository_FindByID(t *testing.T) {
	db := newTestJobDB(t)
	defer db.Close()

	repo := NewJobRepository(db)
	job := &domain.AsyncJob{
		Type:      "test_job",
		Status:    "ready",
		Payload:   `{"test": "data"}`,
		CreatedAt: time.Now(),
	}
	_ = repo.Save(job)

	found, err := repo.FindByID(job.ID)
	if err != nil {
		t.Fatalf("Failed to find job: %v", err)
	}

	if found.Type != "test_job" {
		t.Errorf("Expected type 'test_job', got '%s'", found.Type)
	}
}

func TestJobRepository_FindByStatus(t *testing.T) {
	db := newTestJobDB(t)
	defer db.Close()

	repo := NewJobRepository(db)

	// Create jobs with different statuses
	for i := 0; i < 3; i++ {
		job := &domain.AsyncJob{
			Type:      "test_job",
			Status:    "ready",
			CreatedAt: time.Now(),
		}
		_ = repo.Save(job)
	}

	failedJob := &domain.AsyncJob{
		Type:      "test_job",
		Status:    "failed",
		Error:     ptr("something went wrong"),
		CreatedAt: time.Now(),
	}
	_ = repo.Save(failedJob)

	readyJobs, err := repo.FindByStatus("ready")
	if err != nil {
		t.Fatalf("Failed to find jobs by status: %v", err)
	}

	if len(readyJobs) != 3 {
		t.Errorf("Expected 3 ready jobs, got %d", len(readyJobs))
	}
}

func TestJobRepository_FindRecent(t *testing.T) {
	db := newTestJobDB(t)
	defer db.Close()

	repo := NewJobRepository(db)

	// Create some jobs
	for i := 0; i < 5; i++ {
		job := &domain.AsyncJob{
			Type:      "test_job",
			Status:    "finished",
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Hour),
		}
		_ = repo.Save(job)
	}

	recent, err := repo.FindRecent(3)
	if err != nil {
		t.Fatalf("Failed to find recent jobs: %v", err)
	}

	if len(recent) != 3 {
		t.Errorf("Expected 3 recent jobs, got %d", len(recent))
	}
}

func TestJobRepository_FindFailed(t *testing.T) {
	db := newTestJobDB(t)
	defer db.Close()

	repo := NewJobRepository(db)

	// Create failed jobs
	for i := 0; i < 3; i++ {
		job := &domain.AsyncJob{
			Type:      "test_job",
			Status:    "failed",
			Error:     ptr("error message"),
			CreatedAt: time.Now(),
		}
		_ = repo.Save(job)
	}

	// Create a successful job
	successJob := &domain.AsyncJob{
		Type:      "test_job",
		Status:    "finished",
		CreatedAt: time.Now(),
	}
	_ = repo.Save(successJob)

	failed, err := repo.FindFailed()
	if err != nil {
		t.Fatalf("Failed to find failed jobs: %v", err)
	}

	if len(failed) != 3 {
		t.Errorf("Expected 3 failed jobs, got %d", len(failed))
	}
}

func TestJobRepository_UpdateStatus(t *testing.T) {
	db := newTestJobDB(t)
	defer db.Close()

	repo := NewJobRepository(db)
	job := &domain.AsyncJob{
		Type:      "test_job",
		Status:    "failed",
		Error:     ptr("original error"),
		CreatedAt: time.Now(),
	}
	_ = repo.Save(job)

	// Update status to ready for retry
	err := repo.UpdateStatus(job.ID, "ready", nil)
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	updated, err := repo.FindByID(job.ID)
	if err != nil {
		t.Fatalf("Failed to find updated job: %v", err)
	}

	if updated.Status != "ready" {
		t.Errorf("Expected status 'ready', got '%s'", updated.Status)
	}
	if updated.Error != nil {
		t.Errorf("Expected error to be nil, got '%s'", *updated.Error)
	}
}

func TestJobRepository_CountByStatus(t *testing.T) {
	db := newTestJobDB(t)
	defer db.Close()

	repo := NewJobRepository(db)

	// Create jobs with different statuses
	for i := 0; i < 2; i++ {
		job := &domain.AsyncJob{
			Type:      "test_job",
			Status:    "running",
			CreatedAt: time.Now(),
		}
		_ = repo.Save(job)
	}

	for i := 0; i < 3; i++ {
		job := &domain.AsyncJob{
			Type:      "test_job",
			Status:    "ready",
			CreatedAt: time.Now(),
		}
		_ = repo.Save(job)
	}

	runningCount, err := repo.CountByStatus("running")
	if err != nil {
		t.Fatalf("Failed to count running jobs: %v", err)
	}

	if runningCount != 2 {
		t.Errorf("Expected 2 running jobs, got %d", runningCount)
	}

	readyCount, err := repo.CountByStatus("ready")
	if err != nil {
		t.Fatalf("Failed to count ready jobs: %v", err)
	}

	if readyCount != 3 {
		t.Errorf("Expected 3 ready jobs, got %d", readyCount)
	}
}

func TestJobRepository_PreservesPlatformTaskAssociation(t *testing.T) {
	db := newTestJobDB(t)
	defer db.Close()

	repo := NewJobRepository(db)
	platformTaskID := int64(42)
	job := &domain.AsyncJob{
		PlatformTaskID: &platformTaskID,
		Type:           "thumbnail_generate",
		Status:         "ready",
		Payload:        `{"image_id": 1}`,
		CreatedAt:      time.Now(),
	}

	if err := repo.Save(job); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	found, err := repo.FindByPlatformTaskID(platformTaskID)
	if err != nil {
		t.Fatalf("FindByPlatformTaskID() error = %v", err)
	}
	if len(found) != 1 || found[0].PlatformTaskID == nil || *found[0].PlatformTaskID != platformTaskID {
		t.Fatalf("FindByPlatformTaskID() = %+v, want one job linked to %d", found, platformTaskID)
	}

	nextPlatformTaskID := int64(84)
	job.PlatformTaskID = &nextPlatformTaskID
	if err := repo.Update(job); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	reloaded, err := repo.FindByID(job.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if reloaded.PlatformTaskID == nil || *reloaded.PlatformTaskID != nextPlatformTaskID {
		t.Fatalf("updated PlatformTaskID = %v, want %d", reloaded.PlatformTaskID, nextPlatformTaskID)
	}
}

func ptr(s string) *string {
	return &s
}
