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
	// FindSimilarByTagIDs 按标签重叠数查询相似图片，排除 excludeImageID，按重叠数降序、view_count 降序排序，limit 控制数量。
	FindSimilarByTagIDs(ctx context.Context, tagIDs []int64, excludeImageID int64, limit int) ([]do.Image, error)
	// FindSimilarByCategory 按分类查询相似图片，排除 excludeImageIDs，按 view_count 降序排序，limit 控制数量。
	FindSimilarByCategory(ctx context.Context, category string, excludeImageIDs []int64, limit int) ([]do.Image, error)
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
		image.ViewCount++
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

// similarImageLimit 控制详情页相似推荐返回数量。
const similarImageLimit = 6

// newDetailResponse 组装图片详情响应，包含标签与相似推荐。
func (s *ImageService) newDetailResponse(ctx context.Context, image do.Image) (dto.ImageDetailResponse, error) {
	response := s.toImageResponse(image)
	// 单次标签查询同时派生名称（给 Tags 字段）和 ID（给相似推荐查询）。
	tags, err := s.imageTags(ctx, image.ID)
	if err != nil {
		return dto.ImageDetailResponse{}, err
	}
	tagNames := make([]string, 0, len(tags))
	tagIDs := make([]int64, 0, len(tags))
	for _, tag := range tags {
		tagNames = append(tagNames, tag.Name)
		tagIDs = append(tagIDs, tag.ID)
	}
	similar, err := s.findSimilarImages(ctx, image, tagIDs, similarImageLimit)
	if err != nil {
		return dto.ImageDetailResponse{}, err
	}
	return dto.ImageDetailResponse{
		Image:         response,
		Tags:          tagNames,
		AvgScore:      image.AvgScore,
		RatingCount:   image.RatingCount,
		FavoriteCount: image.FavoriteCount,
		MyRating:      nil,
		IsCollected:   false,
		SimilarImages: similar,
	}, nil
}

// imageTags 查询图片关联标签，返回领域标签对象。
func (s *ImageService) imageTags(ctx context.Context, imageID int64) ([]do.Tag, error) {
	if s.tags == nil {
		return []do.Tag{}, nil
	}
	tags, err := s.tags.ListByImageID(ctx, imageID)
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "list image tags")
	}
	return tags, nil
}

// findSimilarImages 按标签重叠为主、分类回退为辅查询相似图片。
func (s *ImageService) findSimilarImages(ctx context.Context, image do.Image, tagIDs []int64, limit int) ([]dto.ImageResponse, error) {
	byTag, err := s.repo.FindSimilarByTagIDs(ctx, tagIDs, image.ID, limit)
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "find similar by tags")
	}
	if len(byTag) >= limit {
		return s.toImageResponseList(byTag[:limit]), nil
	}
	// 标签重叠不足时，用同 category 按 view_count 降序补足，排除当前图片与已选结果。
	remaining := limit - len(byTag)
	excludeIDs := make([]int64, 0, len(byTag)+1)
	excludeIDs = append(excludeIDs, image.ID)
	for _, img := range byTag {
		excludeIDs = append(excludeIDs, img.ID)
	}
	byCategory, err := s.repo.FindSimilarByCategory(ctx, image.Category, excludeIDs, remaining)
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "find similar by category")
	}
	byTag = append(byTag, byCategory...)
	return s.toImageResponseList(byTag), nil
}

// toImageResponseList 将图片领域对象列表转换为响应 DTO 列表。
func (s *ImageService) toImageResponseList(images []do.Image) []dto.ImageResponse {
	list := make([]dto.ImageResponse, 0, len(images))
	for _, image := range images {
		list = append(list, s.toImageResponse(image))
	}
	return list
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
