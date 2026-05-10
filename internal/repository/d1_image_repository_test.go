package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/wonichan/acgwarehouse-backend/internal/d1client"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

func TestD1FindImagesWithoutAITagsUsesAutoUntaggedCriteria(t *testing.T) {
	t.Parallel()

	var req d1client.QueryRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/query" {
			t.Fatalf("path = %s, want /api/query", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"results":[]}`))
	}))
	defer server.Close()

	repo := NewD1ImageRepositoryWithTags(d1client.NewClientWithAPIKey(server.URL, "test-key"), nil)
	if _, err := repo.FindImagesWithoutAITags(context.Background(), 25); err != nil {
		t.Fatalf("FindImagesWithoutAITags() error = %v", err)
	}

	requiredSQL := []string{
		"i.thumbnail_small_url IS NOT NULL",
		"i.thumbnail_small_url != ''",
		"it.review_state != ?",
		"it.source = ?",
		"pt.status IN (?, ?, ?)",
	}
	for _, fragment := range requiredSQL {
		if !strings.Contains(req.SQL, fragment) {
			t.Fatalf("D1 SQL missing %q:\n%s", fragment, req.SQL)
		}
	}

	wantParams := []any{
		domain.ReviewStateRejected,
		domain.ImageTagSourceAI,
		domain.ReviewStateRejected,
		domain.PlatformTaskTypeAITagGeneration,
		domain.PlatformTaskStatusPending,
		domain.PlatformTaskStatusQueued,
		domain.PlatformTaskStatusRunning,
		float64(25),
	}
	if len(req.Params) != len(wantParams) {
		t.Fatalf("len(params) = %d, want %d: %#v", len(req.Params), len(wantParams), req.Params)
	}
	for i := range wantParams {
		if req.Params[i] != wantParams[i] {
			t.Fatalf("params[%d] = %#v, want %#v; all params=%#v", i, req.Params[i], wantParams[i], req.Params)
		}
	}
}

func TestD1FindByGalleryFilterKeepsExactAndSubtreeSemantics(t *testing.T) {
	t.Parallel()

	requests := make([]d1client.QueryRequest, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/query" {
			t.Fatalf("path = %s, want /api/query", r.URL.Path)
		}
		var req d1client.QueryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		requests = append(requests, req)

		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(req.SQL, "WITH RECURSIVE descendants") {
			_, _ = w.Write([]byte(`{"success":true,"results":[{"id":20},{"id":21}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"success":true,"results":[]}`))
	}))
	defer server.Close()

	tagRepo := NewD1TagRepository(d1client.NewClientWithAPIKey(server.URL, "test-key"))
	repo := NewD1ImageRepositoryWithTags(d1client.NewClientWithAPIKey(server.URL, "test-key"), tagRepo)
	if _, err := repo.FindByGalleryFilter(context.Background(), []int64{10}, []int64{20}, 25, 5, "created_at", "desc"); err != nil {
		t.Fatalf("FindByGalleryFilter() error = %v", err)
	}

	if len(requests) != 2 {
		t.Fatalf("len(requests) = %d, want 2", len(requests))
	}
	filterReq := requests[1]
	if strings.Contains(filterReq.SQL, "it.tag_id IN (?, ?, ?)") {
		t.Fatalf("exact tag was expanded in SQL:\n%s", filterReq.SQL)
	}
	requiredSQL := []string{
		"it.tag_id = ? AND it.review_state != 'rejected'",
		"it.tag_id IN (?, ?)",
	}
	for _, fragment := range requiredSQL {
		if !strings.Contains(filterReq.SQL, fragment) {
			t.Fatalf("D1 SQL missing %q:\n%s", fragment, filterReq.SQL)
		}
	}
	wantParams := []any{float64(10), float64(20), float64(21), float64(25), float64(5)}
	if !reflect.DeepEqual(filterReq.Params, wantParams) {
		t.Fatalf("params = %#v, want %#v", filterReq.Params, wantParams)
	}
}
