package po

import "time"

const rankingTableName = "ranking"

// Ranking 表示热榜预计算缓存持久化对象。
type Ranking struct {
	Period        string    `gorm:"size:16;primaryKey"`
	ImageID       int64     `gorm:"primaryKey;index"`
	Score         float64   `gorm:"not null;default:0;index"`
	Rank          int       `gorm:"not null;index"`
	BayesianScore float64   `gorm:"not null;default:0"`
	RatingCount   int64     `gorm:"not null;default:0"`
	FavoriteCount int64     `gorm:"not null;default:0"`
	ViewCount     int64     `gorm:"not null;default:0"`
	ComputedAt    time.Time `gorm:"not null;index"`
	Image         Image     `gorm:"foreignKey:ImageID"`
}

// TableName 指定热榜缓存表名。
func (Ranking) TableName() string {
	return rankingTableName
}
