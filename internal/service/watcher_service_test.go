package service

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

func TestWatcherImportsNewImageAndQueuesJob(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "watcher-test.db"))
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

	watcher, err := NewWatcherService(scanner, metadataSvc, imageRepo, jobRepo, []string{root})
	if err != nil {
		t.Fatalf("NewWatcherService() error = %v", err)
	}
	defer watcher.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watchErr := make(chan error, 1)
	go func() {
		watchErr <- watcher.Start(ctx)
	}()

	time.Sleep(200 * time.Millisecond)
	imagePath := filepath.Join(root, "watched.png")
	if err := os.WriteFile(imagePath, tinyPNGFixture(), 0o600); err != nil {
		t.Fatalf("write watched image: %v", err)
	}

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		var images int
		var jobs int
		if err := db.QueryRow("SELECT COUNT(*) FROM images").Scan(&images); err == nil && images == 1 {
			if err := db.QueryRow("SELECT COUNT(*) FROM async_jobs").Scan(&jobs); err == nil && jobs >= 1 {
				cancel()
				select {
				case err := <-watchErr:
					if err != nil && err != context.Canceled {
						t.Fatalf("watcher returned error: %v", err)
					}
				case <-time.After(500 * time.Millisecond):
				}
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	cancel()
	select {
	case err := <-watchErr:
		if err != nil && err != context.Canceled {
			t.Fatalf("watcher returned error: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
	}
	t.Fatal("watcher did not import image and queue job before timeout")
}
