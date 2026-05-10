package repository

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

func TestImageMoveRepositoryFindBySourceDirsAndTag(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()
	tagRepo := NewTagRepository(db)
	imageTagRepo := NewImageTagRepository(db)

	tag := &domain.Tag{PreferredLabel: "target", Slug: "target", ReviewState: "confirmed"}
	otherTag := &domain.Tag{PreferredLabel: "other", Slug: "other", ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, tag); err != nil {
		t.Fatalf("save tag: %v", err)
	}
	if err := tagRepo.Save(ctx, otherTag); err != nil {
		t.Fatalf("save other tag: %v", err)
	}

	sourceA := filepath.Join(t.TempDir(), "source-a")
	sourceB := filepath.Join(t.TempDir(), "source-b")
	sourcePrefixSibling := sourceA + "2"

	matchingA := saveImageMoveRepoImage(t, repo, filepath.Join(sourceA, "a.png"), sourceA)
	matchingB := saveImageMoveRepoImage(t, repo, filepath.Join(sourceB, "b.png"), sourceB)
	wrongTag := saveImageMoveRepoImage(t, repo, filepath.Join(sourceA, "wrong-tag.png"), sourceA)
	wrongDir := saveImageMoveRepoImage(t, repo, filepath.Join(sourcePrefixSibling, "sibling.png"), sourcePrefixSibling)

	for _, imageID := range []int64{matchingA.ID, matchingB.ID, wrongDir.ID} {
		if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: imageID, TagID: tag.ID, ReviewState: domain.ReviewStateConfirmed}); err != nil {
			t.Fatalf("save target image-tag: %v", err)
		}
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: wrongTag.ID, TagID: otherTag.ID, ReviewState: domain.ReviewStateConfirmed}); err != nil {
		t.Fatalf("save other image-tag: %v", err)
	}

	got, err := repo.FindBySourceDirsAndTag(ctx, []string{sourceA, sourceB}, tag.ID, 10, 0)
	if err != nil {
		t.Fatalf("FindBySourceDirsAndTag() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].ID != matchingA.ID || got[1].ID != matchingB.ID {
		t.Fatalf("got IDs = [%d %d], want [%d %d]", got[0].ID, got[1].ID, matchingA.ID, matchingB.ID)
	}

	count, err := repo.CountBySourceDirsAndTag(ctx, []string{sourceA, sourceB}, tag.ID)
	if err != nil {
		t.Fatalf("CountBySourceDirsAndTag() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
}

func TestImageMoveRepositoryUpdateImageLocationOnlyTouchesTargetImage(t *testing.T) {
	t.Parallel()

	_, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()
	oldRoot := filepath.Join(t.TempDir(), "old")
	newRoot := filepath.Join(t.TempDir(), "new")

	target := saveImageMoveRepoImage(t, repo, filepath.Join(oldRoot, "a.png"), oldRoot)
	other := saveImageMoveRepoImage(t, repo, filepath.Join(oldRoot, "b.png"), oldRoot)

	newPath := filepath.Join(newRoot, "a.png")
	if err := repo.UpdateImageLocation(ctx, target.ID, newPath, "a.png", newRoot); err != nil {
		t.Fatalf("UpdateImageLocation() error = %v", err)
	}

	updated, err := repo.FindByID(target.ID)
	if err != nil {
		t.Fatalf("FindByID(target) error = %v", err)
	}
	if updated.Path != newPath || updated.Filename != "a.png" || updated.SourceRoot != newRoot {
		t.Fatalf("updated location = (%q, %q, %q), want (%q, %q, %q)", updated.Path, updated.Filename, updated.SourceRoot, newPath, "a.png", newRoot)
	}

	unchanged, err := repo.FindByID(other.ID)
	if err != nil {
		t.Fatalf("FindByID(other) error = %v", err)
	}
	if unchanged.Path != other.Path || unchanged.SourceRoot != oldRoot {
		t.Fatalf("other image changed: path=%q source_root=%q", unchanged.Path, unchanged.SourceRoot)
	}
}

func TestImageMoveHistoryRepositoryPersistsBatchAndItems(t *testing.T) {
	t.Parallel()

	db, _ := newImageRepositoryTestDB(t)
	repo := NewImageMoveHistoryRepository(db)
	ctx := context.Background()
	batch := &domain.ImageMoveBatch{
		TagID:            7,
		SourceDirs:       []string{`E:\src`},
		TargetDir:        `E:\dst`,
		ConflictStrategy: domain.ImageMoveConflictRename,
		TotalMatched:     1,
		Status:           domain.ImageMoveBatchStatusRunning,
	}
	if err := repo.CreateImageMoveBatch(ctx, batch); err != nil {
		t.Fatalf("CreateImageMoveBatch() error = %v", err)
	}
	if err := repo.AddImageMoveItem(ctx, batch.ID, domain.ImageMoveItem{
		ImageID:    10,
		Filename:   "a.png",
		SourcePath: filepath.Join(`E:\src`, "a.png"),
		TargetPath: filepath.Join(`E:\dst`, "a.png"),
		Status:     domain.ImageMoveStatusFailed,
		Reason:     domain.ImageMoveReasonMoveFailed,
	}); err != nil {
		t.Fatalf("AddImageMoveItem() error = %v", err)
	}
	batch.Failed = 1
	batch.Status = domain.ImageMoveBatchStatusFailed
	if err := repo.UpdateImageMoveBatch(ctx, batch); err != nil {
		t.Fatalf("UpdateImageMoveBatch() error = %v", err)
	}

	got, err := repo.FindImageMoveBatch(ctx, batch.ID)
	if err != nil {
		t.Fatalf("FindImageMoveBatch() error = %v", err)
	}
	if got.Status != domain.ImageMoveBatchStatusFailed || got.Progress.Processed != 1 || len(got.Items) != 1 {
		t.Fatalf("batch = %+v", got)
	}
	if !got.Items[0].Retryable {
		t.Fatalf("item retryable = false, want true")
	}
	if got.FinishedAt == nil {
		t.Fatalf("FinishedAt = nil, want timestamp")
	}
}

func saveImageMoveRepoImage(t *testing.T, repo ImageRepository, path, sourceRoot string) *domain.Image {
	t.Helper()
	image := &domain.Image{
		Path:       path,
		Filename:   filepath.Base(path),
		SourceRoot: sourceRoot,
		FileSize:   10,
		Format:     "png",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if _, err := repo.SaveImage(image); err != nil {
		t.Fatalf("save image %q: %v", path, err)
	}
	return image
}
