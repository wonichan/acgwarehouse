package repository_test

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/repository"
)

func Test_ImageRepository_UpsertByCOSKey_updates_existing_image_when_key_repeated(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	repo := repository.NewImageRepository(database.Read, database.Write)
	first := do.Image{
		COSKey:       "thumbnails/miku.png",
		Filename:     "miku.png",
		Size:         100,
		LastModified: fixedImageTime(),
		Width:        640,
		Height:       480,
		Category:     "",
	}
	created, err := repo.UpsertByCOSKey(context.Background(), first)
	if err != nil {
		t.Fatalf("insert image: %v", err)
	}

	// When
	updated, err := repo.UpsertByCOSKey(context.Background(), do.Image{
		COSKey:       first.COSKey,
		Filename:     "miku-new.png",
		Size:         200,
		LastModified: fixedImageTime().Add(time.Hour),
		Width:        800,
		Height:       600,
		Category:     "illustration",
	})

	// Then
	if err != nil {
		t.Fatalf("upsert image: %v", err)
	}
	if updated.ID != created.ID {
		t.Fatalf("id = %d, want existing id %d", updated.ID, created.ID)
	}
	if updated.Filename != "miku-new.png" || updated.Size != 200 || updated.Category != "illustration" {
		t.Fatalf("updated image = %#v, want new metadata", updated)
	}
	count, err := repo.CountActive(context.Background())
	if err != nil {
		t.Fatalf("count active: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func Test_ImageRepository_ListActive_excludes_soft_deleted_images(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	repo := repository.NewImageRepository(database.Read, database.Write)
	active, err := repo.UpsertByCOSKey(context.Background(), do.Image{
		COSKey:       "thumbnails/active.png",
		Filename:     "active.png",
		LastModified: fixedImageTime(),
	})
	if err != nil {
		t.Fatalf("insert active image: %v", err)
	}
	_, err = repo.UpsertByCOSKey(context.Background(), do.Image{
		COSKey:       "thumbnails/deleted.png",
		Filename:     "deleted.png",
		Status:       do.ImageStatusDeleted,
		DeletedAt:    fixedImageTime(),
		LastModified: fixedImageTime(),
	})
	if err != nil {
		t.Fatalf("insert deleted image: %v", err)
	}

	// When
	images, err := repo.ListActive(context.Background(), repository.ImageListQuery{Page: 1, Size: 10})

	// Then
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(images) != 1 || images[0].ID != active.ID {
		t.Fatalf("images = %#v, want only active image", images)
	}
}

func Test_ImageRepository_UpsertByCOSKey_keeps_deleted_image_excluded_when_sync_sees_same_key(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	repo := repository.NewImageRepository(database.Read, database.Write)
	deleted, err := repo.UpsertByCOSKey(context.Background(), do.Image{
		COSKey:       "thumbnails/deleted.png",
		Filename:     "deleted.png",
		Status:       do.ImageStatusDeleted,
		DeletedAt:    fixedImageTime(),
		LastModified: fixedImageTime(),
	})
	if err != nil {
		t.Fatalf("insert deleted image: %v", err)
	}

	// When
	updated, err := repo.UpsertByCOSKey(context.Background(), do.Image{
		COSKey:       deleted.COSKey,
		Filename:     "deleted-new.png",
		Size:         200,
		LastModified: fixedImageTime().Add(time.Hour),
		Width:        800,
		Height:       600,
		Category:     "illustration",
	})

	// Then
	if err != nil {
		t.Fatalf("sync upsert deleted image: %v", err)
	}
	if updated.Status != do.ImageStatusDeleted || updated.DeletedAt.IsZero() {
		t.Fatalf("updated image = %#v, want deleted state preserved", updated)
	}
	images, err := repo.ListActive(context.Background(), repository.ImageListQuery{Page: 1, Size: 10})
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(images) != 0 {
		t.Fatalf("images = %#v, want deleted image excluded", images)
	}
}

func Test_ImageRepository_ListActive_filters_filename_and_counts_matching_images(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	repo := repository.NewImageRepository(database.Read, database.Write)
	for _, image := range []do.Image{
		{COSKey: "thumbnails/miku.png", Filename: "miku.png", LastModified: fixedImageTime()},
		{COSKey: "thumbnails/luka.png", Filename: "luka.png", LastModified: fixedImageTime()},
		{COSKey: "thumbnails/miku-deleted.png", Filename: "miku-deleted.png", LastModified: fixedImageTime()},
	} {
		if _, err := repo.UpsertByCOSKey(context.Background(), image); err != nil {
			t.Fatalf("insert image: %v", err)
		}
	}
	deleted, err := repo.FindByCOSKey(context.Background(), "thumbnails/miku-deleted.png")
	if err != nil {
		t.Fatalf("find deleted image: %v", err)
	}
	if err := repo.SoftDelete(context.Background(), deleted.ID, fixedImageTime()); err != nil {
		t.Fatalf("soft delete image: %v", err)
	}
	query := repository.ImageListQuery{Filename: "miku", Page: 1, Size: 10, Sort: "created_at", Order: "desc"}

	// When
	images, err := repo.ListActive(context.Background(), query)
	total, countErr := repo.CountActiveByQuery(context.Background(), query)

	// Then
	if err != nil {
		t.Fatalf("list filtered images: %v", err)
	}
	if countErr != nil {
		t.Fatalf("count filtered images: %v", countErr)
	}
	if total != 1 || len(images) != 1 || images[0].Filename != "miku.png" {
		t.Fatalf("total=%d images=%#v, want only active miku.png", total, images)
	}
}

func Test_ImageRepository_ListActive_filters_by_tag_name(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	tagRepo := repository.NewTagRepository(database.Read, database.Write, imageRepo)
	mikuImage := mustCreateImage(t, imageRepo, "thumbnails/miku.png")
	lukaImage := mustCreateImage(t, imageRepo, "thumbnails/luka.png")
	mikuTag := mustCreateTag(t, tagRepo, "miku")
	lukaTag := mustCreateTag(t, tagRepo, "luka")
	if _, err := tagRepo.AssignToImages(context.Background(), []int64{mikuImage.ID}, []int64{mikuTag.ID}); err != nil {
		t.Fatalf("assign miku tag: %v", err)
	}
	if _, err := tagRepo.AssignToImages(context.Background(), []int64{lukaImage.ID}, []int64{lukaTag.ID}); err != nil {
		t.Fatalf("assign luka tag: %v", err)
	}
	query := repository.ImageListQuery{Tag: "miku", Page: 1, Size: 10}

	// When
	images, err := imageRepo.ListActive(context.Background(), query)
	total, countErr := imageRepo.CountActiveByQuery(context.Background(), query)

	// Then
	if err != nil {
		t.Fatalf("list by tag: %v", err)
	}
	if countErr != nil {
		t.Fatalf("count by tag: %v", countErr)
	}
	if total != 1 || len(images) != 1 || images[0].ID != mikuImage.ID {
		t.Fatalf("total=%d images=%#v, want only miku image", total, images)
	}
}

func Test_ImageRepository_ListActive_sorts_by_engagement_fields(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	repo := repository.NewImageRepository(database.Read, database.Write)
	for _, image := range []do.Image{
		{COSKey: "thumbnails/low.png", Filename: "low.png", AvgScore: 10, FavoriteCount: 1, ViewCount: 50, LastModified: fixedImageTime()},
		{COSKey: "thumbnails/high.png", Filename: "high.png", AvgScore: 90, FavoriteCount: 9, ViewCount: 5, LastModified: fixedImageTime()},
	} {
		if _, err := repo.UpsertByCOSKey(context.Background(), image); err != nil {
			t.Fatalf("insert image: %v", err)
		}
	}

	tests := []struct {
		name string
		sort string
		want string
	}{
		{name: "score", sort: "avg_score", want: "high.png"},
		{name: "favorites", sort: "favorite_count", want: "high.png"},
		{name: "views", sort: "view_count", want: "low.png"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			images, err := repo.ListActive(context.Background(), repository.ImageListQuery{
				Page:  1,
				Size:  10,
				Sort:  tt.sort,
				Order: "desc",
			})

			// Then
			if err != nil {
				t.Fatalf("list sorted images: %v", err)
			}
			if len(images) == 0 || images[0].Filename != tt.want {
				t.Fatalf("first image = %#v, want %s first", images, tt.want)
			}
		})
	}
}

func Test_ImageRepository_SoftDelete_hides_image_then_restore_returns_it(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	repo := repository.NewImageRepository(database.Read, database.Write)
	stored, err := repo.UpsertByCOSKey(context.Background(), do.Image{
		COSKey:       "thumbnails/restore.png",
		Filename:     "restore.png",
		LastModified: fixedImageTime(),
	})
	if err != nil {
		t.Fatalf("insert image: %v", err)
	}

	// When
	if err := repo.SoftDelete(context.Background(), stored.ID, fixedImageTime()); err != nil {
		t.Fatalf("soft delete image: %v", err)
	}
	_, hiddenErr := repo.FindActiveByID(context.Background(), stored.ID)
	restored, restoreErr := repo.Restore(context.Background(), stored.ID)

	// Then
	if !stderrors.Is(hiddenErr, repository.ErrImageNotFound) {
		t.Fatalf("hidden error = %v, want image not found", hiddenErr)
	}
	if restoreErr != nil {
		t.Fatalf("restore image: %v", restoreErr)
	}
	if restored.Status != do.ImageStatusActive || !restored.DeletedAt.IsZero() {
		t.Fatalf("restored image = %#v, want active without deleted_at", restored)
	}
}

func fixedImageTime() time.Time {
	return time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
}
