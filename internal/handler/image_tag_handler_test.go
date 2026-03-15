package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

func TestImageTagGetImageTagsGroupsTagsByReviewState(t *testing.T) {
	t.Parallel()

	router, _ := newImageTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/1/tags", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Confirmed []map[string]any `json:"confirmed"`
		Pending   []map[string]any `json:"pending"`
		Rejected  []map[string]any `json:"rejected"`
	}
	decodeJSONBody(t, w.Body.Bytes(), &resp)

	if len(resp.Confirmed) != 1 || len(resp.Pending) != 1 || len(resp.Rejected) != 1 {
		t.Fatalf("unexpected grouping lengths: %+v", resp)
	}
}

func TestImageTagAddImageTagAssociatesExistingTag(t *testing.T) {
	t.Parallel()

	router, repos := newImageTagHandlerTestRouter(t)
	body := bytes.NewBufferString(`{"tag_id":2}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/2/tags", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	items, err := repos.imageTagRepo.FindByImageID(context.Background(), 2)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].TagID != 2 || items[0].ReviewState != "confirmed" {
		t.Fatalf("unexpected item: %+v", items[0])
	}
}

func TestImageTagAddImageTagCreatesMissingTagFromLabel(t *testing.T) {
	t.Parallel()

	router, repos := newImageTagHandlerTestRouter(t)
	body := bytes.NewBufferString(`{"tag_label":"new aura"}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/2/tags", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	tag, err := repos.tagRepo.FindByLabel(context.Background(), "new aura")
	if err != nil {
		t.Fatalf("FindByLabel() error = %v", err)
	}
	if tag.ID == 0 {
		t.Fatal("expected created tag id")
	}
	items, err := repos.imageTagRepo.FindByImageID(context.Background(), 2)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	if len(items) != 1 || items[0].TagID != tag.ID {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestImageTagRemoveImageTagDeletesAssociation(t *testing.T) {
	t.Parallel()

	router, repos := newImageTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/images/1/tags/2", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	items, err := repos.imageTagRepo.FindByImageID(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	for _, item := range items {
		if item.TagID == 2 {
			t.Fatal("expected association to be removed")
		}
	}
}

func TestImageTagReviewTagUpdatesState(t *testing.T) {
	t.Parallel()

	router, repos := newImageTagHandlerTestRouter(t)
	body := bytes.NewBufferString(`{"action":"confirm"}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/1/tags/2/review", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	items, err := repos.imageTagRepo.FindByImageID(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	for _, item := range items {
		if item.TagID == 2 && item.ReviewState != "confirmed" {
			t.Fatalf("review_state = %q, want confirmed", item.ReviewState)
		}
	}
}

func TestImageTagBatchReviewUpdatesMultipleStates(t *testing.T) {
	t.Parallel()

	router, repos := newImageTagHandlerTestRouter(t)
	body := bytes.NewBufferString(`{"tag_ids":[2,3],"action":"reject"}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/1/tags/batch-review", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	items, err := repos.imageTagRepo.FindByImageID(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	states := map[int64]string{}
	for _, item := range items {
		states[item.TagID] = item.ReviewState
	}
	if states[2] != "rejected" || states[3] != "rejected" {
		t.Fatalf("unexpected states: %+v", states)
	}
}

type imageTagHandlerTestRepos struct {
	tagRepo      repository.TagRepository
	obsRepo      repository.TagObservationRepository
	imageTagRepo repository.ImageTagRepository
	imageRepo    repository.ImageRepository
	governance   *service.TagGovernanceService
	db           *sql.DB
}

func newImageTagHandlerTestRouter(t *testing.T) (*gin.Engine, *imageTagHandlerTestRepos) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "image-tag-handler.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}

	seedImageTagHandlerData(t, db)

	repos := &imageTagHandlerTestRepos{
		tagRepo:      repository.NewTagRepository(db),
		obsRepo:      repository.NewTagObservationRepository(db),
		imageTagRepo: repository.NewImageTagRepository(db),
		imageRepo:    repository.NewImageRepository(db),
		db:           db,
	}
	repos.governance = service.NewTagGovernanceService(repos.tagRepo, nil, repos.obsRepo, repos.imageTagRepo)
	h := NewImageTagHandler(repos.imageTagRepo, repos.tagRepo, repos.imageRepo, repos.governance)

	router := gin.New()
	api := router.Group("/api/v1")
	api.GET("/images/:id/tags", h.GetImageTags)
	api.POST("/images/:id/tags", h.AddImageTag)
	api.DELETE("/images/:id/tags/:tag_id", h.RemoveImageTag)
	api.POST("/images/:id/tags/:tag_id/review", h.ReviewTag)
	api.POST("/images/:id/tags/batch-review", h.BatchReview)

	return router, repos
}

func seedImageTagHandlerData(t *testing.T, db *sql.DB) {
	t.Helper()

	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO images (id, path, filename, source_root, file_size, width, height, format, created_at, updated_at)
		VALUES
			(1, '/images/1.png', '1.png', '/images', 100, 100, 100, 'png', ?, ?),
			(2, '/images/2.png', '2.png', '/images', 100, 100, 100, 'png', ?, ?)
	`, now, now, now, now)
	if err != nil {
		t.Fatalf("seed images: %v", err)
	}

	tagRepo := repository.NewTagRepository(db)
	for _, tag := range []*domain.Tag{
		{ID: 1, PreferredLabel: "blue sky", Slug: "blue-sky", ReviewState: "confirmed", UsageCount: 10},
		{ID: 2, PreferredLabel: "sunrise", Slug: "sunrise", ReviewState: "pending", UsageCount: 5},
		{ID: 3, PreferredLabel: "cloud", Slug: "cloud", ReviewState: "rejected", UsageCount: 2},
	} {
		if err := tagRepo.Save(context.Background(), tag); err != nil {
			t.Fatalf("seed tag %d: %v", tag.ID, err)
		}
	}

	imageTagRepo := repository.NewImageTagRepository(db)
	for _, item := range []*domain.ImageTag{
		{ImageID: 1, TagID: 1, ReviewState: "confirmed", Confidence: 0.9},
		{ImageID: 1, TagID: 2, ReviewState: "pending", Confidence: 0.8},
		{ImageID: 1, TagID: 3, ReviewState: "rejected", Confidence: 0.2},
	} {
		if err := imageTagRepo.Save(context.Background(), item); err != nil {
			t.Fatalf("seed image tag (%d,%d): %v", item.ImageID, item.TagID, err)
		}
	}
}

func decodeJSONBody(t *testing.T, body []byte, target any) {
	t.Helper()

	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; body = %s", err, string(body))
	}
}
