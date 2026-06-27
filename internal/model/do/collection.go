package do

import "time"

const (
	// CollectionVisibilityPrivate 表示仅收藏夹 owner 可见。
	CollectionVisibilityPrivate CollectionVisibility = "private"
	// CollectionVisibilityPublic 表示所有访问者可见。
	CollectionVisibilityPublic CollectionVisibility = "public"
)

// CollectionVisibility 定义收藏夹可见性。
type CollectionVisibility string

// Collection 表示用户命名收藏夹领域对象。
type Collection struct {
	ID         int64
	UserID     int64
	Name       string
	Visibility CollectionVisibility
	CreatedAt  time.Time
	Items      []CollectionItem
}

// CollectionItem 表示收藏夹内图片条目领域对象。
type CollectionItem struct {
	CollectionID int64
	ImageID      int64
	CreatedAt    time.Time
}

// IsValid 判断收藏夹可见性是否受支持。
func (v CollectionVisibility) IsValid() bool {
	return v == CollectionVisibilityPrivate || v == CollectionVisibilityPublic
}
