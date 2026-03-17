package handler

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

func TestSearchHandler_Search(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Create test images
	for i := 0; i < 5; i++ {
		_, err := db.Exec(`
			INSERT INTO images (path, filename, source_root, file_size, width, height, format, created_at, updated_at)
			VALUES (?, ?, '', 1000, 100, 100, 'jpg', datetime('now'), datetime('now'))
		`, "/images/test"+string(rune('0'+i))+".jpg", "test_image_"+string(rune('0'+i))+".jpg")
		if err != nil {
			t.Fatalf("Failed to create test image: %v", err)
		}
	}

	// Rebuild FTS
	if err := repository.RebuildFTSIndex(db); err != nil {
		t.Fatalf("Failed to rebuild FTS: %v", err)
	}

	// Create handler
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	searchRepo := repository.NewSearchRepository(db)
	searchService := service.NewSearchService(imageRepo, tagRepo, searchRepo)
	handler := NewSearchHandler(searchService)

	tests := []struct {
		name       string
		query      string
		expectCode int
		expectLen  int
	}{
		{
			name:       "search for test",
			query:      "q=test",
			expectCode: http.StatusOK,
			expectLen:  5,
		},
		{
			name:       "empty query returns all",
			query:      "",
			expectCode: http.StatusOK,
			expectLen:  5,
		},
		{
			name:       "no results",
			query:      "q=nonexistent",
			expectCode: http.StatusOK,
			expectLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/api/v1/search?"+tt.query, nil)

			handler.Search(c)

			if w.Code != tt.expectCode {
				t.Errorf("Expected status %d, got %d", tt.expectCode, w.Code)
			}
		})
	}
}

func TestSearchHandler_SearchWithPagination(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Create 25 test images
	for i := 0; i < 25; i++ {
		img := &domain.Image{
			Path:     "/images/pagination_test_" + string(rune('a'+i%26)) + ".jpg",
			Filename: "pagination_test_" + string(rune('a'+i%26)) + ".jpg",
		}
		_, err := db.Exec(`
			INSERT INTO images (path, filename, source_root, file_size, width, height, format, created_at, updated_at)
			VALUES (?, ?, '', 1000, 100, 100, 'jpg', datetime('now'), datetime('now'))
		`, img.Path, img.Filename)
		if err != nil {
			t.Fatalf("Failed to create test image: %v", err)
		}
	}

	// Rebuild FTS
	if err := repository.RebuildFTSIndex(db); err != nil {
		t.Fatalf("Failed to rebuild FTS: %v", err)
	}

	// Create handler
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	searchRepo := repository.NewSearchRepository(db)
	searchService := service.NewSearchService(imageRepo, tagRepo, searchRepo)
	handler := NewSearchHandler(searchService)

	// Test first page
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/search?q=pagination&limit=10&offset=0", nil)

	handler.Search(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestSearchHandler_SearchWithTags(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Create repositories
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)

	// Create tags
	tag1 := &domain.Tag{PreferredLabel: "animal", Slug: "animal"}
	if err := tagRepo.Save(context.Background(), tag1); err != nil {
		t.Fatalf("Failed to create tag: %v", err)
	}

	// Create images
	img1 := &domain.Image{Path: "/img1.jpg", Filename: "tagged_image.jpg"}
	if err := imageRepo.SaveImage(img1); err != nil {
		t.Fatalf("Failed to save image: %v", err)
	}

	// Tag image
	_, err = db.Exec(`INSERT INTO image_tags (image_id, tag_id) VALUES (?, ?)`, img1.ID, tag1.ID)
	if err != nil {
		t.Fatalf("Failed to tag image: %v", err)
	}

	// Create handler
	searchRepo := repository.NewSearchRepository(db)
	searchService := service.NewSearchService(imageRepo, tagRepo, searchRepo)
	handler := NewSearchHandler(searchService)

	// Test tag search
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/search?tag_ids="+string(rune('0'+tag1.ID)), nil)

	handler.Search(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestSearchHandler_SearchByFilename(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Create test images
	_, err = db.Exec(`
		INSERT INTO images (path, filename, source_root, file_size, width, height, format, created_at, updated_at)
		VALUES ('/img1.jpg', 'special_filename.jpg', '', 1000, 100, 100, 'jpg', datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	// Create handler
	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	searchRepo := repository.NewSearchRepository(db)
	searchService := service.NewSearchService(imageRepo, tagRepo, searchRepo)
	handler := NewSearchHandler(searchService)

	// Test filename search
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/search/filename?pattern=special", nil)

	handler.SearchByFilename(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Test missing pattern
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest("GET", "/api/v1/search/filename", nil)

	handler.SearchByFilename(c2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for missing pattern, got %d", http.StatusBadRequest, w2.Code)
	}
}
