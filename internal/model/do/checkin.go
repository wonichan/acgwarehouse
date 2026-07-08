package do

import "time"

// CheckIn 表示用户签到领域对象。
type CheckIn struct {
	ID            int64
	UserID        int64
	CheckInDate   string
	PointsAwarded int
	CreatedAt     time.Time
}

// CheckInResult 表示一次签到尝试的结果。
type CheckInResult struct {
	CheckedIn     bool // true=本次完成首签, false=今日已签过
	PointsAwarded int  // 本次实际发放积分（首签=10, 已签=0）
}

// MonthlyCheckIns 表示月度签到查询结果。
type MonthlyCheckIns struct {
	Dates       []string // 已签到日期列表 ["2026-07-01", ...]
	TotalPoints int64    // 用户当前累计积分
}
