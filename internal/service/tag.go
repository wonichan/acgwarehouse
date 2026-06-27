package service

import (
	"context"
	"strings"

	pkgerrors "github.com/pkg/errors"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
)

const defaultSuggestLimit = 20

var (
	// ErrInvalidTagInput 表示标签输入非法。
	ErrInvalidTagInput = pkgerrors.New("service: invalid tag input")
)

// TagRepository 定义标签服务依赖的仓储能力。
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

// TagService 提供全局标签管理能力。
type TagService struct {
	repo    TagRepository
	indexer ImageIndexer
}

// NewTagService 创建标签服务。
func NewTagService(repo TagRepository, indexer ImageIndexer) *TagService {
	return &TagService{repo: repo, indexer: indexer}
}

// Create 创建或复用全局共享标签。
func (s *TagService) Create(ctx context.Context, tag do.Tag) (do.Tag, error) {
	prepared, err := prepareTagInput(tag)
	if err != nil {
		return do.Tag{}, err
	}
	created, err := s.repo.Create(ctx, prepared)
	if err != nil {
		return do.Tag{}, pkgerrors.WithMessage(err, "create tag")
	}
	return created, nil
}

// List 返回全局标签列表。
func (s *TagService) List(ctx context.Context) ([]do.Tag, error) {
	tags, err := s.repo.List(ctx)
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "list tags")
	}
	return tags, nil
}

// Update 管理员更新标签名称。
func (s *TagService) Update(ctx context.Context, role do.UserRole, tag do.Tag) (do.Tag, error) {
	if !isAdmin(role) {
		return do.Tag{}, pkgerrors.WithMessage(ErrForbidden, "update tag")
	}
	prepared, err := prepareTagInput(tag)
	if err != nil {
		return do.Tag{}, err
	}
	updated, err := s.repo.Update(ctx, prepared)
	if err != nil {
		return do.Tag{}, pkgerrors.WithMessage(err, "update tag")
	}
	return updated, nil
}

// Delete 管理员删除标签。
func (s *TagService) Delete(ctx context.Context, role do.UserRole, id int64) error {
	if !isAdmin(role) {
		return pkgerrors.WithMessage(ErrForbidden, "delete tag")
	}
	if id < 1 {
		return pkgerrors.WithMessage(ErrInvalidTagInput, "validate tag id")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return pkgerrors.WithMessage(err, "delete tag")
	}
	return nil
}

// Suggest 返回标签建议。
func (s *TagService) Suggest(ctx context.Context, prefix string, limit int) ([]do.Tag, error) {
	if limit < 1 {
		limit = defaultSuggestLimit
	}
	tags, err := s.repo.Suggest(ctx, strings.TrimSpace(prefix), limit)
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "suggest tags")
	}
	return tags, nil
}

// ListByImageID 返回图片标签列表。
func (s *TagService) ListByImageID(ctx context.Context, imageID int64) ([]do.Tag, error) {
	tags, err := s.repo.ListByImageID(ctx, imageID)
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "list image tags")
	}
	return tags, nil
}

// AssignToImages 批量给图片打标签并同步搜索索引。
func (s *TagService) AssignToImages(ctx context.Context, imageIDs []int64, tagIDs []int64) ([]do.Image, error) {
	images, err := s.repo.AssignToImages(ctx, imageIDs, tagIDs)
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "assign tags to images")
	}
	if err := s.indexImages(ctx, images); err != nil {
		return nil, pkgerrors.WithMessage(err, "index tagged images")
	}
	return images, nil
}

// UnassignFromImages 批量取消图片标签并同步搜索索引。
func (s *TagService) UnassignFromImages(ctx context.Context, imageIDs []int64, tagIDs []int64) ([]do.Image, error) {
	images, err := s.repo.UnassignFromImages(ctx, imageIDs, tagIDs)
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "unassign tags from images")
	}
	if err := s.indexImages(ctx, images); err != nil {
		return nil, pkgerrors.WithMessage(err, "index untagged images")
	}
	return images, nil
}

// indexImages 将受影响图片写回搜索索引。
func (s *TagService) indexImages(ctx context.Context, images []do.Image) error {
	if s.indexer == nil {
		return nil
	}
	for _, image := range images {
		if err := s.indexer.Index(ctx, image); err != nil {
			return pkgerrors.WithMessage(err, "index image")
		}
	}
	return nil
}

// prepareTagInput 规范化服务层标签输入。
func prepareTagInput(tag do.Tag) (do.Tag, error) {
	tag.Name = strings.TrimSpace(tag.Name)
	if tag.Name == "" || len(tag.Name) > 64 {
		return do.Tag{}, pkgerrors.WithMessage(ErrInvalidTagInput, "validate tag name")
	}
	return tag, nil
}

// isAdmin 判断角色是否具备管理员权限。
func isAdmin(role do.UserRole) bool {
	return role == do.UserRoleAdmin
}
