package service

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

func TestScannerImportsImagesAndQueuesAsyncJob(t *testing.T) {
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
	scanner := NewScannerService(metadataSvc, imageRepo, jobRepo, 1)

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

	var images int
	if err := db.QueryRow("SELECT COUNT(*) FROM images").Scan(&images); err != nil {
		t.Fatalf("query images count: %v", err)
	}
	if images != 1 {
		t.Fatalf("images row count = %d, want 1", images)
	}

	var jobs int
	if err := db.QueryRow("SELECT COUNT(*) FROM async_jobs").Scan(&jobs); err != nil {
		t.Fatalf("query async_jobs count: %v", err)
	}
	if jobs < 1 {
		t.Fatalf("async_jobs row count = %d, want >= 1", jobs)
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
