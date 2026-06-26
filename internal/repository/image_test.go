package repository_test

import (
	"context"
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

func fixedImageTime() time.Time {
	return time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
}
