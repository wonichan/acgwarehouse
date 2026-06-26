package repository

import (
	"context"
	stderrors "errors"
	"sort"
	"strings"
	"time"

	pkgerrors "github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/po"
)

const defaultTagSuggestLimit = 20

var (
	// ErrTagNotFound 表示标签不存在。
	ErrTagNotFound = pkgerrors.New("repository: tag not found")
	// ErrInvalidTagInput 表示标签输入非法。
	ErrInvalidTagInput = pkgerrors.New("repository: invalid tag input")
)

// TagRepository 提供标签持久化访问。
type TagRepository struct {
	readDB  *gorm.DB
	writeDB *gorm.DB
}

// NewTagRepository 创建标签仓储。
func NewTagRepository(readDB *gorm.DB, writeDB *gorm.DB) *TagRepository {
	return &TagRepository{readDB: readDB, writeDB: writeDB}
}

// Create 创建标签；名称已存在时返回既有标签。
func (r *TagRepository) Create(ctx context.Context, tag do.Tag) (do.Tag, error) {
	prepared, err := prepareTag(tag)
	if err != nil {
		return do.Tag{}, err
	}
	created := tagToPO(prepared)
	result := r.writeDB.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&created)
	if result.Error != nil {
		return do.Tag{}, pkgerrors.WithMessage(result.Error, "create tag")
	}
	return r.FindByName(ctx, prepared.Name)
}

// List 返回全部标签，按使用频次和名称稳定排序。
func (r *TagRepository) List(ctx context.Context) ([]do.Tag, error) {
	var tags []po.Tag
	if err := r.readDB.WithContext(ctx).Order("usage_count desc").Order("name asc").Find(&tags).Error; err != nil {
		return nil, pkgerrors.WithMessage(err, "list tags")
	}
	return tagsToDO(tags), nil
}

// FindByID 按 ID 查询标签。
func (r *TagRepository) FindByID(ctx context.Context, id int64) (do.Tag, error) {
	var tag po.Tag
	err := r.readDB.WithContext(ctx).First(&tag, id).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return do.Tag{}, pkgerrors.WithMessage(ErrTagNotFound, "find tag by id")
	}
	if err != nil {
		return do.Tag{}, pkgerrors.WithMessage(err, "find tag by id")
	}
	return tagToDO(tag), nil
}

// FindByName 按名称查询标签。
func (r *TagRepository) FindByName(ctx context.Context, name string) (do.Tag, error) {
	var tag po.Tag
	err := r.readDB.WithContext(ctx).Where("name = ?", strings.TrimSpace(name)).First(&tag).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return do.Tag{}, pkgerrors.WithMessage(ErrTagNotFound, "find tag by name")
	}
	if err != nil {
		return do.Tag{}, pkgerrors.WithMessage(err, "find tag by name")
	}
	return tagToDO(tag), nil
}

// Update 更新标签名称。
func (r *TagRepository) Update(ctx context.Context, tag do.Tag) (do.Tag, error) {
	prepared, err := prepareTag(tag)
	if err != nil {
		return do.Tag{}, err
	}
	result := r.writeDB.WithContext(ctx).Model(&po.Tag{}).Where("id = ?", prepared.ID).Updates(map[string]interface{}{
		"name":       prepared.Name,
		"updated_at": time.Now().UTC(),
	})
	if result.Error != nil {
		return do.Tag{}, pkgerrors.WithMessage(result.Error, "update tag")
	}
	if result.RowsAffected == 0 {
		return do.Tag{}, pkgerrors.WithMessage(ErrTagNotFound, "update tag")
	}
	return r.FindByID(ctx, prepared.ID)
}

// Delete 删除标签及其图片关联。
func (r *TagRepository) Delete(ctx context.Context, id int64) error {
	return r.writeDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tag_id = ?", id).Delete(&po.ImageTag{}).Error; err != nil {
			return pkgerrors.WithMessage(err, "delete image tag links")
		}
		result := tx.Delete(&po.Tag{}, id)
		if result.Error != nil {
			return pkgerrors.WithMessage(result.Error, "delete tag")
		}
		if result.RowsAffected == 0 {
			return pkgerrors.WithMessage(ErrTagNotFound, "delete tag")
		}
		return nil
	})
}

// Suggest 按前缀返回标签建议，使用频次优先。
func (r *TagRepository) Suggest(ctx context.Context, prefix string, limit int) ([]do.Tag, error) {
	if limit < 1 {
		limit = defaultTagSuggestLimit
	}
	var tags []po.Tag
	query := r.readDB.WithContext(ctx).Order("usage_count desc").Order("name asc").Limit(limit)
	trimmed := strings.TrimSpace(prefix)
	if trimmed != "" {
		query = query.Where("name LIKE ?", trimmed+"%")
	}
	if err := query.Find(&tags).Error; err != nil {
		return nil, pkgerrors.WithMessage(err, "suggest tags")
	}
	return tagsToDO(tags), nil
}

// ListByImageID 查询图片已关联的标签。
func (r *TagRepository) ListByImageID(ctx context.Context, imageID int64) ([]do.Tag, error) {
	var tags []po.Tag
	err := r.readDB.WithContext(ctx).
		Joins("JOIN image_tag ON image_tag.tag_id = tag.id").
		Where("image_tag.image_id = ?", imageID).
		Order("tag.name asc").
		Find(&tags).Error
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "list tags by image")
	}
	return tagsToDO(tags), nil
}

// AssignToImages 批量给图片打标签，并返回受影响图片及其最新标签。
func (r *TagRepository) AssignToImages(ctx context.Context, imageIDs []int64, tagIDs []int64) ([]do.Image, error) {
	imageIDs = uniquePositiveIDs(imageIDs)
	tagIDs = uniquePositiveIDs(tagIDs)
	if len(imageIDs) == 0 || len(tagIDs) == 0 {
		return []do.Image{}, nil
	}
	now := time.Now().UTC()
	if err := r.writeDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, imageID := range imageIDs {
			for _, tagID := range tagIDs {
				created, err := createImageTag(ctx, tx, imageID, tagID, now)
				if err != nil {
					return err
				}
				if created {
					if err := incrementTagUsage(ctx, tx, tagID, 1); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}); err != nil {
		return nil, pkgerrors.WithMessage(err, "assign tags to images")
	}
	return r.FindActiveImagesWithTags(ctx, imageIDs)
}

// UnassignFromImages 批量取消图片标签，并返回受影响图片及其最新标签。
func (r *TagRepository) UnassignFromImages(ctx context.Context, imageIDs []int64, tagIDs []int64) ([]do.Image, error) {
	imageIDs = uniquePositiveIDs(imageIDs)
	tagIDs = uniquePositiveIDs(tagIDs)
	if len(imageIDs) == 0 || len(tagIDs) == 0 {
		return []do.Image{}, nil
	}
	if err := r.writeDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, imageID := range imageIDs {
			for _, tagID := range tagIDs {
				result := tx.WithContext(ctx).Where("image_id = ? AND tag_id = ?", imageID, tagID).Delete(&po.ImageTag{})
				if result.Error != nil {
					return pkgerrors.WithMessage(result.Error, "delete image tag link")
				}
				if result.RowsAffected > 0 {
					if err := incrementTagUsage(ctx, tx, tagID, -1); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}); err != nil {
		return nil, pkgerrors.WithMessage(err, "unassign tags from images")
	}
	return r.FindActiveImagesWithTags(ctx, imageIDs)
}

// FindActiveImagesWithTags 查询图片并填充标签名称。
func (r *TagRepository) FindActiveImagesWithTags(ctx context.Context, imageIDs []int64) ([]do.Image, error) {
	imageRepo := NewImageRepository(r.readDB, r.writeDB)
	images, err := imageRepo.FindActiveByIDs(ctx, imageIDs)
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "find affected images")
	}
	for index, image := range images {
		tags, err := r.ListByImageID(ctx, image.ID)
		if err != nil {
			return nil, pkgerrors.WithMessage(err, "list affected image tags")
		}
		images[index].Tags = tagNames(tags)
	}
	return images, nil
}

// prepareTag 规范化标签输入。
func prepareTag(tag do.Tag) (do.Tag, error) {
	tag.Name = strings.TrimSpace(tag.Name)
	if tag.Name == "" || len(tag.Name) > 64 {
		return do.Tag{}, pkgerrors.WithMessage(ErrInvalidTagInput, "validate tag name")
	}
	if tag.CreatedAt.IsZero() {
		tag.CreatedAt = time.Now().UTC()
	}
	tag.UpdatedAt = time.Now().UTC()
	return tag, nil
}

// createImageTag 创建图片标签关联，返回是否新增。
func createImageTag(ctx context.Context, tx *gorm.DB, imageID int64, tagID int64, now time.Time) (bool, error) {
	link := po.ImageTag{ImageID: imageID, TagID: tagID, CreatedAt: now}
	result := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&link)
	if result.Error != nil {
		return false, pkgerrors.WithMessage(result.Error, "create image tag link")
	}
	return result.RowsAffected > 0, nil
}

// incrementTagUsage 调整标签使用次数并限制最小值为 0。
func incrementTagUsage(ctx context.Context, tx *gorm.DB, tagID int64, delta int) error {
	result := tx.WithContext(ctx).Model(&po.Tag{}).Where("id = ?", tagID).
		UpdateColumn("usage_count", gorm.Expr("MAX(usage_count + ?, 0)", delta))
	if result.Error != nil {
		return pkgerrors.WithMessage(result.Error, "update tag usage count")
	}
	if result.RowsAffected == 0 {
		return pkgerrors.WithMessage(ErrTagNotFound, "update tag usage count")
	}
	return nil
}

// uniquePositiveIDs 去重并保留正整数 ID。
func uniquePositiveIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id < 1 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Slice(result, func(i int, j int) bool { return result[i] < result[j] })
	return result
}

// tagNames 提取标签名称。
func tagNames(tags []do.Tag) []string {
	names := make([]string, 0, len(tags))
	for _, tag := range tags {
		names = append(names, tag.Name)
	}
	return names
}
