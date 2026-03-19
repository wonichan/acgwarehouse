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

func TestImageRepositoryFindByTagIDsFiltersByStandardTags(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	// Create tags
	tagRepo := NewTagRepository(db)
	tags := []*domain.Tag{
		{PreferredLabel: "blue sky", Slug: "blue-sky", ReviewState: "confirmed", UsageCount: 5},
		{PreferredLabel: "sunset", Slug: "sunset", ReviewState: "confirmed", UsageCount: 3},
		{PreferredLabel: "ocean", Slug: "ocean", ReviewState: "confirmed", UsageCount: 1},
	}
	for _, tag := range tags {
		if err := tagRepo.Save(ctx, tag); err != nil {
			t.Fatalf("save tag: %v", err)
		}
	}

	// Create images
	images := []*domain.Image{
		{Path: "/img1.png", Filename: "img1.png", SourceRoot: "/", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Path: "/img2.png", Filename: "img2.png", SourceRoot: "/", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Path: "/img3.png", Filename: "img3.png", SourceRoot: "/", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	for _, img := range images {
		if _, err := repo.SaveImage(img); err != nil {
			t.Fatalf("save image: %v", err)
		}
	}

	// Create image-tag associations (AND semantics: img1 has both tag1 and tag2)
	imageTagRepo := NewImageTagRepository(db)
	associations := []*domain.ImageTag{
		{ImageID: images[0].ID, TagID: tags[0].ID, ReviewState: "confirmed"},
		{ImageID: images[0].ID, TagID: tags[1].ID, ReviewState: "confirmed"},
		{ImageID: images[1].ID, TagID: tags[0].ID, ReviewState: "confirmed"}, // only tag1
		{ImageID: images[2].ID, TagID: tags[2].ID, ReviewState: "confirmed"}, // tag3 only
	}
	for _, assoc := range associations {
		if err := imageTagRepo.Save(ctx, assoc); err != nil {
			t.Fatalf("save image-tag: %v", err)
		}
	}

	// Test 1: Filter by tag_ids=1,2 should return only img1 (has both tags - AND semantics)
	filtered, err := repo.FindByTagIDs(ctx, []int64{tags[0].ID, tags[1].ID}, 10, 0, "id", "asc")
	if err != nil {
		t.Fatalf("FindByTagIDs() error = %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("len(filtered) = %d, want 1 (only images with ALL requested tags)", len(filtered))
	}
	if filtered[0].ID != images[0].ID {
		t.Fatalf("filtered[0].ID = %d, want %d", filtered[0].ID, images[0].ID)
	}

	// Test 2: Filter by single tag should return images with that tag
	filteredSingle, err := repo.FindByTagIDs(ctx, []int64{tags[0].ID}, 10, 0, "id", "asc")
	if err != nil {
		t.Fatalf("FindByTagIDs() error = %v", err)
	}
	if len(filteredSingle) != 2 {
		t.Fatalf("len(filteredSingle) = %d, want 2 (images with tag1)", len(filteredSingle))
	}

	// Test 3: Empty tag_ids should return empty (no filter = no results per API contract)
	filteredEmpty, err := repo.FindByTagIDs(ctx, []int64{}, 10, 0, "id", "asc")
	if err != nil {
		t.Fatalf("FindByTagIDs() error = %v", err)
	}
	if len(filteredEmpty) != 0 {
		t.Fatalf("len(filteredEmpty) = %d, want 0", len(filteredEmpty))
	}
}

func TestImageRepositoryFindByTagIDSSupportsPagination(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	// Create tag
	tagRepo := NewTagRepository(db)
	tag := &domain.Tag{PreferredLabel: "test", Slug: "test", ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, tag); err != nil {
		t.Fatalf("save tag: %v", err)
	}

	// Create 3 images with the tag
	imageTagRepo := NewImageTagRepository(db)
	for i := 0; i < 3; i++ {
		img := &domain.Image{
			Path:       filepath.Join("/img", string(rune('a'+i))),
			Filename:   string(rune('a' + i)),
			SourceRoot: "/img",
			Format:     "png",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if _, err := repo.SaveImage(img); err != nil {
			t.Fatalf("save image: %v", err)
		}
		if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: img.ID, TagID: tag.ID, ReviewState: "confirmed"}); err != nil {
			t.Fatalf("save image-tag: %v", err)
		}
	}

	// Test pagination: limit 2, offset 0
	page1, err := repo.FindByTagIDs(ctx, []int64{tag.ID}, 2, 0, "id", "asc")
	if err != nil {
		t.Fatalf("FindByTagIDs() error = %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("len(page1) = %d, want 2", len(page1))
	}

	// Test pagination: limit 2, offset 2
	page2, err := repo.FindByTagIDs(ctx, []int64{tag.ID}, 2, 2, "id", "asc")
	if err != nil {
		t.Fatalf("FindByTagIDs() error = %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("len(page2) = %d, want 1", len(page2))
	}
}

func TestImageRepositoryCountByTagIDsReturnsCorrectCount(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	// Create tags
	tagRepo := NewTagRepository(db)
	tag1 := &domain.Tag{PreferredLabel: "tag1", Slug: "tag1", ReviewState: "confirmed"}
	tag2 := &domain.Tag{PreferredLabel: "tag2", Slug: "tag2", ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, tag1); err != nil {
		t.Fatalf("save tag1: %v", err)
	}
	if err := tagRepo.Save(ctx, tag2); err != nil {
		t.Fatalf("save tag2: %v", err)
	}

	// Create images with tags
	imageTagRepo := NewImageTagRepository(db)
	for i := 0; i < 3; i++ {
		img := &domain.Image{
			Path:       filepath.Join("/img", string(rune('a'+i))),
			Filename:   string(rune('a' + i)),
			SourceRoot: "/img",
			Format:     "png",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if _, err := repo.SaveImage(img); err != nil {
			t.Fatalf("save image: %v", err)
		}
		// img0 and img1 have tag1; img2 does not
		if i < 2 {
			if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: img.ID, TagID: tag1.ID, ReviewState: "confirmed"}); err != nil {
				t.Fatalf("save image-tag: %v", err)
			}
		}
	}

	// Count images with tag1
	count, err := repo.CountByTagIDs(ctx, []int64{tag1.ID})
	if err != nil {
		t.Fatalf("CountByTagIDs() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}

	// Count with empty tag_ids
	countEmpty, err := repo.CountByTagIDs(ctx, []int64{})
	if err != nil {
		t.Fatalf("CountByTagIDs() error = %v", err)
	}
	if countEmpty != 0 {
		t.Fatalf("countEmpty = %d, want 0", countEmpty)
	}
}

func newImageRepositoryTestDB(t *testing.T) (*sql.DB, ImageRepository) {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "image-repo.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	return db, NewImageRepository(db)
}
