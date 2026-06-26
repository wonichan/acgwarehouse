package po

import "time"

const tagTableName = "tag"

// Tag 表示全局共享标签持久化对象。
type Tag struct {
	ID         int64     `gorm:"primaryKey"`
	Name       string    `gorm:"size:64;not null;uniqueIndex"`
	UsageCount int64     `gorm:"not null;default:0;index"`
	CreatedAt  time.Time `gorm:"not null;index"`
	UpdatedAt  time.Time `gorm:"not null"`
}

// TableName 指定标签表名。
func (Tag) TableName() string {
	return tagTableName
}
