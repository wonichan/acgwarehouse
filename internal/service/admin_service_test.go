package service_test

import (
	"context"
	"database/sql"
	"os"
	"strings"
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
	tempFile, err := os.CreateTemp("", "admin-service-*.db")
	if err != nil {
		t.Fatalf("Failed to create temporary database file: %v", err)
	}
	dbPath := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temporary database file handle: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(dbPath)
	})

	db, err := sql.Open("sqlite3", dbPath)
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

func TestAdminService_RetryFailedBatchTasks_CreatesNewBatchFromFailedTasks(t *testing.T) {
	db := newTestAdminDB(t)
	defer db.Close()

	jobRepo := repository.NewJobRepository(db)
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	collectionRepo := repository.NewCollectionRepository(db)
	jobManager := worker.NewManager(jobRepo)
	taskBatchRepo := repository.NewTaskBatchRepository(db)
	taskRepo := repository.NewPlatformTaskRepository(db)
	taskPlatformSvc := service.NewTaskPlatformService(taskBatchRepo, taskRepo, jobRepo)

	cfg := &config.Config{}
	svc := service.NewAdminService(cfg, jobRepo, imageRepo, tagRepo, collectionRepo, jobManager, taskPlatformSvc, taskBatchRepo, taskRepo)

	ctx := context.Background()
	image := &domain.Image{Path: "/test/retry-batch.png", Filename: "retry-batch.png", SourceRoot: "/test", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if _, err := imageRepo.SaveImage(image); err != nil {
		t.Fatalf("SaveImage() error = %v", err)
	}

	plan, err := taskPlatformSvc.PlanBatch(ctx, service.TaskPlatformPlanRequest{
		SourceType:   domain.TaskBatchSourceImportScan,
		SummaryLabel: "retry batch seed",
		SourceRoots:  []string{"/test"},
		TaskTypes:    []string{domain.PlatformTaskTypeThumbnailGenerate, domain.PlatformTaskTypeAITagGeneration},
		Items: []service.TaskPlatformPlanItem{{
			ImageID:          image.ID,
			ImageVersionKey:  service.BuildImageVersionKey(image),
			SourceDescriptor: image.Path,
		}},
	})
	if err != nil {
		t.Fatalf("PlanBatch() error = %v", err)
	}
	if len(plan.CreatedTasks) != 2 {
		t.Fatalf("len(CreatedTasks) = %d, want 2", len(plan.CreatedTasks))
	}

	for i := range plan.CreatedTasks {
		payload := `{"image_id":1,"path":"/test/retry-batch.png","filename":"retry-batch.png"}`
		job, err := taskPlatformSvc.QueueTask(ctx, &plan.CreatedTasks[i], plan.CreatedTasks[i].TaskType, payload)
		if err != nil {
			t.Fatalf("QueueTask(%d) error = %v", i, err)
		}
		if err := jobRepo.UpdateStatus(job.ID, "failed", strPtr("queue failed")); err != nil {
			t.Fatalf("UpdateStatus(failed job) error = %v", err)
		}
	}

	failedTask := plan.CreatedTasks[0]
	failedTask.Status = domain.PlatformTaskStatusFailed
	failedTask.ErrorSummary = strPtr("thumbnail failed")
	failedFinished := time.Now()
	failedTask.FinishedAt = &failedFinished
	if err := taskRepo.Update(ctx, &failedTask); err != nil {
		t.Fatalf("Update(failed task) error = %v", err)
	}

	completedTask := plan.CreatedTasks[1]
	completedTask.Status = domain.PlatformTaskStatusCompleted
	completedFinished := time.Now()
	completedTask.FinishedAt = &completedFinished
	if err := taskRepo.Update(ctx, &completedTask); err != nil {
		t.Fatalf("Update(completed task) error = %v", err)
	}

	result, err := svc.RetryFailedBatchTasks(ctx, plan.Batch.ID)
	if err != nil {
		t.Fatalf("RetryFailedBatchTasks() error = %v", err)
	}

	if result.Batch == nil {
		t.Fatal("expected retry batch result")
	}
	if result.Batch.ID == plan.Batch.ID {
		t.Fatalf("retry batch ID = %d, want new batch", result.Batch.ID)
	}
	if result.Batch.SourceType != domain.TaskBatchSourceRetry {
		t.Fatalf("retry batch source type = %q, want %q", result.Batch.SourceType, domain.TaskBatchSourceRetry)
	}
	if len(result.CreatedTasks) != 1 {
		t.Fatalf("len(CreatedTasks) = %d, want 1", len(result.CreatedTasks))
	}
	if result.CreatedTasks[0].Status != domain.PlatformTaskStatusQueued {
		t.Fatalf("retry task status = %q, want %q", result.CreatedTasks[0].Status, domain.PlatformTaskStatusQueued)
	}

	refreshedBatch, err := taskBatchRepo.FindByID(ctx, result.Batch.ID)
	if err != nil {
		t.Fatalf("FindByID(retry batch) error = %v", err)
	}
	if refreshedBatch.Status != domain.TaskBatchStatusRunning && refreshedBatch.Status != domain.TaskBatchStatusPending {
		t.Fatalf("retry batch status = %q, want pending or running", refreshedBatch.Status)
	}

	reloadedFailed, err := taskRepo.FindByID(ctx, failedTask.ID)
	if err != nil {
		t.Fatalf("FindByID(failed task) error = %v", err)
	}
	if reloadedFailed.Status != domain.PlatformTaskStatusFailed {
		t.Fatalf("failed task status = %q, want %q", reloadedFailed.Status, domain.PlatformTaskStatusFailed)
	}

	reloadedCompleted, err := taskRepo.FindByID(ctx, completedTask.ID)
	if err != nil {
		t.Fatalf("FindByID(completed task) error = %v", err)
	}
	if reloadedCompleted.Status != domain.PlatformTaskStatusCompleted {
		t.Fatalf("completed task status = %q, want %q", reloadedCompleted.Status, domain.PlatformTaskStatusCompleted)
	}
}

func TestAdminService_RetryFailedTask_RejectsNonFailedTasks(t *testing.T) {
	db := newTestAdminDB(t)
	defer db.Close()

	jobRepo := repository.NewJobRepository(db)
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	collectionRepo := repository.NewCollectionRepository(db)
	jobManager := worker.NewManager(jobRepo)
	taskBatchRepo := repository.NewTaskBatchRepository(db)
	taskRepo := repository.NewPlatformTaskRepository(db)
	taskPlatformSvc := service.NewTaskPlatformService(taskBatchRepo, taskRepo, jobRepo)

	svc := service.NewAdminService(&config.Config{}, jobRepo, imageRepo, tagRepo, collectionRepo, jobManager, taskPlatformSvc, taskBatchRepo, taskRepo)

	ctx := context.Background()
	image := &domain.Image{Path: "/test/retry-task.png", Filename: "retry-task.png", SourceRoot: "/test", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if _, err := imageRepo.SaveImage(image); err != nil {
		t.Fatalf("SaveImage() error = %v", err)
	}
	plan, err := taskPlatformSvc.PlanBatch(ctx, service.TaskPlatformPlanRequest{
		SourceType:   domain.TaskBatchSourceImportScan,
		SummaryLabel: "retry task seed",
		TaskTypes:    []string{domain.PlatformTaskTypeThumbnailGenerate},
		Items: []service.TaskPlatformPlanItem{{
			ImageID:          image.ID,
			ImageVersionKey:  service.BuildImageVersionKey(image),
			SourceDescriptor: image.Path,
		}},
	})
	if err != nil {
		t.Fatalf("PlanBatch() error = %v", err)
	}
	payload := `{"image_id":1,"path":"/test/retry-task.png","filename":"retry-task.png"}`
	job, err := taskPlatformSvc.QueueTask(ctx, &plan.CreatedTasks[0], plan.CreatedTasks[0].TaskType, payload)
	if err != nil {
		t.Fatalf("QueueTask() error = %v", err)
	}
	if err := jobRepo.UpdateStatus(job.ID, "failed", strPtr("queue failed")); err != nil {
		t.Fatalf("UpdateStatus(failed job) error = %v", err)
	}
	failedTask := plan.CreatedTasks[0]
	failedTask.Status = domain.PlatformTaskStatusCompleted
	finished := time.Now()
	failedTask.FinishedAt = &finished
	if err := taskRepo.Update(ctx, &failedTask); err != nil {
		t.Fatalf("Update(completed task) error = %v", err)
	}

	_, err = svc.RetryFailedTask(ctx, failedTask.ID)
	if err == nil {
		t.Fatal("expected error retrying non-failed task")
	}
	if !strings.Contains(err.Error(), "failed") {
		t.Fatalf("expected failed-only retry error, got %v", err)
	}
}

func TestAdminService_RetryFailedTask_CreatesNewBatchForSingleFailedTask(t *testing.T) {
	db := newTestAdminDB(t)
	defer db.Close()

	jobRepo := repository.NewJobRepository(db)
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	collectionRepo := repository.NewCollectionRepository(db)
	jobManager := worker.NewManager(jobRepo)
	taskBatchRepo := repository.NewTaskBatchRepository(db)
	taskRepo := repository.NewPlatformTaskRepository(db)
	taskPlatformSvc := service.NewTaskPlatformService(taskBatchRepo, taskRepo, jobRepo)

	svc := service.NewAdminService(&config.Config{}, jobRepo, imageRepo, tagRepo, collectionRepo, jobManager, taskPlatformSvc, taskBatchRepo, taskRepo)

	ctx := context.Background()
	image := &domain.Image{Path: "/test/retry-task-failed.png", Filename: "retry-task-failed.png", SourceRoot: "/test", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if _, err := imageRepo.SaveImage(image); err != nil {
		t.Fatalf("SaveImage() error = %v", err)
	}
	plan, err := taskPlatformSvc.PlanBatch(ctx, service.TaskPlatformPlanRequest{
		SourceType:   domain.TaskBatchSourceImportScan,
		SummaryLabel: "retry failed task seed",
		TaskTypes:    []string{domain.PlatformTaskTypeThumbnailGenerate},
		Items: []service.TaskPlatformPlanItem{{
			ImageID:          image.ID,
			ImageVersionKey:  service.BuildImageVersionKey(image),
			SourceDescriptor: image.Path,
		}},
	})
	if err != nil {
		t.Fatalf("PlanBatch() error = %v", err)
	}
	payload := `{"image_id":1,"path":"/test/retry-task-failed.png","filename":"retry-task-failed.png"}`
	job, err := taskPlatformSvc.QueueTask(ctx, &plan.CreatedTasks[0], plan.CreatedTasks[0].TaskType, payload)
	if err != nil {
		t.Fatalf("QueueTask() error = %v", err)
	}
	if err := jobRepo.UpdateStatus(job.ID, "failed", strPtr("queue failed")); err != nil {
		t.Fatalf("UpdateStatus(failed job) error = %v", err)
	}
	failedTask := plan.CreatedTasks[0]
	failedTask.Status = domain.PlatformTaskStatusFailed
	failedError := "thumbnail failed"
	failedTask.ErrorSummary = &failedError
	finished := time.Now()
	failedTask.FinishedAt = &finished
	if err := taskRepo.Update(ctx, &failedTask); err != nil {
		t.Fatalf("Update(failed task) error = %v", err)
	}

	result, err := svc.RetryFailedTask(ctx, failedTask.ID)
	if err != nil {
		t.Fatalf("RetryFailedTask() error = %v", err)
	}
	if result.Batch == nil {
		t.Fatal("expected retry batch result")
	}
	if result.Batch.ID == plan.Batch.ID {
		t.Fatalf("retry batch ID = %d, want a new batch", result.Batch.ID)
	}
	if len(result.CreatedTasks) != 1 {
		t.Fatalf("len(CreatedTasks) = %d, want 1", len(result.CreatedTasks))
	}
	if result.CreatedTasks[0].TaskType != domain.PlatformTaskTypeThumbnailGenerate {
		t.Fatalf("retry task type = %q, want %q", result.CreatedTasks[0].TaskType, domain.PlatformTaskTypeThumbnailGenerate)
	}
	if result.CreatedTasks[0].Status != domain.PlatformTaskStatusQueued {
		t.Fatalf("retry task status = %q, want %q", result.CreatedTasks[0].Status, domain.PlatformTaskStatusQueued)
	}
	if result.Batch.SourceType != domain.TaskBatchSourceRetry {
		t.Fatalf("retry batch source type = %q, want %q", result.Batch.SourceType, domain.TaskBatchSourceRetry)
	}
}

func TestAdminService_RetryFailedJobs_IncludesPartialFailedBatches(t *testing.T) {
	db := newTestAdminDB(t)
	defer db.Close()

	jobRepo := repository.NewJobRepository(db)
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	collectionRepo := repository.NewCollectionRepository(db)
	jobManager := worker.NewManager(jobRepo)
	taskBatchRepo := repository.NewTaskBatchRepository(db)
	taskRepo := repository.NewPlatformTaskRepository(db)
	taskPlatformSvc := service.NewTaskPlatformService(taskBatchRepo, taskRepo, jobRepo)
	taskReadSvc := service.NewTaskReadService(repository.NewTaskBatchReadRepository(db))

	svc := service.NewAdminService(&config.Config{}, jobRepo, imageRepo, tagRepo, collectionRepo, jobManager, taskReadSvc, taskBatchRepo, taskRepo)
	ctx := context.Background()

	failedBatchTask := seedRetryableFailedTask(t, ctx, imageRepo, jobRepo, taskRepo, taskBatchRepo, taskPlatformSvc, "/test/retry-all-failed.png", []string{domain.PlatformTaskStatusFailed})
	partialFailedTask := seedRetryableFailedTask(t, ctx, imageRepo, jobRepo, taskRepo, taskBatchRepo, taskPlatformSvc, "/test/retry-all-partial.png", []string{domain.PlatformTaskStatusFailed, domain.PlatformTaskStatusCompleted})

	count, err := svc.RetryFailedJobs(ctx)
	if err != nil {
		t.Fatalf("RetryFailedJobs() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("RetryFailedJobs() count = %d, want 2", count)
	}

	failedRetries, err := taskRepo.ListByImageAndTypes(ctx, failedBatchTask.ImageID, []string{failedBatchTask.TaskType})
	if err != nil {
		t.Fatalf("ListByImageAndTypes(failed batch task) error = %v", err)
	}
	if countTasksByStatus(failedRetries, domain.PlatformTaskStatusQueued) != 1 {
		t.Fatalf("queued retries for failed batch = %d, want 1", countTasksByStatus(failedRetries, domain.PlatformTaskStatusQueued))
	}

	partialRetries, err := taskRepo.ListByImageAndTypes(ctx, partialFailedTask.ImageID, []string{partialFailedTask.TaskType})
	if err != nil {
		t.Fatalf("ListByImageAndTypes(partial failed task) error = %v", err)
	}
	if countTasksByStatus(partialRetries, domain.PlatformTaskStatusQueued) != 1 {
		t.Fatalf("queued retries for partial failed batch = %d, want 1", countTasksByStatus(partialRetries, domain.PlatformTaskStatusQueued))
	}
}

func TestAdminService_RetryFailedJobs_SkipsChainsWithCompletedRetry(t *testing.T) {
	db := newTestAdminDB(t)
	defer db.Close()

	jobRepo := repository.NewJobRepository(db)
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	collectionRepo := repository.NewCollectionRepository(db)
	jobManager := worker.NewManager(jobRepo)
	taskBatchRepo := repository.NewTaskBatchRepository(db)
	taskRepo := repository.NewPlatformTaskRepository(db)
	taskPlatformSvc := service.NewTaskPlatformService(taskBatchRepo, taskRepo, jobRepo)
	taskReadSvc := service.NewTaskReadService(repository.NewTaskBatchReadRepository(db))

	svc := service.NewAdminService(&config.Config{}, jobRepo, imageRepo, tagRepo, collectionRepo, jobManager, taskReadSvc, taskBatchRepo, taskRepo)
	ctx := context.Background()

	failedTask := seedRetryableFailedTask(t, ctx, imageRepo, jobRepo, taskRepo, taskBatchRepo, taskPlatformSvc, "/test/retry-completed-chain.png", []string{domain.PlatformTaskStatusFailed})

	firstRetry, err := svc.RetryFailedTask(ctx, failedTask.ID)
	if err != nil {
		t.Fatalf("RetryFailedTask() error = %v", err)
	}
	if len(firstRetry.CreatedTasks) != 1 {
		t.Fatalf("len(CreatedTasks) = %d, want 1", len(firstRetry.CreatedTasks))
	}

	completedRetry := firstRetry.CreatedTasks[0]
	completedRetry.Status = domain.PlatformTaskStatusCompleted
	finished := time.Now()
	completedRetry.FinishedAt = &finished
	if err := taskRepo.Update(ctx, &completedRetry); err != nil {
		t.Fatalf("Update(completed retry task) error = %v", err)
	}
	if _, err := taskBatchRepo.RefreshStatus(ctx, completedRetry.BatchID); err != nil {
		t.Fatalf("RefreshStatus(completed retry batch) error = %v", err)
	}

	retryCount, err := svc.RetryFailedJobs(ctx)
	if err != nil {
		t.Fatalf("RetryFailedJobs() error = %v", err)
	}
	if retryCount != 0 {
		t.Fatalf("RetryFailedJobs() count = %d, want 0", retryCount)
	}

	history, err := taskRepo.ListByImageAndTypes(ctx, failedTask.ImageID, []string{failedTask.TaskType})
	if err != nil {
		t.Fatalf("ListByImageAndTypes() error = %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("len(history) = %d, want 2", len(history))
	}
	if countTasksByStatus(history, domain.PlatformTaskStatusQueued) != 0 {
		t.Fatalf("queued retries after completed retry = %d, want 0", countTasksByStatus(history, domain.PlatformTaskStatusQueued))
	}
}

func TestAdminService_RetryFailedTask_StopsAfterFiveFailures(t *testing.T) {
	db := newTestAdminDB(t)
	defer db.Close()

	jobRepo := repository.NewJobRepository(db)
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	collectionRepo := repository.NewCollectionRepository(db)
	jobManager := worker.NewManager(jobRepo)
	taskBatchRepo := repository.NewTaskBatchRepository(db)
	taskRepo := repository.NewPlatformTaskRepository(db)
	taskPlatformSvc := service.NewTaskPlatformService(taskBatchRepo, taskRepo, jobRepo)

	svc := service.NewAdminService(&config.Config{}, jobRepo, imageRepo, tagRepo, collectionRepo, jobManager, taskPlatformSvc, taskBatchRepo, taskRepo)
	ctx := context.Background()

	failedTask := seedRetryableFailedTask(t, ctx, imageRepo, jobRepo, taskRepo, taskBatchRepo, taskPlatformSvc, "/test/retry-limit.png", []string{domain.PlatformTaskStatusFailed})
	retryBatch := &domain.TaskBatch{
		SourceType:   domain.TaskBatchSourceRetry,
		SummaryLabel: "historical retries",
		Status:       domain.TaskBatchStatusFailed,
		TotalImages:  4,
		CreatedAt:    time.Now(),
	}
	if err := taskBatchRepo.Create(ctx, retryBatch); err != nil {
		t.Fatalf("Create(retry batch) error = %v", err)
	}
	for range 4 {
		historicalTask := domain.PlatformTask{
			BatchID:         retryBatch.ID,
			ImageID:         failedTask.ImageID,
			TaskType:        failedTask.TaskType,
			SourceType:      domain.TaskBatchSourceRetry,
			Status:          domain.PlatformTaskStatusFailed,
			DedupeKey:       failedTask.DedupeKey,
			ImageVersionKey: failedTask.ImageVersionKey,
			CreatedAt:       time.Now(),
		}
		if err := taskRepo.Create(ctx, &historicalTask); err != nil {
			t.Fatalf("Create(historical failed task) error = %v", err)
		}
	}

	_, err := svc.RetryFailedTask(ctx, failedTask.ID)
	if err == nil {
		t.Fatal("expected retry limit error")
	}
	if !strings.Contains(err.Error(), "retry limit") {
		t.Fatalf("expected retry limit error, got %v", err)
	}

	allTasks, err := taskRepo.ListByImageAndTypes(ctx, failedTask.ImageID, []string{failedTask.TaskType})
	if err != nil {
		t.Fatalf("ListByImageAndTypes() error = %v", err)
	}
	if countTasksByStatus(allTasks, domain.PlatformTaskStatusQueued) != 0 {
		t.Fatalf("queued retries = %d, want 0", countTasksByStatus(allTasks, domain.PlatformTaskStatusQueued))
	}
}

func seedRetryableFailedTask(
	t *testing.T,
	ctx context.Context,
	imageRepo repository.ImageRepository,
	jobRepo repository.JobRepository,
	taskRepo repository.PlatformTaskRepository,
	taskBatchRepo repository.TaskBatchRepository,
	taskPlatformSvc *service.TaskPlatformService,
	path string,
	taskStatuses []string,
) domain.PlatformTask {
	t.Helper()

	image := &domain.Image{Path: path, Filename: path[strings.LastIndex(path, "/")+1:], SourceRoot: "/test", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if _, err := imageRepo.SaveImage(image); err != nil {
		t.Fatalf("SaveImage() error = %v", err)
	}

	taskTypes := make([]string, 0, len(taskStatuses))
	for index := range taskStatuses {
		if index == 0 {
			taskTypes = append(taskTypes, domain.PlatformTaskTypeThumbnailGenerate)
			continue
		}
		taskTypes = append(taskTypes, domain.PlatformTaskTypeAITagGeneration)
	}

	plan, err := taskPlatformSvc.PlanBatch(ctx, service.TaskPlatformPlanRequest{
		SourceType:   domain.TaskBatchSourceImportScan,
		SummaryLabel: "retry seed",
		SourceRoots:  []string{"/test"},
		TaskTypes:    taskTypes,
		Items: []service.TaskPlatformPlanItem{{
			ImageID:          image.ID,
			ImageVersionKey:  service.BuildImageVersionKey(image),
			SourceDescriptor: image.Path,
		}},
	})
	if err != nil {
		t.Fatalf("PlanBatch() error = %v", err)
	}

	for i := range plan.CreatedTasks {
		payload := `{"image_id":1,"path":"` + image.Path + `","filename":"` + image.Filename + `"}`
		job, err := taskPlatformSvc.QueueTask(ctx, &plan.CreatedTasks[i], plan.CreatedTasks[i].TaskType, payload)
		if err != nil {
			t.Fatalf("QueueTask(%d) error = %v", i, err)
		}
		status := "finished"
		errorSummary := (*string)(nil)
		if taskStatuses[i] == domain.PlatformTaskStatusFailed {
			status = "failed"
			errorSummary = strPtr("queue failed")
		}
		if err := jobRepo.UpdateStatus(job.ID, status, errorSummary); err != nil {
			t.Fatalf("UpdateStatus(%s job) error = %v", status, err)
		}

		updatedTask := plan.CreatedTasks[i]
		updatedTask.Status = taskStatuses[i]
		finishedAt := time.Now()
		updatedTask.FinishedAt = &finishedAt
		if taskStatuses[i] == domain.PlatformTaskStatusFailed {
			updatedTask.ErrorSummary = strPtr("task failed")
		}
		if err := taskRepo.Update(ctx, &updatedTask); err != nil {
			t.Fatalf("Update(task status=%s) error = %v", taskStatuses[i], err)
		}
		plan.CreatedTasks[i] = updatedTask
	}

	if _, err := taskBatchRepo.RefreshStatus(ctx, plan.Batch.ID); err != nil {
		t.Fatalf("RefreshStatus() error = %v", err)
	}

	return plan.CreatedTasks[0]
}

func countTasksByStatus(tasks []domain.PlatformTask, status string) int {
	count := 0
	for _, task := range tasks {
		if task.Status == status {
			count++
		}
	}
	return count
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
	taskBatchRepo := repository.NewTaskBatchRepository(db)
	taskRepo := repository.NewPlatformTaskRepository(db)
	cfg := &config.Config{}
	svc := service.NewAdminService(cfg, jobRepo, imageRepo, tagRepo, collectionRepo, jobManager, taskReadSvc, taskBatchRepo, taskRepo)

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
		svc := service.NewAdminService(cfg, jobRepo, imageRepo, tagRepo, collectionRepo, jobManager, repository.NewTaskBatchRepository(db), repository.NewPlatformTaskRepository(db))

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
			repository.NewTaskBatchRepository(db),
			repository.NewPlatformTaskRepository(db),
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

	svc := service.NewAdminService(cfg, jobRepo, imageRepo, tagRepo, collectionRepo, jobManager, repository.NewTaskBatchRepository(db), repository.NewPlatformTaskRepository(db))

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

func TestAdminService_GetTaskPlatformOverview_IncludesSidecarDiagnostics(t *testing.T) {
	db := newTestAdminDB(t)
	defer db.Close()

	jobRepo := repository.NewJobRepository(db)
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	collectionRepo := repository.NewCollectionRepository(db)
	jobManager := worker.NewManager(jobRepo)

	cfg := &config.Config{}
	svc := service.NewAdminService(cfg, jobRepo, imageRepo, tagRepo, collectionRepo, jobManager)

	overview, err := svc.GetTaskPlatformOverview(context.Background())
	if err != nil {
		t.Fatalf("GetTaskPlatformOverview() error = %v", err)
	}

	if overview.Sidecar.State == "" {
		t.Fatalf("expected sidecar.state to be populated")
	}
	if overview.Sidecar.LastProbeAt == "" {
		t.Fatalf("expected sidecar.last_probe_at to be populated")
	}
	if overview.Sidecar.LastProbeResult == "" {
		t.Fatalf("expected sidecar.last_probe_result to be populated")
	}
	if overview.Sidecar.LastErrorSummary == "" {
		t.Fatalf("expected sidecar.last_error_summary to be populated")
	}
}

func TestAdminService_ClearQueueAndCancelControls(t *testing.T) {
	db := newTestAdminDB(t)
	defer db.Close()

	jobRepo := repository.NewJobRepository(db)
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	collectionRepo := repository.NewCollectionRepository(db)
	jobManager := worker.NewManager(jobRepo)
	taskBatchRepo := repository.NewTaskBatchRepository(db)
	taskRepo := repository.NewPlatformTaskRepository(db)
	taskReadSvc := service.NewTaskReadService(repository.NewTaskBatchReadRepository(db))
	cfg := &config.Config{}

	ctx := context.Background()
	image := &domain.Image{Path: "/test/cancel.png", Filename: "cancel.png", SourceRoot: "/test", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if _, err := imageRepo.SaveImage(image); err != nil {
		t.Fatalf("SaveImage() error = %v", err)
	}
	batch := &domain.TaskBatch{SourceType: domain.TaskBatchSourceImportScan, SummaryLabel: "admin controls", Status: domain.TaskBatchStatusPending, CreatedAt: time.Now()}
	if err := taskBatchRepo.Create(ctx, batch); err != nil {
		t.Fatalf("Create(batch) error = %v", err)
	}
	makeTask := func(status string) *domain.PlatformTask {
		task := &domain.PlatformTask{
			BatchID:         batch.ID,
			ImageID:         image.ID,
			TaskType:        domain.PlatformTaskTypeThumbnailGenerate,
			SourceType:      domain.TaskBatchSourceImportScan,
			Status:          status,
			ImageVersionKey: "test-version",
			DedupeKey:       "test-version:" + status,
			CreatedAt:       time.Now(),
		}
		if status == domain.PlatformTaskStatusRunning || status == domain.PlatformTaskStatusCompleted {
			now := time.Now()
			task.StartedAt = &now
		}
		if status == domain.PlatformTaskStatusCompleted || status == domain.PlatformTaskStatusCancelled {
			now := time.Now()
			task.FinishedAt = &now
		}
		if err := taskRepo.Create(ctx, task); err != nil {
			t.Fatalf("Create(task %s) error = %v", status, err)
		}
		return task
	}
	pendingTask := makeTask(domain.PlatformTaskStatusPending)
	queuedTask := makeTask(domain.PlatformTaskStatusQueued)
	runningTask := makeTask(domain.PlatformTaskStatusRunning)
	_ = makeTask(domain.PlatformTaskStatusCompleted)
	if err := taskRepo.SetLatestAsyncJob(ctx, queuedTask.ID, nil); err != nil {
		t.Fatalf("SetLatestAsyncJob() error = %v", err)
	}
	if err := taskRepo.SetLatestAsyncJob(ctx, runningTask.ID, nil); err != nil {
		t.Fatalf("SetLatestAsyncJob(running) error = %v", err)
	}

	svc := service.NewAdminService(cfg, jobRepo, imageRepo, tagRepo, collectionRepo, jobManager, taskReadSvc, taskBatchRepo, taskRepo)

	cleared, err := svc.ClearTaskQueue(ctx)
	if err != nil {
		t.Fatalf("ClearTaskQueue() error = %v", err)
	}
	if cleared != 2 {
		t.Fatalf("ClearTaskQueue() = %d, want 2", cleared)
	}
	clearedPending, _ := taskRepo.FindByID(ctx, pendingTask.ID)
	if clearedPending.Status != domain.PlatformTaskStatusCancelled {
		t.Fatalf("pending task status = %q, want %q", clearedPending.Status, domain.PlatformTaskStatusCancelled)
	}
	clearedQueued, _ := taskRepo.FindByID(ctx, queuedTask.ID)
	if clearedQueued.Status != domain.PlatformTaskStatusCancelled {
		t.Fatalf("queued task status = %q, want %q", clearedQueued.Status, domain.PlatformTaskStatusCancelled)
	}
	stillRunning, _ := taskRepo.FindByID(ctx, runningTask.ID)
	if stillRunning.Status != domain.PlatformTaskStatusRunning {
		t.Fatalf("running task status = %q, want %q", stillRunning.Status, domain.PlatformTaskStatusRunning)
	}

	secondBatch := &domain.TaskBatch{SourceType: domain.TaskBatchSourceImportScan, SummaryLabel: "batch cancel", Status: domain.TaskBatchStatusPending, CreatedAt: time.Now()}
	if err := taskBatchRepo.Create(ctx, secondBatch); err != nil {
		t.Fatalf("Create(second batch) error = %v", err)
	}
	batchTaskOne := &domain.PlatformTask{BatchID: secondBatch.ID, ImageID: image.ID, TaskType: domain.PlatformTaskTypeThumbnailGenerate, SourceType: domain.TaskBatchSourceImportScan, Status: domain.PlatformTaskStatusPending, ImageVersionKey: "test-version-2", DedupeKey: "test-version-2:pending", CreatedAt: time.Now()}
	batchTaskTwo := &domain.PlatformTask{BatchID: secondBatch.ID, ImageID: image.ID, TaskType: domain.PlatformTaskTypeThumbnailGenerate, SourceType: domain.TaskBatchSourceImportScan, Status: domain.PlatformTaskStatusRunning, ImageVersionKey: "test-version-2", DedupeKey: "test-version-2:running", CreatedAt: time.Now()}
	started := time.Now()
	batchTaskTwo.StartedAt = &started
	if err := taskRepo.Create(ctx, batchTaskOne); err != nil {
		t.Fatalf("Create(batchTaskOne) error = %v", err)
	}
	if err := taskRepo.Create(ctx, batchTaskTwo); err != nil {
		t.Fatalf("Create(batchTaskTwo) error = %v", err)
	}

	cancelled, err := svc.CancelTaskBatch(ctx, secondBatch.ID)
	if err != nil {
		t.Fatalf("CancelTaskBatch() error = %v", err)
	}
	if cancelled != 2 {
		t.Fatalf("CancelTaskBatch() = %d, want 2", cancelled)
	}
	refreshedBatch, err := taskBatchRepo.FindByID(ctx, secondBatch.ID)
	if err != nil {
		t.Fatalf("FindByID(second batch) error = %v", err)
	}
	if refreshedBatch.Status != domain.TaskBatchStatusCancelled {
		t.Fatalf("batch status = %q, want %q", refreshedBatch.Status, domain.TaskBatchStatusCancelled)
	}

	thirdBatch := &domain.TaskBatch{SourceType: domain.TaskBatchSourceImportScan, SummaryLabel: "single cancel", Status: domain.TaskBatchStatusPending, CreatedAt: time.Now()}
	if err := taskBatchRepo.Create(ctx, thirdBatch); err != nil {
		t.Fatalf("Create(third batch) error = %v", err)
	}
	singleTask := &domain.PlatformTask{BatchID: thirdBatch.ID, ImageID: image.ID, TaskType: domain.PlatformTaskTypeThumbnailGenerate, SourceType: domain.TaskBatchSourceImportScan, Status: domain.PlatformTaskStatusPending, ImageVersionKey: "test-version-3", DedupeKey: "test-version-3:pending", CreatedAt: time.Now()}
	if err := taskRepo.Create(ctx, singleTask); err != nil {
		t.Fatalf("Create(single task) error = %v", err)
	}
	singleCancelCount, err := svc.CancelTask(ctx, singleTask.ID)
	if err != nil {
		t.Fatalf("CancelTask() error = %v", err)
	}
	if singleCancelCount != 1 {
		t.Fatalf("CancelTask() = %d, want 1", singleCancelCount)
	}
	reloadedSingle, err := taskRepo.FindByID(ctx, singleTask.ID)
	if err != nil {
		t.Fatalf("FindByID(single task) error = %v", err)
	}
	if reloadedSingle.Status != domain.PlatformTaskStatusCancelled {
		t.Fatalf("single task status = %q, want %q", reloadedSingle.Status, domain.PlatformTaskStatusCancelled)
	}
	if reloadedSingle.FinishedAt == nil {
		t.Fatal("expected finished_at on cancelled task")
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
