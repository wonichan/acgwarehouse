package po

import "time"

const ratingTableName = "rating"

// Rating 表示用户评分持久化对象。
type Rating struct {
	UserID    int64     `gorm:"primaryKey;autoIncrement:false;index"`
	ImageID   int64     `gorm:"primaryKey;autoIncrement:false;index"`
	Score     int       `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null;index"`
}

// TableName 指定评分表名。
func (Rating) TableName() string {
	return ratingTableName
}
