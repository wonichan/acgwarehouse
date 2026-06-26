package repository

import (
	"time"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/po"
)

// orderImagesByIDs 按给定 ID 顺序重排图片列表。
func orderImagesByIDs(ids []int64, images []do.Image) []do.Image {
	byID := make(map[int64]do.Image, len(images))
	for _, image := range images {
		byID[image.ID] = image
	}
	result := make([]do.Image, 0, len(images))
	for _, id := range ids {
		image, ok := byID[id]
		if ok {
			result = append(result, image)
		}
	}
	return result
}

// imageEventsToPO 将图片事件领域对象列表转换为持久化对象列表。
func imageEventsToPO(events []do.ImageEvent) []po.ImageEvent {
	result := make([]po.ImageEvent, 0, len(events))
	for _, event := range events {
		result = append(result, imageEventToPO(event))
	}
	return result
}

// imageEventToPO 将图片事件领域对象转换为持久化对象。
func imageEventToPO(event do.ImageEvent) po.ImageEvent {
	var userID *int64
	if event.UserID > 0 {
		value := event.UserID
		userID = &value
	}
	return po.ImageEvent{
		ID:        event.ID,
		ImageID:   event.ImageID,
		UserID:    userID,
		Type:      string(event.Type),
		Value:     event.Value,
		CreatedAt: event.CreatedAt.UTC(),
	}
}

// viewCountsByImage 汇总图片浏览事件的计数增量。
func viewCountsByImage(events []do.ImageEvent) map[int64]int {
	counts := make(map[int64]int)
	for _, event := range events {
		if event.Type == do.ImageEventTypeView {
			counts[event.ImageID]++
		}
	}
	return counts
}

// imagesToDO 将图片持久化对象列表转换为领域对象列表。
func imagesToDO(images []po.Image) []do.Image {
	result := make([]do.Image, 0, len(images))
	for _, image := range images {
		result = append(result, imageToDO(image))
	}
	return result
}

// imageToDO 将图片持久化对象转换为领域对象。
func imageToDO(image po.Image) do.Image {
	var deletedAt time.Time
	if image.DeletedAt != nil {
		deletedAt = image.DeletedAt.UTC()
	}
	return do.Image{
		ID:            image.ID,
		COSKey:        image.COSKey,
		Filename:      image.Filename,
		Size:          image.Size,
		LastModified:  image.LastModified.UTC(),
		Width:         image.Width,
		Height:        image.Height,
		Category:      image.Category,
		AvgScore:      image.AvgScore,
		RatingCount:   image.RatingCount,
		FavoriteCount: image.FavoriteCount,
		ViewCount:     image.ViewCount,
		Status:        do.ImageStatus(image.Status),
		DeletedAt:     deletedAt,
		CreatedAt:     image.CreatedAt.UTC(),
	}
}

// imageToPO 将图片领域对象转换为持久化对象。
func imageToPO(image do.Image) po.Image {
	var deletedAt *time.Time
	if !image.DeletedAt.IsZero() {
		value := image.DeletedAt.UTC()
		deletedAt = &value
	}
	return po.Image{
		ID:            image.ID,
		COSKey:        image.COSKey,
		Filename:      image.Filename,
		Size:          image.Size,
		LastModified:  image.LastModified,
		Width:         image.Width,
		Height:        image.Height,
		Category:      image.Category,
		AvgScore:      image.AvgScore,
		RatingCount:   image.RatingCount,
		FavoriteCount: image.FavoriteCount,
		ViewCount:     image.ViewCount,
		Status:        string(image.Status),
		DeletedAt:     deletedAt,
		CreatedAt:     image.CreatedAt,
	}
}
