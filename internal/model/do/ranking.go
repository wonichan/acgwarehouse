package do

import "time"

const (
	// RankingPeriodDay 表示最近 24 小时榜单。
	RankingPeriodDay RankingPeriod = "day"
	// RankingPeriodWeek 表示最近 7 天榜单。
	RankingPeriodWeek RankingPeriod = "week"
	// RankingPeriodMonth 表示最近 30 天榜单。
	RankingPeriodMonth RankingPeriod = "month"
)

const (
	rankingPeriodDayDefaultSize   = 20
	rankingPeriodWeekDefaultSize  = 50
	rankingPeriodMonthDefaultSize = 100
)

// RankingPeriod 定义热榜时间窗口。
type RankingPeriod string

// Ranking 表示单张图片的热榜缓存领域对象。
type Ranking struct {
	Period        RankingPeriod
	Image         Image
	Score         float64
	Rank          int
	BayesianScore float64
	RatingCount   int64
	FavoriteCount int64
	ViewCount     int64
	ComputedAt    time.Time
}

// RankingMetrics 表示热榜窗口聚合指标。
type RankingMetrics struct {
	ImageID       int64
	SumScore      float64
	RatingCount   int64
	FavoriteCount int64
	ViewCount     int64
}

// RankingScore 表示待写入缓存的热榜分值。
type RankingScore struct {
	Period        RankingPeriod
	ImageID       int64
	Score         float64
	Rank          int
	BayesianScore float64
	RatingCount   int64
	FavoriteCount int64
	ViewCount     int64
	ComputedAt    time.Time
}

// RankingListResult 表示热榜缓存列表结果。
type RankingListResult struct {
	Total int64
	Page  int
	Size  int
	List  []Ranking
}

// IsValid 判断热榜周期是否受支持。
func (p RankingPeriod) IsValid() bool {
	return p == RankingPeriodDay || p == RankingPeriodWeek || p == RankingPeriodMonth
}

// Window 返回热榜周期对应的时间窗口长度。
func (p RankingPeriod) Window() time.Duration {
	switch p {
	case RankingPeriodDay:
		return 24 * time.Hour
	case RankingPeriodWeek:
		return 7 * 24 * time.Hour
	case RankingPeriodMonth:
		return 30 * 24 * time.Hour
	default:
		return 0
	}
}

// DefaultSize 返回热榜周期默认展示数量。
func (p RankingPeriod) DefaultSize() int {
	switch p {
	case "", RankingPeriodDay:
		return rankingPeriodDayDefaultSize
	case RankingPeriodWeek:
		return rankingPeriodWeekDefaultSize
	case RankingPeriodMonth:
		return rankingPeriodMonthDefaultSize
	default:
		return rankingPeriodDayDefaultSize
	}
}

// RankingPeriods 返回所有受支持的热榜周期。
func RankingPeriods() []RankingPeriod {
	return []RankingPeriod{RankingPeriodDay, RankingPeriodWeek, RankingPeriodMonth}
}
