package repository

import (
	"context"
	"time"

	pkgerrors "github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/po"
)

// createCollectionItem 创建收藏夹图片条目并返回是否实际新增。
func createCollectionItem(ctx context.Context, tx *gorm.DB, item do.CollectionItem) (bool, error) {
	stored := collectionItemToPO(item)
	result := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&stored)
	if result.Error != nil {
		return false, pkgerrors.WithMessage(result.Error, "create collection item")
	}
	return result.RowsAffected > 0, nil
}

// collectionItemImageIDs 查询收藏夹内去重图片 ID。
func collectionItemImageIDs(ctx context.Context, tx *gorm.DB, collectionID int64) ([]int64, error) {
	var imageIDs []int64
	err := tx.WithContext(ctx).Model(&po.CollectionItem{}).
		Where("collection_id = ?", collectionID).
		Distinct("image_id").
		Pluck("image_id", &imageIDs).Error
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "list collection item image ids")
	}
	return imageIDs, nil
}

// deleteCollectionItems 删除收藏夹条目。
func deleteCollectionItems(ctx context.Context, tx *gorm.DB, collectionID int64) error {
	if err := tx.WithContext(ctx).Where("collection_id = ?", collectionID).Delete(&po.CollectionItem{}).Error; err != nil {
		return pkgerrors.WithMessage(err, "delete collection items")
	}
	return nil
}

// hasUserFavorite 判断用户是否仍在任一收藏夹收藏指定图片。
func hasUserFavorite(ctx context.Context, tx *gorm.DB, userID int64, imageID int64) (bool, error) {
	var count int64
	err := tx.WithContext(ctx).Model(&po.CollectionItem{}).
		Joins("JOIN collection ON collection.id = collection_item.collection_id").
		Where("collection.user_id = ? AND collection_item.image_id = ?", userID, imageID).
		Count(&count).Error
	if err != nil {
		return false, pkgerrors.WithMessage(err, "count user favorite")
	}
	return count > 0, nil
}

// applyFavoriteDelta 更新去重收藏计数并记录收藏事件。
func applyFavoriteDelta(ctx context.Context, tx *gorm.DB, userID int64, imageID int64, delta int, now time.Time) error {
	if err := updateFavoriteCount(ctx, tx, imageID, delta); err != nil {
		return err
	}
	return createFavoriteEvent(ctx, tx, userID, imageID, delta, now)
}

// updateFavoriteCount 按去重收藏用户数更新图片冗余计数。
func updateFavoriteCount(ctx context.Context, tx *gorm.DB, imageID int64, delta int) error {
	update := tx.WithContext(ctx).Model(&po.Image{}).Where("id = ?", imageID).
		UpdateColumn("favorite_count", gorm.Expr("MAX(favorite_count + ?, 0)", delta))
	if update.Error != nil {
		return pkgerrors.WithMessage(update.Error, "update favorite count")
	}
	return nil
}

// createFavoriteEvent 记录收藏或取消收藏事件。
func createFavoriteEvent(ctx context.Context, tx *gorm.DB, userID int64, imageID int64, value int, now time.Time) error {
	event := imageEventToPO(do.ImageEvent{
		ImageID:   imageID,
		UserID:    userID,
		Type:      do.ImageEventTypeFavorite,
		Value:     value,
		CreatedAt: now,
	})
	if err := tx.WithContext(ctx).Create(&event).Error; err != nil {
		return pkgerrors.WithMessage(err, "create favorite event")
	}
	return nil
}
