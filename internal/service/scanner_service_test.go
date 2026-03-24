package service

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

func TestScannerCreatesImportBatchAndPlatformTask(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	imagePath := filepath.Join(root, "scan.png")
	textPath := filepath.Join(root, "skip.txt")

	if err := os.WriteFile(imagePath, tinyPNGFixture(), 0o600); err != nil {
		t.Fatalf("write image fixture: %v", err)
	}
	if err := os.WriteFile(textPath, []byte("not image"), 0o600); err != nil {
		t.Fatalf("write text fixture: %v", err)
	}

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "scan-test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	metadataSvc := NewMetadataService()
	imageRepo := repository.NewImageRepository(db)
	jobRepo := repository.NewJobRepository(db)
	taskPlatformSvc := NewTaskPlatformService(
		repository.NewTaskBatchRepository(db),
		repository.NewPlatformTaskRepository(db),
		jobRepo,
	)
	scanner := NewScannerService(metadataSvc, imageRepo, jobRepo, taskPlatformSvc, 1)

	result, err := scanner.Scan(context.Background(), []string{root})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if result.TotalFiles != 1 {
		t.Fatalf("TotalFiles = %d, want 1", result.TotalFiles)
	}
	if result.Imported != 1 {
		t.Fatalf("Imported = %d, want 1", result.Imported)
	}
	if result.BatchID == 0 {
		t.Fatal("expected scan result to expose batch id")
	}

	var images int
	if err := db.QueryRow("SELECT COUNT(*) FROM images").Scan(&images); err != nil {
		t.Fatalf("query images count: %v", err)
	}
	if images != 1 {
		t.Fatalf("images row count = %d, want 1", images)
	}

	batchRepo := repository.NewTaskBatchRepository(db)
	batch, err := batchRepo.FindByID(context.Background(), result.BatchID)
	if err != nil {
		t.Fatalf("FindByID(batch) error = %v", err)
	}
	if batch.SourceType != domain.TaskBatchSourceImportScan {
		t.Fatalf("batch source_type = %q, want %q", batch.SourceType, domain.TaskBatchSourceImportScan)
	}
	if batch.NewImages != 1 {
		t.Fatalf("batch new_images = %d, want 1", batch.NewImages)
	}
	if len(batch.Sources) != 1 || batch.Sources[0].SourceRoot != root {
		t.Fatalf("batch sources = %+v, want root %q", batch.Sources, root)
	}

	tasks, err := repository.NewPlatformTaskRepository(db).List(context.Background(), repository.PlatformTaskListFilter{BatchID: &result.BatchID, Limit: 10})
	if err != nil {
		t.Fatalf("List(tasks) error = %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("len(tasks) = %d, want 1", len(tasks))
	}
	if tasks[0].TaskType != domain.PlatformTaskTypeThumbnailGenerate {
		t.Fatalf("task type = %q, want %q", tasks[0].TaskType, domain.PlatformTaskTypeThumbnailGenerate)
	}
	if tasks[0].SourceType != domain.TaskBatchSourceImportScan {
		t.Fatalf("task source_type = %q, want %q", tasks[0].SourceType, domain.TaskBatchSourceImportScan)
	}

	var jobs int
	if err := db.QueryRow("SELECT COUNT(*) FROM async_jobs").Scan(&jobs); err != nil {
		t.Fatalf("query async_jobs count: %v", err)
	}
	if jobs != 0 {
		t.Fatalf("async_jobs row count = %d, want 0 before dispatcher wiring", jobs)
	}
	if result.CreatedTasks != 1 {
		t.Fatalf("CreatedTasks = %d, want 1", result.CreatedTasks)
	}
	if result.SkippedTasks != 0 {
		t.Fatalf("SkippedTasks = %d, want 0", result.SkippedTasks)
	}
	if result.PlatformStatus != domain.TaskBatchStatusRunning {
		t.Fatalf("PlatformStatus = %q, want %q", result.PlatformStatus, domain.TaskBatchStatusRunning)
	}
	if result.BatchSourceType != domain.TaskBatchSourceImportScan {
		t.Fatalf("BatchSourceType = %q, want %q", result.BatchSourceType, domain.TaskBatchSourceImportScan)
	}
	if result.SummaryLabel == "" {
		t.Fatal("expected summary label to be populated")
	}
	if result.BatchStatus == "" {
		t.Fatal("expected batch status in scan result")
	}
	if result.BatchStatus != domain.TaskBatchStatusRunning {
		t.Fatalf("BatchStatus = %q, want %q", result.BatchStatus, domain.TaskBatchStatusRunning)
	}
	if result.BatchNewImages != 1 {
		t.Fatalf("BatchNewImages = %d, want 1", result.BatchNewImages)
	}
	if result.BatchSkippedImages != 0 {
		t.Fatalf("BatchSkippedImages = %d, want 0", result.BatchSkippedImages)
	}
	if result.BatchSkippedDuplicateTasks != 0 {
		t.Fatalf("BatchSkippedDuplicateTasks = %d, want 0", result.BatchSkippedDuplicateTasks)
	}
	if result.BatchSkippedUnchanged != 0 {
		t.Fatalf("BatchSkippedUnchanged = %d, want 0", result.BatchSkippedUnchanged)
	}
	if len(result.SourceRoots) != 1 || result.SourceRoots[0] != root {
		t.Fatalf("SourceRoots = %+v, want [%q]", result.SourceRoots, root)
	}
	if len(result.PlannedTaskTypes) != 1 || result.PlannedTaskTypes[0] != domain.PlatformTaskTypeThumbnailGenerate {
		t.Fatalf("PlannedTaskTypes = %+v, want [%q]", result.PlannedTaskTypes, domain.PlatformTaskTypeThumbnailGenerate)
	}
	if result.TotalImagesInBatch != 1 {
		t.Fatalf("TotalImagesInBatch = %d, want 1", result.TotalImagesInBatch)
	}
	if result.ImportedImageIDs[0] == 0 {
		t.Fatalf("ImportedImageIDs = %+v, want non-zero image id", result.ImportedImageIDs)
	}
	if len(result.CreatedPlatformTaskIDs) != 1 || result.CreatedPlatformTaskIDs[0] == 0 {
		t.Fatalf("CreatedPlatformTaskIDs = %+v, want one task id", result.CreatedPlatformTaskIDs)
	}
	if len(result.ImportedImagePaths) != 1 || result.ImportedImagePaths[0] != imagePath {
		t.Fatalf("ImportedImagePaths = %+v, want [%q]", result.ImportedImagePaths, imagePath)
	}
}

func TestScannerCreatesNewBatchButSkipsUnchangedImageTasks(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	imagePath := filepath.Join(root, "repeat.png")
	if err := os.WriteFile(imagePath, tinyPNGFixture(), 0o600); err != nil {
		t.Fatalf("write image fixture: %v", err)
	}

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "scan-repeat.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	metadataSvc := NewMetadataService()
	imageRepo := repository.NewImageRepository(db)
	jobRepo := repository.NewJobRepository(db)
	taskRepo := repository.NewPlatformTaskRepository(db)
	taskPlatformSvc := NewTaskPlatformService(repository.NewTaskBatchRepository(db), taskRepo, jobRepo)
	scanner := NewScannerService(metadataSvc, imageRepo, jobRepo, taskPlatformSvc, 1)

	first, err := scanner.Scan(context.Background(), []string{root})
	if err != nil {
		t.Fatalf("first Scan() error = %v", err)
	}
	second, err := scanner.Scan(context.Background(), []string{root})
	if err != nil {
		t.Fatalf("second Scan() error = %v", err)
	}
	if first.BatchID == second.BatchID {
		t.Fatalf("expected unique batch ids, got %d and %d", first.BatchID, second.BatchID)
	}
	if second.Imported != 0 {
		t.Fatalf("second Imported = %d, want 0 for unchanged image", second.Imported)
	}
	if second.CreatedTasks != 0 {
		t.Fatalf("second CreatedTasks = %d, want 0", second.CreatedTasks)
	}
	if second.SkippedTasks != 1 {
		t.Fatalf("second SkippedTasks = %d, want 1", second.SkippedTasks)
	}

	secondBatch, err := repository.NewTaskBatchRepository(db).FindByID(context.Background(), second.BatchID)
	if err != nil {
		t.Fatalf("FindByID(second batch) error = %v", err)
	}
	if secondBatch.SkippedImages != 1 {
		t.Fatalf("second batch skipped_images = %d, want 1", secondBatch.SkippedImages)
	}
	if secondBatch.SkippedDuplicateTasks != 1 {
		t.Fatalf("second batch skipped_duplicate_tasks = %d, want 1", secondBatch.SkippedDuplicateTasks)
	}
	if secondBatch.SkippedUnchanged != 1 {
		t.Fatalf("second batch skipped_unchanged = %d, want 1", secondBatch.SkippedUnchanged)
	}

	allTasks, err := taskRepo.List(context.Background(), repository.PlatformTaskListFilter{Limit: 20})
	if err != nil {
		t.Fatalf("List(all tasks) error = %v", err)
	}
	if len(allTasks) != 1 {
		t.Fatalf("len(all tasks) = %d, want 1 without duplicate enqueue", len(allTasks))
	}
}

func tinyPNGFixture() []byte {
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	}
}
