package po

import "time"

const (
	dailyRecommendationTableName      = "daily_recommendation"
	dailyRecommendationPoolTableName  = "daily_recommendation_pool"
	dailyRecommendationStateTableName = "daily_recommendation_state"
)

// DailyRecommendation 表示某自然日持久化后的全站推荐结果。
type DailyRecommendation struct {
	Date      string    `gorm:"size:10;primaryKey;uniqueIndex:idx_daily_recommendation_position"`
	ImageID   int64     `gorm:"primaryKey;index"`
	Position  int       `gorm:"not null;uniqueIndex:idx_daily_recommendation_position"`
	Cycle     int64     `gorm:"not null;index"`
	CreatedAt time.Time `gorm:"not null;index"`
	Image     Image     `gorm:"foreignKey:ImageID"`
}

// TableName 指定每日推荐结果表名。
func (DailyRecommendation) TableName() string {
	return dailyRecommendationTableName
}

// DailyRecommendationPool 表示当前公平随机周期中的剩余推荐池。
type DailyRecommendationPool struct {
	ImageID   int64     `gorm:"primaryKey"`
	Cycle     int64     `gorm:"not null;index"`
	Position  int       `gorm:"not null;index"`
	CreatedAt time.Time `gorm:"not null;index"`
	Image     Image     `gorm:"foreignKey:ImageID"`
}

// TableName 指定每日推荐池表名。
func (DailyRecommendationPool) TableName() string {
	return dailyRecommendationPoolTableName
}

// DailyRecommendationState 保存全站每日推荐当前随机周期。
type DailyRecommendationState struct {
	Key       string    `gorm:"size:32;primaryKey"`
	Cycle     int64     `gorm:"not null;default:0"`
	UpdatedAt time.Time `gorm:"not null"`
}

// TableName 指定每日推荐状态表名。
func (DailyRecommendationState) TableName() string {
	return dailyRecommendationStateTableName
}
