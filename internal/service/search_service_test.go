package service

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

func TestSearchService_FullTextSearch(t *testing.T) {
	// Create in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Ensure schema
	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Create repositories
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	searchRepo := repository.NewSearchRepository(db)

	// Create test images
	testImages := []struct {
		filename string
		path     string
	}{
		{"cat_in_garden.jpg", "/images/cat_in_garden.jpg"},
		{"dog_running.png", "/images/dog_running.png"},
		{"cat_sleeping.jpg", "/images/cat_sleeping.jpg"},
	}

	for _, img := range testImages {
		_, err := db.Exec(`
			INSERT INTO images (path, filename, source_root, file_size, width, height, format, created_at, updated_at)
			VALUES (?, ?, '', 1000, 100, 100, 'jpg', datetime('now'), datetime('now'))
		`, img.path, img.filename)
		if err != nil {
			t.Fatalf("Failed to create test image: %v", err)
		}
	}

	// Rebuild FTS index
	if err := repository.RebuildFTSIndex(db); err != nil {
		t.Fatalf("Failed to rebuild FTS index: %v", err)
	}

	// Create service
	searchService := NewSearchService(imageRepo, tagRepo, searchRepo)

	// Test search
	result, err := searchService.Search(context.Background(), SearchOptions{
		Query:  "cat",
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(result.Images) != 2 {
		t.Errorf("Expected 2 images, got %d", len(result.Images))
	}
	if result.Total != 2 {
		t.Errorf("Expected total 2, got %d", result.Total)
	}
}

func TestSearchService_TagSearch(t *testing.T) {
	// Create in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Ensure schema
	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Create repositories
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	searchRepo := repository.NewSearchRepository(db)

	// Create tags
	tag1 := &domain.Tag{PreferredLabel: "animal", Slug: "animal"}
	tag2 := &domain.Tag{PreferredLabel: "nature", Slug: "nature"}
	if err := tagRepo.Save(context.Background(), tag1); err != nil {
		t.Fatalf("Failed to create tag1: %v", err)
	}
	if err := tagRepo.Save(context.Background(), tag2); err != nil {
		t.Fatalf("Failed to create tag2: %v", err)
	}

	// Create images
	img1 := &domain.Image{Path: "/img1.jpg", Filename: "image1.jpg"}
	img2 := &domain.Image{Path: "/img2.jpg", Filename: "image2.jpg"}
	if _, err := imageRepo.SaveImage(img1); err != nil {
		t.Fatalf("Failed to save img1: %v", err)
	}
	if _, err := imageRepo.SaveImage(img2); err != nil {
		t.Fatalf("Failed to save img2: %v", err)
	}

	// Tag images
	_, err = db.Exec(`INSERT INTO image_tags (image_id, tag_id) VALUES (?, ?)`, img1.ID, tag1.ID)
	if err != nil {
		t.Fatalf("Failed to tag img1: %v", err)
	}
	_, err = db.Exec(`INSERT INTO image_tags (image_id, tag_id) VALUES (?, ?)`, img2.ID, tag1.ID)
	if err != nil {
		t.Fatalf("Failed to tag img2: %v", err)
	}

	// Create service
	searchService := NewSearchService(imageRepo, tagRepo, searchRepo)

	// Test tag search
	result, err := searchService.Search(context.Background(), SearchOptions{
		TagIDs: []int64{tag1.ID},
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(result.Images) != 2 {
		t.Errorf("Expected 2 images, got %d", len(result.Images))
	}
}

func TestSearchService_Pagination(t *testing.T) {
	// Create in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Ensure schema
	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Create repositories
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	searchRepo := repository.NewSearchRepository(db)

	// Create 25 test images
	for i := 0; i < 25; i++ {
		img := &domain.Image{
			Path:     "/images/test_image_" + string(rune('a'+i%26)) + ".jpg",
			Filename: "test_image_" + string(rune('a'+i%26)) + ".jpg",
		}
		if _, err := imageRepo.SaveImage(img); err != nil {
			t.Fatalf("Failed to save image: %v", err)
		}
	}

	// Rebuild FTS index
	if err := repository.RebuildFTSIndex(db); err != nil {
		t.Fatalf("Failed to rebuild FTS index: %v", err)
	}

	// Create service
	searchService := NewSearchService(imageRepo, tagRepo, searchRepo)

	// Test first page
	result1, err := searchService.Search(context.Background(), SearchOptions{
		Query:  "test",
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(result1.Images) != 10 {
		t.Errorf("Expected 10 images on first page, got %d", len(result1.Images))
	}
	if !result1.HasMore {
		t.Error("Expected HasMore to be true")
	}

	// Test second page
	result2, err := searchService.Search(context.Background(), SearchOptions{
		Query:  "test",
		Limit:  10,
		Offset: 10,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(result2.Images) > 10 {
		t.Errorf("Expected at most 10 images on second page, got %d", len(result2.Images))
	}
}

func TestSearchService_EmptyQuery(t *testing.T) {
	// Create in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Ensure schema
	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Create repositories
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	searchRepo := repository.NewSearchRepository(db)

	// Create test images
	for i := 0; i < 5; i++ {
		img := &domain.Image{
			Path:     "/images/img" + string(rune('0'+i)) + ".jpg",
			Filename: "img" + string(rune('0'+i)) + ".jpg",
		}
		if _, err := imageRepo.SaveImage(img); err != nil {
			t.Fatalf("Failed to save image: %v", err)
		}
	}

	// Create service
	searchService := NewSearchService(imageRepo, tagRepo, searchRepo)

	// Test empty query (should return all images)
	result, err := searchService.Search(context.Background(), SearchOptions{
		Query:  "",
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(result.Images) != 5 {
		t.Errorf("Expected 5 images, got %d", len(result.Images))
	}
}

func TestSearchService_ViewerWindowSearchWithTagsUsesRealTagFiltering(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	searchRepo := repository.NewSearchRepository(db)
	imageTagRepo := repository.NewImageTagRepository(db)
	searchService := NewSearchService(imageRepo, tagRepo, searchRepo)
	ctx := context.Background()
	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)

	tag := &domain.Tag{PreferredLabel: "keep", Slug: "keep", ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, tag); err != nil {
		t.Fatalf("save tag: %v", err)
	}

	for i := 0; i < 4; i++ {
		img := &domain.Image{
			Path:       fmt.Sprintf("/images/viewer-%d-cat.jpg", i),
			Filename:   fmt.Sprintf("viewer-%d-cat.jpg", i),
			SourceRoot: "/images",
			FileSize:   100,
			Format:     "jpg",
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if _, err := imageRepo.SaveImage(img); err != nil {
			t.Fatalf("save image: %v", err)
		}
		if i == 0 || i == 2 {
			if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: img.ID, TagID: tag.ID, ReviewState: "confirmed"}); err != nil {
				t.Fatalf("save image tag: %v", err)
			}
		}
	}

	if err := repository.RebuildFTSIndex(db); err != nil {
		t.Fatalf("rebuild fts: %v", err)
	}

	window, err := searchService.ViewerWindow(ctx, SearchOptions{
		Query:     "cat",
		TagIDs:    []int64{tag.ID},
		SortBy:    "created_at",
		SortOrder: "asc",
	}, 1, 3)
	if err != nil {
		t.Fatalf("ViewerWindow failed: %v", err)
	}

	if window.Total != 2 {
		t.Fatalf("window.Total = %d, want 2", window.Total)
	}
	if len(window.Images) != 2 {
		t.Fatalf("len(window.Images) = %d, want 2", len(window.Images))
	}
	if window.WindowStart != 0 {
		t.Fatalf("window.WindowStart = %d, want 0", window.WindowStart)
	}
	if window.Images[0].ID >= window.Images[1].ID {
		t.Fatalf("expected deterministic id order, got %d then %d", window.Images[0].ID, window.Images[1].ID)
	}
	if window.Images[window.SelectedIndexInWindow].ID != window.Images[1].ID {
		t.Fatalf("selected image id = %d, want %d", window.Images[window.SelectedIndexInWindow].ID, window.Images[1].ID)
	}
}

func TestSearchService_CombinedSearchRespectsTagFilter(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	ctx := context.Background()
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	searchRepo := repository.NewSearchRepository(db)
	imageTagRepo := repository.NewImageTagRepository(db)
	searchService := NewSearchService(imageRepo, tagRepo, searchRepo)

	tag := &domain.Tag{PreferredLabel: "required", Slug: "required", ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, tag); err != nil {
		t.Fatalf("save tag: %v", err)
	}

	for i := 0; i < 3; i++ {
		img := &domain.Image{Path: fmt.Sprintf("/images/cat-%d.jpg", i), Filename: fmt.Sprintf("cat-%d.jpg", i)}
		if _, err := imageRepo.SaveImage(img); err != nil {
			t.Fatalf("save image: %v", err)
		}
		if i == 1 {
			if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: img.ID, TagID: tag.ID, ReviewState: "confirmed"}); err != nil {
				t.Fatalf("save image tag: %v", err)
			}
		}
	}

	if err := repository.RebuildFTSIndex(db); err != nil {
		t.Fatalf("rebuild fts: %v", err)
	}

	result, err := searchService.Search(ctx, SearchOptions{Query: "cat", TagIDs: []int64{tag.ID}, Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("result.Total = %d, want 1", result.Total)
	}
	if len(result.Images) != 1 {
		t.Fatalf("len(result.Images) = %d, want 1", len(result.Images))
	}
	if result.Images[0].Filename != "cat-1.jpg" {
		t.Fatalf("result.Images[0].Filename = %q, want %q", result.Images[0].Filename, "cat-1.jpg")
	}
}

func TestSearchService_ViewerWindowAllImagesBeyondTenThousand(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	searchRepo := repository.NewSearchRepository(db)
	searchService := NewSearchService(imageRepo, tagRepo, searchRepo)

	for i := 0; i < 10005; i++ {
		img := &domain.Image{Path: fmt.Sprintf("/images/all-%05d.jpg", i), Filename: fmt.Sprintf("all-%05d.jpg", i)}
		if _, err := imageRepo.SaveImage(img); err != nil {
			t.Fatalf("save image %d: %v", i, err)
		}
	}

	window, err := searchService.ViewerWindow(context.Background(), SearchOptions{SortBy: "id", SortOrder: "asc"}, 10002, 5)
	if err != nil {
		t.Fatalf("ViewerWindow failed: %v", err)
	}
	if window.Total != 10005 {
		t.Fatalf("window.Total = %d, want 10005", window.Total)
	}
	if len(window.Images) != 5 {
		t.Fatalf("len(window.Images) = %d, want 5", len(window.Images))
	}
	if window.Images[window.SelectedIndexInWindow].Filename != "all-10002.jpg" {
		t.Fatalf("selected filename = %q, want %q", window.Images[window.SelectedIndexInWindow].Filename, "all-10002.jpg")
	}
}

func TestSearchService_ViewerWindowFTSQueryBeyondTenThousand(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	searchRepo := repository.NewSearchRepository(db)
	searchService := NewSearchService(imageRepo, tagRepo, searchRepo)

	for i := 0; i < 10005; i++ {
		img := &domain.Image{Path: fmt.Sprintf("/images/cat-%05d.jpg", i), Filename: fmt.Sprintf("cat-%05d.jpg", i)}
		if _, err := imageRepo.SaveImage(img); err != nil {
			t.Fatalf("save image %d: %v", i, err)
		}
	}
	if err := repository.RebuildFTSIndex(db); err != nil {
		t.Fatalf("rebuild fts: %v", err)
	}

	window, err := searchService.ViewerWindow(context.Background(), SearchOptions{Query: "cat", SortBy: "relevance", SortOrder: "desc"}, 10002, 5)
	if err != nil {
		t.Fatalf("ViewerWindow failed: %v", err)
	}
	if window.Total != 10005 {
		t.Fatalf("window.Total = %d, want 10005", window.Total)
	}
	if len(window.Images) != 5 {
		t.Fatalf("len(window.Images) = %d, want 5", len(window.Images))
	}
	if window.Images[window.SelectedIndexInWindow].Filename != "cat-10002.jpg" {
		t.Fatalf("selected filename = %q, want %q", window.Images[window.SelectedIndexInWindow].Filename, "cat-10002.jpg")
	}
}
