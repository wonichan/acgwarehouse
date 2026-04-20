package main

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

func TestRewriteThumbnailURLsToRelative(t *testing.T) {
	t.Parallel()

	db := openRewriteThumbnailTestDB(t)
	seedRewriteThumbnailTestImage(t, db, 1,
		"http://118.25.139.30:19003/acg/thumbnails/20260419/example-small.jpg",
		"http://118.25.139.30:19003/acg/thumbnails/20260419/example-large.jpg",
	)
	seedRewriteThumbnailTestImage(t, db, 2,
		"acg/thumbnails/20260419/already-small.jpg",
		"acg/thumbnails/20260419/already-large.jpg",
	)

	result, err := rewriteThumbnailURLsToRelative(db, false)
	if err != nil {
		t.Fatalf("rewriteThumbnailURLsToRelative() error = %v", err)
	}
	if result.RewrittenImages != 1 {
		t.Fatalf("RewrittenImages = %d, want 1", result.RewrittenImages)
	}

	small1, large1 := loadThumbnailURLs(t, db, 1)
	if small1 != "acg/thumbnails/20260419/example-small.jpg" {
		t.Fatalf("image 1 small = %q", small1)
	}
	if large1 != "acg/thumbnails/20260419/example-large.jpg" {
		t.Fatalf("image 1 large = %q", large1)
	}

	small2, large2 := loadThumbnailURLs(t, db, 2)
	if small2 != "acg/thumbnails/20260419/already-small.jpg" || large2 != "acg/thumbnails/20260419/already-large.jpg" {
		t.Fatalf("image 2 urls changed unexpectedly: %q %q", small2, large2)
	}
}

func TestRewriteThumbnailURLsToRelativeDryRunDoesNotPersist(t *testing.T) {
	t.Parallel()

	db := openRewriteThumbnailTestDB(t)
	seedRewriteThumbnailTestImage(t, db, 1,
		"http://118.25.139.30:19003/acg/thumbnails/20260419/example-small.jpg",
		"http://118.25.139.30:19003/acg/thumbnails/20260419/example-large.jpg",
	)

	result, err := rewriteThumbnailURLsToRelative(db, true)
	if err != nil {
		t.Fatalf("rewriteThumbnailURLsToRelative() error = %v", err)
	}
	if result.RewrittenImages != 1 {
		t.Fatalf("RewrittenImages = %d, want 1", result.RewrittenImages)
	}

	small, large := loadThumbnailURLs(t, db, 1)
	if small != "http://118.25.139.30:19003/acg/thumbnails/20260419/example-small.jpg" {
		t.Fatalf("small = %q", small)
	}
	if large != "http://118.25.139.30:19003/acg/thumbnails/20260419/example-large.jpg" {
		t.Fatalf("large = %q", large)
	}
}

func openRewriteThumbnailTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "rewrite-thumbnails.db"))
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	return db
}

func seedRewriteThumbnailTestImage(t *testing.T, db *sql.DB, id int64, smallURL, largeURL string) {
	t.Helper()

	now := time.Now()
	imagePath := filepath.Join("/images", fmt.Sprintf("test-%d.png", id))
	if _, err := db.Exec(`
		INSERT INTO images (id, path, filename, source_root, file_size, width, height, format, thumbnail_small_url, thumbnail_large_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, imagePath, "test.png", "/images", 123, 100, 100, "png", smallURL, largeURL, now, now); err != nil {
		t.Fatalf("seed image %d: %v", id, err)
	}
}

func loadThumbnailURLs(t *testing.T, db *sql.DB, id int64) (string, string) {
	t.Helper()

	var small, large string
	if err := db.QueryRow(`SELECT COALESCE(thumbnail_small_url, ''), COALESCE(thumbnail_large_url, '') FROM images WHERE id = ?`, id).Scan(&small, &large); err != nil {
		t.Fatalf("load thumbnail urls for %d: %v", id, err)
	}
	return small, large
}
