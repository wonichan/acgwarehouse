package service

import (
	"context"
	"strings"
	"time"

	pkgerrors "github.com/pkg/errors"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/dto"
	"github.com/yachiyo/acgwarehouse/internal/repository"
)

// RepositoryImageQuery 定义图片仓储列表查询条件。
type RepositoryImageQuery = repository.ImageListQuery

// ImageQuery 定义图片列表查询条件。
type ImageQuery struct {
	Filename string
	Tag      string
	Page     int
	Size     int
	Sort     string
	Order    string
}

// SearchQuery 定义图片全文搜索条件。
type SearchQuery = do.ImageSearchQuery

// ImageListResult 定义图片列表业务结果。
type ImageListResult struct {
	Total int64
	Page  int
	Size  int
	List  []dto.ImageResponse
}

// ImageRepository 定义图片服务依赖的仓储能力。
type ImageRepository interface {
	ListActive(ctx context.Context, query RepositoryImageQuery) ([]do.Image, error)
	CountActiveByQuery(ctx context.Context, query RepositoryImageQuery) (int64, error)
	FindActiveByID(ctx context.Context, id int64) (do.Image, error)
	FindActiveByIDs(ctx context.Context, ids []int64) ([]do.Image, error)
	SoftDelete(ctx context.Context, id int64, deletedAt time.Time) error
	Restore(ctx context.Context, id int64) (do.Image, error)
}

// ImageSearcher 定义图片搜索依赖能力。
type ImageSearcher interface {
	Search(ctx context.Context, query SearchQuery) (do.ImageSearchResult, error)
	Index(ctx context.Context, image do.Image) error
	Delete(ctx context.Context, imageID int64) error
}

// ViewRecorder 定义浏览事件记录能力。
type ViewRecorder interface {
	RecordView(ctx context.Context, event do.ImageEvent) error
}

// ImageTagReader 定义图片标签读取能力。
type ImageTagReader interface {
	ListByImageID(ctx context.Context, imageID int64) ([]do.Tag, error)
}

// ImageService 提供图片查询、搜索和生命周期管理能力。
type ImageService struct {
	repo     ImageRepository
	searcher ImageSearcher
	views    ViewRecorder
	tags     ImageTagReader
	cosBase  string
}

// NewImageService 创建图片服务。
func NewImageService(repo ImageRepository, searcher ImageSearcher, views ViewRecorder, cosBase string) *ImageService {
	return NewImageServiceWithTags(repo, searcher, views, nil, cosBase)
}

// NewImageServiceWithTags 创建带标签读取能力的图片服务。
func NewImageServiceWithTags(
	repo ImageRepository,
	searcher ImageSearcher,
	views ViewRecorder,
	tags ImageTagReader,
	cosBase string,
) *ImageService {
	return &ImageService{
		repo:     repo,
		searcher: searcher,
		views:    views,
		tags:     tags,
		cosBase:  strings.TrimRight(cosBase, "/"),
	}
}

// List 查询可公开展示图片列表。
func (s *ImageService) List(ctx context.Context, query ImageQuery) (ImageListResult, error) {
	repoQuery := toRepositoryImageQuery(query)
	images, err := s.repo.ListActive(ctx, repoQuery)
	if err != nil {
		return ImageListResult{}, pkgerrors.WithMessage(err, "list active images")
	}
	total, err := s.repo.CountActiveByQuery(ctx, repoQuery)
	if err != nil {
		return ImageListResult{}, pkgerrors.WithMessage(err, "count active images")
	}
	return s.newListResult(total, query.Page, query.Size, images), nil
}

// Detail 查询图片详情并记录浏览事件。
func (s *ImageService) Detail(ctx context.Context, id int64, userID int64) (dto.ImageDetailResponse, error) {
	image, err := s.repo.FindActiveByID(ctx, id)
	if err != nil {
		return dto.ImageDetailResponse{}, pkgerrors.WithMessage(err, "find image detail")
	}
	if s.views != nil {
		if err := s.views.RecordView(ctx, do.ImageEvent{
			ImageID:   image.ID,
			UserID:    userID,
			Type:      do.ImageEventTypeView,
			Value:     1,
			CreatedAt: time.Now().UTC(),
		}); err != nil {
			return dto.ImageDetailResponse{}, pkgerrors.WithMessage(err, "record image view")
		}
	}
	detail, err := s.newDetailResponse(ctx, image)
	if err != nil {
		return dto.ImageDetailResponse{}, err
	}
	return detail, nil
}

// Search 搜索可公开展示图片列表。
func (s *ImageService) Search(ctx context.Context, query SearchQuery) (ImageListResult, error) {
	searchResult, err := s.searcher.Search(ctx, query)
	if err != nil {
		return ImageListResult{}, pkgerrors.WithMessage(err, "search image ids")
	}
	images, err := s.repo.FindActiveByIDs(ctx, searchResult.IDs)
	if err != nil {
		return ImageListResult{}, pkgerrors.WithMessage(err, "find searched images")
	}
	return s.newListResult(searchResult.Total, query.Page, query.Size, images), nil
}

// SoftDelete 软删除图片。
func (s *ImageService) SoftDelete(ctx context.Context, id int64) error {
	if err := s.repo.SoftDelete(ctx, id, time.Now().UTC()); err != nil {
		return pkgerrors.WithMessage(err, "soft delete image")
	}
	if err := s.searcher.Delete(ctx, id); err != nil {
		return pkgerrors.WithMessage(err, "delete image index")
	}
	return nil
}

// Restore 恢复图片并返回最新图片。
func (s *ImageService) Restore(ctx context.Context, id int64) (do.Image, error) {
	image, err := s.repo.Restore(ctx, id)
	if err != nil {
		return do.Image{}, pkgerrors.WithMessage(err, "restore image")
	}
	if err := s.searcher.Index(ctx, image); err != nil {
		return do.Image{}, pkgerrors.WithMessage(err, "restore image index")
	}
	return image, nil
}

// toRepositoryImageQuery 将服务查询转换为仓储查询。
func toRepositoryImageQuery(query ImageQuery) RepositoryImageQuery {
	return RepositoryImageQuery{
		Filename: query.Filename,
		Tag:      query.Tag,
		Page:     query.Page,
		Size:     query.Size,
		Sort:     query.Sort,
		Order:    query.Order,
	}
}

// newListResult 将领域图片列表转换为服务列表结果。
func (s *ImageService) newListResult(total int64, page int, size int, images []do.Image) ImageListResult {
	list := make([]dto.ImageResponse, 0, len(images))
	for _, image := range images {
		list = append(list, s.toImageResponse(image))
	}
	return ImageListResult{Total: total, Page: page, Size: size, List: list}
}

// newDetailResponse 创建阶段三稳定的图片详情响应。
func (s *ImageService) newDetailResponse(ctx context.Context, image do.Image) (dto.ImageDetailResponse, error) {
	response := s.toImageResponse(image)
	tags, err := s.imageTagNames(ctx, image.ID)
	if err != nil {
		return dto.ImageDetailResponse{}, err
	}
	return dto.ImageDetailResponse{
		Image:         response,
		Tags:          tags,
		AvgScore:      image.AvgScore,
		RatingCount:   image.RatingCount,
		FavoriteCount: image.FavoriteCount,
		MyRating:      nil,
		IsCollected:   false,
		SimilarImages: []dto.ImageResponse{},
	}, nil
}

// imageTagNames 查询图片标签名称。
func (s *ImageService) imageTagNames(ctx context.Context, imageID int64) ([]string, error) {
	if s.tags == nil {
		return []string{}, nil
	}
	tags, err := s.tags.ListByImageID(ctx, imageID)
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "list image tags")
	}
	names := make([]string, 0, len(tags))
	for _, tag := range tags {
		names = append(names, tag.Name)
	}
	return names, nil
}

// toImageResponse 将图片领域对象转换为公开响应 DTO。
func (s *ImageService) toImageResponse(image do.Image) dto.ImageResponse {
	return dto.ImageResponse{
		ID:            image.ID,
		COSKey:        image.COSKey,
		Filename:      image.Filename,
		URL:           s.imageURL(image.COSKey),
		Size:          image.Size,
		LastModified:  formatAPITime(image.LastModified),
		Width:         image.Width,
		Height:        image.Height,
		Category:      image.Category,
		AvgScore:      image.AvgScore,
		RatingCount:   image.RatingCount,
		FavoriteCount: image.FavoriteCount,
		ViewCount:     image.ViewCount,
		CreatedAt:     formatAPITime(image.CreatedAt),
	}
}

// imageURL 拼接图片 COS 公开访问地址。
func (s *ImageService) imageURL(cosKey string) string {
	if s.cosBase == "" {
		return cosKey
	}
	return s.cosBase + "/" + strings.TrimLeft(cosKey, "/")
}

// formatAPITime 将时间转换为 UTC RFC3339 字符串。
func formatAPITime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
