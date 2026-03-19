package service

import (
	"context"
	"database/sql"
	"testing"

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
