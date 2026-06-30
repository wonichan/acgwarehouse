package repository

import (
	"context"
	"math/rand"
	"slices"
	"time"

	pkgerrors "github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/po"
)

func activeDailyRecommendationRows(database *gorm.DB, date string) *gorm.DB {
	return database.Model(&po.DailyRecommendation{}).
		Joins("JOIN image ON image.id = daily_recommendation.image_id").
		Where("daily_recommendation.date = ?", date).
		Where("image.status = ? AND image.deleted_at IS NULL", string(do.ImageStatusActive))
}

func listActiveImageIDs(ctx context.Context, tx *gorm.DB) ([]int64, error) {
	var ids []int64
	err := activeImages(tx.WithContext(ctx)).
		Select("image.id").
		Order("image.id asc").
		Pluck("id", &ids).Error
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "list active image ids")
	}
	return ids, nil
}

func countTodayRows(ctx context.Context, tx *gorm.DB, date string) (int64, error) {
	var count int64
	err := tx.WithContext(ctx).
		Model(&po.DailyRecommendation{}).
		Where("date = ?", date).
		Count(&count).Error
	if err != nil {
		return 0, pkgerrors.WithMessage(err, "count daily recommendation rows")
	}
	return count, nil
}

func maxTodayPosition(ctx context.Context, tx *gorm.DB, date string) (int, error) {
	var position int
	if err := tx.WithContext(ctx).
		Model(&po.DailyRecommendation{}).
		Where("date = ?", date).
		Select("COALESCE(MAX(position), 0)").
		Scan(&position).Error; err != nil {
		return 0, pkgerrors.WithMessage(err, "max daily recommendation position")
	}
	return position, nil
}

func listPoolRows(ctx context.Context, tx *gorm.DB, cycle int64, date string) ([]po.DailyRecommendationPool, error) {
	var rows []po.DailyRecommendationPool
	if err := tx.WithContext(ctx).
		Where("cycle = ?", cycle).
		Where("image_id NOT IN (SELECT image_id FROM daily_recommendation WHERE date = ?)", date).
		Order("position asc").
		Find(&rows).Error; err != nil {
		return nil, pkgerrors.WithMessage(err, "list daily recommendation pool")
	}
	return rows, nil
}

func listCurrentPoolIDs(ctx context.Context, tx *gorm.DB, cycle int64) (map[int64]struct{}, error) {
	var ids []int64
	if err := tx.WithContext(ctx).
		Model(&po.DailyRecommendationPool{}).
		Where("cycle = ?", cycle).
		Pluck("image_id", &ids).Error; err != nil {
		return nil, pkgerrors.WithMessage(err, "list current pool ids")
	}
	return idSet(ids), nil
}

func listUsedCycleIDs(ctx context.Context, tx *gorm.DB, cycle int64) (map[int64]struct{}, error) {
	var ids []int64
	if err := tx.WithContext(ctx).
		Model(&po.DailyRecommendation{}).
		Where("cycle = ?", cycle).
		Pluck("image_id", &ids).Error; err != nil {
		return nil, pkgerrors.WithMessage(err, "list used daily recommendation ids")
	}
	return idSet(ids), nil
}

func idSet(ids []int64) map[int64]struct{} {
	byID := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		byID[id] = struct{}{}
	}
	return byID
}

func mergeIDSets(left map[int64]struct{}, right map[int64]struct{}) map[int64]struct{} {
	merged := make(map[int64]struct{}, len(left)+len(right))
	for id := range left {
		merged[id] = struct{}{}
	}
	for id := range right {
		merged[id] = struct{}{}
	}
	return merged
}

func maxPoolPosition(ctx context.Context, tx *gorm.DB, cycle int64) (int, error) {
	var position int
	if err := tx.WithContext(ctx).
		Model(&po.DailyRecommendationPool{}).
		Where("cycle = ?", cycle).
		Select("COALESCE(MAX(position), 0)").
		Scan(&position).Error; err != nil {
		return 0, pkgerrors.WithMessage(err, "max daily recommendation pool position")
	}
	return position, nil
}

func createPoolRows(
	ctx context.Context,
	tx *gorm.DB,
	cycle int64,
	startPosition int,
	ids []int64,
	nowUTC time.Time,
) error {
	if len(ids) == 0 {
		return nil
	}
	rows := make([]po.DailyRecommendationPool, 0, len(ids))
	for index, id := range ids {
		rows = append(rows, po.DailyRecommendationPool{
			ImageID:   id,
			Cycle:     cycle,
			Position:  startPosition + index + 1,
			CreatedAt: nowUTC,
		})
	}
	if err := tx.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(rows).Error; err != nil {
		return pkgerrors.WithMessage(err, "create daily recommendation pool rows")
	}
	return nil
}

func createTodayRow(
	ctx context.Context,
	tx *gorm.DB,
	date string,
	imageID int64,
	position int,
	cycle int64,
	nowUTC time.Time,
) (bool, error) {
	row := po.DailyRecommendation{
		Date:      date,
		ImageID:   imageID,
		Position:  position,
		Cycle:     cycle,
		CreatedAt: nowUTC,
	}
	result := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
	if result.Error != nil {
		return false, pkgerrors.WithMessage(result.Error, "create daily recommendation row")
	}
	return result.RowsAffected > 0, nil
}

func deletePoolRow(ctx context.Context, tx *gorm.DB, imageID int64) error {
	if err := tx.WithContext(ctx).Where("image_id = ?", imageID).Delete(&po.DailyRecommendationPool{}).Error; err != nil {
		return pkgerrors.WithMessage(err, "delete daily recommendation pool row")
	}
	return nil
}

func deleteInactivePoolRows(ctx context.Context, tx *gorm.DB) error {
	deleteQuery := `image_id NOT IN (
		SELECT id FROM image WHERE status = ? AND deleted_at IS NULL
	)`
	if err := tx.WithContext(ctx).
		Where(deleteQuery, string(do.ImageStatusActive)).
		Delete(&po.DailyRecommendationPool{}).Error; err != nil {
		return pkgerrors.WithMessage(err, "delete inactive daily recommendation pool rows")
	}
	return nil
}

func updateStateCycle(ctx context.Context, tx *gorm.DB, cycle int64, nowUTC time.Time) error {
	updates := map[string]interface{}{"cycle": cycle, "updated_at": nowUTC}
	if err := tx.WithContext(ctx).
		Model(&po.DailyRecommendationState{}).
		Where("key = ?", dailyRecommendationStateKey).
		Updates(updates).Error; err != nil {
		return pkgerrors.WithMessage(err, "update daily recommendation state")
	}
	return nil
}

func missingActiveIDs(activeIDs []int64, poolIDs map[int64]struct{}) []int64 {
	missing := make([]int64, 0)
	for _, id := range activeIDs {
		if _, ok := poolIDs[id]; !ok {
			missing = append(missing, id)
		}
	}
	return missing
}

func dailyRecommendationRowsToImages(rows []po.DailyRecommendation) []do.Image {
	images := make([]do.Image, 0, len(rows))
	for _, row := range rows {
		images = append(images, imageToDO(row.Image))
	}
	return images
}

func capImages(images []do.Image, limit int) []do.Image {
	if len(images) <= limit {
		return images
	}
	return images[:limit]
}

func shuffleInt64s(ids []int64) {
	slices.Sort(ids)
	rand.Shuffle(len(ids), func(left int, right int) {
		ids[left], ids[right] = ids[right], ids[left]
	})
}
