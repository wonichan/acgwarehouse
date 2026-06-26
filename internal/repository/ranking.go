package repository

import (
	"context"
	"sort"
	"time"

	pkgerrors "github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/po"
)

// RankingListQuery 定义热榜缓存列表查询条件。
type RankingListQuery struct {
	Page int
	Size int
}

// RankingRepository 提供热榜聚合与缓存访问。
type RankingRepository struct {
	readDB  *gorm.DB
	writeDB *gorm.DB
}

// NewRankingRepository 创建热榜仓储。
func NewRankingRepository(readDB *gorm.DB, writeDB *gorm.DB) *RankingRepository {
	return &RankingRepository{readDB: readDB, writeDB: writeDB}
}

// AggregateMetrics 按时间窗口聚合可展示图片的行为事件指标。
func (r *RankingRepository) AggregateMetrics(
	ctx context.Context,
	period do.RankingPeriod,
	from time.Time,
) ([]do.RankingMetrics, error) {
	if !period.IsValid() {
		return nil, pkgerrors.New("repository: invalid ranking period")
	}
	var rows []rankingMetricRow
	query := activeImages(r.readDB.WithContext(ctx)).
		Select(rankingMetricSelect()).
		Joins("JOIN image_event ON image_event.image_id = image.id").
		Where("image_event.created_at >= ?", from.UTC()).
		Group("image.id")
	if err := query.Scan(&rows).Error; err != nil {
		return nil, pkgerrors.WithMessage(err, "aggregate ranking metrics")
	}
	return rankingMetricRowsToDO(rows), nil
}

// ReplacePeriod 覆盖写入指定周期的热榜缓存。
func (r *RankingRepository) ReplacePeriod(ctx context.Context, period do.RankingPeriod, scores []do.RankingScore) error {
	if !period.IsValid() {
		return pkgerrors.New("repository: invalid ranking period")
	}
	return r.writeDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("period = ?", string(period)).Delete(&po.Ranking{}).Error; err != nil {
			return pkgerrors.WithMessage(err, "delete ranking period")
		}
		if len(scores) == 0 {
			return nil
		}
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(rankingScoresToPO(scores)).Error; err != nil {
			return pkgerrors.WithMessage(err, "create ranking rows")
		}
		return nil
	})
}

// ListCached 查询指定周期的热榜缓存并关联图片元数据。
func (r *RankingRepository) ListCached(
	ctx context.Context,
	period do.RankingPeriod,
	query RankingListQuery,
) (do.RankingListResult, error) {
	if !period.IsValid() {
		return do.RankingListResult{}, pkgerrors.New("repository: invalid ranking period")
	}
	var total int64
	base := activeRankingRows(r.readDB.WithContext(ctx), period)
	if err := base.Count(&total).Error; err != nil {
		return do.RankingListResult{}, pkgerrors.WithMessage(err, "count cached rankings")
	}
	var rankings []po.Ranking
	listQuery := activeRankingRows(r.readDB.WithContext(ctx), period).
		Preload("Image").
		Order("ranking.rank asc")
	if query.Size > 0 {
		listQuery = listQuery.Limit(query.Size).Offset(rankingOffset(query))
	}
	if err := listQuery.Find(&rankings).Error; err != nil {
		return do.RankingListResult{}, pkgerrors.WithMessage(err, "list cached rankings")
	}
	return do.RankingListResult{
		Total: total,
		Page:  query.Page,
		Size:  query.Size,
		List:  rankingsToDO(rankings),
	}, nil
}

// rankingMetricSelect 返回窗口指标聚合 SQL 片段。
func rankingMetricSelect() string {
	return `image.id AS image_id,
		COALESCE(SUM(CASE WHEN image_event.type = 'rating' THEN image_event.value ELSE 0 END), 0) AS sum_score,
		COALESCE(SUM(CASE WHEN image_event.type = 'rating' THEN 1 ELSE 0 END), 0) AS rating_count,
		COALESCE(COUNT(DISTINCT CASE WHEN image_event.type = 'favorite' AND image_event.value > 0 THEN image_event.user_id END), 0) AS favorite_count,
		COALESCE(SUM(CASE WHEN image_event.type = 'view' THEN 1 ELSE 0 END), 0) AS view_count`
}

// activeRankingRows 限定热榜缓存对应图片仍可公开展示。
func activeRankingRows(database *gorm.DB, period do.RankingPeriod) *gorm.DB {
	return database.Model(&po.Ranking{}).
		Joins("JOIN image ON image.id = ranking.image_id").
		Where("ranking.period = ?", string(period)).
		Where("image.status = ? AND image.deleted_at IS NULL", string(do.ImageStatusActive))
}

// rankingOffset 计算热榜分页偏移量。
func rankingOffset(query RankingListQuery) int {
	if query.Page < 1 || query.Size < 1 {
		return 0
	}
	return (query.Page - 1) * query.Size
}

// rankingMetricRowsToDO 将指标查询行转换为领域指标。
func rankingMetricRowsToDO(rows []rankingMetricRow) []do.RankingMetrics {
	metrics := make([]do.RankingMetrics, 0, len(rows))
	for _, row := range rows {
		metrics = append(metrics, do.RankingMetrics{
			ImageID:       row.ImageID,
			SumScore:      row.SumScore,
			RatingCount:   row.RatingCount,
			FavoriteCount: row.FavoriteCount,
			ViewCount:     row.ViewCount,
		})
	}
	return metrics
}

// rankingScoresToPO 将热榜分值转换为持久化对象并保证顺序确定。
func rankingScoresToPO(scores []do.RankingScore) []po.Ranking {
	sortedScores := append([]do.RankingScore(nil), scores...)
	sort.SliceStable(sortedScores, func(left int, right int) bool {
		return sortedScores[left].Rank < sortedScores[right].Rank
	})
	rows := make([]po.Ranking, 0, len(sortedScores))
	for _, score := range sortedScores {
		rows = append(rows, rankingScoreToPO(score))
	}
	return rows
}

// rankingScoreToPO 将单个热榜分值转换为持久化对象。
func rankingScoreToPO(score do.RankingScore) po.Ranking {
	return po.Ranking{
		Period:        string(score.Period),
		ImageID:       score.ImageID,
		Score:         score.Score,
		Rank:          score.Rank,
		BayesianScore: score.BayesianScore,
		RatingCount:   score.RatingCount,
		FavoriteCount: score.FavoriteCount,
		ViewCount:     score.ViewCount,
		ComputedAt:    score.ComputedAt.UTC(),
	}
}

// rankingsToDO 将热榜缓存持久化对象转换为领域对象。
func rankingsToDO(rankings []po.Ranking) []do.Ranking {
	result := make([]do.Ranking, 0, len(rankings))
	for _, ranking := range rankings {
		result = append(result, rankingToDO(ranking))
	}
	return result
}

// rankingToDO 将热榜缓存持久化对象转换为领域对象。
func rankingToDO(ranking po.Ranking) do.Ranking {
	return do.Ranking{
		Period:        do.RankingPeriod(ranking.Period),
		Image:         imageToDO(ranking.Image),
		Score:         ranking.Score,
		Rank:          ranking.Rank,
		BayesianScore: ranking.BayesianScore,
		RatingCount:   ranking.RatingCount,
		FavoriteCount: ranking.FavoriteCount,
		ViewCount:     ranking.ViewCount,
		ComputedAt:    ranking.ComputedAt.UTC(),
	}
}

type rankingMetricRow struct {
	ImageID       int64
	SumScore      float64
	RatingCount   int64
	FavoriteCount int64
	ViewCount     int64
}
