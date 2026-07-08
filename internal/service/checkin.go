package service

import (
	"context"
	"time"

	pkgerrors "github.com/pkg/errors"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/ports"
)

const (
	// pointsPerDay 表示每次签到发放的积分。
	pointsPerDay = 10
)

// cstLocation 表示亚洲/上海时区（UTC+8）。
var cstLocation = time.FixedZone("CST", 8*3600)

// CheckInService 提供用户每日签到与月度签到查询能力。
type CheckInService struct {
	repo     ports.CheckInRepository
	userRepo UserRepository
}

// NewCheckInService 创建签到服务。
func NewCheckInService(repo ports.CheckInRepository, userRepo UserRepository) *CheckInService {
	return &CheckInService{repo: repo, userRepo: userRepo}
}

// CheckInToday 计算亚洲/上海时区当日日期并完成幂等签到。
func (s *CheckInService) CheckInToday(ctx context.Context, userID int64) (do.CheckInResult, error) {
	today := time.Now().In(cstLocation).Format("2006-01-02")
	checkedIn, err := s.repo.CheckInToday(ctx, userID, today, pointsPerDay)
	if err != nil {
		return do.CheckInResult{}, pkgerrors.WithMessage(err, "check in today")
	}
	awarded := 0
	if checkedIn {
		awarded = pointsPerDay
	}
	return do.CheckInResult{CheckedIn: checkedIn, PointsAwarded: awarded}, nil
}

// ListMonthly 查询月度签到记录，返回日期列表与用户累计积分。
func (s *CheckInService) ListMonthly(ctx context.Context, userID int64, year int, month int) (do.MonthlyCheckIns, error) {
	records, err := s.repo.ListByMonth(ctx, userID, year, month)
	if err != nil {
		return do.MonthlyCheckIns{}, pkgerrors.WithMessage(err, "list monthly check-ins")
	}
	dates := make([]string, 0, len(records))
	for _, record := range records {
		dates = append(dates, record.CheckInDate)
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return do.MonthlyCheckIns{}, pkgerrors.WithMessage(err, "find user points")
	}
	return do.MonthlyCheckIns{Dates: dates, TotalPoints: user.Points}, nil
}
