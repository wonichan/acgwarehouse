package do

import "time"

// Tag 表示全局共享标签领域对象。
type Tag struct {
	ID         int64
	Name       string
	UsageCount int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
