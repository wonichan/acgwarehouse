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

func TestImageTagRepositorySaveCreatesAssociation(t *testing.T) {
	t.Parallel()

	repo := newImageTagRepositoryForTest(t)
	observationID := int64(1)
	imageTag := &domain.ImageTag{ImageID: 1, TagID: 1, SourceObservationID: &observationID, Confidence: 0.88, ReviewState: "pending"}

	if err := repo.Save(context.Background(), imageTag); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	items, err := repo.FindByImageID(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
}

func TestImageTagRepositoryFindByImageIDReturnsImageTags(t *testing.T) {
	t.Parallel()

	repo := newImageTagRepositoryForTest(t)
	mustSaveImageTag(t, repo, &domain.ImageTag{ImageID: 1, TagID: 1, ReviewState: "pending"})
	mustSaveImageTag(t, repo, &domain.ImageTag{ImageID: 1, TagID: 2, ReviewState: "confirmed"})

	items, err := repo.FindByImageID(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
}

func TestImageTagRepositoryFindByTagIDReturnsAssociatedImages(t *testing.T) {
	t.Parallel()

	repo := newImageTagRepositoryForTest(t)
	mustSaveImageTag(t, repo, &domain.ImageTag{ImageID: 1, TagID: 2, ReviewState: "pending"})
	mustSaveImageTag(t, repo, &domain.ImageTag{ImageID: 2, TagID: 2, ReviewState: "pending"})
	mustSaveImageTag(t, repo, &domain.ImageTag{ImageID: 3, TagID: 1, ReviewState: "pending"})

	items, err := repo.FindByTagID(context.Background(), 2, 10, 0)
	if err != nil {
		t.Fatalf("FindByTagID() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
}

func TestImageTagRepositoryUpdateReviewStateUpdatesAssociation(t *testing.T) {
	t.Parallel()

	repo := newImageTagRepositoryForTest(t)
	mustSaveImageTag(t, repo, &domain.ImageTag{ImageID: 1, TagID: 1, ReviewState: "pending"})

	if err := repo.UpdateReviewState(context.Background(), 1, 1, "confirmed"); err != nil {
		t.Fatalf("UpdateReviewState() error = %v", err)
	}

	items, err := repo.FindByImageID(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	if items[0].ReviewState != "confirmed" {
		t.Fatalf("ReviewState = %q, want confirmed", items[0].ReviewState)
	}
}

func TestImageTagRepositoryDeleteRemovesAssociation(t *testing.T) {
	t.Parallel()

	repo := newImageTagRepositoryForTest(t)
	mustSaveImageTag(t, repo, &domain.ImageTag{ImageID: 1, TagID: 1, ReviewState: "pending"})

	if err := repo.Delete(context.Background(), 1, 1); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	items, err := repo.FindByImageID(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(items))
	}
}

func newImageTagRepositoryForTest(t *testing.T) ImageTagRepository {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "image-tag-repo.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	mustSeedImageTagData(t, db)

	return NewImageTagRepository(db)
}

func mustSeedImageTagData(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO images (id, path, filename, source_root, file_size, width, height, format, created_at, updated_at)
		VALUES
			(1, '/images/1.png', '1.png', '/images', 100, 100, 100, 'png', ?, ?),
			(2, '/images/2.png', '2.png', '/images', 100, 100, 100, 'png', ?, ?),
			(3, '/images/3.png', '3.png', '/images', 100, 100, 100, 'png', ?, ?)
	`, time.Now(), time.Now(), time.Now(), time.Now(), time.Now(), time.Now())
	if err != nil {
		t.Fatalf("seed images: %v", err)
	}

	tagRepo := NewTagRepository(db)
	if err := tagRepo.Save(context.Background(), &domain.Tag{ID: 1, PreferredLabel: "blue sky", Slug: "blue-sky", UsageCount: 5}); err != nil {
		t.Fatalf("seed tag 1: %v", err)
	}
	if err := tagRepo.Save(context.Background(), &domain.Tag{ID: 2, PreferredLabel: "rain night", Slug: "rain-night", UsageCount: 9}); err != nil {
		t.Fatalf("seed tag 2: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO tag_observations (id, image_id, raw_text, confidence, evidence_type, provider, model_name, prompt_version, created_at)
		VALUES (1, 1, 'blue sky', 0.9, 'ai_generated', 'qwen', 'qwen-vl-max', 'v1', ?)
	`, time.Now())
	if err != nil {
		t.Fatalf("seed observations: %v", err)
	}
}

func mustSaveImageTag(t *testing.T, repo ImageTagRepository, imageTag *domain.ImageTag) *domain.ImageTag {
	t.Helper()

	if err := repo.Save(context.Background(), imageTag); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	return imageTag
}
