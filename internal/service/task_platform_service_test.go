package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

func TestTaskPlatformServiceSkipsDuplicateTaskForExistingDedupeKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	env := newTaskPlatformServiceTestEnv(t)
	image := saveTaskPlatformServiceImage(t, env.db, "duplicate.png")
	seedTaskPlatformBatchAndTask(t, env, image.ID, domain.PlatformTaskTypeThumbnailGenerate, domain.PlatformTaskStatusCompleted, "image:duplicate:v1")

	result, err := env.service.PlanBatch(ctx, TaskPlatformPlanRequest{
		SourceType:   domain.TaskBatchSourceImportScan,
		SummaryLabel: "duplicate protection",
		SourceRoots:  []string{"/task-platform"},
		TaskTypes:    []string{domain.PlatformTaskTypeThumbnailGenerate},
		Items: []TaskPlatformPlanItem{{
			ImageID:          image.ID,
			ImageVersionKey:  "image:duplicate:v1",
			SourceDescriptor: image.Path,
		}},
	})
	if err != nil {
		t.Fatalf("PlanBatch() error = %v", err)
	}
	if len(result.CreatedTasks) != 0 {
		t.Fatalf("len(CreatedTasks) = %d, want 0", len(result.CreatedTasks))
	}
	if result.Batch.SkippedDuplicateTasks != 1 {
		t.Fatalf("SkippedDuplicateTasks = %d, want 1", result.Batch.SkippedDuplicateTasks)
	}
}

func TestTaskPlatformServiceBackfillsOnlyMissingTaskTypes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	env := newTaskPlatformServiceTestEnv(t)
	image := saveTaskPlatformServiceImage(t, env.db, "backfill.png")
	seedTaskPlatformBatchAndTask(t, env, image.ID, domain.PlatformTaskTypeThumbnailGenerate, domain.PlatformTaskStatusCompleted, "image:backfill:v1")

	result, err := env.service.PlanBatch(ctx, TaskPlatformPlanRequest{
		SourceType:   domain.TaskBatchSourceImportScan,
		SummaryLabel: "backfill missing task type",
		SourceRoots:  []string{"/task-platform"},
		TaskTypes: []string{
			domain.PlatformTaskTypeThumbnailGenerate,
			domain.PlatformTaskTypeAITagGeneration,
		},
		Items: []TaskPlatformPlanItem{{
			ImageID:          image.ID,
			ImageVersionKey:  "image:backfill:v1",
			SourceDescriptor: image.Path,
		}},
	})
	if err != nil {
		t.Fatalf("PlanBatch() error = %v", err)
	}
	if len(result.CreatedTasks) != 1 {
		t.Fatalf("len(CreatedTasks) = %d, want 1", len(result.CreatedTasks))
	}
	if result.CreatedTasks[0].TaskType != domain.PlatformTaskTypeAITagGeneration {
		t.Fatalf("created task type = %q, want %q", result.CreatedTasks[0].TaskType, domain.PlatformTaskTypeAITagGeneration)
	}
	if result.Batch.SkippedDuplicateTasks != 1 {
		t.Fatalf("SkippedDuplicateTasks = %d, want 1 for completed thumbnail dedupe", result.Batch.SkippedDuplicateTasks)
	}
}

func TestTaskPlatformServiceRefreshBatchStatusAggregatesRunningAndPartialFailed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	env := newTaskPlatformServiceTestEnv(t)
	image := saveTaskPlatformServiceImage(t, env.db, "lifecycle.png")

	result, err := env.service.PlanBatch(ctx, TaskPlatformPlanRequest{
		SourceType:   domain.TaskBatchSourceImportScan,
		SummaryLabel: "lifecycle aggregate",
		SourceRoots:  []string{"/task-platform"},
		TaskTypes: []string{
			domain.PlatformTaskTypeThumbnailGenerate,
			domain.PlatformTaskTypeAITagGeneration,
		},
		Items: []TaskPlatformPlanItem{{
			ImageID:          image.ID,
			ImageVersionKey:  "image:lifecycle:v1",
			SourceDescriptor: image.Path,
		}},
	})
	if err != nil {
		t.Fatalf("PlanBatch() error = %v", err)
	}

	if len(result.CreatedTasks) != 2 {
		t.Fatalf("len(CreatedTasks) = %d, want 2", len(result.CreatedTasks))
	}

	result.CreatedTasks[0].Status = domain.PlatformTaskStatusRunning
	if err := env.taskRepo.Update(ctx, &result.CreatedTasks[0]); err != nil {
		t.Fatalf("Update(running) error = %v", err)
	}
	refreshed, err := env.service.RefreshBatchStatus(ctx, result.Batch.ID)
	if err != nil {
		t.Fatalf("RefreshBatchStatus(running) error = %v", err)
	}
	if refreshed.Status != domain.TaskBatchStatusRunning {
		t.Fatalf("running batch status = %q, want %q", refreshed.Status, domain.TaskBatchStatusRunning)
	}

	finishedAt := time.Now()
	result.CreatedTasks[0].Status = domain.PlatformTaskStatusCompleted
	result.CreatedTasks[0].FinishedAt = &finishedAt
	result.CreatedTasks[1].Status = domain.PlatformTaskStatusFailed
	result.CreatedTasks[1].FinishedAt = &finishedAt
	for i := range result.CreatedTasks {
		if err := env.taskRepo.Update(ctx, &result.CreatedTasks[i]); err != nil {
			t.Fatalf("Update(terminal %d) error = %v", i, err)
		}
	}

	refreshed, err = env.service.RefreshBatchStatus(ctx, result.Batch.ID)
	if err != nil {
		t.Fatalf("RefreshBatchStatus(partial_failed) error = %v", err)
	}
	if refreshed.Status != domain.TaskBatchStatusPartialFailed {
		t.Fatalf("partial_failed batch status = %q, want %q", refreshed.Status, domain.TaskBatchStatusPartialFailed)
	}
}

func TestTaskPlatformServiceQueueTaskLinksAsyncJobAndMarksTaskQueued(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	env := newTaskPlatformServiceTestEnv(t)
	image := saveTaskPlatformServiceImage(t, env.db, "queued.png")
	plan, err := env.service.PlanBatch(ctx, TaskPlatformPlanRequest{
		SourceType:   domain.TaskBatchSourceManualSingle,
		SummaryLabel: "queue ai task",
		TaskTypes:    []string{domain.PlatformTaskTypeAITagGeneration},
		Items: []TaskPlatformPlanItem{{
			ImageID:          image.ID,
			ImageVersionKey:  BuildImageVersionKey(image),
			SourceDescriptor: image.Path,
		}},
	})
	if err != nil {
		t.Fatalf("PlanBatch() error = %v", err)
	}
	if len(plan.CreatedTasks) != 1 {
		t.Fatalf("len(CreatedTasks) = %d, want 1", len(plan.CreatedTasks))
	}
	payload, err := json.Marshal(map[string]any{"image_id": image.ID, "path": image.Path})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	job, err := env.service.QueueTask(ctx, &plan.CreatedTasks[0], domain.PlatformTaskTypeAITagGeneration, string(payload))
	if err != nil {
		t.Fatalf("QueueTask() error = %v", err)
	}
	if job.PlatformTaskID == nil || *job.PlatformTaskID != plan.CreatedTasks[0].ID {
		t.Fatalf("job.PlatformTaskID = %+v, want %d", job.PlatformTaskID, plan.CreatedTasks[0].ID)
	}
	reloadedTask, err := env.taskRepo.FindByID(ctx, plan.CreatedTasks[0].ID)
	if err != nil {
		t.Fatalf("FindByID(task) error = %v", err)
	}
	if reloadedTask.Status != domain.PlatformTaskStatusQueued {
		t.Fatalf("task status = %q, want %q", reloadedTask.Status, domain.PlatformTaskStatusQueued)
	}
	if reloadedTask.LatestAsyncJobID == nil || *reloadedTask.LatestAsyncJobID != job.ID {
		t.Fatalf("LatestAsyncJobID = %+v, want %d", reloadedTask.LatestAsyncJobID, job.ID)
	}
}

func TestTaskPlatformServiceSyncsJobLifecycleBackToPlatformTask(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	env := newTaskPlatformServiceTestEnv(t)
	image := saveTaskPlatformServiceImage(t, env.db, "lifecycle-sync.png")
	plan, err := env.service.PlanBatch(ctx, TaskPlatformPlanRequest{
		SourceType:   domain.TaskBatchSourceImportScan,
		SummaryLabel: "sync lifecycle",
		TaskTypes:    []string{domain.PlatformTaskTypeThumbnailGenerate},
		Items: []TaskPlatformPlanItem{{
			ImageID:          image.ID,
			ImageVersionKey:  BuildImageVersionKey(image),
			SourceDescriptor: image.Path,
		}},
	})
	if err != nil {
		t.Fatalf("PlanBatch() error = %v", err)
	}
	payload := `{"image_id":1,"path":"/task-platform/lifecycle-sync.png","filename":"lifecycle-sync"}`
	job, err := env.service.QueueTask(ctx, &plan.CreatedTasks[0], domain.PlatformTaskTypeThumbnailGenerate, payload)
	if err != nil {
		t.Fatalf("QueueTask() error = %v", err)
	}
	if err := env.service.MarkJobRunning(ctx, job.ID); err != nil {
		t.Fatalf("MarkJobRunning() error = %v", err)
	}
	runningTask, err := env.taskRepo.FindByID(ctx, plan.CreatedTasks[0].ID)
	if err != nil {
		t.Fatalf("FindByID(running task) error = %v", err)
	}
	if runningTask.Status != domain.PlatformTaskStatusRunning {
		t.Fatalf("running task status = %q, want %q", runningTask.Status, domain.PlatformTaskStatusRunning)
	}
	if err := env.service.MarkJobCompleted(ctx, job.ID); err != nil {
		t.Fatalf("MarkJobCompleted() error = %v", err)
	}
	completedTask, err := env.taskRepo.FindByID(ctx, plan.CreatedTasks[0].ID)
	if err != nil {
		t.Fatalf("FindByID(completed task) error = %v", err)
	}
	if completedTask.Status != domain.PlatformTaskStatusCompleted {
		t.Fatalf("completed task status = %q, want %q", completedTask.Status, domain.PlatformTaskStatusCompleted)
	}
	completedBatch, err := env.batchRepo.FindByID(ctx, plan.Batch.ID)
	if err != nil {
		t.Fatalf("FindByID(completed batch) error = %v", err)
	}
	if completedBatch.Status != domain.TaskBatchStatusCompleted {
		t.Fatalf("completed batch status = %q, want %q", completedBatch.Status, domain.TaskBatchStatusCompleted)
	}

	failedPlan, err := env.service.PlanBatch(ctx, TaskPlatformPlanRequest{
		SourceType:   domain.TaskBatchSourceImportScan,
		SummaryLabel: "sync failure",
		TaskTypes:    []string{domain.PlatformTaskTypeThumbnailGenerate},
		Items: []TaskPlatformPlanItem{{
			ImageID:          image.ID,
			ImageVersionKey:  "image:lifecycle-sync:v2",
			SourceDescriptor: image.Path,
		}},
	})
	if err != nil {
		t.Fatalf("PlanBatch(failed) error = %v", err)
	}
	failedJob, err := env.service.QueueTask(ctx, &failedPlan.CreatedTasks[0], domain.PlatformTaskTypeThumbnailGenerate, payload)
	if err != nil {
		t.Fatalf("QueueTask(failed) error = %v", err)
	}
	if err := env.service.MarkJobFailed(ctx, failedJob.ID, "thumbnail failed"); err != nil {
		t.Fatalf("MarkJobFailed() error = %v", err)
	}
	failedTask, err := env.taskRepo.FindByID(ctx, failedPlan.CreatedTasks[0].ID)
	if err != nil {
		t.Fatalf("FindByID(failed task) error = %v", err)
	}
	if failedTask.Status != domain.PlatformTaskStatusFailed {
		t.Fatalf("failed task status = %q, want %q", failedTask.Status, domain.PlatformTaskStatusFailed)
	}
	if failedTask.ErrorSummary == nil || *failedTask.ErrorSummary != "thumbnail failed" {
		t.Fatalf("ErrorSummary = %+v, want thumbnail failed", failedTask.ErrorSummary)
	}
	failedBatch, err := env.batchRepo.FindByID(ctx, failedPlan.Batch.ID)
	if err != nil {
		t.Fatalf("FindByID(failed batch) error = %v", err)
	}
	if failedBatch.Status != domain.TaskBatchStatusFailed {
		t.Fatalf("failed batch status = %q, want %q", failedBatch.Status, domain.TaskBatchStatusFailed)
	}
}

type taskPlatformServiceTestEnv struct {
	db        *sql.DB
	service   *TaskPlatformService
	batchRepo repository.TaskBatchRepository
	taskRepo  repository.PlatformTaskRepository
	jobRepo   repository.JobRepository
}

func newTaskPlatformServiceTestEnv(t *testing.T) *taskPlatformServiceTestEnv {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "task-platform-service.db"))
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	batchRepo := repository.NewTaskBatchRepository(db)
	taskRepo := repository.NewPlatformTaskRepository(db)
	jobRepo := repository.NewJobRepository(db)

	return &taskPlatformServiceTestEnv{
		db:        db,
		service:   NewTaskPlatformService(batchRepo, taskRepo, jobRepo),
		batchRepo: batchRepo,
		taskRepo:  taskRepo,
		jobRepo:   jobRepo,
	}
}

func saveTaskPlatformServiceImage(t *testing.T, db *sql.DB, filename string) *domain.Image {
	t.Helper()

	now := time.Now()
	image := &domain.Image{
		Path:       "/task-platform/" + filename,
		Filename:   filename,
		SourceRoot: "/task-platform",
		FileSize:   512,
		Width:      128,
		Height:     128,
		Format:     "png",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if _, err := repository.NewImageRepository(db).SaveImage(image); err != nil {
		t.Fatalf("SaveImage(%s) error = %v", filename, err)
	}
	return image
}

func seedTaskPlatformBatchAndTask(t *testing.T, env *taskPlatformServiceTestEnv, imageID int64, taskType, status, versionKey string) {
	t.Helper()

	ctx := context.Background()
	batch := &domain.TaskBatch{
		SourceType:   domain.TaskBatchSourceImportScan,
		SummaryLabel: "seed batch",
		Status:       domain.TaskBatchStatusPending,
		CreatedAt:    time.Now(),
	}
	if err := env.batchRepo.Create(ctx, batch); err != nil {
		t.Fatalf("Create(seed batch) error = %v", err)
	}

	task := &domain.PlatformTask{
		BatchID:         batch.ID,
		ImageID:         imageID,
		TaskType:        taskType,
		SourceType:      domain.TaskBatchSourceImportScan,
		Status:          status,
		ImageVersionKey: versionKey,
		DedupeKey:       versionKey + ":" + taskType,
		CreatedAt:       time.Now(),
	}
	if status == domain.PlatformTaskStatusCompleted || status == domain.PlatformTaskStatusFailed || status == domain.PlatformTaskStatusSkipped || status == domain.PlatformTaskStatusCancelled {
		finishedAt := time.Now()
		task.FinishedAt = &finishedAt
	}
	if err := env.taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("Create(seed task) error = %v", err)
	}
}
