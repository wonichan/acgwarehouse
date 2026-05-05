package service

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

func TestSearchMaintenanceServiceRebuildsFTSIndex(t *testing.T) {
	t.Parallel()

	db := newSearchMaintenanceTestDB(t)
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	imageTagRepo := repository.NewImageTagRepository(db)
	ctx := context.Background()

	image := &domain.Image{
		Path:       "/library/cat.png",
		Filename:   "cat.png",
		SourceRoot: "/library",
		FileSize:   100,
		Width:      100,
		Height:     100,
		Format:     "png",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if _, err := imageRepo.SaveImage(image); err != nil {
		t.Fatalf("SaveImage() error = %v", err)
	}
	tag := &domain.Tag{PreferredLabel: "calico", Slug: "calico", Level: domain.TagLevelChild, ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, tag); err != nil {
		t.Fatalf("Save(tag) error = %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: image.ID, TagID: tag.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("Save(image tag) error = %v", err)
	}
	if _, err := db.Exec(`DELETE FROM images_fts`); err != nil {
		t.Fatalf("clear fts: %v", err)
	}

	if err := NewSearchMaintenanceService(db).RebuildFTSIndex(ctx); err != nil {
		t.Fatalf("RebuildFTSIndex() error = %v", err)
	}

	ids, err := repository.NewSearchRepository(db).FTSFullTextSearch(ctx, "calico", 10, 0)
	if err != nil {
		t.Fatalf("FTSFullTextSearch() error = %v", err)
	}
	if len(ids) != 1 || ids[0] != image.ID {
		t.Fatalf("FTSFullTextSearch() = %v, want [%d]", ids, image.ID)
	}
}

func newSearchMaintenanceTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "search-maintenance.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}
	return db
}
