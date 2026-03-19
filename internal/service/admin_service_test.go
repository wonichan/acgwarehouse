package service_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
	"github.com/wonichan/acgwarehouse-backend/internal/worker"
)

func newTestAdminDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	// Create tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS async_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			status TEXT DEFAULT 'ready',
			payload TEXT,
			progress REAL DEFAULT 0.0,
			error TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			started_at TIMESTAMP,
			finished_at TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS images (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT UNIQUE NOT NULL,
			filename TEXT NOT NULL,
			source_root TEXT NOT NULL,
			file_size INTEGER,
			width INTEGER,
			height INTEGER,
			format TEXT,
			phash INTEGER,
			thumbnail_small_url TEXT,
			thumbnail_large_url TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS tags (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			preferred_label TEXT UNIQUE NOT NULL,
			slug TEXT UNIQUE NOT NULL,
			primary_category TEXT,
			review_state TEXT DEFAULT 'pending',
			trust_score REAL DEFAULT 0.0,
			usage_count INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS collections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			cover_image_id INTEGER,
			image_count INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}
	return db
}

func TestAdminService_GetSummary(t *testing.T) {
	db := newTestAdminDB(t)
	defer db.Close()

	jobRepo := repository.NewJobRepository(db)
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	collectionRepo := repository.NewCollectionRepository(db)
	jobManager := worker.NewManager(jobRepo)

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Env:  "development",
		},
		Admin: config.AdminConfig{
			Username: "admin",
			Password: "secret123",
		},
	}

	svc := service.NewAdminService(cfg, jobRepo, imageRepo, tagRepo, collectionRepo, jobManager)

	// Add some test data
	job := &domain.AsyncJob{
		Type:      "test_job",
		Status:    "ready",
		CreatedAt: time.Now(),
	}
	_ = jobRepo.Save(job)

	img := &domain.Image{
		Path:              "/test/image1.jpg",
		Filename:          "image1.jpg",
		SourceRoot:        "/test",
		ThumbnailSmallUrl: "",
		ThumbnailLargeUrl: "",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	_, err := imageRepo.SaveImage(img)
	if err != nil {
		t.Fatalf("Failed to save test image: %v", err)
	}

	summary, err := svc.GetSummary(context.Background())
	if err != nil {
		t.Fatalf("Failed to get summary: %v", err)
	}

	// Verify health status
	if summary.Health.Status != "healthy" {
		t.Errorf("Expected health status 'healthy', got '%s'", summary.Health.Status)
	}

	// Verify config summary doesn't expose secrets
	if summary.Config.AdminUsername != "admin" {
		t.Errorf("Expected admin username 'admin', got '%s'", summary.Config.AdminUsername)
	}

	// Verify task counts
	if summary.Tasks.Total < 1 {
		t.Errorf("Expected at least 1 task, got %d", summary.Tasks.Total)
	}

	// Verify library stats
	if summary.Library.TotalImages < 1 {
		t.Errorf("Expected at least 1 image, got %d", summary.Library.TotalImages)
	}
}

func TestAdminService_GetSummary_HidesSecrets(t *testing.T) {
	db := newTestAdminDB(t)
	defer db.Close()

	jobRepo := repository.NewJobRepository(db)
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	collectionRepo := repository.NewCollectionRepository(db)
	jobManager := worker.NewManager(jobRepo)

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			Env:  "development",
		},
		AI: config.AIConfig{
			APIKey: "super-secret-api-key",
		},
		COS: config.COSConfig{
			SecretID:  "cos-secret-id",
			SecretKey: "cos-secret-key",
		},
		Admin: config.AdminConfig{
			Password: "admin-password",
		},
	}

	svc := service.NewAdminService(cfg, jobRepo, imageRepo, tagRepo, collectionRepo, jobManager)

	summary, err := svc.GetSummary(context.Background())
	if err != nil {
		t.Fatalf("Failed to get summary: %v", err)
	}

	// Verify secrets are NOT exposed (only indication that they exist)
	if !summary.Config.HasAIKey {
		t.Error("AI API key should be indicated as configured")
	}
	if !summary.Config.HasCOSSecretKey {
		t.Error("COS secret key should be indicated as configured")
	}
	if !summary.Config.HasAdminPassword {
		t.Error("Admin password should be indicated as configured")
	}
}

func TestAdminService_GetJobs(t *testing.T) {
	db := newTestAdminDB(t)
	defer db.Close()

	jobRepo := repository.NewJobRepository(db)
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	collectionRepo := repository.NewCollectionRepository(db)
	jobManager := worker.NewManager(jobRepo)

	cfg := &config.Config{}
	svc := service.NewAdminService(cfg, jobRepo, imageRepo, tagRepo, collectionRepo, jobManager)

	// Create some jobs
	for i := 0; i < 5; i++ {
		job := &domain.AsyncJob{
			Type:      "test_job",
			Status:    "finished",
			CreatedAt: time.Now(),
		}
		_ = jobRepo.Save(job)
	}

	jobs, err := svc.GetJobs(context.Background(), 10)
	if err != nil {
		t.Fatalf("Failed to get jobs: %v", err)
	}

	if len(jobs) != 5 {
		t.Errorf("Expected 5 jobs, got %d", len(jobs))
	}
}

func TestAdminService_TriggerScan(t *testing.T) {
	db := newTestAdminDB(t)
	defer db.Close()

	jobRepo := repository.NewJobRepository(db)
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	collectionRepo := repository.NewCollectionRepository(db)
	jobManager := worker.NewManager(jobRepo)

	cfg := &config.Config{
		Storage: config.StorageConfig{
			ScanRoots: []string{"/test/path"},
		},
	}
	svc := service.NewAdminService(cfg, jobRepo, imageRepo, tagRepo, collectionRepo, jobManager)

	jobID, err := svc.TriggerScan(context.Background())
	if err != nil {
		t.Fatalf("Failed to trigger scan: %v", err)
	}

	if jobID == 0 {
		t.Error("Expected job ID to be set")
	}

	// Verify the job was created
	job, err := jobRepo.FindByID(jobID)
	if err != nil {
		t.Fatalf("Failed to find created job: %v", err)
	}

	if job.Type != "manual_scan" {
		t.Errorf("Expected job type 'manual_scan', got '%s'", job.Type)
	}
}

func TestAdminService_RetryFailedJobs(t *testing.T) {
	db := newTestAdminDB(t)
	defer db.Close()

	jobRepo := repository.NewJobRepository(db)
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	collectionRepo := repository.NewCollectionRepository(db)
	jobManager := worker.NewManager(jobRepo)

	cfg := &config.Config{}
	svc := service.NewAdminService(cfg, jobRepo, imageRepo, tagRepo, collectionRepo, jobManager)

	// Create failed jobs
	var failedJobIDs []int64
	for i := 0; i < 3; i++ {
		job := &domain.AsyncJob{
			Type:      "test_job",
			Status:    "failed",
			Error:     strPtr("some error"),
			CreatedAt: time.Now(),
		}
		_ = jobRepo.Save(job)
		failedJobIDs = append(failedJobIDs, job.ID)
	}

	// Create a running job - should NOT be retried
	runningJob := &domain.AsyncJob{
		Type:      "test_job",
		Status:    "running",
		CreatedAt: time.Now(),
	}
	_ = jobRepo.Save(runningJob)

	// Create a finished job - should NOT be retried
	finishedJob := &domain.AsyncJob{
		Type:      "test_job",
		Status:    "finished",
		CreatedAt: time.Now(),
	}
	_ = jobRepo.Save(finishedJob)

	// Retry failed jobs
	count, err := svc.RetryFailedJobs(context.Background())
	if err != nil {
		t.Fatalf("Failed to retry failed jobs: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 jobs retried, got %d", count)
	}

	// Verify failed jobs are now ready
	for _, id := range failedJobIDs {
		job, err := jobRepo.FindByID(id)
		if err != nil {
			t.Fatalf("Failed to find job %d: %v", id, err)
		}
		if job.Status != "ready" {
			t.Errorf("Expected job %d status 'ready', got '%s'", id, job.Status)
		}
		if job.Error != nil {
			t.Errorf("Expected job %d error to be cleared, got '%s'", id, *job.Error)
		}
	}

	// Verify running job unchanged
	running, _ := jobRepo.FindByID(runningJob.ID)
	if running.Status != "running" {
		t.Errorf("Running job should not be affected, status: '%s'", running.Status)
	}

	// Verify finished job unchanged
	finished, _ := jobRepo.FindByID(finishedJob.ID)
	if finished.Status != "finished" {
		t.Errorf("Finished job should not be affected, status: '%s'", finished.Status)
	}
}

func TestAdminService_PauseResumeBackgroundTasks(t *testing.T) {
	db := newTestAdminDB(t)
	defer db.Close()

	jobRepo := repository.NewJobRepository(db)
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	collectionRepo := repository.NewCollectionRepository(db)
	jobManager := worker.NewManager(jobRepo)

	cfg := &config.Config{}
	svc := service.NewAdminService(cfg, jobRepo, imageRepo, tagRepo, collectionRepo, jobManager)

	// Initially running
	if !svc.IsBackgroundRunning() {
		t.Error("Expected background tasks to be running initially")
	}

	// Pause
	err := svc.PauseBackgroundTasks(context.Background())
	if err != nil {
		t.Fatalf("Failed to pause background tasks: %v", err)
	}

	if svc.IsBackgroundRunning() {
		t.Error("Expected background tasks to be paused")
	}

	// Resume
	err = svc.ResumeBackgroundTasks(context.Background())
	if err != nil {
		t.Fatalf("Failed to resume background tasks: %v", err)
	}

	if !svc.IsBackgroundRunning() {
		t.Error("Expected background tasks to be running after resume")
	}
}

func strPtr(s string) *string {
	return &s
}
