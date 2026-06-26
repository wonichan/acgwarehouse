package repository

import (
	"context"
	"strings"
	"time"

	pkgerrors "github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/po"
)

const defaultImageSort = "created_at desc"

// ImageListQuery 定义图片仓储列表查询条件。
type ImageListQuery struct {
	Page  int
	Size  int
	Sort  string
	Order string
}

// ImageRepository 提供图片持久化访问。
type ImageRepository struct {
	readDB  *gorm.DB
	writeDB *gorm.DB
}

// NewImageRepository 创建图片仓储。
func NewImageRepository(readDB *gorm.DB, writeDB *gorm.DB) *ImageRepository {
	return &ImageRepository{readDB: readDB, writeDB: writeDB}
}

// UpsertByCOSKey 按 COS key 幂等写入图片元数据。
func (r *ImageRepository) UpsertByCOSKey(ctx context.Context, image do.Image) (do.Image, error) {
	stored := imageToPO(image.NormalizeForCreate(time.Now().UTC()))
	assignments := clause.AssignmentColumns([]string{
		"filename",
		"size",
		"last_modified",
		"width",
		"height",
		"category",
	})
	err := r.writeDB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "cos_key"}},
		DoUpdates: assignments,
	}).Create(&stored).Error
	if err != nil {
		return do.Image{}, pkgerrors.WithMessage(err, "upsert image by cos key")
	}
	return r.FindByCOSKey(ctx, image.COSKey)
}

// FindByCOSKey 按 COS key 查询图片。
func (r *ImageRepository) FindByCOSKey(ctx context.Context, cosKey string) (do.Image, error) {
	var image po.Image
	if err := r.readDB.WithContext(ctx).Where("cos_key = ?", cosKey).First(&image).Error; err != nil {
		return do.Image{}, pkgerrors.WithMessage(err, "find image by cos key")
	}
	return imageToDO(image), nil
}

// ListActive 查询未软删除图片列表。
func (r *ImageRepository) ListActive(ctx context.Context, query ImageListQuery) ([]do.Image, error) {
	var images []po.Image
	database := activeImages(r.readDB.WithContext(ctx)).Order(imageOrder(query))
	if query.Size > 0 {
		database = database.Limit(query.Size).Offset(imageOffset(query))
	}
	if err := database.Find(&images).Error; err != nil {
		return nil, pkgerrors.WithMessage(err, "list active images")
	}
	return imagesToDO(images), nil
}

// CountActive 统计未软删除图片数量。
func (r *ImageRepository) CountActive(ctx context.Context) (int64, error) {
	var total int64
	if err := activeImages(r.readDB.WithContext(ctx)).Count(&total).Error; err != nil {
		return 0, pkgerrors.WithMessage(err, "count active images")
	}
	return total, nil
}

// activeImages 限定图片处于可展示状态。
func activeImages(database *gorm.DB) *gorm.DB {
	return database.Model(&po.Image{}).Where("status = ? AND deleted_at IS NULL", string(do.ImageStatusActive))
}

// imageOffset 计算分页偏移量。
func imageOffset(query ImageListQuery) int {
	if query.Page < 1 || query.Size < 1 {
		return 0
	}
	return (query.Page - 1) * query.Size
}

// imageOrder 返回白名单排序表达式。
func imageOrder(query ImageListQuery) string {
	field := strings.ToLower(strings.TrimSpace(query.Sort))
	order := strings.ToLower(strings.TrimSpace(query.Order))
	if order != "asc" {
		order = "desc"
	}
	switch field {
	case "created_at", "size", "filename", "category", "last_modified":
		return field + " " + order
	default:
		return defaultImageSort
	}
}

// imagesToDO 将图片持久化对象列表转换为领域对象列表。
func imagesToDO(images []po.Image) []do.Image {
	result := make([]do.Image, 0, len(images))
	for _, image := range images {
		result = append(result, imageToDO(image))
	}
	return result
}

// imageToDO 将图片持久化对象转换为领域对象。
func imageToDO(image po.Image) do.Image {
	var deletedAt time.Time
	if image.DeletedAt != nil {
		deletedAt = image.DeletedAt.UTC()
	}
	return do.Image{
		ID:            image.ID,
		COSKey:        image.COSKey,
		Filename:      image.Filename,
		Size:          image.Size,
		LastModified:  image.LastModified.UTC(),
		Width:         image.Width,
		Height:        image.Height,
		Category:      image.Category,
		AvgScore:      image.AvgScore,
		RatingCount:   image.RatingCount,
		FavoriteCount: image.FavoriteCount,
		ViewCount:     image.ViewCount,
		Status:        do.ImageStatus(image.Status),
		DeletedAt:     deletedAt,
		CreatedAt:     image.CreatedAt.UTC(),
	}
}

// imageToPO 将图片领域对象转换为持久化对象。
func imageToPO(image do.Image) po.Image {
	var deletedAt *time.Time
	if !image.DeletedAt.IsZero() {
		value := image.DeletedAt.UTC()
		deletedAt = &value
	}
	return po.Image{
		ID:            image.ID,
		COSKey:        image.COSKey,
		Filename:      image.Filename,
		Size:          image.Size,
		LastModified:  image.LastModified,
		Width:         image.Width,
		Height:        image.Height,
		Category:      image.Category,
		AvgScore:      image.AvgScore,
		RatingCount:   image.RatingCount,
		FavoriteCount: image.FavoriteCount,
		ViewCount:     image.ViewCount,
		Status:        string(image.Status),
		DeletedAt:     deletedAt,
		CreatedAt:     image.CreatedAt,
	}
}
