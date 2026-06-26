package search_test

import (
	"context"
	"testing"
	"time"

	"github.com/yachiyo/acgwarehouse/internal/infra/search"
	"github.com/yachiyo/acgwarehouse/internal/model/do"
)

func Test_Index_Search_returns_image_when_query_matches_full_pinyin(t *testing.T) {
	// Given
	idx, err := search.Open(t.TempDir() + "/images.bleve")
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	t.Cleanup(func() {
		if err := idx.Close(); err != nil {
			t.Fatalf("close index: %v", err)
		}
	})
	image := do.Image{
		ID:        7,
		COSKey:    "thumbnails/miku.png",
		Filename:  "初音未来.png",
		Tags:      []string{"歌姬"},
		Size:      1024,
		CreatedAt: time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC),
		Status:    do.ImageStatusActive,
	}
	if err := idx.Index(context.Background(), image); err != nil {
		t.Fatalf("index image: %v", err)
	}

	// When
	result, err := idx.Search(context.Background(), search.Query{Text: "chuyin", Page: 1, Size: 10})

	// Then
	if err != nil {
		t.Fatalf("search image: %v", err)
	}
	if result.Total != 1 || len(result.IDs) != 1 || result.IDs[0] != image.ID {
		t.Fatalf("result = %#v, want one hit for image %d", result, image.ID)
	}
}

func Test_Index_Delete_removes_image_from_results(t *testing.T) {
	// Given
	idx, err := search.Open(t.TempDir() + "/images.bleve")
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	t.Cleanup(func() {
		if err := idx.Close(); err != nil {
			t.Fatalf("close index: %v", err)
		}
	})
	image := do.Image{ID: 8, COSKey: "thumbnails/a.png", Filename: "avatar.png", Status: do.ImageStatusActive}
	if err := idx.Index(context.Background(), image); err != nil {
		t.Fatalf("index image: %v", err)
	}

	// When
	if err := idx.Delete(context.Background(), image.ID); err != nil {
		t.Fatalf("delete image: %v", err)
	}
	result, err := idx.Search(context.Background(), search.Query{Text: "avatar", Page: 1, Size: 10})

	// Then
	if err != nil {
		t.Fatalf("search image: %v", err)
	}
	if result.Total != 0 || len(result.IDs) != 0 {
		t.Fatalf("result = %#v, want empty result", result)
	}
}

func Test_Index_Search_returns_total_for_all_hits_when_page_is_limited(t *testing.T) {
	// Given
	idx, err := search.Open(t.TempDir() + "/images.bleve")
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	t.Cleanup(func() {
		if err := idx.Close(); err != nil {
			t.Fatalf("close index: %v", err)
		}
	})
	for _, image := range []do.Image{
		{ID: 11, COSKey: "thumbnails/one.png", Filename: "avatar-one.png", Status: do.ImageStatusActive},
		{ID: 12, COSKey: "thumbnails/two.png", Filename: "avatar-two.png", Status: do.ImageStatusActive},
	} {
		if err := idx.Index(context.Background(), image); err != nil {
			t.Fatalf("index image: %v", err)
		}
	}

	// When
	result, err := idx.Search(context.Background(), search.Query{Text: "avatar", Page: 1, Size: 1})

	// Then
	if err != nil {
		t.Fatalf("search image: %v", err)
	}
	if result.Total != 2 || len(result.IDs) != 1 {
		t.Fatalf("result = %#v, want total 2 with one paged id", result)
	}
}
