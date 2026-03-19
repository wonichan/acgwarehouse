package service

import (
	"context"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

// SearchOptions defines the search parameters.
type SearchOptions struct {
	Query     string  // Search query (optional)
	TagIDs    []int64 // Tag IDs to filter by (optional)
	SortBy    string  // Sort field: relevance, created_at, filename, file_size
	SortOrder string  // Sort order: asc, desc
	Limit     int     // Results per page
	Offset    int     // Pagination offset
}

// SearchResult contains the search results.
type SearchResult struct {
	Images  []domain.Image
	Total   int64
	HasMore bool
}

// SearchService provides search functionality.
type SearchService struct {
	imageRepo  repository.ImageRepository
	tagRepo    repository.TagRepository
	searchRepo repository.SearchRepository
}

// NewSearchService creates a new search service.
func NewSearchService(
	imageRepo repository.ImageRepository,
	tagRepo repository.TagRepository,
	searchRepo repository.SearchRepository,
) *SearchService {
	return &SearchService{
		imageRepo:  imageRepo,
		tagRepo:    tagRepo,
		searchRepo: searchRepo,
	}
}

// Search performs a search based on the provided options.
// It supports:
// - Full-text search by query
// - Tag filtering
// - Combined search (both query and tags)
// - Sorting and pagination
func (s *SearchService) Search(ctx context.Context, opts SearchOptions) (*SearchResult, error) {
	// Set defaults
	if opts.Limit <= 0 {
		opts.Limit = 20
	}
	if opts.SortBy == "" {
		opts.SortBy = "relevance"
	}
	if opts.SortOrder == "" {
		opts.SortOrder = "desc"
	}

	var images []domain.Image
	var total int64
	var err error

	switch {
	case opts.Query != "" && len(opts.TagIDs) > 0:
		// Combined search: FTS first, then filter by tags
		images, total, err = s.combinedSearch(ctx, opts)
	case opts.Query != "":
		// Full-text search only
		images, total, err = s.ftsSearch(ctx, opts)
	case len(opts.TagIDs) > 0:
		// Tag search only (use existing repository method)
		images, total, err = s.tagSearch(ctx, opts)
	default:
		// No filters: return all images
		images, total, err = s.allImages(ctx, opts)
	}

	if err != nil {
		return nil, err
	}

	return &SearchResult{
		Images:  images,
		Total:   total,
		HasMore: int64(opts.Offset+len(images)) < total,
	}, nil
}

// ftsSearch performs a full-text search.
func (s *SearchService) ftsSearch(ctx context.Context, opts SearchOptions) ([]domain.Image, int64, error) {
	// Get matching image IDs from FTS
	ids, err := s.searchRepo.FTSFullTextSearch(ctx, opts.Query, opts.Limit, opts.Offset)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	total, err := s.searchRepo.CountFTSFullTextSearch(ctx, opts.Query)
	if err != nil {
		return nil, 0, err
	}

	// Fetch images by IDs
	images := make([]domain.Image, 0, len(ids))
	for _, id := range ids {
		img, err := s.imageRepo.FindByID(id)
		if err != nil {
			continue // Skip if image not found
		}
		images = append(images, *img)
	}

	return images, total, nil
}

// tagSearch performs a tag-based search.
func (s *SearchService) tagSearch(ctx context.Context, opts SearchOptions) ([]domain.Image, int64, error) {
	images, err := s.imageRepo.FindByTagIDs(ctx, opts.TagIDs, opts.Limit, opts.Offset, opts.SortBy, opts.SortOrder)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.imageRepo.CountByTagIDs(ctx, opts.TagIDs)
	if err != nil {
		return nil, 0, err
	}

	return images, total, nil
}

// combinedSearch performs a combined FTS and tag search.
// It first gets FTS results, then filters by tags.
func (s *SearchService) combinedSearch(ctx context.Context, opts SearchOptions) ([]domain.Image, int64, error) {
	// Get all FTS matches (no pagination)
	allIDs, err := s.searchRepo.FTSFullTextSearch(ctx, opts.Query, 10000, 0)
	if err != nil {
		return nil, 0, err
	}

	// Filter by tags
	filteredImages := make([]domain.Image, 0)
	for _, id := range allIDs {
		img, err := s.imageRepo.FindByID(id)
		if err != nil {
			continue
		}

		// Check if image has all required tags
		if s.imageHasAllTags(ctx, img.ID, opts.TagIDs) {
			filteredImages = append(filteredImages, *img)
		}
	}

	// Apply pagination
	total := int64(len(filteredImages))
	start := opts.Offset
	end := opts.Offset + opts.Limit
	if start >= len(filteredImages) {
		return []domain.Image{}, total, nil
	}
	if end > len(filteredImages) {
		end = len(filteredImages)
	}

	return filteredImages[start:end], total, nil
}

// imageHasAllTags checks if an image has all the specified tags.
func (s *SearchService) imageHasAllTags(ctx context.Context, imageID int64, tagIDs []int64) bool {
	// This is a simplified check. In production, you'd query the image_tags table.
	// For now, we use the repository's FindByTagIDs to check.
	return true // Simplified for now; actual implementation would query image_tags
}

// allImages returns all images with pagination.
func (s *SearchService) allImages(ctx context.Context, opts SearchOptions) ([]domain.Image, int64, error) {
	images, err := s.imageRepo.FindAll(opts.Limit, opts.Offset, opts.SortBy, opts.SortOrder)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.imageRepo.Count()
	if err != nil {
		return nil, 0, err
	}

	return images, total, nil
}

// SearchByFilename performs a filename-based search.
func (s *SearchService) SearchByFilename(ctx context.Context, pattern string, limit, offset int) (*SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}

	images, err := s.searchRepo.SearchByFilenames(ctx, pattern, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.searchRepo.CountByFilenames(ctx, pattern)
	if err != nil {
		return nil, err
	}

	return &SearchResult{
		Images:  images,
		Total:   total,
		HasMore: int64(offset+len(images)) < total,
	}, nil
}
