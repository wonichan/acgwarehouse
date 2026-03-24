package repository

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

func TestTaskBatchSchemaCreatesBatchTables(t *testing.T) {
	t.Parallel()

	db := newTaskPlatformSchemaTestDB(t)
	defer db.Close()

	mustHaveTable(t, db, "task_batches")
	mustHaveTable(t, db, "task_batch_sources")
	mustHaveTable(t, db, "platform_tasks")
}

func TestSchemaAddsNullablePlatformTaskIDToAsyncJobs(t *testing.T) {
	t.Parallel()

	db := newTaskPlatformSchemaTestDB(t)
	defer db.Close()

	columnType, notNull, found := columnInfo(t, db, "async_jobs", "platform_task_id")
	if !found {
		t.Fatal("expected async_jobs.platform_task_id column to exist")
	}
	if columnType != "INTEGER" {
		t.Fatalf("platform_task_id type = %q, want %q", columnType, "INTEGER")
	}
	if notNull {
		t.Fatal("expected async_jobs.platform_task_id to be nullable")
	}
}

func TestTaskBatchRepositoryRefreshStatusAggregatesLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newTaskPlatformSchemaTestDB(t)
	defer db.Close()

	batchRepo := NewTaskBatchRepository(db)
	taskRepo := NewPlatformTaskRepository(db)
	image := saveTaskPlatformTestImage(t, db, "batch-lifecycle.png")

	batch := &domain.TaskBatch{
		SourceType:   domain.TaskBatchSourceImportScan,
		SummaryLabel: "batch lifecycle",
		Status:       domain.TaskBatchStatusPending,
		CreatedAt:    time.Now(),
	}
	if err := batchRepo.Create(ctx, batch); err != nil {
		t.Fatalf("Create(batch) error = %v", err)
	}

	versionKey := "image:" + image.Filename + ":v1"
	for _, candidate := range []domain.PlatformTask{
		{
			BatchID:         batch.ID,
			ImageID:         image.ID,
			TaskType:        domain.PlatformTaskTypeThumbnailGenerate,
			SourceType:      domain.TaskBatchSourceImportScan,
			Status:          domain.PlatformTaskStatusPending,
			ImageVersionKey: versionKey,
			DedupeKey:       versionKey + ":thumbnail",
			CreatedAt:       time.Now(),
		},
		{
			BatchID:         batch.ID,
			ImageID:         image.ID,
			TaskType:        domain.PlatformTaskTypeAITagGeneration,
			SourceType:      domain.TaskBatchSourceImportScan,
			Status:          domain.PlatformTaskStatusCompleted,
			ImageVersionKey: versionKey,
			DedupeKey:       versionKey + ":ai",
			CreatedAt:       time.Now(),
		},
	} {
		task := candidate
		if err := taskRepo.Create(ctx, &task); err != nil {
			t.Fatalf("Create(task %s) error = %v", task.TaskType, err)
		}
	}

	refreshed, err := batchRepo.RefreshStatus(ctx, batch.ID)
	if err != nil {
		t.Fatalf("RefreshStatus(running) error = %v", err)
	}
	if refreshed.Status != domain.TaskBatchStatusRunning {
		t.Fatalf("running aggregate status = %q, want %q", refreshed.Status, domain.TaskBatchStatusRunning)
	}

	tasks, err := taskRepo.List(ctx, PlatformTaskListFilter{BatchID: &batch.ID, Limit: 10})
	if err != nil {
		t.Fatalf("List(tasks) error = %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("len(tasks) = %d, want 2", len(tasks))
	}
	tasks[0].Status = domain.PlatformTaskStatusFailed
	finishedAt := time.Now()
	tasks[0].FinishedAt = &finishedAt
	if err := taskRepo.Update(ctx, &tasks[0]); err != nil {
		t.Fatalf("Update(failed task) error = %v", err)
	}

	refreshed, err = batchRepo.RefreshStatus(ctx, batch.ID)
	if err != nil {
		t.Fatalf("RefreshStatus(partial_failed) error = %v", err)
	}
	if refreshed.Status != domain.TaskBatchStatusPartialFailed {
		t.Fatalf("partial failure aggregate status = %q, want %q", refreshed.Status, domain.TaskBatchStatusPartialFailed)
	}
	if refreshed.FinishedAt == nil {
		t.Fatal("expected finished_at to be set once all tasks reached terminal states")
	}
}

func newTaskPlatformSchemaTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "task-platform-schema.db"))
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}
	return db
}

func mustHaveTable(t *testing.T, db *sql.DB, tableName string) {
	t.Helper()

	var found string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, tableName).Scan(&found)
	if err != nil {
		if err == sql.ErrNoRows {
			t.Fatalf("expected table %q to exist", tableName)
		}
		t.Fatalf("failed to lookup table %q: %v", tableName, err)
	}
	if found != tableName {
		t.Fatalf("found table = %q, want %q", found, tableName)
	}
}

func columnInfo(t *testing.T, db *sql.DB, tableName, columnName string) (columnType string, notNull bool, found bool) {
	t.Helper()

	rows, err := db.Query(`PRAGMA table_info(` + tableName + `)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info(%s) error = %v", tableName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			declType   string
			notNullInt int
			defaultVal any
			pk         int
		)
		if err := rows.Scan(&cid, &name, &declType, &notNullInt, &defaultVal, &pk); err != nil {
			t.Fatalf("scan table_info row error = %v", err)
		}
		if name == columnName {
			return declType, notNullInt == 1, true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table_info rows error = %v", err)
	}
	return "", false, false
}

func saveTaskPlatformTestImage(t *testing.T, db *sql.DB, filename string) *domain.Image {
	t.Helper()

	now := time.Now()
	image := &domain.Image{
		Path:       "/task-platform/" + filename,
		Filename:   filename,
		SourceRoot: "/task-platform",
		FileSize:   256,
		Width:      64,
		Height:     64,
		Format:     "png",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if _, err := NewImageRepository(db).SaveImage(image); err != nil {
		t.Fatalf("SaveImage(%s) error = %v", filename, err)
	}
	return image
}
