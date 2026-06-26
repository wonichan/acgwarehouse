package do

import "time"

// Rating 表示用户对图片的评分领域对象。
type Rating struct {
	UserID    int64
	ImageID   int64
	Score     int
	UpdatedAt time.Time
}
