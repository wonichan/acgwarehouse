package validators

import (
	"time"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
)

func NormalizeImageForCreate(image do.Image, now time.Time) do.Image {
	if image.Status == "" {
		image.Status = do.ImageStatusActive
	}
	if image.CreatedAt.IsZero() {
		image.CreatedAt = now.UTC()
	}
	if !image.LastModified.IsZero() {
		image.LastModified = image.LastModified.UTC()
	}
	return image
}

func NormalizeCollectionForCreate(collection do.Collection, now time.Time) do.Collection {
	if collection.Visibility == "" {
		collection.Visibility = do.CollectionVisibilityPrivate
	}
	if collection.CreatedAt.IsZero() {
		collection.CreatedAt = now.UTC()
	}
	return collection
}