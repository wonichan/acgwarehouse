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
	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("Failed to ensure scan schema: %v", err)
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

func TestAdminService_GetTaskPlatformOverview_ReturnsQueueAndPlatformCounts(t *testing.T) {
	db := newTestAdminDB(t)
	defer db.Close()

	jobRepo := repository.NewJobRepository(db)
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	collectionRepo := repository.NewCollectionRepository(db)
	jobManager := worker.NewManagerWithConfig(jobRepo, 3, 32)
	jobManager.Pause()

	ctx := context.Background()
	if _, err := jobManager.AddJob(ctx, "test_job", `{"n":1}`); err != nil {
		t.Fatalf("Failed to add queued job #1: %v", err)
	}
	if _, err := jobManager.AddJob(ctx, "test_job", `{"n":2}`); err != nil {
		t.Fatalf("Failed to add queued job #2: %v", err)
	}

	batchOneID := insertTaskBatchForOverviewTest(t, db, "running")
	batchTwoID := insertTaskBatchForOverviewTest(t, db, "partial_failed")

	insertImageForOverviewTest(t, db, 1001, "/test/overview-1.jpg", "overview-1.jpg")
	insertImageForOverviewTest(t, db, 1002, "/test/overview-2.jpg", "overview-2.jpg")
	insertImageForOverviewTest(t, db, 1003, "/test/overview-3.jpg", "overview-3.jpg")
	insertImageForOverviewTest(t, db, 1004, "/test/overview-4.jpg", "overview-4.jpg")
	insertImageForOverviewTest(t, db, 1005, "/test/overview-5.jpg", "overview-5.jpg")
	insertImageForOverviewTest(t, db, 1006, "/test/overview-6.jpg", "overview-6.jpg")

	insertPlatformTaskForOverviewTest(t, db, batchOneID, 1001, "pending")
	insertPlatformTaskForOverviewTest(t, db, batchOneID, 1002, "queued")
	insertPlatformTaskForOverviewTest(t, db, batchOneID, 1003, "running")
	insertPlatformTaskForOverviewTest(t, db, batchTwoID, 1004, "completed")
	insertPlatformTaskForOverviewTest(t, db, batchTwoID, 1005, "failed")
	insertPlatformTaskForOverviewTest(t, db, batchTwoID, 1006, "cancelled")

	taskReadSvc := service.NewTaskReadService(repository.NewTaskBatchReadRepository(db))
	cfg := &config.Config{}
	svc := service.NewAdminService(cfg, jobRepo, imageRepo, tagRepo, collectionRepo, jobManager, taskReadSvc)

	overview, err := svc.GetTaskPlatformOverview(ctx)
	if err != nil {
		t.Fatalf("Failed to get task platform overview: %v", err)
	}

	if !overview.Queue.IsPaused {
		t.Fatalf("Expected queue to be paused")
	}
	if overview.Queue.QueueSize != 2 {
		t.Fatalf("Expected queue size 2, got %d", overview.Queue.QueueSize)
	}
	if overview.Queue.WorkerCount != 3 {
		t.Fatalf("Expected worker count 3, got %d", overview.Queue.WorkerCount)
	}

	if overview.Batches["running"] != 1 {
		t.Fatalf("Expected running batch count 1, got %d", overview.Batches["running"])
	}
	if overview.Batches["partial_failed"] != 1 {
		t.Fatalf("Expected partial_failed batch count 1, got %d", overview.Batches["partial_failed"])
	}

	if overview.Tasks["pending"] != 1 {
		t.Fatalf("Expected pending task count 1, got %d", overview.Tasks["pending"])
	}
	if overview.Tasks["queued"] != 1 {
		t.Fatalf("Expected queued task count 1, got %d", overview.Tasks["queued"])
	}
	if overview.Tasks["running"] != 1 {
		t.Fatalf("Expected running task count 1, got %d", overview.Tasks["running"])
	}
	if overview.Tasks["completed"] != 1 {
		t.Fatalf("Expected completed task count 1, got %d", overview.Tasks["completed"])
	}
	if overview.Tasks["failed"] != 1 {
		t.Fatalf("Expected failed task count 1, got %d", overview.Tasks["failed"])
	}
	if overview.Tasks["cancelled"] != 1 {
		t.Fatalf("Expected cancelled task count 1, got %d", overview.Tasks["cancelled"])
	}
}

func TestAdminService_GetTaskPlatformOverview_HandlesMissingTaskReadData(t *testing.T) {
	db := newTestAdminDB(t)
	defer db.Close()

	jobRepo := repository.NewJobRepository(db)
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	collectionRepo := repository.NewCollectionRepository(db)
	jobManager := worker.NewManager(jobRepo)

	cfg := &config.Config{}

	t.Run("without task read service", func(t *testing.T) {
		svc := service.NewAdminService(cfg, jobRepo, imageRepo, tagRepo, collectionRepo, jobManager)

		overview, err := svc.GetTaskPlatformOverview(context.Background())
		if err != nil {
			t.Fatalf("Expected no error without taskReadSvc, got: %v", err)
		}
		if len(overview.Batches) != 0 {
			t.Fatalf("Expected empty batch counts, got: %#v", overview.Batches)
		}
		if len(overview.Tasks) != 0 {
			t.Fatalf("Expected empty task counts, got: %#v", overview.Tasks)
		}
	})

	t.Run("with task read service but no batches", func(t *testing.T) {
		svc := service.NewAdminService(
			cfg,
			jobRepo,
			imageRepo,
			tagRepo,
			collectionRepo,
			jobManager,
			service.NewTaskReadService(repository.NewTaskBatchReadRepository(db)),
		)

		overview, err := svc.GetTaskPlatformOverview(context.Background())
		if err != nil {
			t.Fatalf("Expected no error with empty batches, got: %v", err)
		}
		if len(overview.Batches) != 0 {
			t.Fatalf("Expected empty batch counts, got: %#v", overview.Batches)
		}
		if len(overview.Tasks) != 0 {
			t.Fatalf("Expected empty task counts, got: %#v", overview.Tasks)
		}
	})
}

func TestAdminService_GetTaskPlatformOverview_PreservesLegacySummaryFields(t *testing.T) {
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
			Port: 8090,
			Env:  "testing",
		},
		AI: config.AIConfig{
			APIKey: "test-key",
		},
		COS: config.COSConfig{
			SecretKey: "cos-secret",
		},
		Admin: config.AdminConfig{
			Username: "ops-admin",
			Password: "ops-secret",
		},
	}

	svc := service.NewAdminService(cfg, jobRepo, imageRepo, tagRepo, collectionRepo, jobManager)

	overview, err := svc.GetTaskPlatformOverview(context.Background())
	if err != nil {
		t.Fatalf("Failed to get task platform overview: %v", err)
	}

	if overview.Health.Status != "healthy" {
		t.Fatalf("Expected health status healthy, got %s", overview.Health.Status)
	}
	if overview.Config.ServerHost != "localhost" {
		t.Fatalf("Expected server host localhost, got %s", overview.Config.ServerHost)
	}
	if !overview.Config.HasAIKey {
		t.Fatalf("Expected has_ai_key=true")
	}
	if !overview.Config.HasCOSSecretKey {
		t.Fatalf("Expected has_cos_secret_key=true")
	}
	if overview.Config.AdminUsername != "ops-admin" {
		t.Fatalf("Expected admin username ops-admin, got %s", overview.Config.AdminUsername)
	}

	if overview.Library.TotalImages != 0 || overview.Library.TotalTags != 0 || overview.Library.TotalCollections != 0 {
		t.Fatalf("Expected empty library stats in fresh DB, got %+v", overview.Library)
	}
}

func insertTaskBatchForOverviewTest(t *testing.T, db *sql.DB, status string) int64 {
	t.Helper()

	result, err := db.Exec(
		`INSERT INTO task_batches (source_type, trigger_key, summary_label, status, total_images, new_images, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"import_scan",
		"overview-trigger",
		"overview batch",
		status,
		6,
		6,
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("Failed to insert task batch: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to fetch task batch id: %v", err)
	}
	return id
}

func insertImageForOverviewTest(t *testing.T, db *sql.DB, imageID int64, path, filename string) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO images (id, path, filename, source_root, thumbnail_small_url, thumbnail_large_url, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		imageID,
		path,
		filename,
		"/test",
		"thumb-small",
		"thumb-large",
		time.Now().UTC(),
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("Failed to insert image: %v", err)
	}
}

func insertPlatformTaskForOverviewTest(t *testing.T, db *sql.DB, batchID, imageID int64, status string) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO platform_tasks (batch_id, image_id, task_type, source_type, status, dedupe_key, image_version_key, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		batchID,
		imageID,
		"ai_tag_generation",
		"import_scan",
		status,
		"dedupe-"+status+"-"+time.Now().UTC().Format(time.RFC3339Nano),
		"image-version-"+time.Now().UTC().Format(time.RFC3339Nano),
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("Failed to insert platform task: %v", err)
	}
}

func strPtr(s string) *string {
	return &s
}
