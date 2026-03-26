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

	if _, err := repo.Delete(context.Background(), 1, 1); err != nil {
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
	// Add tag 3 for merge tests
	if err := tagRepo.Save(context.Background(), &domain.Tag{ID: 3, PreferredLabel: "sunset", Slug: "sunset", ReviewState: "confirmed"}); err != nil {
		t.Fatalf("seed tag 3: %v", err)
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

func TestImageTagRepositoryMergeReassignsPendingTag(t *testing.T) {
	t.Parallel()

	repo := newImageTagRepositoryForTest(t)
	ctx := context.Background()

	// First create an image-tag association for (image 1, tag 1)
	obsID := int64(1)
	if err := repo.Save(ctx, &domain.ImageTag{ImageID: 1, TagID: 1, ReviewState: "pending", SourceObservationID: &obsID, Confidence: 0.9}); err != nil {
		t.Fatalf("save initial image-tag: %v", err)
	}

	// Tag 3 is already seeded in mustSeedImageTagData

	// Merge image 1's tag 1 to tag 3
	err := repo.MergeImageTag(ctx, 1, 1, 3)
	if err != nil {
		t.Fatalf("MergeImageTag() error = %v", err)
	}

	// Verify: tag 1 should no longer be on image 1
	items, err := repo.FindByImageID(ctx, 1)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	for _, item := range items {
		if item.TagID == 1 {
			t.Fatal("tag 1 should be removed from image 1 after merge")
		}
	}

	// Verify: tag 3 should now be on image 1
	found := false
	for _, item := range items {
		if item.TagID == 3 {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("tag 3 should be added to image 1 after merge")
	}
}

func TestImageTagRepositoryGetTagStatsReturnsStatistics(t *testing.T) {
	t.Parallel()

	repo := newImageTagRepositoryForTest(t)
	ctx := context.Background()

	// Add more image-tag associations with different states
	// Image 1, Tag 2 - pending with AI source (observation_id set)
	obsID := int64(1)
	if err := repo.Save(ctx, &domain.ImageTag{ImageID: 1, TagID: 2, ReviewState: "pending", SourceObservationID: &obsID, Confidence: 0.9}); err != nil {
		t.Fatalf("save image-tag: %v", err)
	}
	// Image 2, Tag 2 - confirmed without observation (manual)
	if err := repo.Save(ctx, &domain.ImageTag{ImageID: 2, TagID: 2, ReviewState: "confirmed", Confidence: 1.0}); err != nil {
		t.Fatalf("save image-tag: %v", err)
	}

	// Get stats for tag 2
	stats, err := repo.GetTagStats(ctx, 2)
	if err != nil {
		t.Fatalf("GetTagStats() error = %v", err)
	}

	// Tag 2 should have: 2 usages, 1 confirmed, 1 pending, 1 AI, 1 manual
	if stats.UsageCount != 2 {
		t.Fatalf("usage_count = %d, want 2", stats.UsageCount)
	}
	if stats.ConfirmedCount != 1 {
		t.Fatalf("confirmed_count = %d, want 1", stats.ConfirmedCount)
	}
	if stats.PendingCount != 1 {
		t.Fatalf("pending_count = %d, want 1", stats.PendingCount)
	}
	if stats.AICount != 1 {
		t.Fatalf("ai_count = %d, want 1", stats.AICount)
	}
	if stats.ManualCount != 1 {
		t.Fatalf("manual_count = %d, want 1", stats.ManualCount)
	}
}

func TestImageTagRepositorySavePersistsSource(t *testing.T) {
	t.Parallel()

	repo := newImageTagRepositoryForTest(t)
	ctx := context.Background()

	if err := repo.Save(ctx, &domain.ImageTag{ImageID: 1, TagID: 1, Source: domain.ImageTagSourceAI, ReviewState: "pending"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	items, err := repo.FindByImageID(ctx, 1)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Source != domain.ImageTagSourceAI {
		t.Fatalf("items[0].Source = %q, want %q", items[0].Source, domain.ImageTagSourceAI)
	}
}

func TestImageTagRepositorySaveDefaultsSourceToManual(t *testing.T) {
	t.Parallel()

	repo := newImageTagRepositoryForTest(t)
	ctx := context.Background()

	if err := repo.Save(ctx, &domain.ImageTag{ImageID: 2, TagID: 2, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	items, err := repo.FindByImageID(ctx, 2)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Source != domain.ImageTagSourceManual {
		t.Fatalf("items[0].Source = %q, want %q", items[0].Source, domain.ImageTagSourceManual)
	}
}

// getDBFromRepo extracts the underlying db from the repository test setup
func getDBFromRepo(t *testing.T, repo ImageTagRepository) *sql.DB {
	t.Helper()
	// We need to recreate the db access for additional seeding
	// This is a test helper, so we open a new connection
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "extra-tag.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}
	return db
}
