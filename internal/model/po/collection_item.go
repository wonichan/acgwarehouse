package po

import "time"

const collectionItemTableName = "collection_item"

// CollectionItem 表示收藏夹图片条目持久化对象。
type CollectionItem struct {
	CollectionID int64      `gorm:"primaryKey;autoIncrement:false"`
	ImageID      int64      `gorm:"primaryKey;autoIncrement:false;index"`
	CreatedAt    time.Time  `gorm:"not null;index"`
	Collection   Collection `gorm:"foreignKey:CollectionID"`
	Image        Image      `gorm:"foreignKey:ImageID"`
}

// TableName 指定收藏夹条目表名。
func (CollectionItem) TableName() string {
	return collectionItemTableName
}
