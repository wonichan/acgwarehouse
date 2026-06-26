package po

import "time"

const imageEventTableName = "image_event"

// ImageEvent 表示图片行为事件持久化对象。
type ImageEvent struct {
	ID        int64     `gorm:"primaryKey"`
	ImageID   int64     `gorm:"not null;index"`
	UserID    *int64    `gorm:"index"`
	Type      string    `gorm:"size:16;not null;index"`
	Value     int       `gorm:"not null;default:1"`
	CreatedAt time.Time `gorm:"not null;index"`
	Image     Image     `gorm:"foreignKey:ImageID"`
}

// TableName 指定图片事件表名。
func (ImageEvent) TableName() string {
	return imageEventTableName
}
