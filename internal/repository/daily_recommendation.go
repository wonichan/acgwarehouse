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

const dailyRecommendationStateKey = "global"

// DailyRecommendationShuffler 打乱候选图片 ID，测试可注入确定性实现。
type DailyRecommendationShuffler func([]int64)

// IdentityDailyRecommendationShuffler 保持输入顺序，供确定性测试使用。
func IdentityDailyRecommendationShuffler(_ []int64) {}

// DailyRecommendationRepository 提供每日推荐公平随机池访问。
type DailyRecommendationRepository struct {
	readDB   *gorm.DB
	writeDB  *gorm.DB
	shuffler DailyRecommendationShuffler
}

// NewDailyRecommendationRepository 创建每日推荐仓储。
func NewDailyRecommendationRepository(readDB *gorm.DB, writeDB *gorm.DB) *DailyRecommendationRepository {
	return NewDailyRecommendationRepositoryWithShuffler(readDB, writeDB, shuffleInt64s)
}

// NewDailyRecommendationRepositoryWithShuffler 创建带指定洗牌策略的每日推荐仓储。
func NewDailyRecommendationRepositoryWithShuffler(
	readDB *gorm.DB,
	writeDB *gorm.DB,
	shuffler DailyRecommendationShuffler,
) *DailyRecommendationRepository {
	if shuffler == nil {
		shuffler = shuffleInt64s
	}
	return &DailyRecommendationRepository{readDB: readDB, writeDB: writeDB, shuffler: shuffler}
}

// GetOrCreateToday 返回指定日期的稳定全站每日推荐，必要时补齐缺失项。
func (r *DailyRecommendationRepository) GetOrCreateToday(
	ctx context.Context,
	date string,
	limit int,
	nowUTC time.Time,
) (do.DailyRecommendationList, error) {
	if limit < 1 {
		return do.DailyRecommendationList{Date: date, Images: []do.Image{}}, nil
	}
	var result do.DailyRecommendationList
	err := r.writeDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		images, err := r.getActiveTodayImages(ctx, tx, date)
		if err != nil {
			return err
		}
		if len(images) < limit {
			request := fillTodayRequest{Date: date, Needed: limit - len(images), NowUTC: nowUTC.UTC()}
			if err := r.fillToday(ctx, tx, request); err != nil {
				return err
			}
			images, err = r.getActiveTodayImages(ctx, tx, date)
			if err != nil {
				return err
			}
		}
		result = do.DailyRecommendationList{Date: date, Images: capImages(images, limit)}
		return nil
	})
	if err != nil {
		return do.DailyRecommendationList{}, pkgerrors.WithMessage(err, "get or create daily recommendations")
	}
	return result, nil
}

func (r *DailyRecommendationRepository) getActiveTodayImages(
	ctx context.Context,
	tx *gorm.DB,
	date string,
) ([]do.Image, error) {
	if err := r.deleteInactiveTodayRows(ctx, tx, date); err != nil {
		return nil, err
	}
	var rows []po.DailyRecommendation
	if err := activeDailyRecommendationRows(tx.WithContext(ctx), date).
		Preload("Image").
		Order("daily_recommendation.position asc").
		Find(&rows).Error; err != nil {
		return nil, pkgerrors.WithMessage(err, "list daily recommendation rows")
	}
	return dailyRecommendationRowsToImages(rows), nil
}

func (r *DailyRecommendationRepository) fillToday(ctx context.Context, tx *gorm.DB, request fillTodayRequest) error {
	activeIDs, err := listActiveImageIDs(ctx, tx)
	if err != nil {
		return err
	}
	if len(activeIDs) == 0 {
		return nil
	}
	state, err := r.ensureState(ctx, tx, request.NowUTC)
	if err != nil {
		return err
	}
	if err := r.syncCurrentPool(ctx, tx, state.Cycle, activeIDs, request.NowUTC); err != nil {
		return err
	}
	currentCount, err := countTodayRows(ctx, tx, request.Date)
	if err != nil {
		return err
	}
	positionBase, err := maxTodayPosition(ctx, tx, request.Date)
	if err != nil {
		return err
	}
	return r.takeRows(ctx, tx, takeRowsRequest{
		Date:         request.Date,
		Needed:       min(request.Needed, len(activeIDs)-int(currentCount)),
		PositionBase: positionBase,
		Cycle:        state.Cycle,
		NowUTC:       request.NowUTC,
		ActiveIDs:    activeIDs,
	})
}

type fillTodayRequest struct {
	Date   string
	Needed int
	NowUTC time.Time
}

type takeRowsRequest struct {
	Date         string
	Needed       int
	PositionBase int
	Cycle        int64
	NowUTC       time.Time
	ActiveIDs    []int64
}

func (r *DailyRecommendationRepository) takeRows(ctx context.Context, tx *gorm.DB, request takeRowsRequest) error {
	remaining := request.Needed
	positionBase := request.PositionBase
	cycle := request.Cycle
	for remaining > 0 {
		poolRows, err := listPoolRows(ctx, tx, cycle, request.Date)
		if err != nil {
			return err
		}
		if len(poolRows) == 0 {
			cycle++
			if err := r.replacePool(ctx, tx, cycle, request.ActiveIDs, request.NowUTC); err != nil {
				return err
			}
			if err := updateStateCycle(ctx, tx, cycle, request.NowUTC); err != nil {
				return err
			}
			continue
		}
		count := min(remaining, len(poolRows))
		for index := range count {
			row := poolRows[index]
			inserted, err := createTodayRow(ctx, tx, request.Date, row.ImageID, positionBase+1, cycle, request.NowUTC)
			if err != nil {
				return err
			}
			if !inserted {
				continue
			}
			positionBase++
			if err := deletePoolRow(ctx, tx, row.ImageID); err != nil {
				return err
			}
			remaining--
		}
	}
	return nil
}

func (r *DailyRecommendationRepository) ensureState(
	ctx context.Context,
	tx *gorm.DB,
	nowUTC time.Time,
) (po.DailyRecommendationState, error) {
	var state po.DailyRecommendationState
	err := tx.WithContext(ctx).First(&state, "key = ?", dailyRecommendationStateKey).Error
	if err == nil {
		return state, nil
	}
	if !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return po.DailyRecommendationState{}, pkgerrors.WithMessage(err, "find daily recommendation state")
	}
	state = po.DailyRecommendationState{Key: dailyRecommendationStateKey, Cycle: 1, UpdatedAt: nowUTC}
	if err := tx.WithContext(ctx).Create(&state).Error; err != nil {
		return po.DailyRecommendationState{}, pkgerrors.WithMessage(err, "create daily recommendation state")
	}
	return state, nil
}

func (r *DailyRecommendationRepository) syncCurrentPool(
	ctx context.Context,
	tx *gorm.DB,
	cycle int64,
	activeIDs []int64,
	nowUTC time.Time,
) error {
	if err := deleteInactivePoolRows(ctx, tx); err != nil {
		return err
	}
	poolIDs, err := listCurrentPoolIDs(ctx, tx, cycle)
	if err != nil {
		return err
	}
	usedIDs, err := listUsedCycleIDs(ctx, tx, cycle)
	if err != nil {
		return err
	}
	missing := missingActiveIDs(activeIDs, mergeIDSets(poolIDs, usedIDs))
	if len(missing) == 0 {
		return nil
	}
	r.shuffler(missing)
	start, err := maxPoolPosition(ctx, tx, cycle)
	if err != nil {
		return err
	}
	return createPoolRows(ctx, tx, cycle, start, missing, nowUTC)
}

func (r *DailyRecommendationRepository) replacePool(
	ctx context.Context,
	tx *gorm.DB,
	cycle int64,
	activeIDs []int64,
	nowUTC time.Time,
) error {
	if err := tx.WithContext(ctx).Where("cycle = ?", cycle).Delete(&po.DailyRecommendationPool{}).Error; err != nil {
		return pkgerrors.WithMessage(err, "delete daily recommendation pool")
	}
	ids := append([]int64(nil), activeIDs...)
	r.shuffler(ids)
	return createPoolRows(ctx, tx, cycle, 0, ids, nowUTC)
}

func (r *DailyRecommendationRepository) deleteInactiveTodayRows(ctx context.Context, tx *gorm.DB, date string) error {
	deleteQuery := `date = ? AND image_id NOT IN (
		SELECT id FROM image WHERE status = ? AND deleted_at IS NULL
	)`
	if err := tx.WithContext(ctx).
		Where(deleteQuery, date, string(do.ImageStatusActive)).
		Delete(&po.DailyRecommendation{}).Error; err != nil {
		return pkgerrors.WithMessage(err, "delete inactive daily recommendations")
	}
	return nil
}
