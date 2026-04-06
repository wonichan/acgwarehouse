package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/config"
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
	db           *sql.DB
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
		db:           db,
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

func TestImageHandlerListImagesFiltersByHasTagsFalse(t *testing.T) {
	t.Parallel()

	router, repos := newImageHandlerTestRouter(t)

	// Create tag
	tag := &domain.Tag{PreferredLabel: "test", Slug: "test", ReviewState: "confirmed"}
	if err := repos.tagRepo.Save(context.Background(), tag); err != nil {
		t.Fatalf("save tag: %v", err)
	}

	// Tag images 1 and 2, leave image 3 untagged
	if err := repos.imageTagRepo.Save(context.Background(), &domain.ImageTag{ImageID: 1, TagID: tag.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("tag image 1: %v", err)
	}
	if err := repos.imageTagRepo.Save(context.Background(), &domain.ImageTag{ImageID: 2, TagID: tag.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("tag image 2: %v", err)
	}

	// Test: has_tags=false should return only untagged images
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/images?has_tags=false", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Images []map[string]any `json:"images"`
		Total  int64            `json:"total"`
	}
	decodeJSONResponse(t, w, &resp)

	if len(resp.Images) != 1 {
		t.Fatalf("len(images) = %d, want 1 (only untagged image)", len(resp.Images))
	}
	if resp.Images[0]["id"].(float64) != 3 {
		t.Fatalf("images[0].id = %v, want 3 (the untagged image)", resp.Images[0]["id"])
	}
	if resp.Total != 1 {
		t.Fatalf("total = %d, want 1", resp.Total)
	}
}

func TestImageHandlerListImagesHasTagsTrueReturnsAllImages(t *testing.T) {
	t.Parallel()

	router, _ := newImageHandlerTestRouter(t)

	// Test: has_tags=true should return all images (same as default)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/images?has_tags=true", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Images []map[string]any `json:"images"`
		Total  int64            `json:"total"`
	}
	decodeJSONResponse(t, w, &resp)

	if len(resp.Images) != 3 {
		t.Fatalf("len(images) = %d, want 3 (all images)", len(resp.Images))
	}
	if resp.Total != 3 {
		t.Fatalf("total = %d, want 3", resp.Total)
	}
}

func TestImageHandlerListImagesHasTagsFalseWithTagIDsReturnsError(t *testing.T) {
	t.Parallel()

	router, repos := newImageHandlerTestRouter(t)

	// Create tag
	tag := &domain.Tag{PreferredLabel: "test", Slug: "test", ReviewState: "confirmed"}
	if err := repos.tagRepo.Save(context.Background(), tag); err != nil {
		t.Fatalf("save tag: %v", err)
	}

	// Test: has_tags=false AND tag_ids should return 400
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/images?has_tags=false&tag_ids="+itoa(tag.ID), nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]any
	decodeJSONResponse(t, w, &resp)

	if resp["error"] == nil {
		t.Fatal("response missing 'error' field")
	}
}

func TestImageHandlerListImagesHasTagsFalseSupportsPagination(t *testing.T) {
	t.Parallel()

	// Create custom test DB with more untagged images
	gin.SetMode(gin.TestMode)
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "image-handler-pagination.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	// Create 5 images
	now := time.Now()
	for i := 1; i <= 5; i++ {
		_, err := db.Exec(`
			INSERT INTO images (id, path, filename, source_root, file_size, width, height, format, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, i, filepath.Join("/img", string(rune('a'+i-1))), string(rune('a'+i-1)), "/img", 100, 100, 100, "png", now, now)
		if err != nil {
			t.Fatalf("seed image %d: %v", i, err)
		}
	}

	// Tag first 2 images
	tagRepo := repository.NewTagRepository(db)
	imageTagRepo := repository.NewImageTagRepository(db)
	tag := &domain.Tag{PreferredLabel: "test", Slug: "test", ReviewState: "confirmed"}
	if err := tagRepo.Save(context.Background(), tag); err != nil {
		t.Fatalf("save tag: %v", err)
	}
	for i := 1; i <= 2; i++ {
		if err := imageTagRepo.Save(context.Background(), &domain.ImageTag{ImageID: int64(i), TagID: tag.ID, ReviewState: "confirmed"}); err != nil {
			t.Fatalf("tag image %d: %v", i, err)
		}
	}

	imageRepo := repository.NewImageRepository(db)
	h := NewImageHandler(imageRepo, tagRepo, imageTagRepo)
	router := gin.New()
	api := router.Group("/api/v1")
	api.GET("/images", h.ListImages)

	// Request first page with limit=1 (3 untagged images total)
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/images?has_tags=false&limit=1", nil)
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

	if len(resp1.Images) != 1 {
		t.Fatalf("page 1: len(images) = %d, want 1", len(resp1.Images))
	}
	if resp1.Total != 3 {
		t.Fatalf("page 1: total = %d, want 3 (3 untagged images)", resp1.Total)
	}
	if !resp1.HasMore {
		t.Fatal("page 1: has_more = false, want true")
	}
}

func TestImageHandler_TriggerImportReturnsAcceptedWithQueuedJob(t *testing.T) {
	t.Parallel()

	router := newImageHandlerTriggerImportRouter(t, &mockAdminService{scanJobID: 42}, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/scan", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusAccepted, w.Body.String())
	}

	var resp struct {
		Status string `json:"status"`
		JobID  int64  `json:"job_id"`
	}
	decodeJSONResponse(t, w, &resp)

	if resp.Status != "queued" {
		t.Fatalf("status = %q, want %q", resp.Status, "queued")
	}
	if resp.JobID != 42 {
		t.Fatalf("job_id = %d, want 42", resp.JobID)
	}
}

func TestImageHandler_TriggerImportFailureReturnsStructuredError(t *testing.T) {
	t.Parallel()

	router := newImageHandlerTriggerImportRouter(t, &mockAdminService{err: errors.New("queue unavailable")}, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/scan", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusInternalServerError, w.Body.String())
	}

	var resp struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	decodeJSONResponse(t, w, &resp)

	if resp.Status != "failed" {
		t.Fatalf("status = %q, want %q", resp.Status, "failed")
	}
	if resp.Error == "" {
		t.Fatal("error is empty, want non-empty error")
	}
}

func TestImageHandler_TriggerImportRouteIsProductHandlerNotPlaceholder(t *testing.T) {
	t.Parallel()

	router := newImageHandlerTriggerImportRouter(t, &mockAdminService{scanJobID: 7}, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/scan", nil)
	router.ServeHTTP(w, req)

	if w.Code == http.StatusNotImplemented {
		t.Fatalf("status = %d, route still behaves like placeholder", w.Code)
	}
}

func TestImageHandlerViewerWindowGalleryReturnsWindow(t *testing.T) {
	t.Parallel()

	router, _ := newViewerWindowTestRouter(t)
	body := map[string]any{
		"source":            "gallery",
		"selected_index":    6,
		"selected_image_id": 7,
		"limit":             99,
		"snapshot": map[string]any{
			"sort_by":  "created_at",
			"sort_dir": "asc",
		},
	}

	w := performViewerWindowRequest(t, router, body)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Items                 []map[string]any `json:"items"`
		WindowStartIndex      int              `json:"window_start_index"`
		SelectedIndex         int              `json:"selected_index"`
		SelectedIndexInWindow int              `json:"selected_index_in_window"`
		Total                 int              `json:"total"`
		HasPrevious           bool             `json:"has_previous"`
		HasNext               bool             `json:"has_next"`
	}
	decodeJSONResponse(t, w, &resp)

	if len(resp.Items) != 10 {
		t.Fatalf("len(items) = %d, want 10", len(resp.Items))
	}
	if resp.WindowStartIndex != 1 {
		t.Fatalf("window_start_index = %d, want 1", resp.WindowStartIndex)
	}
	if resp.SelectedIndex != 6 {
		t.Fatalf("selected_index = %d, want 6", resp.SelectedIndex)
	}
	if resp.SelectedIndexInWindow != 5 {
		t.Fatalf("selected_index_in_window = %d, want 5", resp.SelectedIndexInWindow)
	}
	if resp.Total != 12 {
		t.Fatalf("total = %d, want 12", resp.Total)
	}
	if !resp.HasPrevious || !resp.HasNext {
		t.Fatalf("has_previous=%v has_next=%v, want true/true", resp.HasPrevious, resp.HasNext)
	}
	if resp.Items[resp.SelectedIndexInWindow]["id"].(float64) != 7 {
		t.Fatalf("selected item id = %v, want 7", resp.Items[resp.SelectedIndexInWindow]["id"])
	}
}

func TestImageHandlerViewerWindowGalleryRejectsOutOfRangeIndex(t *testing.T) {
	t.Parallel()

	router, _ := newViewerWindowTestRouter(t)
	w := performViewerWindowRequest(t, router, map[string]any{
		"source":            "gallery",
		"selected_index":    99,
		"selected_image_id": 1,
		"limit":             10,
		"snapshot": map[string]any{
			"sort_by":  "id",
			"sort_dir": "asc",
		},
	})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}

	var resp map[string]any
	decodeJSONResponse(t, w, &resp)
	if resp["error"] != "invalid_viewer_request" {
		t.Fatalf("error = %v, want invalid_viewer_request", resp["error"])
	}
}

func TestImageHandlerViewerWindowGalleryDetectsSnapshotDrift(t *testing.T) {
	t.Parallel()

	router, _ := newViewerWindowTestRouter(t)
	w := performViewerWindowRequest(t, router, map[string]any{
		"source":            "gallery",
		"selected_index":    2,
		"selected_image_id": 999,
		"limit":             10,
		"snapshot": map[string]any{
			"sort_by":  "id",
			"sort_dir": "asc",
		},
	})

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}

	var resp map[string]any
	decodeJSONResponse(t, w, &resp)
	if resp["error"] != "viewer_snapshot_drift" {
		t.Fatalf("error = %v, want viewer_snapshot_drift", resp["error"])
	}
}

func TestImageHandlerViewerWindowSearchReturnsWindow(t *testing.T) {
	t.Parallel()

	router, repos := newViewerWindowTestRouter(t)
	seedViewerWindowSearchData(t, repos)

	w := performViewerWindowRequest(t, router, map[string]any{
		"source":            "search",
		"selected_index":    1,
		"selected_image_id": 3,
		"limit":             10,
		"snapshot": map[string]any{
			"q":          "cat",
			"tag_ids":    []int64{1},
			"sort_by":    "created_at",
			"sort_order": "asc",
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Items                 []map[string]any `json:"items"`
		WindowStartIndex      int              `json:"window_start_index"`
		SelectedIndexInWindow int              `json:"selected_index_in_window"`
		Total                 int              `json:"total"`
	}
	decodeJSONResponse(t, w, &resp)

	if resp.Total != 2 {
		t.Fatalf("total = %d, want 2", resp.Total)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(resp.Items))
	}
	if resp.WindowStartIndex != 0 || resp.SelectedIndexInWindow != 1 {
		t.Fatalf("window_start_index=%d selected_index_in_window=%d, want 0 and 1", resp.WindowStartIndex, resp.SelectedIndexInWindow)
	}
	if resp.Items[0]["id"].(float64) != 1 || resp.Items[1]["id"].(float64) != 3 {
		t.Fatalf("ids = [%v %v], want [1 3]", resp.Items[0]["id"], resp.Items[1]["id"])
	}
}

func TestImageHandlerViewerWindowSearchHandlesWindowDriftWithoutPanic(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewImageHandler(nil, nil, nil, viewerWindowSearchServiceStub{
		result: &service.ViewerWindowResult{
			Images:                []domain.Image{{ID: 1, Filename: "cat-1.jpg"}},
			Total:                 2,
			WindowStart:           1,
			SelectedIndex:         1,
			SelectedIndexInWindow: 1,
			HasPrevious:           true,
			HasNext:               false,
		},
	})
	router.POST("/api/v1/viewer/window", h.ViewerWindow)

	w := performViewerWindowRequest(t, router, map[string]any{
		"source":            "search",
		"selected_index":    1,
		"selected_image_id": 1,
		"limit":             10,
		"snapshot": map[string]any{
			"q":          "cat",
			"sort_by":    "relevance",
			"sort_order": "desc",
		},
	})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}

	var resp map[string]any
	decodeJSONResponse(t, w, &resp)
	if resp["error"] != "invalid_viewer_request" {
		t.Fatalf("error = %v, want invalid_viewer_request", resp["error"])
	}
}

type viewerWindowSearchServiceStub struct {
	result *service.ViewerWindowResult
	err    error
}

func (s viewerWindowSearchServiceStub) ViewerWindow(ctx context.Context, opts service.SearchOptions, selectedIndex, limit int) (*service.ViewerWindowResult, error) {
	return s.result, s.err
}

func newImageHandlerTriggerImportRouter(t *testing.T, adminSvc AdminServiceInterface, adminCfg *config.Config) *gin.Engine {
	t.Helper()

	_, repos := newImageHandlerTestRouter(t)

	router := gin.New()
	deps := &Dependencies{
		ImageRepo:    repos.imageRepo,
		TagRepo:      repos.tagRepo,
		ImageTagRepo: repos.imageTagRepo,
		AdminSvc:     adminSvc,
		AdminCfg:     adminCfg,
	}
	if deps.AdminCfg == nil {
		deps.AdminCfg = &config.Config{}
	}
	SetupRoutes(router, deps)

	return router
}

func newViewerWindowTestRouter(t *testing.T) (*gin.Engine, *imageHandlerTestRepos) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "viewer-window.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	for i := 1; i <= 12; i++ {
		_, err := db.Exec(`
			INSERT INTO images (id, path, filename, source_root, file_size, width, height, format, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, i, filepath.Join("/viewer", itoa(int64(i))+".jpg"), itoa(int64(i))+".jpg", "/viewer", 100, 100, 100, "jpg", now, now)
		if err != nil {
			t.Fatalf("seed image %d: %v", i, err)
		}
	}

	repos := &imageHandlerTestRepos{
		db:           db,
		imageRepo:    repository.NewImageRepository(db),
		tagRepo:      repository.NewTagRepository(db),
		imageTagRepo: repository.NewImageTagRepository(db),
	}
	searchSvc := service.NewSearchService(repos.imageRepo, repos.tagRepo, repository.NewSearchRepository(db))

	router := gin.New()
	deps := &Dependencies{
		ImageRepo:    repos.imageRepo,
		TagRepo:      repos.tagRepo,
		ImageTagRepo: repos.imageTagRepo,
		SearchSvc:    searchSvc,
	}
	SetupRoutes(router, deps)
	return router, repos
}

func seedViewerWindowSearchData(t *testing.T, repos *imageHandlerTestRepos) {
	t.Helper()

	ctx := context.Background()
	tag := &domain.Tag{PreferredLabel: "viewer-search", Slug: "viewer-search", ReviewState: "confirmed"}
	if err := repos.tagRepo.Save(ctx, tag); err != nil {
		t.Fatalf("save tag: %v", err)
	}
	if tag.ID != 1 {
		t.Fatalf("tag.ID = %d, want 1 for stable test payload", tag.ID)
	}
	for _, imageID := range []int64{1, 3} {
		if err := repos.imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: imageID, TagID: tag.ID, ReviewState: "confirmed"}); err != nil {
			t.Fatalf("save image tag: %v", err)
		}
	}
	for _, imageID := range []int64{1, 2, 3, 4} {
		if _, err := repos.db.Exec(`
			UPDATE images SET filename = ?, path = ?, source_root = ?, created_at = ?, updated_at = ? WHERE id = ?
		`, "cat-"+itoa(imageID)+".jpg", "/viewer/cat-"+itoa(imageID)+".jpg", "/viewer", time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC), time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC), imageID); err != nil {
			t.Fatalf("update image %d: %v", imageID, err)
		}
	}
	if err := repository.RebuildFTSIndex(repos.db); err != nil {
		t.Fatalf("rebuild fts: %v", err)
	}
}

func performViewerWindowRequest(t *testing.T, router *gin.Engine, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/viewer/window", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	return w
}
