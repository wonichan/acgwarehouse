package repository

import (
	"context"
	stderrors "errors"
	"time"

	pkgerrors "github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/po"
)

var (
	// ErrCollectionNotFound 表示收藏夹不存在。
	ErrCollectionNotFound = pkgerrors.New("repository: collection not found")
	// ErrCollectionForbidden 表示当前用户无权访问或管理收藏夹。
	ErrCollectionForbidden = pkgerrors.New("repository: collection forbidden")
)

// CollectionRepository 提供收藏夹持久化访问。
type CollectionRepository struct {
	readDB  *gorm.DB
	writeDB *gorm.DB
}

// NewCollectionRepository 创建收藏夹仓储。
func NewCollectionRepository(readDB *gorm.DB, writeDB *gorm.DB) *CollectionRepository {
	return &CollectionRepository{readDB: readDB, writeDB: writeDB}
}

// Create 创建用户命名收藏夹。
func (r *CollectionRepository) Create(ctx context.Context, collection do.Collection) (do.Collection, error) {
	stored := collectionToPO(collection.NormalizeForCreate(time.Now().UTC()))
	if err := r.writeDB.WithContext(ctx).Create(&stored).Error; err != nil {
		return do.Collection{}, pkgerrors.WithMessage(err, "create collection")
	}
	return collectionToDO(stored), nil
}

// ListByOwner 查询指定用户的收藏夹列表。
func (r *CollectionRepository) ListByOwner(ctx context.Context, userID int64) ([]do.Collection, error) {
	var collections []po.Collection
	err := r.readDB.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").Find(&collections).Error
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "list owner collections")
	}
	return collectionsToDO(collections), nil
}

// FindVisible 查询对访问者可见的收藏夹。
func (r *CollectionRepository) FindVisible(ctx context.Context, collectionID int64, viewerID int64) (do.Collection, error) {
	collection, err := r.findByID(ctx, r.readDB, collectionID)
	if err != nil {
		return do.Collection{}, err
	}
	if !canViewCollection(collection, viewerID) {
		return do.Collection{}, pkgerrors.WithMessage(ErrCollectionForbidden, "find visible collection")
	}
	return collectionToDO(collection), nil
}

// Update 更新 owner 自己的收藏夹名称与可见性。
func (r *CollectionRepository) Update(ctx context.Context, collection do.Collection) (do.Collection, error) {
	var updated do.Collection
	err := r.writeDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		stored, err := r.findByID(ctx, tx, collection.ID)
		if err != nil {
			return err
		}
		if stored.UserID != collection.UserID {
			return pkgerrors.WithMessage(ErrCollectionForbidden, "update collection")
		}
		stored.Name = collection.Name
		stored.Visibility = string(collection.Visibility)
		if err := tx.WithContext(ctx).Save(&stored).Error; err != nil {
			return pkgerrors.WithMessage(err, "save collection")
		}
		updated = collectionToDO(stored)
		return nil
	})
	if err != nil {
		return do.Collection{}, pkgerrors.WithMessage(err, "update collection")
	}
	return updated, nil
}

// Delete 删除 owner 自己的收藏夹及其条目，并重算受影响图片收藏人数。
func (r *CollectionRepository) Delete(ctx context.Context, collectionID int64, userID int64) error {
	return r.writeDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		stored, err := r.findByID(ctx, tx, collectionID)
		if err != nil {
			return err
		}
		if stored.UserID != userID {
			return pkgerrors.WithMessage(ErrCollectionForbidden, "delete collection")
		}
		imageIDs, err := collectionItemImageIDs(ctx, tx, collectionID)
		if err != nil {
			return err
		}
		if err := deleteCollectionItems(ctx, tx, collectionID); err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Delete(&po.Collection{}, collectionID).Error; err != nil {
			return pkgerrors.WithMessage(err, "delete collection")
		}
		now := time.Now().UTC()
		for _, imageID := range imageIDs {
			remaining, err := hasUserFavorite(ctx, tx, userID, imageID)
			if err != nil {
				return err
			}
			if !remaining {
				if err := applyFavoriteDelta(ctx, tx, userID, imageID, -1, now); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// AddItem 将图片加入 owner 自己的收藏夹，同夹同图重复调用保持幂等。
func (r *CollectionRepository) AddItem(
	ctx context.Context,
	collectionID int64,
	userID int64,
	imageID int64,
) (do.CollectionItem, error) {
	item := do.CollectionItem{CollectionID: collectionID, ImageID: imageID, CreatedAt: time.Now().UTC()}
	err := r.writeDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureCollectionOwner(ctx, tx, collectionID, userID); err != nil {
			return err
		}
		if err := ensureCollectionActiveImage(ctx, tx, imageID); err != nil {
			return err
		}
		alreadyFavorited, err := hasUserFavorite(ctx, tx, userID, imageID)
		if err != nil {
			return err
		}
		created, err := createCollectionItem(ctx, tx, item)
		if err != nil || !created {
			return err
		}
		if !alreadyFavorited {
			return applyFavoriteDelta(ctx, tx, userID, imageID, 1, item.CreatedAt)
		}
		return nil
	})
	if err != nil {
		return do.CollectionItem{}, pkgerrors.WithMessage(err, "add collection item")
	}
	return item, nil
}

// RemoveItem 从 owner 自己的收藏夹移除图片，并在最后一份收藏移除时扣减去重收藏数。
func (r *CollectionRepository) RemoveItem(ctx context.Context, collectionID int64, userID int64, imageID int64) error {
	return r.writeDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureCollectionOwner(ctx, tx, collectionID, userID); err != nil {
			return err
		}
		result := tx.WithContext(ctx).Where("collection_id = ? AND image_id = ?", collectionID, imageID).
			Delete(&po.CollectionItem{})
		if result.Error != nil {
			return pkgerrors.WithMessage(result.Error, "delete collection item")
		}
		if result.RowsAffected == 0 {
			return nil
		}
		remaining, err := hasUserFavorite(ctx, tx, userID, imageID)
		if err != nil {
			return err
		}
		if !remaining {
			return applyFavoriteDelta(ctx, tx, userID, imageID, -1, time.Now().UTC())
		}
		return nil
	})
}

// findByID 按 ID 查询收藏夹并预加载条目。
func (r *CollectionRepository) findByID(ctx context.Context, database *gorm.DB, collectionID int64) (po.Collection, error) {
	var collection po.Collection
	err := database.WithContext(ctx).Preload("Items").Where("id = ?", collectionID).First(&collection).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return po.Collection{}, pkgerrors.WithMessage(ErrCollectionNotFound, "find collection")
	}
	if err != nil {
		return po.Collection{}, pkgerrors.WithMessage(err, "find collection")
	}
	return collection, nil
}

// canViewCollection 判断访问者是否可读取收藏夹。
func canViewCollection(collection po.Collection, viewerID int64) bool {
	return collection.UserID == viewerID || collection.Visibility == string(do.CollectionVisibilityPublic)
}

// ensureCollectionOwner 确认当前用户是收藏夹 owner。
func ensureCollectionOwner(ctx context.Context, tx *gorm.DB, collectionID int64, userID int64) error {
	var collection po.Collection
	err := tx.WithContext(ctx).Where("id = ?", collectionID).First(&collection).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return pkgerrors.WithMessage(ErrCollectionNotFound, "find collection owner")
	}
	if err != nil {
		return pkgerrors.WithMessage(err, "find collection owner")
	}
	if collection.UserID != userID {
		return pkgerrors.WithMessage(ErrCollectionForbidden, "check collection owner")
	}
	return nil
}

// ensureCollectionActiveImage 确认收藏目标图片可公开展示。
func ensureCollectionActiveImage(ctx context.Context, tx *gorm.DB, imageID int64) error {
	var image po.Image
	err := activeImages(tx.WithContext(ctx)).Where("id = ?", imageID).First(&image).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return pkgerrors.WithMessage(ErrImageNotFound, "find collection image")
	}
	if err != nil {
		return pkgerrors.WithMessage(err, "find collection image")
	}
	return nil
}
