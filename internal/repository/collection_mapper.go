package repository

import (
	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/po"
)

// collectionsToDO 将收藏夹持久化对象列表转换为领域对象列表。
func collectionsToDO(collections []po.Collection) []do.Collection {
	result := make([]do.Collection, 0, len(collections))
	for _, collection := range collections {
		result = append(result, collectionToDO(collection))
	}
	return result
}

// collectionToDO 将收藏夹持久化对象转换为领域对象。
func collectionToDO(collection po.Collection) do.Collection {
	var coverImageID int64
	if collection.CoverImageID != nil {
		coverImageID = *collection.CoverImageID
	}
	return do.Collection{
		ID:           collection.ID,
		UserID:       collection.UserID,
		Name:         collection.Name,
		Visibility:   do.CollectionVisibility(collection.Visibility),
		CoverImageID: coverImageID,
		CreatedAt:    collection.CreatedAt.UTC(),
		Items:        collectionItemsToDO(collection.Items),
	}
}

// collectionToPO 将收藏夹领域对象转换为持久化对象。
func collectionToPO(collection do.Collection) po.Collection {
	var coverImageID *int64
	if collection.CoverImageID > 0 {
		value := collection.CoverImageID
		coverImageID = &value
	}
	return po.Collection{
		ID:           collection.ID,
		UserID:       collection.UserID,
		Name:         collection.Name,
		Visibility:   string(collection.Visibility),
		CoverImageID: coverImageID,
		CreatedAt:    collection.CreatedAt.UTC(),
	}
}

// collectionItemsToDO 将收藏夹条目列表转换为领域对象列表。
func collectionItemsToDO(items []po.CollectionItem) []do.CollectionItem {
	result := make([]do.CollectionItem, 0, len(items))
	for _, item := range items {
		if item.Image.ID == 0 {
			continue
		}
		result = append(result, collectionItemToDO(item))
	}
	return result
}

// collectionItemToDO 将收藏夹条目持久化对象转换为领域对象。
func collectionItemToDO(item po.CollectionItem) do.CollectionItem {
	return do.CollectionItem{
		CollectionID: item.CollectionID,
		ImageID:      item.ImageID,
		CreatedAt:    item.CreatedAt.UTC(),
		Image:        imageToDO(item.Image),
	}
}

// collectionItemToPO 将收藏夹条目领域对象转换为持久化对象。
func collectionItemToPO(item do.CollectionItem) po.CollectionItem {
	return po.CollectionItem{
		CollectionID: item.CollectionID,
		ImageID:      item.ImageID,
		CreatedAt:    item.CreatedAt.UTC(),
	}
}
