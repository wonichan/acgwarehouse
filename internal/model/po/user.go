package po

import "time"

const userTableName = "user"

// User 表示用户持久化对象。
type User struct {
	ID                 int64     `gorm:"primaryKey"`
	Username           string    `gorm:"size:32;not null;uniqueIndex"`
	PasswordHash       string    `gorm:"size:128;not null"`
	Role               string    `gorm:"size:16;not null"`
	Nickname           string    `gorm:"size:20;not null;default:''"`
	FavoriteTags       string    `gorm:"size:120;not null;default:''"`
	Bio                string    `gorm:"size:200;not null;default:''"`
	PublicProfile      bool      `gorm:"not null;default:true"`
	EmailNotifications bool      `gorm:"not null;default:true"`
	SyncCollections    bool      `gorm:"not null;default:true"`
	CreatedAt          time.Time `gorm:"not null"`
}

// TableName 指定用户表名。
func (User) TableName() string {
	return userTableName
}
