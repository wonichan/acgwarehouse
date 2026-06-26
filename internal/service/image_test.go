package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/service"
)

type memoryImageRepository struct {
	images []do.Image
}

func (r *memoryImageRepository) ListActive(_ context.Context, query service.RepositoryImageQuery) ([]do.Image, error) {
	return r.images[:min(query.Size, len(r.images))], nil
}

func (r *memoryImageRepository) CountActiveByQuery(_ context.Context, _ service.RepositoryImageQuery) (int64, error) {
	return int64(len(r.images)), nil
}

func (r *memoryImageRepository) FindActiveByID(_ context.Context, id int64) (do.Image, error) {
	for _, image := range r.images {
		if image.ID == id {
			return image, nil
		}
	}
	return do.Image{}, service.ErrImageNotFound
}

func (r *memoryImageRepository) FindActiveByIDs(_ context.Context, ids []int64) ([]do.Image, error) {
	result := make([]do.Image, 0, len(ids))
	for _, id := range ids {
		for _, image := range r.images {
			if image.ID == id {
				result = append(result, image)
			}
		}
	}
	return result, nil
}

func (r *memoryImageRepository) SoftDelete(_ context.Context, _ int64, _ time.Time) error {
	return nil
}

func (r *memoryImageRepository) Restore(_ context.Context, id int64) (do.Image, error) {
	return r.FindActiveByID(context.Background(), id)
}

type memoryImageSearcher struct {
	ids   []int64
	total int64
}

func (s memoryImageSearcher) Search(_ context.Context, query service.SearchQuery) (do.ImageSearchResult, error) {
	limit := min(query.Size, len(s.ids))
	total := s.total
	if total == 0 {
		total = int64(len(s.ids))
	}
	return do.ImageSearchResult{IDs: s.ids[:limit], Total: total}, nil
}

func (s memoryImageSearcher) Index(_ context.Context, _ do.Image) error {
	return nil
}

func (s memoryImageSearcher) Delete(_ context.Context, _ int64) error {
	return nil
}

type memoryViewRecorder struct {
	events []do.ImageEvent
}

func (r *memoryViewRecorder) RecordView(_ context.Context, event do.ImageEvent) error {
	r.events = append(r.events, event)
	return nil
}

func Test_ImageService_Detail_returns_placeholder_fields_and_records_view(t *testing.T) {
	// Given
	repo := &memoryImageRepository{images: []do.Image{{
		ID:            7,
		COSKey:        "thumbnails/miku.png",
		Filename:      "miku.png",
		CreatedAt:     fixedServiceImageTime(),
		Status:        do.ImageStatusActive,
		AvgScore:      88,
		RatingCount:   2,
		ViewCount:     9,
		FavoriteCount: 3,
	}}}
	recorder := &memoryViewRecorder{}
	svc := service.NewImageService(repo, memoryImageSearcher{}, recorder, "https://cdn.example.com")

	// When
	detail, err := svc.Detail(context.Background(), 7, 0)

	// Then
	if err != nil {
		t.Fatalf("image detail: %v", err)
	}
	if detail.Image.URL != "https://cdn.example.com/thumbnails/miku.png" {
		t.Fatalf("url = %q, want full CDN URL", detail.Image.URL)
	}
	if detail.MyRating != nil || detail.IsCollected || len(detail.Tags) != 0 || len(detail.SimilarImages) != 0 {
		t.Fatalf("detail = %#v, want stable phase-03 placeholders", detail)
	}
	if len(recorder.events) != 1 || recorder.events[0].ImageID != 7 || recorder.events[0].Type != do.ImageEventTypeView {
		t.Fatalf("events = %#v, want one view event", recorder.events)
	}
}

func Test_ImageService_Search_returns_images_in_search_order(t *testing.T) {
	// Given
	repo := &memoryImageRepository{images: []do.Image{
		{ID: 1, COSKey: "thumbnails/one.png", Filename: "one.png", Status: do.ImageStatusActive},
		{ID: 2, COSKey: "thumbnails/two.png", Filename: "two.png", Status: do.ImageStatusActive},
	}}
	svc := service.NewImageService(repo, memoryImageSearcher{ids: []int64{2, 1}}, nil, "https://cdn.example.com")

	// When
	result, err := svc.Search(context.Background(), service.SearchQuery{Text: "two", Page: 1, Size: 10})

	// Then
	if err != nil {
		t.Fatalf("search images: %v", err)
	}
	if result.Total != 2 || len(result.List) != 2 || result.List[0].ID != 2 || result.List[1].ID != 1 {
		t.Fatalf("result = %#v, want search order [2,1]", result)
	}
}

func Test_ImageService_Search_returns_bleve_total_when_deleted_hits_are_filtered(t *testing.T) {
	// Given
	repo := &memoryImageRepository{images: []do.Image{
		{ID: 2, COSKey: "thumbnails/two.png", Filename: "two.png", Status: do.ImageStatusActive},
	}}
	svc := service.NewImageService(repo, memoryImageSearcher{ids: []int64{2, 99}, total: 2}, nil, "https://cdn.example.com")

	// When
	result, err := svc.Search(context.Background(), service.SearchQuery{Text: "two", Page: 1, Size: 10})

	// Then
	if err != nil {
		t.Fatalf("search images: %v", err)
	}
	if result.Total != 2 || len(result.List) != 1 || result.List[0].ID != 2 {
		t.Fatalf("result = %#v, want bleve total 2 with only active image 2 returned", result)
	}
}

func fixedServiceImageTime() time.Time {
	return time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
}
