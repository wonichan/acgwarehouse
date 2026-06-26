package service

import (
	"context"
	stderrors "errors"
	"strings"

	pkgerrors "github.com/pkg/errors"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/repository"
)

var (
	// ErrCollectionNotFound 表示收藏夹不存在。
	ErrCollectionNotFound = repository.ErrCollectionNotFound
	// ErrInvalidCollectionInput 表示收藏夹输入非法。
	ErrInvalidCollectionInput = pkgerrors.New("service: invalid collection input")
)

// CollectionRepository 定义收藏服务依赖的仓储能力。
type CollectionRepository interface {
	Create(ctx context.Context, collection do.Collection) (do.Collection, error)
	ListByOwner(ctx context.Context, userID int64) ([]do.Collection, error)
	FindVisible(ctx context.Context, collectionID int64, viewerID int64) (do.Collection, error)
	Update(ctx context.Context, collection do.Collection) (do.Collection, error)
	Delete(ctx context.Context, collectionID int64, userID int64) error
	AddItem(ctx context.Context, collectionID int64, userID int64, imageID int64) (do.CollectionItem, error)
	RemoveItem(ctx context.Context, collectionID int64, userID int64, imageID int64) error
}

// CollectionService 提供收藏夹管理能力。
type CollectionService struct {
	repo CollectionRepository
}

// NewCollectionService 创建收藏服务。
func NewCollectionService(repo CollectionRepository) *CollectionService {
	return &CollectionService{repo: repo}
}

// Create 创建用户收藏夹。
func (s *CollectionService) Create(ctx context.Context, collection do.Collection) (do.Collection, error) {
	prepared, err := prepareCollectionInput(collection)
	if err != nil {
		return do.Collection{}, err
	}
	created, err := s.repo.Create(ctx, prepared)
	if err != nil {
		return do.Collection{}, mapCollectionError(err, "create collection")
	}
	return created, nil
}

// ListByOwner 返回用户自己的收藏夹列表。
func (s *CollectionService) ListByOwner(ctx context.Context, userID int64) ([]do.Collection, error) {
	if userID < 1 {
		return nil, pkgerrors.WithMessage(ErrInvalidCollectionInput, "validate collection owner")
	}
	collections, err := s.repo.ListByOwner(ctx, userID)
	if err != nil {
		return nil, mapCollectionError(err, "list collections")
	}
	return collections, nil
}

// FindVisible 返回访问者可见的收藏夹。
func (s *CollectionService) FindVisible(ctx context.Context, collectionID int64, viewerID int64) (do.Collection, error) {
	if collectionID < 1 {
		return do.Collection{}, pkgerrors.WithMessage(ErrInvalidCollectionInput, "validate collection id")
	}
	collection, err := s.repo.FindVisible(ctx, collectionID, viewerID)
	if err != nil {
		return do.Collection{}, mapCollectionError(err, "find visible collection")
	}
	return collection, nil
}

// Update 更新用户自己的收藏夹。
func (s *CollectionService) Update(ctx context.Context, collection do.Collection) (do.Collection, error) {
	prepared, err := prepareCollectionInput(collection)
	if err != nil {
		return do.Collection{}, err
	}
	if prepared.ID < 1 {
		return do.Collection{}, pkgerrors.WithMessage(ErrInvalidCollectionInput, "validate collection id")
	}
	updated, err := s.repo.Update(ctx, prepared)
	if err != nil {
		return do.Collection{}, mapCollectionError(err, "update collection")
	}
	return updated, nil
}

// Delete 删除用户自己的收藏夹。
func (s *CollectionService) Delete(ctx context.Context, collectionID int64, userID int64) error {
	if collectionID < 1 || userID < 1 {
		return pkgerrors.WithMessage(ErrInvalidCollectionInput, "validate delete collection")
	}
	if err := s.repo.Delete(ctx, collectionID, userID); err != nil {
		return mapCollectionError(err, "delete collection")
	}
	return nil
}

// AddItem 将图片加入用户自己的收藏夹。
func (s *CollectionService) AddItem(
	ctx context.Context,
	collectionID int64,
	userID int64,
	imageID int64,
) (do.CollectionItem, error) {
	if collectionID < 1 || userID < 1 || imageID < 1 {
		return do.CollectionItem{}, pkgerrors.WithMessage(ErrInvalidCollectionInput, "validate add collection item")
	}
	item, err := s.repo.AddItem(ctx, collectionID, userID, imageID)
	if err != nil {
		return do.CollectionItem{}, mapCollectionError(err, "add collection item")
	}
	return item, nil
}

// RemoveItem 从用户自己的收藏夹移除图片。
func (s *CollectionService) RemoveItem(ctx context.Context, collectionID int64, userID int64, imageID int64) error {
	if collectionID < 1 || userID < 1 || imageID < 1 {
		return pkgerrors.WithMessage(ErrInvalidCollectionInput, "validate remove collection item")
	}
	if err := s.repo.RemoveItem(ctx, collectionID, userID, imageID); err != nil {
		return mapCollectionError(err, "remove collection item")
	}
	return nil
}

// prepareCollectionInput 规范化收藏夹输入。
func prepareCollectionInput(collection do.Collection) (do.Collection, error) {
	collection.Name = strings.TrimSpace(collection.Name)
	if collection.Name == "" || len(collection.Name) > 64 || collection.UserID < 1 {
		return do.Collection{}, pkgerrors.WithMessage(ErrInvalidCollectionInput, "validate collection")
	}
	if collection.Visibility == "" {
		collection.Visibility = do.CollectionVisibilityPrivate
	}
	if !collection.Visibility.IsValid() {
		return do.Collection{}, pkgerrors.WithMessage(ErrInvalidCollectionInput, "validate collection visibility")
	}
	return collection, nil
}

// mapCollectionError 将仓储收藏错误映射为服务错误。
func mapCollectionError(err error, message string) error {
	switch {
	case stderrors.Is(err, repository.ErrCollectionForbidden):
		return pkgerrors.WithMessage(ErrForbidden, message)
	case stderrors.Is(err, repository.ErrCollectionNotFound):
		return pkgerrors.WithMessage(ErrCollectionNotFound, message)
	default:
		return pkgerrors.WithMessage(err, message)
	}
}
