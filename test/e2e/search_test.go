package e2e

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/handler"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

func setupSearchTestServer(t *testing.T) (*gin.Engine, *sql.DB) {
	gin.SetMode(gin.TestMode)

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	searchRepo := repository.NewSearchRepository(db)
	searchSvc := service.NewSearchService(imageRepo, tagRepo, searchRepo)

	r := gin.New()
	api := r.Group("/api/v1")

	searchHandler := handler.NewSearchHandler(searchSvc)
	api.GET("/search", searchHandler.Search)
	api.GET("/search/filename", searchHandler.SearchByFilename)

	return r, db
}

func TestE2E_Search(t *testing.T) {
	r, db := setupSearchTestServer(t)
	defer db.Close()

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

	// Test 1: Keyword search
	t.Run("KeywordSearch", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/search?q=cat", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Errorf("Failed to parse response: %v", err)
		}

		images, ok := resp["images"].([]interface{})
		if !ok {
			t.Error("Expected images array in response")
		}

		t.Logf("Found %d images for 'cat'", len(images))
	})

	// Test 2: Empty search (returns all)
	t.Run("EmptySearch", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/search", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Errorf("Failed to parse response: %v", err)
		}

		images, ok := resp["images"].([]interface{})
		if !ok {
			t.Error("Expected images array in response")
		}

		if len(images) != 3 {
			t.Errorf("Expected 3 images, got %d", len(images))
		}
	})

	// Test 3: Filename search
	t.Run("FilenameSearch", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/search/filename?pattern=cat", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Errorf("Failed to parse response: %v", err)
		}

		images, ok := resp["images"].([]interface{})
		if !ok {
			t.Error("Expected images array in response")
		}

		t.Logf("Found %d images for filename pattern 'cat'", len(images))
	})

	// Test 4: Pagination
	t.Run("Pagination", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/search?limit=2&offset=0", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Errorf("Failed to parse response: %v", err)
		}

		images, ok := resp["images"].([]interface{})
		if !ok {
			t.Error("Expected images array in response")
		}

		if len(images) > 2 {
			t.Errorf("Expected at most 2 images, got %d", len(images))
		}

		hasMore, ok := resp["has_more"].(bool)
		if !ok {
			t.Error("Expected has_more boolean in response")
		}

		t.Logf("Pagination: %d images, has_more: %v", len(images), hasMore)
	})
}

func TestE2E_SearchWithTags(t *testing.T) {
	r, db := setupSearchTestServer(t)
	defer db.Close()

	// Create tags
	tagRepo := repository.NewTagRepository(db)
	tag1 := &domain.Tag{PreferredLabel: "animal", Slug: "animal"}
	tag2 := &domain.Tag{PreferredLabel: "nature", Slug: "nature"}
	ctx := context.Background()
	if err := tagRepo.Save(ctx, tag1); err != nil {
		t.Fatalf("Failed to create tag1: %v", err)
	}
	if err := tagRepo.Save(ctx, tag2); err != nil {
		t.Fatalf("Failed to create tag2: %v", err)
	}

	// Create images
	imageRepo := repository.NewImageRepository(db)
	img1 := &domain.Image{Path: "/img1.jpg", Filename: "tagged_image.jpg"}
	img2 := &domain.Image{Path: "/img2.jpg", Filename: "another_tagged.jpg"}
	if err := imageRepo.SaveImage(img1); err != nil {
		t.Fatalf("Failed to save img1: %v", err)
	}
	if err := imageRepo.SaveImage(img2); err != nil {
		t.Fatalf("Failed to save img2: %v", err)
	}

	// Tag images
	_, err := db.Exec(`INSERT INTO image_tags (image_id, tag_id) VALUES (?, ?)`, img1.ID, tag1.ID)
	if err != nil {
		t.Fatalf("Failed to tag img1: %v", err)
	}
	_, err = db.Exec(`INSERT INTO image_tags (image_id, tag_id) VALUES (?, ?)`, img2.ID, tag1.ID)
	if err != nil {
		t.Fatalf("Failed to tag img2: %v", err)
	}

	// Rebuild FTS
	if err := repository.RebuildFTSIndex(db); err != nil {
		t.Fatalf("Failed to rebuild FTS: %v", err)
	}

	// Test tag search
	t.Run("TagSearch", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/search?tag_ids=1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Errorf("Failed to parse response: %v", err)
		}

		t.Logf("Tag search response: %v", resp)
	})
}

func TestE2E_SearchSorting(t *testing.T) {
	r, db := setupSearchTestServer(t)
	defer db.Close()

	// Create test images
	for i := 0; i < 5; i++ {
		_, err := db.Exec(`
			INSERT INTO images (path, filename, source_root, file_size, width, height, format, created_at, updated_at)
			VALUES (?, ?, '', ?, 100, 100, 'jpg', datetime('now'), datetime('now'))
		`, "/images/img"+string(rune('0'+i))+".jpg", "img"+string(rune('0'+i))+".jpg", (i+1)*1000)
		if err != nil {
			t.Fatalf("Failed to create test image: %v", err)
		}
	}

	// Rebuild FTS
	if err := repository.RebuildFTSIndex(db); err != nil {
		t.Fatalf("Failed to rebuild FTS: %v", err)
	}

	// Test sorting
	t.Run("SortByFileSize", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/search?sort_by=file_size&sort_order=asc", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Errorf("Failed to parse response: %v", err)
		}

		t.Logf("Sorted search response: %v", resp)
	})
}
