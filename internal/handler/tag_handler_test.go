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

func TestTagGetTagsReturnsListSortedByUsageCount(t *testing.T) {
	t.Parallel()

	router, _ := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Tags  []domain.Tag `json:"tags"`
		Total int          `json:"total"`
	}
	decodeJSONResponse(t, w, &resp)

	if resp.Total != 3 {
		t.Fatalf("total = %d, want 3", resp.Total)
	}
	if len(resp.Tags) != 3 {
		t.Fatalf("len(tags) = %d, want 3", len(resp.Tags))
	}
	if resp.Tags[0].PreferredLabel != "blue sky" {
		t.Fatalf("first tag = %q, want %q", resp.Tags[0].PreferredLabel, "blue sky")
	}
}

func TestTagGetTagsSearchesLabelsAndAliases(t *testing.T) {
	t.Parallel()

	router, _ := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags?search=蓝", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Tags  []domain.Tag `json:"tags"`
		Total int          `json:"total"`
	}
	decodeJSONResponse(t, w, &resp)

	if resp.Total != 2 {
		t.Fatalf("total = %d, want 2", resp.Total)
	}
	if len(resp.Tags) != 2 {
		t.Fatalf("len(tags) = %d, want 2", len(resp.Tags))
	}
	if resp.Tags[0].PreferredLabel != "blue sky" {
		t.Fatalf("first tag = %q, want %q", resp.Tags[0].PreferredLabel, "blue sky")
	}
	if resp.Tags[1].PreferredLabel != "sunrise" {
		t.Fatalf("second tag = %q, want %q", resp.Tags[1].PreferredLabel, "sunrise")
	}
}

func TestTagCreateTagCreatesTag(t *testing.T) {
	t.Parallel()

	router, repos := newTagHandlerTestRouter(t)
	body := bytes.NewBufferString(`{"preferred_label":"蓝天白云","primary_category":"场景"}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tags", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	tag, err := repos.tagRepo.FindByLabel(context.Background(), "蓝天白云")
	if err != nil {
		t.Fatalf("FindByLabel() error = %v", err)
	}
	if tag.Slug != "蓝天白云" {
		t.Fatalf("slug = %q, want %q", tag.Slug, "蓝天白云")
	}
	if tag.PrimaryCategory != "场景" {
		t.Fatalf("primary_category = %q, want %q", tag.PrimaryCategory, "场景")
	}
}

func TestTagUpdateTagUpdatesPreferredLabel(t *testing.T) {
	t.Parallel()

	router, repos := newTagHandlerTestRouter(t)
	body := bytes.NewBufferString(`{"preferred_label":"blue horizon"}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tags/1", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	tag, err := repos.tagRepo.FindByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if tag.PreferredLabel != "blue horizon" {
		t.Fatalf("preferred_label = %q, want %q", tag.PreferredLabel, "blue horizon")
	}
	if tag.Slug != "blue-horizon" {
		t.Fatalf("slug = %q, want %q", tag.Slug, "blue-horizon")
	}
}

func TestTagDeleteTagRemovesTagAndAssociations(t *testing.T) {
	t.Parallel()

	router, repos := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tags/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	if _, err := repos.tagRepo.FindByID(context.Background(), 1); err == nil {
		t.Fatal("expected tag to be deleted")
	}
	aliases, err := repos.aliasRepo.FindByTagID(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindByTagID() error = %v", err)
	}
	if len(aliases) != 0 {
		t.Fatalf("len(aliases) = %d, want 0", len(aliases))
	}
	items, err := repos.imageTagRepo.FindByImageID(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	for _, item := range items {
		if item.TagID == 1 {
			t.Fatal("expected image tag association to be deleted")
		}
	}
}

func TestTagGetAliasesReturnsAliases(t *testing.T) {
	t.Parallel()

	router, _ := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/1/aliases", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Aliases []domain.TagAlias `json:"aliases"`
	}
	decodeJSONResponse(t, w, &resp)

	if len(resp.Aliases) != 1 {
		t.Fatalf("len(aliases) = %d, want 1", len(resp.Aliases))
	}
	if resp.Aliases[0].Label != "蓝天" {
		t.Fatalf("label = %q, want %q", resp.Aliases[0].Label, "蓝天")
	}
}

func TestTagAddAliasCreatesAlias(t *testing.T) {
	t.Parallel()

	router, repos := newTagHandlerTestRouter(t)
	body := bytes.NewBufferString(`{"label":"蓝色天空","alias_type":"synonym"}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tags/1/aliases", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	alias, err := repos.aliasRepo.FindByNormalizedLabel(context.Background(), "蓝色天空")
	if err != nil {
		t.Fatalf("FindByNormalizedLabel() error = %v", err)
	}
	if alias.TagID != 1 {
		t.Fatalf("tag_id = %d, want 1", alias.TagID)
	}
}

func TestTagGetGovernanceReturnsRowsOrderedByUsageCount(t *testing.T) {
	t.Parallel()

	router, _ := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/governance", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Rows []struct {
			TagID              int64    `json:"tag_id"`
			PreferredLabel     string   `json:"preferred_label"`
			PrimaryCategory    string   `json:"primary_category"`
			Aliases            []string `json:"aliases"`
			UsageCount         int64    `json:"usage_count"`
			PendingCount       int64    `json:"pending_count"`
			ConfirmedCount     int64    `json:"confirmed_count"`
			AICount            int64    `json:"ai_count"`
			ManualCount        int64    `json:"manual_count"`
			AffectedImageCount int64    `json:"affected_image_count"`
			CanDelete          bool     `json:"can_delete"`
		} `json:"rows"`
		Total int `json:"total"`
	}
	decodeJSONResponse(t, w, &resp)

	if resp.Total == 0 || len(resp.Rows) == 0 {
		t.Fatal("expected governance rows")
	}
	if resp.Rows[0].TagID == 0 {
		t.Fatal("expected tag_id in governance row")
	}
	if resp.Rows[0].PreferredLabel == "" || resp.Rows[0].PrimaryCategory == "" {
		t.Fatal("expected label/category in governance row")
	}
	if len(resp.Rows[0].Aliases) == 0 {
		t.Fatal("expected aliases in governance row")
	}
	if resp.Rows[0].UsageCount < resp.Rows[len(resp.Rows)-1].UsageCount {
		t.Fatal("expected governance rows sorted by usage_count desc")
	}
}

func TestTagMergeRequiresExplicitTargetTagID(t *testing.T) {
	t.Parallel()

	router, _ := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tags/1/merge", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTagMergeRejectsSourceEqualsTarget(t *testing.T) {
	t.Parallel()

	router, _ := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tags/1/merge", bytes.NewBufferString(`{"target_tag_id":1}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTagMergeMovesSourceImageTagsToExplicitTarget(t *testing.T) {
	t.Parallel()

	router, repos := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tags/1/merge", bytes.NewBufferString(`{"target_tag_id":2}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	if _, err := repos.tagRepo.FindByID(context.Background(), 1); err == nil {
		t.Fatal("expected source tag to be deleted after merge")
	}

	targetRows, err := repos.imageTagRepo.FindByTagID(context.Background(), 2, 20, 0)
	if err != nil {
		t.Fatalf("FindByTagID(target) error = %v", err)
	}
	if len(targetRows) == 0 {
		t.Fatal("expected merged image-tag associations on target")
	}
}

type tagHandlerTestRepos struct {
	tagRepo      repository.TagRepository
	aliasRepo    repository.TagAliasRepository
	imageTagRepo repository.ImageTagRepository
	db           *sql.DB
}

func newTagHandlerTestRouter(t *testing.T) (*gin.Engine, *tagHandlerTestRepos) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "tag-handler.db"))
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

	seedTagHandlerData(t, db)

	repos := &tagHandlerTestRepos{
		tagRepo:      repository.NewTagRepository(db),
		aliasRepo:    repository.NewTagAliasRepository(db),
		imageTagRepo: repository.NewImageTagRepository(db),
		db:           db,
	}
	adminSvc := service.NewTagAdminService(repos.db, repos.tagRepo, repos.aliasRepo, repos.imageTagRepo)
	h := NewTagHandler(repos.tagRepo, repos.aliasRepo, repos.imageTagRepo, adminSvc)

	router := gin.New()
	api := router.Group("/api/v1")
	api.GET("/tags", h.GetTags)
	api.GET("/tags/governance", h.GetGovernanceTags)
	api.POST("/tags", h.CreateTag)
	api.PUT("/tags/:id", h.UpdateTag)
	api.DELETE("/tags/:id", h.DeleteTag)
	api.POST("/tags/:id/merge", h.MergeTag)
	api.GET("/tags/:id/aliases", h.GetAliases)
	api.POST("/tags/:id/aliases", h.AddAlias)

	return router, repos
}

func seedTagHandlerData(t *testing.T, db *sql.DB) {
	t.Helper()

	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO images (id, path, filename, source_root, file_size, width, height, format, created_at, updated_at)
		VALUES (1, '/images/1.png', '1.png', '/images', 100, 100, 100, 'png', ?, ?)
	`, now, now)
	if err != nil {
		t.Fatalf("seed images: %v", err)
	}

	tagRepo := repository.NewTagRepository(db)
	for _, tag := range []*domain.Tag{
		{ID: 1, PreferredLabel: "blue sky", Slug: "blue-sky", PrimaryCategory: "scene", ReviewState: "confirmed", UsageCount: 10},
		{ID: 2, PreferredLabel: "sunrise", Slug: "sunrise", PrimaryCategory: "scene", ReviewState: "pending", UsageCount: 3},
		{ID: 3, PreferredLabel: "cloud", Slug: "cloud", PrimaryCategory: "meta", ReviewState: "confirmed", UsageCount: 1},
	} {
		if err := tagRepo.Save(context.Background(), tag); err != nil {
			t.Fatalf("seed tag %d: %v", tag.ID, err)
		}
	}

	aliasRepo := repository.NewTagAliasRepository(db)
	for _, alias := range []*domain.TagAlias{
		{ID: 1, TagID: 1, Label: "蓝天", AliasType: "translation"},
		{ID: 2, TagID: 2, Label: "蓝色黎明", AliasType: "translation"},
	} {
		if err := aliasRepo.Save(context.Background(), alias); err != nil {
			t.Fatalf("seed alias %d: %v", alias.ID, err)
		}
	}

	imageTagRepo := repository.NewImageTagRepository(db)
	if err := imageTagRepo.Save(context.Background(), &domain.ImageTag{ImageID: 1, TagID: 1, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("seed image tag: %v", err)
	}
}

func decodeJSONResponse(t *testing.T, w *httptest.ResponseRecorder, target any) {
	t.Helper()

	if err := json.Unmarshal(w.Body.Bytes(), target); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; body = %s", err, w.Body.String())
	}
}
