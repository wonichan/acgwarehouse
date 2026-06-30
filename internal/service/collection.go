package service

import (
	"context"
	stderrors "errors"
	"strings"

	pkgerrors "github.com/pkg/errors"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/ports"
)

var (
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
	repo    CollectionRepository
	cosBase string
}

// NewCollectionService 创建收藏服务。
func NewCollectionService(repo CollectionRepository, cosBase string) *CollectionService {
	return &CollectionService{repo: repo, cosBase: strings.TrimRight(cosBase, "/")}
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
	return s.attachCoverImageURL(created), nil
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
	for i := range collections {
		collections[i] = s.attachCoverImageURL(collections[i])
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
	return s.attachCoverImageURL(collection), nil
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
	if prepared.CoverImageIDSet && prepared.CoverImageID > 0 {
		existing, err := s.repo.FindVisible(ctx, prepared.ID, prepared.UserID)
		if err != nil {
			return do.Collection{}, mapCollectionError(err, "validate collection cover")
		}
		if !collectionContainsImage(existing, prepared.CoverImageID) {
			return do.Collection{}, pkgerrors.WithMessage(ErrInvalidCollectionInput, "validate collection cover image")
		}
	}
	updated, err := s.repo.Update(ctx, prepared)
	if err != nil {
		return do.Collection{}, mapCollectionError(err, "update collection")
	}
	return s.attachCoverImageURL(updated), nil
}

// collectionContainsImage 判断指定图片是否在收藏夹条目中。
func collectionContainsImage(collection do.Collection, imageID int64) bool {
	for _, item := range collection.Items {
		if item.ImageID == imageID {
			return true
		}
	}
	return false
}

// attachCoverImageURL 为收藏夹计算封面图片 URL 并填充条目图片 URL。
func (s *CollectionService) attachCoverImageURL(collection do.Collection) do.Collection {
	for i := range collection.Items {
		if collection.Items[i].Image.ID > 0 && collection.Items[i].Image.COSKey != "" {
			collection.Items[i].Image.URL = s.imageURL(collection.Items[i].Image.COSKey)
		}
	}
	collection.CoverImageURL = s.resolveCoverImageURL(collection)
	return collection
}

// resolveCoverImageURL 解析收藏夹封面图片 URL。
// 优先使用 CoverImageID 指定的图片；未设置时 fallback 第一张条目图片。
func (s *CollectionService) resolveCoverImageURL(collection do.Collection) string {
	var target do.Image
	if collection.CoverImageID > 0 {
		for _, item := range collection.Items {
			if item.ImageID == collection.CoverImageID {
				target = item.Image
				break
			}
		}
	}
	if target.ID == 0 && len(collection.Items) > 0 {
		target = collection.Items[0].Image
	}
	if target.ID == 0 || target.COSKey == "" {
		return ""
	}
	return s.imageURL(target.COSKey)
}

// imageURL 拼接图片 COS 公开访问地址。
func (s *CollectionService) imageURL(cosKey string) string {
	if s.cosBase == "" {
		return cosKey
	}
	return s.cosBase + "/" + strings.TrimLeft(cosKey, "/")
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
	case stderrors.Is(err, ports.ErrCollectionForbidden):
		return pkgerrors.WithMessage(ErrForbidden, message)
	case stderrors.Is(err, ports.ErrCollectionNotFound):
		return pkgerrors.WithMessage(ports.ErrCollectionNotFound, message)
	default:
		return pkgerrors.WithMessage(err, message)
	}
}
