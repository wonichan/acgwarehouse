package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

func TestSearchRepository_FTSFullTextSearch(t *testing.T) {
	// Create in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Ensure schema
	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Create test images
	testImages := []struct {
		filename string
		path     string
	}{
		{"cat_in_garden.jpg", "/images/cat_in_garden.jpg"},
		{"dog_running.png", "/images/dog_running.png"},
		{"cat_sleeping.jpg", "/images/cat_sleeping.jpg"},
		{"flower_garden.jpg", "/images/flower_garden.jpg"},
		{"anime_cat_girl.png", "/images/anime_cat_girl.png"},
	}

	for _, img := range testImages {
		_, err := db.Exec(`
			INSERT INTO images (path, filename, source_root, file_size, width, height, format, created_at, updated_at)
			VALUES (?, ?, '', 1000, 100, 100, 'jpg', datetime('now'), datetime('now'))
		`, img.path, img.filename)
		if err != nil {
			t.Fatalf("Failed to create test image %s: %v", img.filename, err)
		}
	}

	// Rebuild FTS index
	if err := RebuildFTSIndex(db); err != nil {
		t.Fatalf("Failed to rebuild FTS index: %v", err)
	}

	searchRepo := NewSearchRepository(db)

	tests := []struct {
		name            string
		query           string
		expectCount     int
		expectFilenames []string
	}{
		{
			name:        "search for cat",
			query:       "cat",
			expectCount: 3,
		},
		{
			name:        "search for garden",
			query:       "garden",
			expectCount: 2,
		},
		{
			name:        "search for anime",
			query:       "anime",
			expectCount: 1,
		},
		{
			name:        "search for dog",
			query:       "dog",
			expectCount: 1,
		},
		{
			name:        "search for nonexistent",
			query:       "nonexistent",
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids, err := searchRepo.FTSFullTextSearch(context.Background(), tt.query, 100, 0)
			if err != nil {
				t.Fatalf("FTSFullTextSearch failed: %v", err)
			}

			if len(ids) != tt.expectCount {
				t.Errorf("Expected %d results, got %d (ids: %v)", tt.expectCount, len(ids), ids)
			}

			// Verify count matches
			count, err := searchRepo.CountFTSFullTextSearch(context.Background(), tt.query)
			if err != nil {
				t.Fatalf("CountFTSFullTextSearch failed: %v", err)
			}
			if int(count) != tt.expectCount {
				t.Errorf("CountFTSFullTextSearch: expected %d, got %d", tt.expectCount, count)
			}
		})
	}
}

func TestSearchRepository_SearchByFilenames(t *testing.T) {
	// Create in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Ensure schema
	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Create test images
	testImages := []struct {
		id       int64
		filename string
		path     string
	}{
		{1, "test_image.jpg", "/images/test_image.jpg"},
		{2, "another_test.png", "/images/another_test.png"},
		{3, "different.jpg", "/images/different.jpg"},
	}

	for _, img := range testImages {
		_, err := db.Exec(`
			INSERT INTO images (id, path, filename, source_root, file_size, width, height, format, created_at, updated_at)
			VALUES (?, ?, ?, '', 1000, 100, 100, 'jpg', datetime('now'), datetime('now'))
		`, img.id, img.path, img.filename)
		if err != nil {
			t.Fatalf("Failed to create test image: %v", err)
		}
	}

	searchRepo := NewSearchRepository(db)

	// Test filename search
	images, err := searchRepo.SearchByFilenames(context.Background(), "test", 100, 0)
	if err != nil {
		t.Fatalf("SearchByFilenames failed: %v", err)
	}

	if len(images) != 2 {
		t.Errorf("Expected 2 images, got %d", len(images))
	}

	// Verify count
	count, err := searchRepo.CountByFilenames(context.Background(), "test")
	if err != nil {
		t.Fatalf("CountByFilenames failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
}

func TestSearchRepository_CombinedSearch(t *testing.T) {
	// Create in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Ensure schema
	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Create repositories
	imageRepo := NewImageRepository(db)

	// Create tags
	tagRepo := NewTagRepository(db)
	tag1 := &domain.Tag{PreferredLabel: "cat", Slug: "cat"}
	tag2 := &domain.Tag{PreferredLabel: "dog", Slug: "dog"}
	if err := tagRepo.Save(context.Background(), tag1); err != nil {
		t.Fatalf("Failed to create tag1: %v", err)
	}
	if err := tagRepo.Save(context.Background(), tag2); err != nil {
		t.Fatalf("Failed to create tag2: %v", err)
	}

	// Create images
	image1 := &domain.Image{Path: "/img1.jpg", Filename: "cat_image.jpg"}
	image2 := &domain.Image{Path: "/img2.jpg", Filename: "dog_image.jpg"}
	image3 := &domain.Image{Path: "/img3.jpg", Filename: "bird_image.jpg"}

	if err := imageRepo.SaveImage(image1); err != nil {
		t.Fatalf("Failed to save image1: %v", err)
	}
	if err := imageRepo.SaveImage(image2); err != nil {
		t.Fatalf("Failed to save image2: %v", err)
	}
	if err := imageRepo.SaveImage(image3); err != nil {
		t.Fatalf("Failed to save image3: %v", err)
	}

	// Add tags to images
	_, err = db.Exec(`INSERT INTO image_tags (image_id, tag_id) VALUES (?, ?)`, image1.ID, tag1.ID)
	if err != nil {
		t.Fatalf("Failed to tag image1: %v", err)
	}
	_, err = db.Exec(`INSERT INTO image_tags (image_id, tag_id) VALUES (?, ?)`, image2.ID, tag2.ID)
	if err != nil {
		t.Fatalf("Failed to tag image2: %v", err)
	}

	// Rebuild FTS
	if err := RebuildFTSIndex(db); err != nil {
		t.Fatalf("Failed to rebuild FTS: %v", err)
	}

	// Test that FTS can find images by tag
	searchRepo := NewSearchRepository(db)
	ids, err := searchRepo.FTSFullTextSearch(context.Background(), "cat", 100, 0)
	if err != nil {
		t.Fatalf("FTSFullTextSearch failed: %v", err)
	}

	if len(ids) < 1 {
		t.Errorf("Expected at least 1 result for 'cat', got %d", len(ids))
	}
}
