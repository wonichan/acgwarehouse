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

// TestImageHandlerListImagesReturnsConsistentContract verifies the JSON response
// has a consistent field shape that matches the Flutter client expectations.
func TestImageHandlerListImagesReturnsConsistentContract(t *testing.T) {
	t.Parallel()

	router, _ := newImageHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/images?limit=1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Response must have exactly these fields: images, next_cursor, has_more, total
	var resp map[string]any
	decodeJSONResponse(t, w, &resp)

	// Verify required fields exist
	if _, ok := resp["images"]; !ok {
		t.Fatal("response missing 'images' field")
	}
	if _, ok := resp["next_cursor"]; !ok {
		t.Fatal("response missing 'next_cursor' field")
	}
	if _, ok := resp["has_more"]; !ok {
		t.Fatal("response missing 'has_more' field")
	}
	if _, ok := resp["total"]; !ok {
		t.Fatal("response missing 'total' field")
	}

	// Verify types
	images, ok := resp["images"].([]any)
	if !ok {
		t.Fatalf("images is %T, want []any", resp["images"])
	}
	if len(images) == 0 {
		t.Fatal("images array is empty")
	}

	// Verify image object has required fields
	firstImage := images[0].(map[string]any)
	requiredFields := []string{"id", "path", "filename", "source_root", "file_size", "width", "height", "format", "created_at", "updated_at"}
	for _, field := range requiredFields {
		if _, ok := firstImage[field]; !ok {
			t.Fatalf("image missing required field '%s'", field)
		}
	}
}

// TestImageHandlerPaginationBehavior verifies stable next_cursor/has_more
// behavior across multiple pages for large result sets.
func TestImageHandlerPaginationBehavior(t *testing.T) {
	t.Parallel()

	router, _ := newImageHandlerTestRouter(t)

	// Request first page with limit=1 (total 3 images in test DB)
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/images?limit=1", nil)
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("page 1 status = %d, want %d", w1.Code, http.StatusOK)
	}

	var resp1 struct {
		Images     []map[string]any `json:"images"`
		NextCursor string           `json:"next_cursor"`
		HasMore    bool             `json:"has_more"`
		Total      int64            `json:"total"`
	}
	decodeJSONResponse(t, w1, &resp1)

	// Page 1: should have has_more=true and next_cursor set
	if len(resp1.Images) != 1 {
		t.Fatalf("page 1: len(images) = %d, want 1", len(resp1.Images))
	}
	if !resp1.HasMore {
		t.Fatal("page 1: has_more = false, want true (more pages exist)")
	}
	if resp1.NextCursor == "" {
		t.Fatal("page 1: next_cursor is empty, want non-empty cursor")
	}
	if resp1.Total != 3 {
		t.Fatalf("page 1: total = %d, want 3", resp1.Total)
	}

	// Request second page using next_cursor as offset
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/images?limit=1&offset="+resp1.NextCursor, nil)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("page 2 status = %d, want %d", w2.Code, http.StatusOK)
	}

	var resp2 struct {
		Images     []map[string]any `json:"images"`
		NextCursor string           `json:"next_cursor"`
		HasMore    bool             `json:"has_more"`
	}
	decodeJSONResponse(t, w2, &resp2)

	// Page 2: should still have has_more=true (1 more image remains)
	if len(resp2.Images) != 1 {
		t.Fatalf("page 2: len(images) = %d, want 1", len(resp2.Images))
	}
	if !resp2.HasMore {
		t.Fatal("page 2: has_more = false, want true (1 more page exists)")
	}

	// Request third/final page
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/images?limit=1&offset="+resp2.NextCursor, nil)
	router.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Fatalf("page 3 status = %d, want %d", w3.Code, http.StatusOK)
	}

	var resp3 struct {
		Images     []map[string]any `json:"images"`
		NextCursor string           `json:"next_cursor"`
		HasMore    bool             `json:"has_more"`
	}
	decodeJSONResponse(t, w3, &resp3)

	// Page 3: last page, should have has_more=false and empty next_cursor
	if len(resp3.Images) != 1 {
		t.Fatalf("page 3: len(images) = %d, want 1", len(resp3.Images))
	}
	if resp3.HasMore {
		t.Fatal("page 3: has_more = true, want false (no more pages)")
	}
	if resp3.NextCursor != "" {
		t.Fatalf("page 3: next_cursor = %q, want empty string", resp3.NextCursor)
	}
}

// TestImageHandlerTagFilteredPaginationPreservesContract verifies tag-filtered
// queries use the same response contract and maintain correct total counts.
func TestImageHandlerTagFilteredPaginationPreservesContract(t *testing.T) {
	t.Parallel()

	router, repos := newImageHandlerTestRouter(t)

	// Create tag
	tag := &domain.Tag{PreferredLabel: "test", Slug: "test", ReviewState: "confirmed"}
	if err := repos.tagRepo.Save(context.Background(), tag); err != nil {
		t.Fatalf("save tag: %v", err)
	}

	// Tag images 1 and 2
	if err := repos.imageTagRepo.Save(context.Background(), &domain.ImageTag{ImageID: 1, TagID: tag.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("tag image 1: %v", err)
	}
	if err := repos.imageTagRepo.Save(context.Background(), &domain.ImageTag{ImageID: 2, TagID: tag.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("tag image 2: %v", err)
	}

	// Request with tag filter, limit=1
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/images?limit=1&tag_ids="+itoa(tag.ID), nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Images     []map[string]any `json:"images"`
		NextCursor string           `json:"next_cursor"`
		HasMore    bool             `json:"has_more"`
		Total      int64            `json:"total"`
	}
	decodeJSONResponse(t, w, &resp)

	// Verify same contract fields exist
	if len(resp.Images) != 1 {
		t.Fatalf("len(images) = %d, want 1", len(resp.Images))
	}
	if !resp.HasMore {
		t.Fatal("has_more = false, want true (2 tagged images, limit 1)")
	}
	if resp.Total != 2 {
		t.Fatalf("total = %d, want 2 (only 2 images have the tag)", resp.Total)
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
