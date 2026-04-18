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

	if resp.Total != 6 {
		t.Fatalf("total = %d, want 6", resp.Total)
	}
	if len(resp.Tags) != 6 {
		t.Fatalf("len(tags) = %d, want 6", len(resp.Tags))
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
	body := bytes.NewBufferString(`{"preferred_label":"蓝天白云","primary_category":"场景","level":"child"}`)

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

func TestTagCreateTagRejectsMissingHierarchyForNewTag(t *testing.T) {
	t.Parallel()

	router, _ := newTagHandlerTestRouter(t)
	body := bytes.NewBufferString(`{"preferred_label":"brand new rootless tag"}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tags", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestTagCreateTagCreatesRequestedHierarchyLevel(t *testing.T) {
	t.Parallel()

	router, repos := newTagHandlerTestRouter(t)
	body := bytes.NewBufferString(`{"preferred_label":"heroine","primary_category":"artist","level":"parent","parent_id":4}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tags", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusCreated, w.Body.String())
	}

	tag, err := repos.tagRepo.FindByLabel(context.Background(), "heroine")
	if err != nil {
		t.Fatalf("FindByLabel() error = %v", err)
	}
	if tag.Level != domain.TagLevelParent {
		t.Fatalf("Level = %q, want %q", tag.Level, domain.TagLevelParent)
	}
	if tag.ParentID == nil || *tag.ParentID != 4 {
		t.Fatalf("ParentID = %v, want 4", tag.ParentID)
	}
}

func TestTagCreateTagReusesExistingExactLabel(t *testing.T) {
	t.Parallel()

	router, _ := newTagHandlerTestRouter(t)
	body := bytes.NewBufferString(`{"preferred_label":"blue sky","level":"root"}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tags", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		ID     int64 `json:"id"`
		Reused bool  `json:"reused"`
	}
	decodeJSONResponse(t, w, &resp)
	if resp.ID != 1 {
		t.Fatalf("id = %d, want 1", resp.ID)
	}
	if !resp.Reused {
		t.Fatal("expected reused=true")
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

func TestTagUpdateTagRejectsHierarchyMutationFields(t *testing.T) {
	t.Parallel()

	router, _ := newTagHandlerTestRouter(t)
	body := bytes.NewBufferString(`{"level":"root"}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tags/1", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestTagDeleteTagDeletesUsedTagAndCleansAssociations(t *testing.T) {
	t.Parallel()

	router, repos := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tags/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Success            bool  `json:"success"`
		DeletedTagID       int64 `json:"deleted_tag_id"`
		AffectedImageCount int64 `json:"affected_image_count"`
		DetachedChildCount int64 `json:"detached_child_count"`
	}
	decodeJSONResponse(t, w, &resp)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	if resp.DeletedTagID != 1 {
		t.Fatalf("deleted_tag_id = %d, want 1", resp.DeletedTagID)
	}
	if resp.AffectedImageCount != 1 {
		t.Fatalf("affected_image_count = %d, want 1", resp.AffectedImageCount)
	}
	if resp.DetachedChildCount != 0 {
		t.Fatalf("detached_child_count = %d, want 0", resp.DetachedChildCount)
	}

	if _, err := repos.tagRepo.FindByID(context.Background(), 1); err == nil {
		t.Fatal("expected tag to be deleted")
	}
	aliases, err := repos.aliasRepo.FindByTagID(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindByTagID() error = %v", err)
	}
	if len(aliases) != 0 {
		t.Fatalf("expected aliases to be deleted, got %d", len(aliases))
	}
	items, err := repos.imageTagRepo.FindByImageID(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	found := false
	for _, item := range items {
		if item.TagID == 1 {
			found = true
			break
		}
	}
	if found {
		t.Fatal("expected image tag association to be removed when tag is deleted")
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

func TestTagGetStatsIncludesDirectAndTreeCounts(t *testing.T) {
	t.Parallel()

	router, _ := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/stats", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Stats []struct {
			TagID            int64  `json:"tag_id"`
			Level            string `json:"level"`
			UsageCount       int64  `json:"usage_count"`
			DirectUsageCount int64  `json:"direct_usage_count"`
			TreeUsageCount   int64  `json:"tree_usage_count"`
		} `json:"stats"`
	}
	decodeJSONResponse(t, w, &resp)
	if len(resp.Stats) == 0 {
		t.Fatal("expected stats rows")
	}
	for _, row := range resp.Stats {
		if row.Level == "" {
			t.Fatal("expected level in stats row")
		}
		if row.UsageCount != row.DirectUsageCount {
			t.Fatalf("usage_count should mirror direct_usage_count, got %d vs %d", row.UsageCount, row.DirectUsageCount)
		}
	}
}

func TestTagGetTreeReturnsHierarchyData(t *testing.T) {
	t.Parallel()

	router, _ := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/tree", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Tree []struct {
			TagID    int64 `json:"tag_id"`
			Children []struct {
				TagID int64 `json:"tag_id"`
			} `json:"children"`
		} `json:"tree"`
	}
	decodeJSONResponse(t, w, &resp)
	if len(resp.Tree) == 0 {
		t.Fatal("expected non-empty tree")
	}
}

func TestTagGetParentCandidatesReturnsRootForParentLevel(t *testing.T) {
	t.Parallel()

	router, _ := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/parent-candidates?level=parent", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Candidates []domain.Tag `json:"candidates"`
	}
	decodeJSONResponse(t, w, &resp)
	if len(resp.Candidates) == 0 || resp.Candidates[0].Level != domain.TagLevelRoot {
		t.Fatalf("unexpected candidates: %+v", resp.Candidates)
	}
}

func TestTagGetParentCandidatesAllowsNullPrimaryCategoryForChildLevel(t *testing.T) {
	t.Parallel()

	router, repos := newTagHandlerTestRouter(t)
	if _, err := repos.db.Exec(`UPDATE tags SET primary_category = NULL WHERE id = ?`, 6); err != nil {
		t.Fatalf("set primary_category null: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/parent-candidates?level=child", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Candidates []domain.Tag `json:"candidates"`
	}
	decodeJSONResponse(t, w, &resp)
	if len(resp.Candidates) != 1 {
		t.Fatalf("len(candidates) = %d, want 1", len(resp.Candidates))
	}
	if resp.Candidates[0].ID != 6 {
		t.Fatalf("candidate id = %d, want 6", resp.Candidates[0].ID)
	}
	if resp.Candidates[0].PrimaryCategory != "" {
		t.Fatalf("PrimaryCategory = %q, want empty string", resp.Candidates[0].PrimaryCategory)
	}
}

func TestTagChangeLevelUpdatesHierarchy(t *testing.T) {
	t.Parallel()

	router, repos := newTagHandlerTestRouter(t)
	body := bytes.NewBufferString(`{"level":"parent","parent_id":4}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tags/3/change-level", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	tag, err := repos.tagRepo.FindByID(context.Background(), 3)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if tag.Level != domain.TagLevelParent {
		t.Fatalf("Level = %q, want %q", tag.Level, domain.TagLevelParent)
	}
	if tag.ParentID == nil || *tag.ParentID != 4 {
		t.Fatalf("ParentID = %v, want 4", tag.ParentID)
	}
}

func TestTagReparentDetachesChild(t *testing.T) {
	t.Parallel()

	router, repos := newTagHandlerTestRouter(t)
	body := bytes.NewBufferString(`{}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tags/5/reparent", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	tag, err := repos.tagRepo.FindByID(context.Background(), 5)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if tag.ParentID != nil {
		t.Fatalf("ParentID = %v, want nil", tag.ParentID)
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

func TestTagGetDeletePreviewReturnsBlockingReasonForUsedTag(t *testing.T) {
	t.Parallel()

	router, _ := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/1/delete-preview", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		TagID              int64  `json:"tag_id"`
		AffectedImageCount int64  `json:"affected_image_count"`
		CanDelete          bool   `json:"can_delete"`
		ChildCount         int64  `json:"child_count"`
		BlockingReason     string `json:"blocking_reason"`
	}
	decodeJSONResponse(t, w, &resp)

	if resp.TagID != 1 {
		t.Fatalf("tag_id = %d, want 1", resp.TagID)
	}
	if resp.AffectedImageCount == 0 {
		t.Fatal("expected affected_image_count > 0 for used tag")
	}
	if !resp.CanDelete {
		t.Fatal("expected used tag preview to remain deletable")
	}
	if resp.BlockingReason != "" {
		t.Fatalf("blocking_reason = %q, want empty", resp.BlockingReason)
	}
	if resp.ChildCount != 0 {
		t.Fatalf("child_count = %d, want 0", resp.ChildCount)
	}
}

func TestTagDeleteTagDeletesUsedTagWithAffectedImageCount(t *testing.T) {
	t.Parallel()

	router, repos := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tags/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Success            bool   `json:"success"`
		AffectedImageCount int64  `json:"affected_image_count"`
		DetachedChildCount int64  `json:"detached_child_count"`
	}
	decodeJSONResponse(t, w, &resp)

	if !resp.Success {
		t.Fatal("expected success=true")
	}
	if resp.AffectedImageCount != 1 {
		t.Fatalf("affected_image_count = %d, want 1", resp.AffectedImageCount)
	}
	if resp.DetachedChildCount != 0 {
		t.Fatalf("detached_child_count = %d, want 0", resp.DetachedChildCount)
	}
	if _, err := repos.tagRepo.FindByID(context.Background(), 1); err == nil {
		t.Fatal("expected tag 1 to be deleted")
	}
}

func TestTagDeleteTagReturnsAffectedImageCountZeroWhenUnused(t *testing.T) {
	t.Parallel()

	router, repos := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tags/3", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Success            bool  `json:"success"`
		DeletedTagID       int64 `json:"deleted_tag_id"`
		AffectedImageCount int64 `json:"affected_image_count"`
		DetachedChildCount int64 `json:"detached_child_count"`
	}
	decodeJSONResponse(t, w, &resp)

	if !resp.Success {
		t.Fatal("expected success=true")
	}
	if resp.DeletedTagID != 3 {
		t.Fatalf("deleted_tag_id = %d, want 3", resp.DeletedTagID)
	}
	if resp.AffectedImageCount != 0 {
		t.Fatalf("affected_image_count = %d, want 0", resp.AffectedImageCount)
	}
	if resp.DetachedChildCount != 0 {
		t.Fatalf("detached_child_count = %d, want 0", resp.DetachedChildCount)
	}

	if _, err := repos.tagRepo.FindByID(context.Background(), 3); err == nil {
		t.Fatal("expected tag 3 to be deleted")
	}
}

func TestTagDeleteTagDetachesDirectChildrenWhenDeletingParent(t *testing.T) {
	t.Parallel()

	router, repos := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tags/4", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Success            bool  `json:"success"`
		DeletedTagID       int64 `json:"deleted_tag_id"`
		AffectedImageCount int64 `json:"affected_image_count"`
		DetachedChildCount int64 `json:"detached_child_count"`
	}
	decodeJSONResponse(t, w, &resp)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	if resp.DeletedTagID != 4 {
		t.Fatalf("deleted_tag_id = %d, want 4", resp.DeletedTagID)
	}
	if resp.DetachedChildCount != 1 {
		t.Fatalf("detached_child_count = %d, want 1", resp.DetachedChildCount)
	}

	if _, err := repos.tagRepo.FindByID(context.Background(), 4); err == nil {
		t.Fatal("expected tag 4 to be deleted")
	}
	child, err := repos.tagRepo.FindByID(context.Background(), 6)
	if err != nil {
		t.Fatalf("FindByID(child) error = %v", err)
	}
	if child.ParentID != nil {
		t.Fatalf("child.ParentID = %v, want nil", child.ParentID)
	}
}

func TestTagCleanupDeletesOnlySelectedUnusedTags(t *testing.T) {
	t.Parallel()

	router, repos := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tags/batch/cleanup", bytes.NewBufferString(`{"tag_ids":[1,3,999]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Deleted []struct {
			TagID          int64  `json:"tag_id"`
			PreferredLabel string `json:"preferred_label"`
		} `json:"deleted"`
		Blocked []struct {
			TagID          int64  `json:"tag_id"`
			PreferredLabel string `json:"preferred_label"`
			BlockingReason string `json:"blocking_reason"`
		} `json:"blocked"`
		Failed []struct {
			TagID          int64  `json:"tag_id"`
			PreferredLabel string `json:"preferred_label"`
			Error          string `json:"error"`
		} `json:"failed"`
	}
	decodeJSONResponse(t, w, &resp)

	if len(resp.Deleted) != 1 || resp.Deleted[0].TagID != 3 {
		t.Fatalf("deleted = %+v, want only tag_id=3", resp.Deleted)
	}
	if len(resp.Blocked) != 1 || resp.Blocked[0].TagID != 1 {
		t.Fatalf("blocked = %+v, want only tag_id=1", resp.Blocked)
	}
	if len(resp.Failed) != 1 || resp.Failed[0].TagID != 999 {
		t.Fatalf("failed = %+v, want only tag_id=999", resp.Failed)
	}

	if _, err := repos.tagRepo.FindByID(context.Background(), 3); err == nil {
		t.Fatal("expected selected unused tag to be deleted")
	}
	if _, err := repos.tagRepo.FindByID(context.Background(), 2); err != nil {
		t.Fatalf("non-selected tag should remain: %v", err)
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
	api.GET("/tags/stats", h.GetTagStats)
	api.GET("/tags/tree", h.GetTagTree)
	api.GET("/tags/parent-candidates", h.GetParentCandidates)
	api.GET("/tags/tree/roots", h.GetTreeRoots)
	api.GET("/tags/tree/children", h.GetTreeChildren)
	api.GET("/tags/orphans", h.GetOrphans)
	api.GET("/tags/:id/delete-preview", h.GetDeletePreview)
	api.POST("/tags", h.CreateTag)
	api.PUT("/tags/:id", h.UpdateTag)
	api.POST("/tags/:id/change-level", h.ChangeTagLevel)
	api.POST("/tags/:id/reparent", h.ReparentTag)
	api.DELETE("/tags/:id", h.DeleteTag)
	api.POST("/tags/:id/merge", h.MergeTag)
	api.POST("/tags/batch/cleanup", h.CleanUnusedTags)
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

	for _, tag := range []*domain.Tag{
		{ID: 1, PreferredLabel: "blue sky", Slug: "blue-sky", Level: domain.TagLevelChild, PrimaryCategory: "scene", ReviewState: "confirmed", UsageCount: 10},
		{ID: 2, PreferredLabel: "sunrise", Slug: "sunrise", Level: domain.TagLevelChild, PrimaryCategory: "scene", ReviewState: "pending", UsageCount: 3},
		{ID: 3, PreferredLabel: "cloud", Slug: "cloud", Level: domain.TagLevelChild, PrimaryCategory: "meta", ReviewState: "confirmed", UsageCount: 1},
		{ID: 4, PreferredLabel: "characters", Slug: "characters", Level: domain.TagLevelRoot, PrimaryCategory: "meta", ReviewState: "confirmed", UsageCount: 2},
		{ID: 6, PreferredLabel: "heroine group", Slug: "heroine-group", Level: domain.TagLevelParent, ParentID: int64PtrHandler(4), PrimaryCategory: "meta", ReviewState: "confirmed", UsageCount: 1},
		{ID: 5, PreferredLabel: "heroine child", Slug: "heroine-child", Level: domain.TagLevelChild, ParentID: int64PtrHandler(6), PrimaryCategory: "meta", ReviewState: "confirmed", UsageCount: 1},
	} {
		tag.CreatedAt = now
		if err := insertSeedTag(db, tag); err != nil {
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
	if err := imageTagRepo.Save(context.Background(), &domain.ImageTag{ImageID: 1, TagID: 5, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("seed image tag child: %v", err)
	}
}

func int64PtrHandler(v int64) *int64 {
	return &v
}

func decodeJSONResponse(t *testing.T, w *httptest.ResponseRecorder, target any) {
	t.Helper()

	if err := json.Unmarshal(w.Body.Bytes(), target); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; body = %s", err, w.Body.String())
	}
}

func TestTagGetTreeRootsReturnsRootTagsWithHasChildren(t *testing.T) {
	t.Parallel()

	router, _ := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/tree/roots", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Items []struct {
			ID              int64  `json:"id"`
			PreferredLabel  string `json:"preferred_label"`
			Level           string `json:"level"`
			HasChildren     bool   `json:"has_children"`
			UsageCount      int    `json:"usage_count"`
			PrimaryCategory string `json:"primary_category"`
		} `json:"items"`
	}
	decodeJSONResponse(t, w, &resp)

	if len(resp.Items) == 0 {
		t.Fatal("expected at least one root tag")
	}

	found := false
	for _, item := range resp.Items {
		if item.ID == 4 {
			found = true
			if item.PreferredLabel != "characters" {
				t.Fatalf("preferred_label = %q, want %q", item.PreferredLabel, "characters")
			}
			if !item.HasChildren {
				t.Fatal("expected tag 4 (characters root) to have has_children=true")
			}
		}
	}
	if !found {
		t.Fatal("expected to find root tag with id=4 (characters)")
	}
}

func TestTagGetTreeChildrenReturnsChildrenWithHasChildren(t *testing.T) {
	t.Parallel()

	router, _ := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/tree/children?parent_id=4", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Items []struct {
			ID              int64  `json:"id"`
			PreferredLabel  string `json:"preferred_label"`
			Level           string `json:"level"`
			HasChildren     bool   `json:"has_children"`
			ParentID        *int64 `json:"parent_id"`
		} `json:"items"`
	}
	decodeJSONResponse(t, w, &resp)

	if len(resp.Items) == 0 {
		t.Fatal("expected at least one child under parent_id=4")
	}

	found := false
	for _, item := range resp.Items {
		if item.ID == 6 {
			found = true
			if item.PreferredLabel != "heroine group" {
				t.Fatalf("preferred_label = %q, want %q", item.PreferredLabel, "heroine group")
			}
			if item.ParentID == nil || *item.ParentID != 4 {
				t.Fatalf("parent_id = %v, want 4", item.ParentID)
			}
			if !item.HasChildren {
				t.Fatal("expected tag 6 (heroine group parent) to have has_children=true")
			}
		}
	}
	if !found {
		t.Fatal("expected to find child tag with id=6 (heroine group)")
	}
}

func TestTagGetTreeChildrenReturns400WhenParentIDMissing(t *testing.T) {
	t.Parallel()

	router, _ := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/tree/children", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestTagGetTreeChildrenReturns400WhenParentIDInvalid(t *testing.T) {
	t.Parallel()

	router, _ := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/tree/children?parent_id=abc", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestTagGetTreeChildrenReturnsEmptyForLeafNode(t *testing.T) {
	t.Parallel()

	router, _ := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/tree/children?parent_id=1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Items []struct {
			ID int64 `json:"id"`
		} `json:"items"`
	}
	decodeJSONResponse(t, w, &resp)

	if len(resp.Items) != 0 {
		t.Fatalf("expected 0 children for leaf tag, got %d", len(resp.Items))
	}
}

func TestTagGetOrphansReturnsOrphanTagsWithPagination(t *testing.T) {
	t.Parallel()

	router, _ := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/orphans?limit=10&offset=0", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Items []struct {
			ID              int64  `json:"id"`
			PreferredLabel  string `json:"preferred_label"`
			Level           string `json:"level"`
			HasChildren     bool   `json:"has_children"`
			PrimaryCategory string `json:"primary_category"`
			UsageCount      int    `json:"usage_count"`
		} `json:"items"`
		Total   int  `json:"total"`
		HasMore bool `json:"has_more"`
	}
	decodeJSONResponse(t, w, &resp)

	if resp.Total == 0 {
		t.Fatal("expected at least one orphan tag")
	}
	if len(resp.Items) == 0 {
		t.Fatal("expected items in orphan response")
	}

	for _, item := range resp.Items {
		if item.Level == "root" {
			t.Fatalf("orphan tag %d (%q) should not be root level", item.ID, item.PreferredLabel)
		}
	}
}

func TestTagGetOrphansPaginationReturnsHasMore(t *testing.T) {
	t.Parallel()

	router, _ := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/orphans?limit=1&offset=0", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Items   []struct {
			ID int64 `json:"id"`
		} `json:"items"`
		Total   int  `json:"total"`
		HasMore bool `json:"has_more"`
	}
	decodeJSONResponse(t, w, &resp)

	if len(resp.Items) != 1 {
		t.Fatalf("expected exactly 1 item with limit=1, got %d", len(resp.Items))
	}
	if resp.Total <= 1 {
		t.Fatalf("expected total > 1 for has_more to be true, got total=%d", resp.Total)
	}
	if !resp.HasMore {
		t.Fatal("expected has_more=true when more items exist beyond limit")
	}
}

func TestTagGetOrphansSearchFiltersResults(t *testing.T) {
	t.Parallel()

	router, _ := newTagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/orphans?search=blue", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Items []struct {
			ID             int64  `json:"id"`
			PreferredLabel string `json:"preferred_label"`
		} `json:"items"`
		Total   int  `json:"total"`
		HasMore bool `json:"has_more"`
	}
	decodeJSONResponse(t, w, &resp)

	if resp.Total == 0 {
		t.Fatal("expected at least one orphan matching 'blue'")
	}
	for _, item := range resp.Items {
		if item.PreferredLabel != "blue sky" {
			t.Fatalf("expected only 'blue sky', got %q", item.PreferredLabel)
		}
	}
}

func insertSeedTag(db *sql.DB, tag *domain.Tag) error {
	if tag.Level == "" {
		tag.Level = domain.TagLevelChild
	}
	if tag.CreatedAt.IsZero() {
		tag.CreatedAt = time.Now()
	}
	_, err := db.Exec(`
		INSERT INTO tags (id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tag.ID, tag.PreferredLabel, tag.Slug, tag.Level, tag.ParentID, tag.PrimaryCategory, tag.ReviewState, tag.TrustScore, tag.UsageCount, tag.CreatedAt)
	return err
}
