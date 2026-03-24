package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

func TestPlatformTaskSchemaStoresDedupeFields(t *testing.T) {
	t.Parallel()

	db := newTaskPlatformSchemaTestDB(t)
	defer db.Close()

	expectedColumns := map[string]string{
		"batch_id":          "INTEGER",
		"image_id":          "INTEGER",
		"task_type":         "TEXT",
		"image_version_key": "TEXT",
		"dedupe_key":        "TEXT",
	}

	for columnName, wantType := range expectedColumns {
		columnType, _, found := columnInfo(t, db, "platform_tasks", columnName)
		if !found {
			t.Fatalf("expected platform_tasks.%s column to exist", columnName)
		}
		if columnType != wantType {
			t.Fatalf("platform_tasks.%s type = %q, want %q", columnName, columnType, wantType)
		}
	}
}

func TestPlatformTaskSchemaCreatesDedupeLookupIndex(t *testing.T) {
	t.Parallel()

	db := newTaskPlatformSchemaTestDB(t)
	defer db.Close()

	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = 'platform_tasks' AND name = ?`, "idx_platform_tasks_dedupe_key").Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			t.Fatal("expected idx_platform_tasks_dedupe_key index to exist")
		}
		t.Fatalf("failed to lookup idx_platform_tasks_dedupe_key: %v", err)
	}
	if name != "idx_platform_tasks_dedupe_key" {
		t.Fatalf("index name = %q, want %q", name, "idx_platform_tasks_dedupe_key")
	}
}

func TestPlatformTaskRepositoryFindActiveByDedupeKeyReturnsRunningOrCompleted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newTaskPlatformSchemaTestDB(t)
	defer db.Close()

	batchRepo := NewTaskBatchRepository(db)
	taskRepo := NewPlatformTaskRepository(db)
	image := saveTaskPlatformTestImage(t, db, "dedupe.png")

	batch := &domain.TaskBatch{
		SourceType:   domain.TaskBatchSourceImportScan,
		SummaryLabel: "dedupe lookup",
		Status:       domain.TaskBatchStatusPending,
		CreatedAt:    time.Now(),
	}
	if err := batchRepo.Create(ctx, batch); err != nil {
		t.Fatalf("Create(batch) error = %v", err)
	}

	versionKey := "image:dedupe:v1"
	task := &domain.PlatformTask{
		BatchID:         batch.ID,
		ImageID:         image.ID,
		TaskType:        domain.PlatformTaskTypeThumbnailGenerate,
		SourceType:      domain.TaskBatchSourceImportScan,
		Status:          domain.PlatformTaskStatusRunning,
		ImageVersionKey: versionKey,
		DedupeKey:       versionKey + ":thumbnail",
		CreatedAt:       time.Now(),
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("Create(task) error = %v", err)
	}

	found, err := taskRepo.FindActiveByDedupeKey(ctx, task.DedupeKey)
	if err != nil {
		t.Fatalf("FindActiveByDedupeKey(running) error = %v", err)
	}
	if found == nil || found.ID != task.ID {
		t.Fatalf("running dedupe lookup = %+v, want task id %d", found, task.ID)
	}

	task.Status = domain.PlatformTaskStatusCompleted
	finishedAt := time.Now()
	task.FinishedAt = &finishedAt
	if err := taskRepo.Update(ctx, task); err != nil {
		t.Fatalf("Update(completed task) error = %v", err)
	}

	found, err = taskRepo.FindActiveByDedupeKey(ctx, task.DedupeKey)
	if err != nil {
		t.Fatalf("FindActiveByDedupeKey(completed) error = %v", err)
	}
	if found == nil || found.Status != domain.PlatformTaskStatusCompleted {
		t.Fatalf("completed dedupe lookup = %+v, want completed task", found)
	}
}

func TestPlatformTaskRepositoryListByImageAndTypesReturnsOnlyRequestedTypes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newTaskPlatformSchemaTestDB(t)
	defer db.Close()

	batchRepo := NewTaskBatchRepository(db)
	taskRepo := NewPlatformTaskRepository(db)
	image := saveTaskPlatformTestImage(t, db, "missing-types.png")

	batch := &domain.TaskBatch{
		SourceType:   domain.TaskBatchSourceImportScan,
		SummaryLabel: "missing types",
		Status:       domain.TaskBatchStatusPending,
		CreatedAt:    time.Now(),
	}
	if err := batchRepo.Create(ctx, batch); err != nil {
		t.Fatalf("Create(batch) error = %v", err)
	}

	for _, taskType := range []string{domain.PlatformTaskTypeThumbnailGenerate, domain.PlatformTaskTypeAITagGeneration} {
		task := &domain.PlatformTask{
			BatchID:         batch.ID,
			ImageID:         image.ID,
			TaskType:        taskType,
			SourceType:      domain.TaskBatchSourceImportScan,
			Status:          domain.PlatformTaskStatusCompleted,
			ImageVersionKey: "image:missing:v1",
			DedupeKey:       "image:missing:v1:" + taskType,
			CreatedAt:       time.Now(),
		}
		if err := taskRepo.Create(ctx, task); err != nil {
			t.Fatalf("Create(%s) error = %v", taskType, err)
		}
	}

	found, err := taskRepo.ListByImageAndTypes(ctx, image.ID, []string{domain.PlatformTaskTypeAITagGeneration})
	if err != nil {
		t.Fatalf("ListByImageAndTypes() error = %v", err)
	}
	if len(found) != 1 || found[0].TaskType != domain.PlatformTaskTypeAITagGeneration {
		t.Fatalf("ListByImageAndTypes() = %+v, want only ai_tag_generation", found)
	}
}
