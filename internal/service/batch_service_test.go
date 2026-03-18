package service

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

type batchServiceTestEnv struct {
	svc            *BatchService
	imageRepo      repository.ImageRepository
	tagRepo        repository.TagRepository
	imageTagRepo   repository.ImageTagRepository
	collectionRepo repository.CollectionRepository
}

func setupBatchServiceTest(t *testing.T) *batchServiceTestEnv {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "batch_service_test_*.db")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	t.Cleanup(func() { _ = os.Remove(tmpPath) })

	db, err := sql.Open("sqlite3", tmpPath)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}

	env := &batchServiceTestEnv{
		imageRepo:      repository.NewImageRepository(db),
		tagRepo:        repository.NewTagRepository(db),
		imageTagRepo:   repository.NewImageTagRepository(db),
		collectionRepo: repository.NewCollectionRepository(db),
	}
	env.svc = NewBatchService(env.imageRepo, env.tagRepo, env.imageTagRepo, env.collectionRepo)

	return env
}

func saveBatchTestImage(t *testing.T, imageRepo repository.ImageRepository, filename string) *domain.Image {
	t.Helper()
	now := time.Now()
	image := &domain.Image{
		Path:       "/batch/" + filename,
		Filename:   filename,
		SourceRoot: "/batch",
		FileSize:   128,
		Width:      100,
		Height:     100,
		Format:     "png",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := imageRepo.SaveImage(image); err != nil {
		t.Fatalf("save image: %v", err)
	}
	return image
}

func saveBatchTestTag(t *testing.T, tagRepo repository.TagRepository, label string) *domain.Tag {
	t.Helper()
	tag := &domain.Tag{PreferredLabel: label, Slug: label, ReviewState: "confirmed", UsageCount: 0}
	if err := tagRepo.Save(context.Background(), tag); err != nil {
		t.Fatalf("save tag: %v", err)
	}
	return tag
}

func TestBatchService_BatchAddAndRemoveTags(t *testing.T) {
	t.Parallel()
	env := setupBatchServiceTest(t)
	ctx := context.Background()

	img1 := saveBatchTestImage(t, env.imageRepo, "one.png")
	img2 := saveBatchTestImage(t, env.imageRepo, "two.png")
	tag1 := saveBatchTestTag(t, env.tagRepo, "tag-a")
	tag2 := saveBatchTestTag(t, env.tagRepo, "tag-b")

	added, err := env.svc.BatchAddTags(ctx, []int64{img1.ID, img2.ID}, []int64{tag1.ID, tag2.ID})
	if err != nil {
		t.Fatalf("BatchAddTags() error = %v", err)
	}
	// BatchAddTags returns count of images processed, not total tags added
	if added != 2 {
		t.Fatalf("BatchAddTags() added = %d, want 2", added)
	}

	img1Tags, err := env.imageTagRepo.FindByImageID(ctx, img1.ID)
	if err != nil {
		t.Fatalf("FindByImageID(img1) error = %v", err)
	}
	if len(img1Tags) != 2 {
		t.Fatalf("len(img1 tags) = %d, want 2", len(img1Tags))
	}

	removed, err := env.svc.BatchRemoveTags(ctx, []int64{img1.ID, img2.ID}, []int64{tag2.ID})
	if err != nil {
		t.Fatalf("BatchRemoveTags() error = %v", err)
	}
	if removed != 2 {
		t.Fatalf("BatchRemoveTags() removed = %d, want 2", removed)
	}

	img2Tags, err := env.imageTagRepo.FindByImageID(ctx, img2.ID)
	if err != nil {
		t.Fatalf("FindByImageID(img2) error = %v", err)
	}
	if len(img2Tags) != 1 || img2Tags[0].TagID != tag1.ID {
		t.Fatalf("remaining img2 tags = %+v, want only tag1", img2Tags)
	}
}

func TestBatchService_BatchMoveAndRemoveFromCollection(t *testing.T) {
	t.Parallel()
	env := setupBatchServiceTest(t)
	ctx := context.Background()

	collection := &domain.Collection{Name: "favorites"}
	if err := env.collectionRepo.Save(ctx, collection); err != nil {
		t.Fatalf("Save(collection) error = %v", err)
	}

	img1 := saveBatchTestImage(t, env.imageRepo, "one.png")
	time.Sleep(10 * time.Millisecond)
	img2 := saveBatchTestImage(t, env.imageRepo, "two.png")

	moved, err := env.svc.BatchMoveToCollection(ctx, []int64{img1.ID, img2.ID}, collection.ID)
	if err != nil {
		t.Fatalf("BatchMoveToCollection() error = %v", err)
	}
	if moved != 2 {
		t.Fatalf("BatchMoveToCollection() moved = %d, want 2", moved)
	}

	updated, err := env.collectionRepo.FindByID(ctx, collection.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if updated.ImageCount != 2 {
		t.Fatalf("ImageCount = %d, want 2", updated.ImageCount)
	}
	if updated.CoverImageID == nil || *updated.CoverImageID != img2.ID {
		t.Fatalf("CoverImageID = %v, want %d", updated.CoverImageID, img2.ID)
	}

	removed, err := env.svc.BatchRemoveFromCollection(ctx, []int64{img2.ID}, collection.ID)
	if err != nil {
		t.Fatalf("BatchRemoveFromCollection() error = %v", err)
	}
	if removed != 1 {
		t.Fatalf("BatchRemoveFromCollection() removed = %d, want 1", removed)
	}

	updated, err = env.collectionRepo.FindByID(ctx, collection.ID)
	if err != nil {
		t.Fatalf("FindByID() after remove error = %v", err)
	}
	if updated.ImageCount != 1 {
		t.Fatalf("ImageCount after remove = %d, want 1", updated.ImageCount)
	}
	if updated.CoverImageID == nil || *updated.CoverImageID != img1.ID {
		t.Fatalf("CoverImageID after remove = %v, want %d", updated.CoverImageID, img1.ID)
	}
}

func TestBatchService_BatchDeleteImages(t *testing.T) {
	t.Parallel()
	env := setupBatchServiceTest(t)
	ctx := context.Background()

	collection := &domain.Collection{Name: "delete-target"}
	if err := env.collectionRepo.Save(ctx, collection); err != nil {
		t.Fatalf("Save(collection) error = %v", err)
	}

	img1 := saveBatchTestImage(t, env.imageRepo, "one.png")
	img2 := saveBatchTestImage(t, env.imageRepo, "two.png")

	if err := env.collectionRepo.AddImage(ctx, collection.ID, img1.ID); err != nil {
		t.Fatalf("AddImage(img1) error = %v", err)
	}
	if err := env.collectionRepo.AddImage(ctx, collection.ID, img2.ID); err != nil {
		t.Fatalf("AddImage(img2) error = %v", err)
	}

	deleted, err := env.svc.BatchDeleteImages(ctx, []int64{img1.ID})
	if err != nil {
		t.Fatalf("BatchDeleteImages() error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("BatchDeleteImages() deleted = %d, want 1", deleted)
	}

	if _, err := env.imageRepo.FindByID(img1.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("FindByID(deleted) error = %v, want %v", err, sql.ErrNoRows)
	}

	// Note: BatchDeleteImages does not auto-update collection ImageCount
	// Collection membership cleanup is handled separately or via cascade
}
