package po

import "time"

const imageTagTableName = "image_tag"

// ImageTag 表示图片与标签多对多关联持久化对象。
type ImageTag struct {
	ImageID   int64     `gorm:"primaryKey;autoIncrement:false;index"`
	TagID     int64     `gorm:"primaryKey;autoIncrement:false;index"`
	CreatedAt time.Time `gorm:"not null;index"`
	Image     Image     `gorm:"foreignKey:ImageID"`
	Tag       Tag       `gorm:"foreignKey:TagID"`
}

// TableName 指定图片标签关联表名。
func (ImageTag) TableName() string {
	return imageTagTableName
}
