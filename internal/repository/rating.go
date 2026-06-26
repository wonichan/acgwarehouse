package repository

import (
	"context"
	stderrors "errors"
	"time"

	pkgerrors "github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/po"
)

var (
	// ErrRatingImageNotFound 表示评分目标图片不存在或不可公开展示。
	ErrRatingImageNotFound = ErrImageNotFound
)

// RatingRepository 提供图片评分持久化访问。
type RatingRepository struct {
	readDB  *gorm.DB
	writeDB *gorm.DB
}

// NewRatingRepository 创建评分仓储。
func NewRatingRepository(readDB *gorm.DB, writeDB *gorm.DB) *RatingRepository {
	return &RatingRepository{readDB: readDB, writeDB: writeDB}
}

// Upsert 创建或覆盖用户评分，并在同一事务内更新图片评分聚合与评分事件。
func (r *RatingRepository) Upsert(ctx context.Context, rating do.Rating) (do.Rating, error) {
	prepared := prepareRating(rating)
	err := r.writeDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureActiveImage(ctx, tx, prepared.ImageID); err != nil {
			return err
		}
		if err := upsertRating(ctx, tx, prepared); err != nil {
			return err
		}
		if err := recomputeImageRating(ctx, tx, prepared.ImageID); err != nil {
			return err
		}
		return createRatingEvent(ctx, tx, prepared)
	})
	if err != nil {
		return do.Rating{}, pkgerrors.WithMessage(err, "upsert rating")
	}
	return r.Find(ctx, prepared.UserID, prepared.ImageID)
}

// Find 查询指定用户对指定图片的评分。
func (r *RatingRepository) Find(ctx context.Context, userID int64, imageID int64) (do.Rating, error) {
	var rating po.Rating
	err := r.readDB.WithContext(ctx).Where("user_id = ? AND image_id = ?", userID, imageID).First(&rating).Error
	if err != nil {
		return do.Rating{}, pkgerrors.WithMessage(err, "find rating")
	}
	return ratingToDO(rating), nil
}

// prepareRating 补齐评分更新时间。
func prepareRating(rating do.Rating) do.Rating {
	if rating.UpdatedAt.IsZero() {
		rating.UpdatedAt = time.Now().UTC()
	} else {
		rating.UpdatedAt = rating.UpdatedAt.UTC()
	}
	return rating
}

// ensureActiveImage 确认评分目标图片可公开展示。
func ensureActiveImage(ctx context.Context, tx *gorm.DB, imageID int64) error {
	var image po.Image
	err := activeImages(tx.WithContext(ctx)).Where("id = ?", imageID).First(&image).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return pkgerrors.WithMessage(ErrRatingImageNotFound, "find rating image")
	}
	if err != nil {
		return pkgerrors.WithMessage(err, "find rating image")
	}
	return nil
}

// upsertRating 写入或覆盖用户评分。
func upsertRating(ctx context.Context, tx *gorm.DB, rating do.Rating) error {
	stored := ratingToPO(rating)
	result := tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "image_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"score",
			"updated_at",
		}),
	}).Create(&stored)
	if result.Error != nil {
		return pkgerrors.WithMessage(result.Error, "upsert rating row")
	}
	return nil
}

// recomputeImageRating 依据 rating 表重新计算图片评分均值和人数。
func recomputeImageRating(ctx context.Context, tx *gorm.DB, imageID int64) error {
	var aggregate ratingAggregate
	if err := tx.WithContext(ctx).Model(&po.Rating{}).
		Select("COALESCE(AVG(score), 0) AS avg_score, COUNT(*) AS rating_count").
		Where("image_id = ?", imageID).
		Scan(&aggregate).Error; err != nil {
		return pkgerrors.WithMessage(err, "aggregate ratings")
	}
	result := tx.WithContext(ctx).Model(&po.Image{}).Where("id = ?", imageID).Updates(map[string]interface{}{
		"avg_score":    aggregate.AvgScore,
		"rating_count": aggregate.RatingCount,
	})
	if result.Error != nil {
		return pkgerrors.WithMessage(result.Error, "update rating aggregate")
	}
	return nil
}

// createRatingEvent 记录评分事件流水。
func createRatingEvent(ctx context.Context, tx *gorm.DB, rating do.Rating) error {
	event := imageEventToPO(do.ImageEvent{
		ImageID:   rating.ImageID,
		UserID:    rating.UserID,
		Type:      do.ImageEventTypeRating,
		Value:     rating.Score,
		CreatedAt: rating.UpdatedAt,
	})
	if err := tx.WithContext(ctx).Create(&event).Error; err != nil {
		return pkgerrors.WithMessage(err, "create rating event")
	}
	return nil
}

type ratingAggregate struct {
	AvgScore    float64
	RatingCount int64
}

// ratingToPO 将评分领域对象转换为持久化对象。
func ratingToPO(rating do.Rating) po.Rating {
	return po.Rating{
		UserID:    rating.UserID,
		ImageID:   rating.ImageID,
		Score:     rating.Score,
		UpdatedAt: rating.UpdatedAt.UTC(),
	}
}

// ratingToDO 将评分持久化对象转换为领域对象。
func ratingToDO(rating po.Rating) do.Rating {
	return do.Rating{
		UserID:    rating.UserID,
		ImageID:   rating.ImageID,
		Score:     rating.Score,
		UpdatedAt: rating.UpdatedAt.UTC(),
	}
}
