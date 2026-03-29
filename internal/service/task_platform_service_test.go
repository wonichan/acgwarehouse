package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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

func TestTaskPlatformServiceIgnoresJobLifecycleForCancelledTasks(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	env := newTaskPlatformServiceTestEnv(t)
	image := saveTaskPlatformServiceImage(t, env.db, "cancelled-task.png")
	plan, err := env.service.PlanBatch(ctx, TaskPlatformPlanRequest{
		SourceType:   domain.TaskBatchSourceImportScan,
		SummaryLabel: "cancelled lifecycle",
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
	job, err := env.service.QueueTask(ctx, &plan.CreatedTasks[0], domain.PlatformTaskTypeThumbnailGenerate, `{}`)
	if err != nil {
		t.Fatalf("QueueTask() error = %v", err)
	}
	task, err := env.taskRepo.FindByID(ctx, plan.CreatedTasks[0].ID)
	if err != nil {
		t.Fatalf("FindByID(task) error = %v", err)
	}
	task.Status = domain.PlatformTaskStatusCancelled
	if err := env.taskRepo.Update(ctx, task); err != nil {
		t.Fatalf("Update(cancelled task) error = %v", err)
	}

	if err := env.service.MarkJobRunning(ctx, job.ID); err != nil {
		t.Fatalf("MarkJobRunning() error = %v", err)
	}
	if err := env.service.MarkJobCompleted(ctx, job.ID); err != nil {
		t.Fatalf("MarkJobCompleted() error = %v", err)
	}
	if err := env.service.MarkJobFailed(ctx, job.ID, "should be ignored"); err != nil {
		t.Fatalf("MarkJobFailed() error = %v", err)
	}

	reloaded, err := env.taskRepo.FindByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("FindByID(reloaded task) error = %v", err)
	}
	if reloaded.Status != domain.PlatformTaskStatusCancelled {
		t.Fatalf("cancelled task status = %q, want %q", reloaded.Status, domain.PlatformTaskStatusCancelled)
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

func TestMarkJobFailedIsolation_OneFailsSiblingSucceeds(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	env := newTaskPlatformServiceTestEnv(t)

	img1 := saveTaskPlatformServiceImage(t, env.db, "isolation-a.png")
	img2 := saveTaskPlatformServiceImage(t, env.db, "isolation-b.png")
	img3 := saveTaskPlatformServiceImage(t, env.db, "isolation-c.png")

	result, err := env.service.PlanBatch(ctx, TaskPlatformPlanRequest{
		SourceType:   domain.TaskBatchSourceImportScan,
		SummaryLabel: "isolation test",
		SourceRoots:  []string{"/task-platform"},
		TaskTypes:    []string{domain.PlatformTaskTypeAITagGeneration},
		Items: []TaskPlatformPlanItem{
			{ImageID: img1.ID, ImageVersionKey: "image:iso-a:v1", SourceDescriptor: img1.Path},
			{ImageID: img2.ID, ImageVersionKey: "image:iso-b:v1", SourceDescriptor: img2.Path},
			{ImageID: img3.ID, ImageVersionKey: "image:iso-c:v1", SourceDescriptor: img3.Path},
		},
	})
	if err != nil {
		t.Fatalf("PlanBatch() error = %v", err)
	}
	if len(result.CreatedTasks) != 3 {
		t.Fatalf("len(CreatedTasks) = %d, want 3", len(result.CreatedTasks))
	}

	// Queue all three tasks
	jobs := make([]*domain.AsyncJob, 3)
	for i := range result.CreatedTasks {
		payload := `{"image_id":` + fmt.Sprintf("%d", result.CreatedTasks[i].ImageID) + `}`
		job, err := env.service.QueueTask(ctx, &result.CreatedTasks[i], domain.PlatformTaskTypeAITagGeneration, payload)
		if err != nil {
			t.Fatalf("QueueTask(%d) error = %v", i, err)
		}
		jobs[i] = job
	}

	// Fail task for img2 (the middle sibling)
	if err := env.service.MarkJobFailed(ctx, jobs[1].ID, "AI provider timeout"); err != nil {
		t.Fatalf("MarkJobFailed(img2) error = %v", err)
	}

	// Complete task for img1 (first sibling)
	if err := env.service.MarkJobCompleted(ctx, jobs[0].ID); err != nil {
		t.Fatalf("MarkJobCompleted(img1) error = %v", err)
	}

	// Complete task for img3 (last sibling)
	if err := env.service.MarkJobCompleted(ctx, jobs[2].ID); err != nil {
		t.Fatalf("MarkJobCompleted(img3) error = %v", err)
	}

	// Verify each task reached its own terminal state independently
	task1, err := env.taskRepo.FindByID(ctx, result.CreatedTasks[0].ID)
	if err != nil {
		t.Fatalf("FindByID(task1) error = %v", err)
	}
	if task1.Status != domain.PlatformTaskStatusCompleted {
		t.Errorf("task1 status = %q, want %q", task1.Status, domain.PlatformTaskStatusCompleted)
	}

	task2, err := env.taskRepo.FindByID(ctx, result.CreatedTasks[1].ID)
	if err != nil {
		t.Fatalf("FindByID(task2) error = %v", err)
	}
	if task2.Status != domain.PlatformTaskStatusFailed {
		t.Errorf("task2 status = %q, want %q", task2.Status, domain.PlatformTaskStatusFailed)
	}
	if task2.ErrorSummary == nil || *task2.ErrorSummary != "AI provider timeout" {
		t.Errorf("task2 ErrorSummary = %+v, want %q", task2.ErrorSummary, "AI provider timeout")
	}

	task3, err := env.taskRepo.FindByID(ctx, result.CreatedTasks[2].ID)
	if err != nil {
		t.Fatalf("FindByID(task3) error = %v", err)
	}
	if task3.Status != domain.PlatformTaskStatusCompleted {
		t.Errorf("task3 status = %q, want %q", task3.Status, domain.PlatformTaskStatusCompleted)
	}

	// Batch should be partial_failed (1 failed, 2 completed)
	batch, err := env.batchRepo.FindByID(ctx, result.Batch.ID)
	if err != nil {
		t.Fatalf("FindByID(batch) error = %v", err)
	}
	if batch.Status != domain.TaskBatchStatusPartialFailed {
		t.Fatalf("batch status = %q, want %q", batch.Status, domain.TaskBatchStatusPartialFailed)
	}
}

func TestMarkJobFailedIsolation_OnlyAffectsOwnTask(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	env := newTaskPlatformServiceTestEnv(t)

	img1 := saveTaskPlatformServiceImage(t, env.db, "own-a.png")
	img2 := saveTaskPlatformServiceImage(t, env.db, "own-b.png")

	// Create two separate batches
	plan1, err := env.service.PlanBatch(ctx, TaskPlatformPlanRequest{
		SourceType:   domain.TaskBatchSourceImportScan,
		SummaryLabel: "batch 1",
		TaskTypes:    []string{domain.PlatformTaskTypeAITagGeneration},
		Items: []TaskPlatformPlanItem{
			{ImageID: img1.ID, ImageVersionKey: "image:own-a:v1", SourceDescriptor: img1.Path},
		},
	})
	if err != nil {
		t.Fatalf("PlanBatch(batch1) error = %v", err)
	}

	plan2, err := env.service.PlanBatch(ctx, TaskPlatformPlanRequest{
		SourceType:   domain.TaskBatchSourceImportScan,
		SummaryLabel: "batch 2",
		TaskTypes:    []string{domain.PlatformTaskTypeAITagGeneration},
		Items: []TaskPlatformPlanItem{
			{ImageID: img2.ID, ImageVersionKey: "image:own-b:v1", SourceDescriptor: img2.Path},
		},
	})
	if err != nil {
		t.Fatalf("PlanBatch(batch2) error = %v", err)
	}

	job1, err := env.service.QueueTask(ctx, &plan1.CreatedTasks[0], domain.PlatformTaskTypeAITagGeneration, `{}`)
	if err != nil {
		t.Fatalf("QueueTask(batch1) error = %v", err)
	}
	_, err = env.service.QueueTask(ctx, &plan2.CreatedTasks[0], domain.PlatformTaskTypeAITagGeneration, `{}`)
	if err != nil {
		t.Fatalf("QueueTask(batch2) error = %v", err)
	}

	// Fail job in batch 1
	if err := env.service.MarkJobFailed(ctx, job1.ID, "config error: missing API key"); err != nil {
		t.Fatalf("MarkJobFailed(job1) error = %v", err)
	}

	// Verify batch 1 task is failed with correct error
	failedTask, err := env.taskRepo.FindByID(ctx, plan1.CreatedTasks[0].ID)
	if err != nil {
		t.Fatalf("FindByID(failed task) error = %v", err)
	}
	if failedTask.Status != domain.PlatformTaskStatusFailed {
		t.Errorf("failed task status = %q, want %q", failedTask.Status, domain.PlatformTaskStatusFailed)
	}
	if failedTask.ErrorSummary == nil || *failedTask.ErrorSummary != "config error: missing API key" {
		t.Errorf("failed task ErrorSummary = %+v, want %q", failedTask.ErrorSummary, "config error: missing API key")
	}

	// Verify batch 2 task is completely unaffected (still queued, no error)
	unrelatedTask, err := env.taskRepo.FindByID(ctx, plan2.CreatedTasks[0].ID)
	if err != nil {
		t.Fatalf("FindByID(unrelated task) error = %v", err)
	}
	if unrelatedTask.Status != domain.PlatformTaskStatusQueued {
		t.Errorf("unrelated task status = %q, want %q (unchanged)", unrelatedTask.Status, domain.PlatformTaskStatusQueued)
	}
	if unrelatedTask.ErrorSummary != nil {
		t.Errorf("unrelated task ErrorSummary = %+v, want nil", unrelatedTask.ErrorSummary)
	}

	// Batch 1 is failed, batch 2 is still running (has queued tasks)
	batch1, err := env.batchRepo.FindByID(ctx, plan1.Batch.ID)
	if err != nil {
		t.Fatalf("FindByID(batch1) error = %v", err)
	}
	if batch1.Status != domain.TaskBatchStatusFailed {
		t.Errorf("batch1 status = %q, want %q", batch1.Status, domain.TaskBatchStatusFailed)
	}

	batch2, err := env.batchRepo.FindByID(ctx, plan2.Batch.ID)
	if err != nil {
		t.Fatalf("FindByID(batch2) error = %v", err)
	}
	if batch2.Status == domain.TaskBatchStatusFailed || batch2.Status == domain.TaskBatchStatusPartialFailed {
		t.Errorf("batch2 status = %q, should not be failed/partial_failed (unrelated failure must not leak)", batch2.Status)
	}
}

func TestIsolation_QueueProcessingContinuesAfterFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	env := newTaskPlatformServiceTestEnv(t)

	img1 := saveTaskPlatformServiceImage(t, env.db, "continue-a.png")
	img2 := saveTaskPlatformServiceImage(t, env.db, "continue-b.png")

	result, err := env.service.PlanBatch(ctx, TaskPlatformPlanRequest{
		SourceType:   domain.TaskBatchSourceImportScan,
		SummaryLabel: "continue after failure",
		TaskTypes:    []string{domain.PlatformTaskTypeAITagGeneration},
		Items: []TaskPlatformPlanItem{
			{ImageID: img1.ID, ImageVersionKey: "image:cont-a:v1", SourceDescriptor: img1.Path},
			{ImageID: img2.ID, ImageVersionKey: "image:cont-b:v1", SourceDescriptor: img2.Path},
		},
	})
	if err != nil {
		t.Fatalf("PlanBatch() error = %v", err)
	}
	if len(result.CreatedTasks) != 2 {
		t.Fatalf("len(CreatedTasks) = %d, want 2", len(result.CreatedTasks))
	}

	job1, err := env.service.QueueTask(ctx, &result.CreatedTasks[0], domain.PlatformTaskTypeAITagGeneration, `{}`)
	if err != nil {
		t.Fatalf("QueueTask(0) error = %v", err)
	}
	job2, err := env.service.QueueTask(ctx, &result.CreatedTasks[1], domain.PlatformTaskTypeAITagGeneration, `{}`)
	if err != nil {
		t.Fatalf("QueueTask(1) error = %v", err)
	}

	// Fail first job
	if err := env.service.MarkJobFailed(ctx, job1.ID, "transient network error"); err != nil {
		t.Fatalf("MarkJobFailed(job1) error = %v", err)
	}

	// Second job should still be processable — mark it running then completed
	if err := env.service.MarkJobRunning(ctx, job2.ID); err != nil {
		t.Fatalf("MarkJobRunning(job2) error = %v", err)
	}
	if err := env.service.MarkJobCompleted(ctx, job2.ID); err != nil {
		t.Fatalf("MarkJobCompleted(job2) error = %v", err)
	}

	// Verify: task1 failed, task2 completed
	task1, _ := env.taskRepo.FindByID(ctx, result.CreatedTasks[0].ID)
	task2, _ := env.taskRepo.FindByID(ctx, result.CreatedTasks[1].ID)

	if task1.Status != domain.PlatformTaskStatusFailed {
		t.Errorf("task1 status = %q, want %q", task1.Status, domain.PlatformTaskStatusFailed)
	}
	if task2.Status != domain.PlatformTaskStatusCompleted {
		t.Errorf("task2 status = %q, want %q", task2.Status, domain.PlatformTaskStatusCompleted)
	}

	// Batch is partial_failed: 1 failed + 1 completed
	batch, err := env.batchRepo.FindByID(ctx, result.Batch.ID)
	if err != nil {
		t.Fatalf("FindByID(batch) error = %v", err)
	}
	if batch.Status != domain.TaskBatchStatusPartialFailed {
		t.Fatalf("batch status = %q, want %q (partial_failed, not blocked)", batch.Status, domain.TaskBatchStatusPartialFailed)
	}
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
