package repository

import (
	"context"
	stderrors "errors"
	"strings"
	"time"

	pkgerrors "github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/po"
	"github.com/yachiyo/acgwarehouse/internal/ports"
	"github.com/yachiyo/acgwarehouse/internal/validators"
)

const defaultImageSort = "image.created_at desc"

var (
	// ErrImageNotFound 表示图片不存在或不可公开展示。
	ErrImageNotFound = ports.ErrImageNotFound
)

// ImageListQuery 定义图片仓储列表查询条件。
type ImageListQuery struct {
	Filename string
	Tag      string
	Page     int
	Size     int
	Sort     string
	Order    string
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
	stored := imageToPO(validators.NormalizeImageForCreate(image, time.Now().UTC()))
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
	database := queryActiveImages(r.readDB.WithContext(ctx), query).Order(imageOrder(query))
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
	return r.CountActiveByQuery(ctx, ImageListQuery{})
}

// CountActiveByQuery 统计符合查询条件的未软删除图片数量。
func (r *ImageRepository) CountActiveByQuery(ctx context.Context, query ImageListQuery) (int64, error) {
	var total int64
	if err := queryActiveImages(r.readDB.WithContext(ctx), query).Count(&total).Error; err != nil {
		return 0, pkgerrors.WithMessage(err, "count active images")
	}
	return total, nil
}

// FindActiveByID 按 ID 查询可公开展示图片。
func (r *ImageRepository) FindActiveByID(ctx context.Context, id int64) (do.Image, error) {
	var image po.Image
	err := activeImages(r.readDB.WithContext(ctx)).Where("id = ?", id).First(&image).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return do.Image{}, pkgerrors.WithMessage(ErrImageNotFound, "find active image by id")
	}
	if err != nil {
		return do.Image{}, pkgerrors.WithMessage(err, "find active image by id")
	}
	return imageToDO(image), nil
}

// FindActiveByIDs 按 ID 列表查询可公开展示图片并保持入参顺序。
func (r *ImageRepository) FindActiveByIDs(ctx context.Context, ids []int64) ([]do.Image, error) {
	if len(ids) == 0 {
		return []do.Image{}, nil
	}
	var images []po.Image
	if err := activeImages(r.readDB.WithContext(ctx)).Where("id IN ?", ids).Find(&images).Error; err != nil {
		return nil, pkgerrors.WithMessage(err, "find active images by ids")
	}
	return orderImagesByIDs(ids, imagesToDO(images)), nil
}

// SoftDelete 将图片标记为已删除。
func (r *ImageRepository) SoftDelete(ctx context.Context, id int64, deletedAt time.Time) error {
	updates := map[string]interface{}{
		"status":     string(do.ImageStatusDeleted),
		"deleted_at": deletedAt.UTC(),
	}
	result := activeImages(r.writeDB.WithContext(ctx)).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return pkgerrors.WithMessage(result.Error, "soft delete image")
	}
	if result.RowsAffected == 0 {
		return pkgerrors.WithMessage(ErrImageNotFound, "soft delete image")
	}
	return nil
}

// Restore 恢复已软删除图片。
func (r *ImageRepository) Restore(ctx context.Context, id int64) (do.Image, error) {
	updates := map[string]interface{}{
		"status":     string(do.ImageStatusActive),
		"deleted_at": nil,
	}
	result := r.writeDB.WithContext(ctx).Model(&po.Image{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return do.Image{}, pkgerrors.WithMessage(result.Error, "restore image")
	}
	if result.RowsAffected == 0 {
		return do.Image{}, pkgerrors.WithMessage(ErrImageNotFound, "restore image")
	}
	return r.FindActiveByID(ctx, id)
}

// CreateImageEvents 批量写入图片行为事件并累加浏览计数。
func (r *ImageRepository) CreateImageEvents(ctx context.Context, events []do.ImageEvent) error {
	if len(events) == 0 {
		return nil
	}
	return r.writeDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(imageEventsToPO(events)).Error; err != nil {
			return pkgerrors.WithMessage(err, "create image events")
		}
		for imageID, count := range viewCountsByImage(events) {
			update := tx.Model(&po.Image{}).Where("id = ?", imageID).
				UpdateColumn("view_count", gorm.Expr("view_count + ?", count))
			if update.Error != nil {
				return pkgerrors.WithMessage(update.Error, "increment image view count")
			}
		}
		return nil
	})
}

// activeImages 限定图片处于可展示状态。
func activeImages(database *gorm.DB) *gorm.DB {
	return database.Model(&po.Image{}).Where("status = ? AND deleted_at IS NULL", string(do.ImageStatusActive))
}

// queryActiveImages 按查询条件限定可展示图片。
func queryActiveImages(database *gorm.DB, query ImageListQuery) *gorm.DB {
	database = activeImages(database)
	filename := strings.TrimSpace(query.Filename)
	if filename != "" {
		database = database.Where("filename LIKE ?", "%"+filename+"%")
	}
	tagName := strings.TrimSpace(query.Tag)
	if tagName != "" {
		database = database.
			Joins("JOIN image_tag ON image_tag.image_id = image.id").
			Joins("JOIN tag ON tag.id = image_tag.tag_id").
			Where("tag.name = ?", tagName)
	}
	return database
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
	case "created_at":
		return "image.created_at " + order
	case "size":
		return "image.size " + order
	case "tag":
		return defaultImageSort
	default:
		return defaultImageSort
	}
}
