package repository

import (
	"context"
	stderrors "errors"
	"fmt"
	"time"

	pkgerrors "github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/po"
	"github.com/yachiyo/acgwarehouse/internal/ports"
)

// 确保 CheckInRepository 实现 ports.CheckInRepository 接口。
var _ ports.CheckInRepository = (*CheckInRepository)(nil)

// CheckInRepository 提供用户签到持久化访问。
type CheckInRepository struct {
	readDB  *gorm.DB
	writeDB *gorm.DB
}

// NewCheckInRepository 创建签到仓储。
func NewCheckInRepository(readDB *gorm.DB, writeDB *gorm.DB) *CheckInRepository {
	return &CheckInRepository{readDB: readDB, writeDB: writeDB}
}

// CheckInToday 原子地完成当日签到。
func (r *CheckInRepository) CheckInToday(ctx context.Context, userID int64, date string, pointsAwarded int) (bool, error) {
	checkedIn := false
	err := r.writeDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing po.CheckIn
		findErr := tx.WithContext(ctx).
			Where("user_id = ? AND check_in_date = ?", userID, date).
			First(&existing).Error
		if findErr == nil {
			checkedIn = false
			return nil
		}
		if !stderrors.Is(findErr, gorm.ErrRecordNotFound) {
			return pkgerrors.WithMessage(findErr, "find existing check-in")
		}
		record := po.CheckIn{
			UserID:        userID,
			CheckInDate:   date,
			PointsAwarded: pointsAwarded,
			CreatedAt:     time.Now().UTC(),
		}
		if createErr := tx.WithContext(ctx).Create(&record).Error; createErr != nil {
			if isUniqueConstraintError(createErr) {
				checkedIn = false
				return nil
			}
			return pkgerrors.WithMessage(createErr, "create check-in")
		}
		updateResult := tx.WithContext(ctx).
			Model(&po.User{}).
			Where("id = ?", userID).
			UpdateColumn("points", gorm.Expr("points + ?", pointsAwarded))
		if updateResult.Error != nil {
			return pkgerrors.WithMessage(updateResult.Error, "increment user points")
		}
		checkedIn = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return checkedIn, nil
}

// ListByMonth 查询指定用户某年某月的全部签到记录，按日期升序。
func (r *CheckInRepository) ListByMonth(ctx context.Context, userID int64, year int, month int) ([]do.CheckIn, error) {
	startDate := fmt.Sprintf("%04d-%02d-01", year, month)
	endDate := fmt.Sprintf("%04d-%02d-31", year, month)
	var records []po.CheckIn
	err := r.readDB.WithContext(ctx).
		Where("user_id = ? AND check_in_date BETWEEN ? AND ?", userID, startDate, endDate).
		Order("check_in_date ASC").
		Find(&records).Error
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "list monthly check-ins")
	}
	result := make([]do.CheckIn, 0, len(records))
	for _, record := range records {
		result = append(result, checkInToDO(record))
	}
	return result, nil
}

// checkInToDO 将签到持久化对象转换为领域对象。
func checkInToDO(record po.CheckIn) do.CheckIn {
	return do.CheckIn{
		ID:            record.ID,
		UserID:        record.UserID,
		CheckInDate:   record.CheckInDate,
		PointsAwarded: record.PointsAwarded,
		CreatedAt:     record.CreatedAt,
	}
}
