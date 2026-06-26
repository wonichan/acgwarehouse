package po

import "time"

const collectionTableName = "collection"

// Collection 表示收藏夹持久化对象。
type Collection struct {
	ID         int64            `gorm:"primaryKey"`
	UserID     int64            `gorm:"not null;index"`
	Name       string           `gorm:"size:64;not null"`
	Visibility string           `gorm:"size:16;not null;default:private;index"`
	CreatedAt  time.Time        `gorm:"not null;index"`
	Items      []CollectionItem `gorm:"foreignKey:CollectionID"`
}

// TableName 指定收藏夹表名。
func (Collection) TableName() string {
	return collectionTableName
}
