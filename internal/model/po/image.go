package po

import "time"

const imageTableName = "image"

// Image 表示图片持久化对象。
type Image struct {
	ID            int64      `gorm:"primaryKey"`
	COSKey        string     `gorm:"column:cos_key;size:512;not null;uniqueIndex"`
	Filename      string     `gorm:"size:255;not null;index"`
	Size          int64      `gorm:"not null;default:0;index"`
	LastModified  time.Time  `gorm:"not null;index"`
	Width         int        `gorm:"not null;default:0"`
	Height        int        `gorm:"not null;default:0"`
	Category      string     `gorm:"size:64;not null;default:'';index"`
	AvgScore      float64    `gorm:"not null;default:0"`
	RatingCount   int64      `gorm:"not null;default:0"`
	FavoriteCount int64      `gorm:"not null;default:0"`
	ViewCount     int64      `gorm:"not null;default:0"`
	Status        string     `gorm:"size:16;not null;default:active;index"`
	DeletedAt     *time.Time `gorm:"index"`
	CreatedAt     time.Time  `gorm:"not null;index"`
}

// TableName 指定图片表名。
func (Image) TableName() string {
	return imageTableName
}
