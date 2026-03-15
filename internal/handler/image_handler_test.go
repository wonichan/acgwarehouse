package handler

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

func TestImageHandlerListImagesReturnsImages(t *testing.T) {
	t.Parallel()

	router, _ := newImageHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/images", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Images     []map[string]any `json:"images"`
		NextCursor string           `json:"next_cursor"`
		HasMore    bool             `json:"has_more"`
	}
	decodeJSONResponse(t, w, &resp)

	if len(resp.Images) != 3 {
		t.Fatalf("len(images) = %d, want 3", len(resp.Images))
	}
}

func TestImageHandlerListImagesFiltersByTagIDs(t *testing.T) {
	t.Parallel()

	router, repos := newImageHandlerTestRouter(t)

	// Create tags
	tag1 := &domain.Tag{PreferredLabel: "blue sky", Slug: "blue-sky", ReviewState: "confirmed"}
	tag2 := &domain.Tag{PreferredLabel: "sunset", Slug: "sunset", ReviewState: "confirmed"}
	if err := repos.tagRepo.Save(context.Background(), tag1); err != nil {
		t.Fatalf("save tag1: %v", err)
	}
	if err := repos.tagRepo.Save(context.Background(), tag2); err != nil {
		t.Fatalf("save tag2: %v", err)
	}

	// Add tags to images: img1 has both tag1 and tag2, img2 has only tag1
	if err := repos.imageTagRepo.Save(context.Background(), &domain.ImageTag{ImageID: 1, TagID: tag1.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("save image-tag: %v", err)
	}
	if err := repos.imageTagRepo.Save(context.Background(), &domain.ImageTag{ImageID: 1, TagID: tag2.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("save image-tag: %v", err)
	}
	if err := repos.imageTagRepo.Save(context.Background(), &domain.ImageTag{ImageID: 2, TagID: tag1.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("save image-tag: %v", err)
	}

	// Test: Filter by tag_ids=tag1.ID,tag2.ID (AND semantics - only img1 has both)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/images?tag_ids="+joinInt64([]int64{tag1.ID, tag2.ID}), nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Images     []map[string]any `json:"images"`
		NextCursor string           `json:"next_cursor"`
		HasMore    bool             `json:"has_more"`
	}
	decodeJSONResponse(t, w, &resp)

	if len(resp.Images) != 1 {
		t.Fatalf("len(images) = %d, want 1 (only images with ALL requested tags)", len(resp.Images))
	}
	if resp.Images[0]["id"].(float64) != float64(1) {
		t.Fatalf("images[0].id = %v, want 1", resp.Images[0]["id"])
	}
}

func TestImageHandlerListImagesFiltersBySingleTagID(t *testing.T) {
	t.Parallel()

	router, repos := newImageHandlerTestRouter(t)

	// Create tag
	tag1 := &domain.Tag{PreferredLabel: "blue sky", Slug: "blue-sky", ReviewState: "confirmed"}
	if err := repos.tagRepo.Save(context.Background(), tag1); err != nil {
		t.Fatalf("save tag1: %v", err)
	}

	// Add tag to img1 and img2
	if err := repos.imageTagRepo.Save(context.Background(), &domain.ImageTag{ImageID: 1, TagID: tag1.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("save image-tag: %v", err)
	}
	if err := repos.imageTagRepo.Save(context.Background(), &domain.ImageTag{ImageID: 2, TagID: tag1.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("save image-tag: %v", err)
	}

	// Test: Filter by single tag_id
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/images?tag_ids="+joinInt64([]int64{tag1.ID}), nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Images []map[string]any `json:"images"`
	}
	decodeJSONResponse(t, w, &resp)

	if len(resp.Images) != 2 {
		t.Fatalf("len(images) = %d, want 2", len(resp.Images))
	}
}

func TestImageHandlerGetImageReturnsImage(t *testing.T) {
	t.Parallel()

	router, _ := newImageHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]any
	decodeJSONResponse(t, w, &resp)

	if resp["id"].(float64) != float64(1) {
		t.Fatalf("id = %v, want 1", resp["id"])
	}
}

func TestImageHandlerGetImageReturnsNotFoundForInvalidID(t *testing.T) {
	t.Parallel()

	router, _ := newImageHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/999", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

type imageHandlerTestRepos struct {
	imageRepo    repository.ImageRepository
	tagRepo      repository.TagRepository
	imageTagRepo repository.ImageTagRepository
	governance   *service.TagGovernanceService
}

func newImageHandlerTestRouter(t *testing.T) (*gin.Engine, *imageHandlerTestRepos) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "image-handler.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	// Seed images
	now := time.Now()
	for i := 1; i <= 3; i++ {
		_, err := db.Exec(`
			INSERT INTO images (id, path, filename, source_root, file_size, width, height, format, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, i, filepath.Join("/img", string(rune('a'+i-1))), string(rune('a'+i-1)), "/img", 100, 100, 100, "png", now, now)
		if err != nil {
			t.Fatalf("seed image %d: %v", i, err)
		}
	}

	repos := &imageHandlerTestRepos{
		imageRepo:    repository.NewImageRepository(db),
		tagRepo:      repository.NewTagRepository(db),
		imageTagRepo: repository.NewImageTagRepository(db),
	}
	// Note: governance not needed for image listing, pass nil for optional repos
	repos.governance = service.NewTagGovernanceService(repos.tagRepo, nil, nil, repos.imageTagRepo)

	h := NewImageHandler(repos.imageRepo, repos.tagRepo, repos.imageTagRepo)

	router := gin.New()
	api := router.Group("/api/v1")
	api.GET("/images", h.ListImages)
	api.GET("/images/:id", h.GetImage)

	return router, repos
}

func joinInt64(ids []int64) string {
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = itoa(id)
	}
	return strings.Join(strs, ",")
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var neg bool
	if n < 0 {
		neg = true
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
