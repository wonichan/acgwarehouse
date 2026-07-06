package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/service"
)

type memoryImageRepository struct {
	images            []do.Image
	similarByTag      []do.Image
	similarByCategory []do.Image
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

// FindSimilarByTagIDs 模拟按标签重叠查询相似图片，过滤排除自身并截断 limit。
func (r *memoryImageRepository) FindSimilarByTagIDs(_ context.Context, tagIDs []int64, excludeImageID int64, limit int) ([]do.Image, error) {
	if len(tagIDs) == 0 || limit <= 0 || excludeImageID < 1 {
		return []do.Image{}, nil
	}
	result := make([]do.Image, 0, limit)
	for _, img := range r.similarByTag {
		if img.ID == excludeImageID {
			continue
		}
		if len(result) >= limit {
			break
		}
		result = append(result, img)
	}
	return result, nil
}

// FindSimilarByCategory 模拟按分类查询相似图片，过滤排除 ID 并截断 limit。
func (r *memoryImageRepository) FindSimilarByCategory(_ context.Context, category string, excludeImageIDs []int64, limit int) ([]do.Image, error) {
	if category == "" || limit <= 0 {
		return []do.Image{}, nil
	}
	excluded := make(map[int64]bool, len(excludeImageIDs))
	for _, id := range excludeImageIDs {
		excluded[id] = true
	}
	result := make([]do.Image, 0, limit)
	for _, img := range r.similarByCategory {
		if excluded[img.ID] {
			continue
		}
		if len(result) >= limit {
			break
		}
		result = append(result, img)
	}
	return result, nil
}

// memoryImageTagReader 模拟图片标签读取器，按 imageID 返回预设标签。
type memoryImageTagReader struct {
	tagsByImage map[int64][]do.Tag
}

// ListByImageID 返回预设的图片标签列表。
func (r *memoryImageTagReader) ListByImageID(_ context.Context, imageID int64) ([]do.Tag, error) {
	return append([]do.Tag{}, r.tagsByImage[imageID]...), nil
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
	if detail.Image.ViewCount != 10 {
		t.Fatalf("view_count = %d, want current view included", detail.Image.ViewCount)
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

func Test_ImageService_Detail_similar_images_tag_overlap_sufficient_no_fallback(t *testing.T) {
	// Given：当前图片有标签，标签重叠返回 6 张，无需 category 回退。
	current := do.Image{ID: 1, COSKey: "thumbnails/cur.png", Filename: "cur.png", Status: do.ImageStatusActive}
	tagSimilar := make([]do.Image, 0, 6)
	for i := int64(10); i < 16; i++ {
		tagSimilar = append(tagSimilar, do.Image{ID: i, COSKey: "thumbnails/s.png", Filename: "s.png", Status: do.ImageStatusActive})
	}
	repo := &memoryImageRepository{
		images:            []do.Image{current},
		similarByTag:      tagSimilar,
		similarByCategory: []do.Image{{ID: 99, COSKey: "thumbnails/leak.png", Filename: "leak.png", Status: do.ImageStatusActive}},
	}
	tagReader := &memoryImageTagReader{tagsByImage: map[int64][]do.Tag{1: {{ID: 1, Name: "tag1"}}}}
	svc := service.NewImageServiceWithTags(repo, memoryImageSearcher{}, nil, tagReader, "https://cdn.example.com")

	// When
	detail, err := svc.Detail(context.Background(), 1, 0)

	// Then
	if err != nil {
		t.Fatalf("image detail: %v", err)
	}
	if len(detail.SimilarImages) != 6 {
		t.Fatalf("similar count = %d, want 6 (tag overlap sufficient)", len(detail.SimilarImages))
	}
	for i, img := range detail.SimilarImages {
		if img.ID != int64(10)+int64(i) {
			t.Fatalf("similar[%d].id = %d, want %d", i, img.ID, 10+i)
		}
	}
}

func Test_ImageService_Detail_similar_images_falls_back_to_category_when_tag_overlap_insufficient(t *testing.T) {
	// Given：标签重叠只返回 2 张，需用同 category 补 3 张，排除当前图片与已选结果。
	current := do.Image{ID: 1, COSKey: "thumbnails/cur.png", Filename: "cur.png", Category: "art", Status: do.ImageStatusActive}
	repo := &memoryImageRepository{
		images: []do.Image{current},
		similarByTag: []do.Image{
			{ID: 10, COSKey: "thumbnails/t1.png", Filename: "t1.png", Status: do.ImageStatusActive},
			{ID: 11, COSKey: "thumbnails/t2.png", Filename: "t2.png", Status: do.ImageStatusActive},
		},
		similarByCategory: []do.Image{
			{ID: 1, COSKey: "thumbnails/cur.png", Filename: "cur.png", Status: do.ImageStatusActive},
			{ID: 10, COSKey: "thumbnails/t1.png", Filename: "t1.png", Status: do.ImageStatusActive},
			{ID: 20, COSKey: "thumbnails/c1.png", Filename: "c1.png", Status: do.ImageStatusActive},
			{ID: 21, COSKey: "thumbnails/c2.png", Filename: "c2.png", Status: do.ImageStatusActive},
			{ID: 22, COSKey: "thumbnails/c3.png", Filename: "c3.png", Status: do.ImageStatusActive},
		},
	}
	tagReader := &memoryImageTagReader{tagsByImage: map[int64][]do.Tag{1: {{ID: 1, Name: "tag1"}}}}
	svc := service.NewImageServiceWithTags(repo, memoryImageSearcher{}, nil, tagReader, "https://cdn.example.com")

	// When
	detail, err := svc.Detail(context.Background(), 1, 0)

	// Then
	if err != nil {
		t.Fatalf("image detail: %v", err)
	}
	if len(detail.SimilarImages) != 5 {
		t.Fatalf("similar count = %d, want 5 (2 tag + 3 category)", len(detail.SimilarImages))
	}
	ids := make([]int64, 0, 5)
	for _, img := range detail.SimilarImages {
		ids = append(ids, img.ID)
	}
	// 前 2 张来自标签重叠，后 3 张来自 category 回退（排除 ID 1、10、11）。
	wantIDs := []int64{10, 11, 20, 21, 22}
	for i, want := range wantIDs {
		if ids[i] != want {
			t.Fatalf("similar ids = %v, want %v", ids, wantIDs)
		}
	}
}

func Test_ImageService_Detail_similar_images_empty_when_no_tags_and_no_category(t *testing.T) {
	// Given：当前图片无标签、无 category，相似推荐应为空。
	current := do.Image{ID: 1, COSKey: "thumbnails/cur.png", Filename: "cur.png", Category: "", Status: do.ImageStatusActive}
	repo := &memoryImageRepository{
		images:            []do.Image{current},
		similarByTag:      []do.Image{{ID: 10, COSKey: "thumbnails/s.png", Filename: "s.png", Status: do.ImageStatusActive}},
		similarByCategory: []do.Image{{ID: 20, COSKey: "thumbnails/c.png", Filename: "c.png", Status: do.ImageStatusActive}},
	}
	tagReader := &memoryImageTagReader{tagsByImage: map[int64][]do.Tag{}}
	svc := service.NewImageServiceWithTags(repo, memoryImageSearcher{}, nil, tagReader, "https://cdn.example.com")

	// When
	detail, err := svc.Detail(context.Background(), 1, 0)

	// Then
	if err != nil {
		t.Fatalf("image detail: %v", err)
	}
	if len(detail.SimilarImages) != 0 {
		t.Fatalf("similar count = %d, want 0 (no tags, no category)", len(detail.SimilarImages))
	}
}
