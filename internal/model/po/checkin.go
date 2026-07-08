package po

import "time"

const checkInTableName = "check_in"

// CheckIn 表示用户签到持久化对象。
type CheckIn struct {
	ID            int64     `gorm:"primaryKey"`
	UserID        int64     `gorm:"not null;uniqueIndex:idx_user_check_in_date,priority:1"`
	CheckInDate   string    `gorm:"size:10;not null;uniqueIndex:idx_user_check_in_date,priority:2"`
	PointsAwarded int       `gorm:"not null"`
	CreatedAt     time.Time `gorm:"not null"`
}

// TableName 指定签到表名。
func (CheckIn) TableName() string {
	return checkInTableName
}
