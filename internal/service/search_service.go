package service

import (
	"context"
	"errors"
	"log"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

var ErrViewerRequestOutOfRange = errors.New("selected_index is out of range for the supplied snapshot")

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

type ViewerWindowResult struct {
	Images                []domain.Image
	Total                 int64
	WindowStart           int
	SelectedIndex         int
	SelectedIndexInWindow int
	HasPrevious           bool
	HasNext               bool
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

	log.Printf("[service] Search started: query_len=%d tag_count=%d sort_by=%s limit=%d", len(opts.Query), len(opts.TagIDs), opts.SortBy, opts.Limit)

	var images []domain.Image
	var total int64
	var err error

	switch {
	case opts.Query != "" && len(opts.TagIDs) > 0:
		images, total, err = s.combinedSearch(ctx, opts)
		if err != nil {
			log.Printf("[service] Search failed: operation=combinedSearch error=%v", err)
		}
	case opts.Query != "":
		images, total, err = s.ftsSearch(ctx, opts)
		if err != nil {
			log.Printf("[service] Search failed: operation=ftsSearch error=%v", err)
		}
	case len(opts.TagIDs) > 0:
		// Tag search only (use existing repository method)
		images, total, err = s.tagSearch(ctx, opts)
		if err != nil {
			log.Printf("[service] Search failed: operation=tagSearch error=%v", err)
		}
	default:
		// No filters: return all images
		images, total, err = s.allImages(ctx, opts)
		if err != nil {
			log.Printf("[service] Search failed: operation=allImages error=%v", err)
		}
	}

	if err != nil {
		return nil, err
	}

	result := &SearchResult{
		Images:  images,
		Total:   total,
		HasMore: int64(opts.Offset+len(images)) < total,
	}
	log.Printf("[service] Search completed: result_count=%d total=%d has_more=%t", len(result.Images), result.Total, result.HasMore)

	return result, nil
}

// ftsSearch performs a full-text search.
func (s *SearchService) ftsSearch(ctx context.Context, opts SearchOptions) ([]domain.Image, int64, error) {
	images, err := s.searchRepo.SearchImages(ctx, repository.SearchQueryOptions{
		Query:     opts.Query,
		SortBy:    opts.SortBy,
		SortOrder: opts.SortOrder,
		Limit:     opts.Limit,
		Offset:    opts.Offset,
	})
	if err != nil {
		log.Printf("[service] ftsSearch failed: operation=searchRepo.SearchImages error=%v", err)
		return nil, 0, err
	}

	total, err := s.searchRepo.CountSearchImages(ctx, repository.SearchQueryOptions{Query: opts.Query, SortBy: opts.SortBy, SortOrder: opts.SortOrder})
	if err != nil {
		log.Printf("[service] ftsSearch failed: operation=searchRepo.CountSearchImages error=%v", err)
		return nil, 0, err
	}

	return images, total, nil
}

// tagSearch performs a tag-based search.
func (s *SearchService) tagSearch(ctx context.Context, opts SearchOptions) ([]domain.Image, int64, error) {
	images, err := s.imageRepo.FindByTagIDs(ctx, opts.TagIDs, opts.Limit, opts.Offset, opts.SortBy, opts.SortOrder)
	if err != nil {
		log.Printf("[service] tagSearch failed: operation=imageRepo.FindByTagIDs error=%v", err)
		return nil, 0, err
	}

	total, err := s.imageRepo.CountByTagIDs(ctx, opts.TagIDs)
	if err != nil {
		log.Printf("[service] tagSearch failed: operation=imageRepo.CountByTagIDs error=%v", err)
		return nil, 0, err
	}

	return images, total, nil
}

// combinedSearch performs a combined FTS and tag search.
// It first gets FTS results, then filters by tags.
func (s *SearchService) combinedSearch(ctx context.Context, opts SearchOptions) ([]domain.Image, int64, error) {
	images, err := s.searchRepo.SearchImages(ctx, repository.SearchQueryOptions{
		Query:     opts.Query,
		TagIDs:    opts.TagIDs,
		SortBy:    opts.SortBy,
		SortOrder: opts.SortOrder,
		Limit:     opts.Limit,
		Offset:    opts.Offset,
	})
	if err != nil {
		log.Printf("[service] combinedSearch failed: operation=searchRepo.SearchImages error=%v", err)
		return nil, 0, err
	}
	total, err := s.searchRepo.CountSearchImages(ctx, repository.SearchQueryOptions{Query: opts.Query, TagIDs: opts.TagIDs, SortBy: opts.SortBy, SortOrder: opts.SortOrder})
	if err != nil {
		log.Printf("[service] combinedSearch failed: operation=searchRepo.CountSearchImages error=%v", err)
		return nil, 0, err
	}
	return images, total, nil
}

// allImages returns all images with pagination.
func (s *SearchService) allImages(_ context.Context, opts SearchOptions) ([]domain.Image, int64, error) {
	images, err := s.imageRepo.FindAll(opts.Limit, opts.Offset, opts.SortBy, opts.SortOrder)
	if err != nil {
		log.Printf("[service] allImages failed: operation=imageRepo.FindAll error=%v", err)
		return nil, 0, err
	}

	total, err := s.imageRepo.Count()
	if err != nil {
		log.Printf("[service] allImages failed: operation=imageRepo.Count error=%v", err)
		return nil, 0, err
	}

	return images, total, nil
}

// SearchByFilename performs a filename-based search.
func (s *SearchService) SearchByFilename(ctx context.Context, pattern string, limit, offset int) (*SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}

	log.Printf("[service] SearchByFilename started: pattern=%s limit=%d", pattern, limit)

	images, err := s.searchRepo.SearchByFilenames(ctx, pattern, limit, offset)
	if err != nil {
		log.Printf("[service] SearchByFilename failed: operation=searchRepo.SearchByFilenames error=%v", err)
		return nil, err
	}

	total, err := s.searchRepo.CountByFilenames(ctx, pattern)
	if err != nil {
		log.Printf("[service] SearchByFilename failed: operation=searchRepo.CountByFilenames error=%v", err)
		return nil, err
	}

	result := &SearchResult{
		Images:  images,
		Total:   total,
		HasMore: int64(offset+len(images)) < total,
	}
	log.Printf("[service] SearchByFilename completed: result_count=%d total=%d has_more=%t", len(result.Images), result.Total, result.HasMore)

	return result, nil
}

func (s *SearchService) ViewerWindow(ctx context.Context, opts SearchOptions, selectedIndex, limit int) (*ViewerWindowResult, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 10 {
		limit = 10
	}

	log.Printf("[service] ViewerWindow started: selected_index=%d limit=%d", selectedIndex, limit)
	if selectedIndex < 0 {
		return nil, ErrViewerRequestOutOfRange
	}

	total, err := s.viewerWindowTotal(ctx, opts)
	if err != nil {
		log.Printf("[service] ViewerWindow failed: operation=viewerWindowTotal error=%v", err)
		return nil, err
	}
	if int64(selectedIndex) >= total {
		return nil, ErrViewerRequestOutOfRange
	}

	windowStart := viewerWindowStart(selectedIndex, limit, int(total))
	images, err := s.viewerWindowImages(ctx, opts, windowStart, limit)
	if err != nil {
		log.Printf("[service] ViewerWindow failed: operation=viewerWindowImages error=%v", err)
		return nil, err
	}

	result := &ViewerWindowResult{
		Images:                images,
		Total:                 total,
		WindowStart:           windowStart,
		SelectedIndex:         selectedIndex,
		SelectedIndexInWindow: selectedIndex - windowStart,
		HasPrevious:           selectedIndex > 0,
		HasNext:               int64(selectedIndex) < total-1,
	}
	log.Printf("[service] ViewerWindow completed: window_size=%d total=%d selected_index=%d", len(result.Images), result.Total, selectedIndex)

	return result, nil
}

func (s *SearchService) viewerWindowTotal(ctx context.Context, opts SearchOptions) (int64, error) {
	if opts.Query != "" {
		total, err := s.searchRepo.CountSearchImages(ctx, repository.SearchQueryOptions{
			Query:     opts.Query,
			TagIDs:    opts.TagIDs,
			SortBy:    opts.SortBy,
			SortOrder: opts.SortOrder,
		})
		if err != nil {
			log.Printf("[service] viewerWindowTotal failed: operation=searchRepo.CountSearchImages error=%v", err)
		}
		return total, err
	}
	if len(opts.TagIDs) > 0 {
		total, err := s.imageRepo.CountByTagIDs(ctx, opts.TagIDs)
		if err != nil {
			log.Printf("[service] viewerWindowTotal failed: operation=imageRepo.CountByTagIDs error=%v", err)
		}
		return total, err
	}
	total, err := s.imageRepo.Count()
	if err != nil {
		log.Printf("[service] viewerWindowTotal failed: operation=imageRepo.Count error=%v", err)
	}
	return total, err
}

func (s *SearchService) viewerWindowImages(ctx context.Context, opts SearchOptions, offset, limit int) ([]domain.Image, error) {
	if opts.Query != "" {
		images, err := s.searchRepo.SearchImages(ctx, repository.SearchQueryOptions{
			Query:     opts.Query,
			TagIDs:    opts.TagIDs,
			SortBy:    opts.SortBy,
			SortOrder: opts.SortOrder,
			Limit:     limit,
			Offset:    offset,
		})
		if err != nil {
			log.Printf("[service] viewerWindowImages failed: operation=searchRepo.SearchImages error=%v", err)
		}
		return images, err
	}
	sortBy := opts.SortBy
	sortOrder := opts.SortOrder
	if sortBy == "" || sortBy == "relevance" {
		sortBy = "id"
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}
	if len(opts.TagIDs) > 0 {
		images, err := s.imageRepo.FindByTagIDs(ctx, opts.TagIDs, limit, offset, sortBy, sortOrder)
		if err != nil {
			log.Printf("[service] viewerWindowImages failed: operation=imageRepo.FindByTagIDs error=%v", err)
		}
		return images, err
	}
	images, err := s.imageRepo.FindAll(limit, offset, sortBy, sortOrder)
	if err != nil {
		log.Printf("[service] viewerWindowImages failed: operation=imageRepo.FindAll error=%v", err)
	}
	return images, err
}

func viewerWindowStart(selectedIndex, limit, total int) int {
	if total <= 0 {
		return 0
	}
	start := selectedIndex - limit/2
	start = max(start, 0)
	maxStart := max(total-limit, 0)
	if start > maxStart {
		start = maxStart
	}
	return start
}
