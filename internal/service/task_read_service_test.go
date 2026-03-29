package service

import (
	"context"
	"testing"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

func TestTaskReadServiceListBatchesAggregatesSourcesAndStatusStats(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	env := newTaskPlatformServiceTestEnv(t)
	service := NewTaskReadService(repository.NewTaskBatchReadRepository(env.db))

	firstImage := saveTaskPlatformServiceImage(t, env.db, "batch-aggregate-a.png")
	secondImage := saveTaskPlatformServiceImage(t, env.db, "batch-aggregate-b.png")

	batch := &domain.TaskBatch{
		SourceType:            domain.TaskBatchSourceImportScan,
		TriggerKey:            "scan-aggregate",
		SummaryLabel:          "import batch aggregate",
		Status:                domain.TaskBatchStatusPartialFailed,
		TotalImages:           4,
		NewImages:             2,
		SkippedImages:         2,
		SkippedUnchanged:      1,
		SkippedDuplicateTasks: 1,
		LatestErrorSummary:    strPtr("ai tag timeout"),
		CreatedAt:             time.Now().Add(-1 * time.Hour),
	}
	if err := env.batchRepo.Create(ctx, batch); err != nil {
		t.Fatalf("Create(batch) error = %v", err)
	}
	if err := env.batchRepo.AddSource(ctx, &domain.TaskBatchSource{BatchID: batch.ID, SourceRoot: "/library/root-a", SourceLabel: "root-a"}); err != nil {
		t.Fatalf("AddSource(root-a) error = %v", err)
	}
	if err := env.batchRepo.AddSource(ctx, &domain.TaskBatchSource{BatchID: batch.ID, SourceRoot: "/library/root-b", SourceLabel: "root-b"}); err != nil {
		t.Fatalf("AddSource(root-b) error = %v", err)
	}

	seedTaskReadPlatformTask(t, env, batch.ID, firstImage.ID, domain.PlatformTaskTypeThumbnailGenerate, domain.PlatformTaskStatusCompleted, "image:aggregate:a:thumb", nil, nil)
	seedTaskReadPlatformTask(t, env, batch.ID, secondImage.ID, domain.PlatformTaskTypeAITagGeneration, domain.PlatformTaskStatusFailed, "image:aggregate:b:ai", nil, strPtr("ai tag timeout"))

	batches, err := service.ListBatches(ctx, TaskBatchReadFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListBatches() error = %v", err)
	}
	if len(batches) != 1 {
		t.Fatalf("len(batches) = %d, want 1", len(batches))
	}
	got := batches[0]
	if got.ID != batch.ID {
		t.Fatalf("batch id = %d, want %d", got.ID, batch.ID)
	}
	if got.SourceSummary != "root-a, root-b" {
		t.Fatalf("SourceSummary = %q, want %q", got.SourceSummary, "root-a, root-b")
	}
	if got.StatusCounts[domain.PlatformTaskStatusCompleted] != 1 {
		t.Fatalf("completed count = %d, want 1", got.StatusCounts[domain.PlatformTaskStatusCompleted])
	}
	if got.StatusCounts[domain.PlatformTaskStatusFailed] != 1 {
		t.Fatalf("failed count = %d, want 1", got.StatusCounts[domain.PlatformTaskStatusFailed])
	}
	if got.TaskTypeCounts[domain.PlatformTaskTypeThumbnailGenerate] != 1 {
		t.Fatalf("thumbnail count = %d, want 1", got.TaskTypeCounts[domain.PlatformTaskTypeThumbnailGenerate])
	}
	if got.TaskTypeCounts[domain.PlatformTaskTypeAITagGeneration] != 1 {
		t.Fatalf("ai count = %d, want 1", got.TaskTypeCounts[domain.PlatformTaskTypeAITagGeneration])
	}
	if got.SkipSummary.Unchanged != 1 || got.SkipSummary.DuplicateTasks != 1 || got.SkipSummary.Total != 2 {
		t.Fatalf("SkipSummary = %+v, want unchanged=1 duplicate=1 total=2", got.SkipSummary)
	}
	if got.FailureSummary != "ai tag timeout" {
		t.Fatalf("FailureSummary = %q, want %q", got.FailureSummary, "ai tag timeout")
	}
}

func TestTaskReadServiceListTasksReturnsBatchTaskDetails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	env := newTaskPlatformServiceTestEnv(t)
	service := NewTaskReadService(repository.NewTaskBatchReadRepository(env.db))

	image := saveTaskPlatformServiceImage(t, env.db, "task-details.png")
	batch := &domain.TaskBatch{
		SourceType:   domain.TaskBatchSourceManualSingle,
		SummaryLabel: "manual single batch",
		Status:       domain.TaskBatchStatusRunning,
		CreatedAt:    time.Now(),
	}
	if err := env.batchRepo.Create(ctx, batch); err != nil {
		t.Fatalf("Create(batch) error = %v", err)
	}

	jobID := seedTaskReadPlatformTask(t, env, batch.ID, image.ID, domain.PlatformTaskTypeAITagGeneration, domain.PlatformTaskStatusQueued, "image:task-details:ai", strPtr(domain.PlatformTaskSkipReasonAlreadyCompleted), nil)

	tasks, err := service.ListTasks(ctx, TaskReadFilter{BatchID: &batch.ID, Limit: 10})
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("len(tasks) = %d, want 1", len(tasks))
	}
	got := tasks[0]
	if got.BatchID != batch.ID {
		t.Fatalf("BatchID = %d, want %d", got.BatchID, batch.ID)
	}
	if got.TaskType != domain.PlatformTaskTypeAITagGeneration {
		t.Fatalf("TaskType = %q, want %q", got.TaskType, domain.PlatformTaskTypeAITagGeneration)
	}
	if got.Status != domain.PlatformTaskStatusQueued {
		t.Fatalf("Status = %q, want %q", got.Status, domain.PlatformTaskStatusQueued)
	}
	if got.SkipReason != domain.PlatformTaskSkipReasonAlreadyCompleted {
		t.Fatalf("SkipReason = %q, want %q", got.SkipReason, domain.PlatformTaskSkipReasonAlreadyCompleted)
	}
	if got.LatestAsyncJobID == nil || *got.LatestAsyncJobID != jobID {
		t.Fatalf("LatestAsyncJobID = %+v, want %d", got.LatestAsyncJobID, jobID)
	}
	if got.ImageFilename != image.Filename {
		t.Fatalf("ImageFilename = %q, want %q", got.ImageFilename, image.Filename)
	}
	if got.ImagePath != image.Path {
		t.Fatalf("ImagePath = %q, want %q", got.ImagePath, image.Path)
	}
}

func TestTaskReadServiceListBatchesSortsDescendingAndSupportsFilters(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	env := newTaskPlatformServiceTestEnv(t)
	service := NewTaskReadService(repository.NewTaskBatchReadRepository(env.db))

	older := &domain.TaskBatch{SourceType: domain.TaskBatchSourceImportScan, SummaryLabel: "older", Status: domain.TaskBatchStatusCompleted, CreatedAt: time.Now().Add(-2 * time.Hour)}
	newer := &domain.TaskBatch{SourceType: domain.TaskBatchSourceManualBatch, SummaryLabel: "newer", Status: domain.TaskBatchStatusRunning, CreatedAt: time.Now().Add(-1 * time.Hour)}
	for _, batch := range []*domain.TaskBatch{older, newer} {
		if err := env.batchRepo.Create(ctx, batch); err != nil {
			t.Fatalf("Create(%s) error = %v", batch.SummaryLabel, err)
		}
	}

	batches, err := service.ListBatches(ctx, TaskBatchReadFilter{Limit: 1})
	if err != nil {
		t.Fatalf("ListBatches(limit) error = %v", err)
	}
	if len(batches) != 1 {
		t.Fatalf("len(batches) = %d, want 1", len(batches))
	}
	if batches[0].ID != newer.ID {
		t.Fatalf("first batch id = %d, want %d", batches[0].ID, newer.ID)
	}

	filteredTasks, err := service.ListTasks(ctx, TaskReadFilter{BatchID: &older.ID, Limit: 10})
	if err != nil {
		t.Fatalf("ListTasks(batch_id) error = %v", err)
	}
	if len(filteredTasks) != 0 {
		t.Fatalf("len(filteredTasks) = %d, want 0", len(filteredTasks))
	}

	filteredBatches, err := service.ListBatches(ctx, TaskBatchReadFilter{SourceType: domain.TaskBatchSourceImportScan, Limit: 10})
	if err != nil {
		t.Fatalf("ListBatches(source_type) error = %v", err)
	}
	if len(filteredBatches) != 1 || filteredBatches[0].ID != older.ID {
		t.Fatalf("filtered batches = %+v, want only older batch", filteredBatches)
	}
}

func TestTaskReadServiceFailureSummary_GroupedReasonsWithCounts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	env := newTaskPlatformServiceTestEnv(t)
	service := NewTaskReadService(repository.NewTaskBatchReadRepository(env.db))

	img1 := saveTaskPlatformServiceImage(t, env.db, "failure-a.png")
	img2 := saveTaskPlatformServiceImage(t, env.db, "failure-b.png")
	img3 := saveTaskPlatformServiceImage(t, env.db, "failure-c.png")
	img4 := saveTaskPlatformServiceImage(t, env.db, "failure-d.png")

	batch := &domain.TaskBatch{
		SourceType:         domain.TaskBatchSourceImportScan,
		SummaryLabel:       "failure grouping test",
		Status:             domain.TaskBatchStatusPartialFailed,
		TotalImages:        4,
		NewImages:          4,
		LatestErrorSummary: strPtr("timeout: AI provider rate limit exceeded"),
		CreatedAt:          time.Now(),
	}
	if err := env.batchRepo.Create(ctx, batch); err != nil {
		t.Fatalf("Create(batch) error = %v", err)
	}

	// 2 tasks with same "timeout" error, 1 with "auth" error, 1 completed
	seedTaskReadPlatformTask(t, env, batch.ID, img1.ID, domain.PlatformTaskTypeAITagGeneration, domain.PlatformTaskStatusFailed, "image:fa:v1", nil, strPtr("timeout: AI provider rate limit exceeded"))
	seedTaskReadPlatformTask(t, env, batch.ID, img2.ID, domain.PlatformTaskTypeAITagGeneration, domain.PlatformTaskStatusFailed, "image:fb:v1", nil, strPtr("timeout: AI provider rate limit exceeded"))
	seedTaskReadPlatformTask(t, env, batch.ID, img3.ID, domain.PlatformTaskTypeAITagGeneration, domain.PlatformTaskStatusFailed, "image:fc:v1", nil, strPtr("auth: invalid API key"))
	seedTaskReadPlatformTask(t, env, batch.ID, img4.ID, domain.PlatformTaskTypeAITagGeneration, domain.PlatformTaskStatusCompleted, "image:fd:v1", nil, nil)

	batches, err := service.ListBatches(ctx, TaskBatchReadFilter{BatchID: &batch.ID, Limit: 10})
	if err != nil {
		t.Fatalf("ListBatches() error = %v", err)
	}
	if len(batches) != 1 {
		t.Fatalf("len(batches) = %d, want 1", len(batches))
	}
	got := batches[0]

	// Must have grouped failure reasons with counts
	if len(got.FailureGroups) < 2 {
		t.Fatalf("len(FailureGroups) = %d, want >= 2", len(got.FailureGroups))
	}

	// Find the timeout group
	var timeoutGroup *TaskBatchFailureGroup
	var authGroup *TaskBatchFailureGroup
	for i := range got.FailureGroups {
		if got.FailureGroups[i].ReasonKey == "timeout" {
			timeoutGroup = &got.FailureGroups[i]
		}
		if got.FailureGroups[i].ReasonKey == "auth" {
			authGroup = &got.FailureGroups[i]
		}
	}
	if timeoutGroup == nil {
		t.Fatal("expected failure group with reason_key 'timeout', not found")
	}
	if timeoutGroup.Count != 2 {
		t.Errorf("timeout group count = %d, want 2", timeoutGroup.Count)
	}
	if authGroup == nil {
		t.Fatal("expected failure group with reason_key 'auth', not found")
	}
	if authGroup.Count != 1 {
		t.Errorf("auth group count = %d, want 1", authGroup.Count)
	}

	// Preserve existing FailureSummary for backward compat
	if got.FailureSummary != "timeout: AI provider rate limit exceeded" {
		t.Fatalf("FailureSummary = %q, want %q", got.FailureSummary, "timeout: AI provider rate limit exceeded")
	}
}

func TestTaskReadServiceRetryHint_TransientVsNonRetryable(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	env := newTaskPlatformServiceTestEnv(t)
	service := NewTaskReadService(repository.NewTaskBatchReadRepository(env.db))

	img1 := saveTaskPlatformServiceImage(t, env.db, "retry-transient.png")
	img2 := saveTaskPlatformServiceImage(t, env.db, "retry-auth.png")
	img3 := saveTaskPlatformServiceImage(t, env.db, "retry-config.png")
	img4 := saveTaskPlatformServiceImage(t, env.db, "retry-network.png")

	batch := &domain.TaskBatch{
		SourceType:         domain.TaskBatchSourceImportScan,
		SummaryLabel:       "retry hint test",
		Status:             domain.TaskBatchStatusFailed,
		TotalImages:        4,
		NewImages:          4,
		LatestErrorSummary: strPtr("network: connection refused"),
		CreatedAt:          time.Now(),
	}
	if err := env.batchRepo.Create(ctx, batch); err != nil {
		t.Fatalf("Create(batch) error = %v", err)
	}

	seedTaskReadPlatformTask(t, env, batch.ID, img1.ID, domain.PlatformTaskTypeAITagGeneration, domain.PlatformTaskStatusFailed, "image:rt:v1", nil, strPtr("timeout: request timed out after 30s"))
	seedTaskReadPlatformTask(t, env, batch.ID, img2.ID, domain.PlatformTaskTypeAITagGeneration, domain.PlatformTaskStatusFailed, "image:ra:v1", nil, strPtr("auth: invalid API key"))
	seedTaskReadPlatformTask(t, env, batch.ID, img3.ID, domain.PlatformTaskTypeAITagGeneration, domain.PlatformTaskStatusFailed, "image:rc:v1", nil, strPtr("config: model not found"))
	seedTaskReadPlatformTask(t, env, batch.ID, img4.ID, domain.PlatformTaskTypeAITagGeneration, domain.PlatformTaskStatusFailed, "image:rn:v1", nil, strPtr("network: connection refused"))

	batches, err := service.ListBatches(ctx, TaskBatchReadFilter{BatchID: &batch.ID, Limit: 10})
	if err != nil {
		t.Fatalf("ListBatches() error = %v", err)
	}
	if len(batches) != 1 {
		t.Fatalf("len(batches) = %d, want 1", len(batches))
	}
	got := batches[0]

	// Build lookup by reason_key
	groupMap := make(map[string]TaskBatchFailureGroup)
	for _, fg := range got.FailureGroups {
		groupMap[fg.ReasonKey] = fg
	}

	// timeout and network should be retry-recommended (transient)
	timeoutGrp, ok := groupMap["timeout"]
	if !ok {
		t.Fatal("expected failure group 'timeout'")
	}
	if !timeoutGrp.RetryRecommended {
		t.Errorf("timeout group RetryRecommended = false, want true (transient failure)")
	}

	networkGrp, ok := groupMap["network"]
	if !ok {
		t.Fatal("expected failure group 'network'")
	}
	if !networkGrp.RetryRecommended {
		t.Errorf("network group RetryRecommended = false, want true (transient failure)")
	}

	// auth and config should NOT be retry-recommended
	authGrp, ok := groupMap["auth"]
	if !ok {
		t.Fatal("expected failure group 'auth'")
	}
	if authGrp.RetryRecommended {
		t.Errorf("auth group RetryRecommended = true, want false (configuration error)")
	}

	configGrp, ok := groupMap["config"]
	if !ok {
		t.Fatal("expected failure group 'config'")
	}
	if configGrp.RetryRecommended {
		t.Errorf("config group RetryRecommended = true, want false (configuration error)")
	}

	// All groups should have non-empty retry hints
	for _, fg := range got.FailureGroups {
		if fg.RetryHint == "" {
			t.Errorf("failure group %q has empty RetryHint", fg.ReasonKey)
		}
	}
}

func TestTaskReadServiceTaskReadReturnsErrorSummaryPerTask(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	env := newTaskPlatformServiceTestEnv(t)
	service := NewTaskReadService(repository.NewTaskBatchReadRepository(env.db))

	image := saveTaskPlatformServiceImage(t, env.db, "task-error.png")
	batch := &domain.TaskBatch{
		SourceType:   domain.TaskBatchSourceImportScan,
		SummaryLabel: "task error test",
		Status:       domain.TaskBatchStatusFailed,
		CreatedAt:    time.Now(),
	}
	if err := env.batchRepo.Create(ctx, batch); err != nil {
		t.Fatalf("Create(batch) error = %v", err)
	}

	seedTaskReadPlatformTask(t, env, batch.ID, image.ID, domain.PlatformTaskTypeAITagGeneration, domain.PlatformTaskStatusFailed, "image:te:v1", nil, strPtr("timeout: AI provider did not respond"))

	tasks, err := service.ListTasks(ctx, TaskReadFilter{BatchID: &batch.ID, Limit: 10})
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("len(tasks) = %d, want 1", len(tasks))
	}
	if tasks[0].ErrorSummary != "timeout: AI provider did not respond" {
		t.Fatalf("ErrorSummary = %q, want %q", tasks[0].ErrorSummary, "timeout: AI provider did not respond")
	}
}

func seedTaskReadPlatformTask(t *testing.T, env *taskPlatformServiceTestEnv, batchID, imageID int64, taskType, status, versionKey string, skipReason, errorSummary *string) int64 {
	t.Helper()

	ctx := context.Background()
	now := time.Now()
	task := &domain.PlatformTask{
		BatchID:         batchID,
		ImageID:         imageID,
		TaskType:        taskType,
		SourceType:      domain.TaskBatchSourceImportScan,
		Status:          status,
		DedupeKey:       versionKey + ":" + taskType,
		ImageVersionKey: versionKey,
		SkipReason:      skipReason,
		ErrorSummary:    errorSummary,
		CreatedAt:       now,
	}
	if status != domain.PlatformTaskStatusPending {
		task.QueuedAt = &now
	}
	if status == domain.PlatformTaskStatusRunning || status == domain.PlatformTaskStatusCompleted || status == domain.PlatformTaskStatusFailed {
		task.StartedAt = &now
	}
	if status == domain.PlatformTaskStatusCompleted || status == domain.PlatformTaskStatusFailed || status == domain.PlatformTaskStatusCancelled || status == domain.PlatformTaskStatusSkipped {
		task.FinishedAt = &now
	}
	if err := env.taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("Create(task) error = %v", err)
	}

	job := &domain.AsyncJob{
		PlatformTaskID: &task.ID,
		Type:           taskType,
		Status:         "ready",
		Payload:        `{"image_id":1}`,
		CreatedAt:      now,
	}
	if err := env.jobRepo.Save(job); err != nil {
		t.Fatalf("Save(job) error = %v", err)
	}
	if err := env.taskRepo.SetLatestAsyncJob(ctx, task.ID, &job.ID); err != nil {
		t.Fatalf("SetLatestAsyncJob() error = %v", err)
	}
	return job.ID
}

func strPtr(value string) *string {
	return &value
}
