// Package ports 定义核心仓储和外部依赖的接口契约。
// 这些接口仅依赖领域对象（do 包），不引入任何基础设施细节（如 GORM、数据库等）。
// Service 层通过这些接口与 Repository 层解耦，实现依赖反转。
package ports

import (
	"context"
	"errors"
	"time"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
)

// 领域错误定义，与具体实现解耦。
var (
	// ErrCollectionNotFound 表示收藏夹不存在。
	ErrCollectionNotFound = errors.New("collection not found")
	// ErrCollectionForbidden 表示当前用户无权访问或管理收藏夹。
	ErrCollectionForbidden = errors.New("collection forbidden")
	// ErrImageNotFound 表示图片不存在。
	ErrImageNotFound = errors.New("image not found")
	// ErrTagNotFound 表示标签不存在。
	ErrTagNotFound = errors.New("tag not found")
	// ErrUserNotFound 表示用户不存在。
	ErrUserNotFound = errors.New("user not found")
	// ErrUsernameExists 表示用户名已存在。
	ErrUsernameExists = errors.New("username exists")
)

// ImageRepository 定义图片持久化访问能力。
type ImageRepository interface {
	ListActive(ctx context.Context, query ImageListQuery) ([]do.Image, error)
	CountActiveByQuery(ctx context.Context, query ImageListQuery) (int64, error)
	FindActiveByID(ctx context.Context, id int64) (do.Image, error)
	FindActiveByIDs(ctx context.Context, ids []int64) ([]do.Image, error)
	// FindSimilarByTagIDs 按标签重叠数查询相似图片，排除 excludeImageID，按重叠数降序、view_count 降序排序，limit 控制数量。
	FindSimilarByTagIDs(ctx context.Context, tagIDs []int64, excludeImageID int64, limit int) ([]do.Image, error)
	// FindSimilarByCategory 按分类查询相似图片，排除 excludeImageIDs，按 view_count 降序排序，limit 控制数量。
	FindSimilarByCategory(ctx context.Context, category string, excludeImageIDs []int64, limit int) ([]do.Image, error)
	SoftDelete(ctx context.Context, id int64, deletedAt time.Time) error
	Restore(ctx context.Context, id int64) (do.Image, error)
}

// ImageListQuery 定义图片列表查询条件。
type ImageListQuery struct {
	Page    int
	Size    int
	TagIDs  []int64
	OrderBy string
}

// ImageSearcher 定义图片全文搜索能力。
type ImageSearcher interface {
	Search(ctx context.Context, query ImageSearchQuery) (do.ImageSearchResult, error)
	Index(ctx context.Context, image do.Image) error
	Delete(ctx context.Context, imageID int64) error
}

// ImageSearchQuery 定义图片全文搜索条件。
type ImageSearchQuery struct {
	Text string
	Page int
	Size int
}

// ViewRecorder 定义浏览事件记录能力。
type ViewRecorder interface {
	RecordView(ctx context.Context, event do.ImageEvent) error
}

// ImageTagReader 定义图片标签读取能力。
type ImageTagReader interface {
	ListByImageID(ctx context.Context, imageID int64) ([]do.Tag, error)
}

// CollectionRepository 定义收藏夹持久化访问能力。
type CollectionRepository interface {
	Create(ctx context.Context, collection do.Collection) (do.Collection, error)
	ListByOwner(ctx context.Context, userID int64) ([]do.Collection, error)
	FindVisible(ctx context.Context, collectionID int64, viewerID int64) (do.Collection, error)
	Update(ctx context.Context, collection do.Collection) (do.Collection, error)
	Delete(ctx context.Context, collectionID int64, userID int64) error
	AddItem(ctx context.Context, collectionID int64, userID int64, imageID int64) (do.CollectionItem, error)
	RemoveItem(ctx context.Context, collectionID int64, userID int64, imageID int64) error
}

// TagRepository 定义标签持久化访问能力。
type TagRepository interface {
	Create(ctx context.Context, tag do.Tag) (do.Tag, error)
	List(ctx context.Context) ([]do.Tag, error)
	FindByID(ctx context.Context, id int64) (do.Tag, error)
	Update(ctx context.Context, tag do.Tag) (do.Tag, error)
	Delete(ctx context.Context, id int64) error
	Suggest(ctx context.Context, prefix string, limit int) ([]do.Tag, error)
	ListByImageID(ctx context.Context, imageID int64) ([]do.Tag, error)
	AssignToImages(ctx context.Context, imageIDs []int64, tagIDs []int64) ([]do.Image, error)
	UnassignFromImages(ctx context.Context, imageIDs []int64, tagIDs []int64) ([]do.Image, error)
}

// ImageIndexer 定义标签变更后需要同步的图片索引能力。
type ImageIndexer interface {
	Index(ctx context.Context, image do.Image) error
}

// RankingRepository 定义热榜缓存访问能力。
type RankingRepository interface {
	ListCached(ctx context.Context, period do.RankingPeriod, query RankingListQuery) (do.RankingListResult, error)
}

// RankingListQuery 定义热榜缓存列表查询条件。
type RankingListQuery struct {
	Page int
	Size int
}

// RatingRepository 定义评分持久化访问能力。
type RatingRepository interface {
	Upsert(ctx context.Context, rating do.Rating) (do.Rating, error)
}

// UserRepository 定义用户持久化访问能力。
type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (do.User, error)
	FindByID(ctx context.Context, id int64) (do.User, error)
	Create(ctx context.Context, user do.User) (do.User, error)
	UpdateProfile(ctx context.Context, user do.User) (do.User, error)
	UpdatePasswordHash(ctx context.Context, userID int64, passwordHash string) error
}

// ImageEventRepository 定义图片行为事件持久化访问能力。
type ImageEventRepository interface {
	CreateImageEvents(ctx context.Context, events []do.ImageEvent) error
}
