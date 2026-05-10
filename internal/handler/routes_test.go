package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRoutesRegistersTagAndImageTagEndpoints(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	SetupRoutes(router)

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/health"},
		{method: http.MethodGet, path: "/ready"},
		{method: http.MethodGet, path: "/api/v1/tags"},
		{method: http.MethodPost, path: "/api/v1/tags"},
		{method: http.MethodGet, path: "/api/v1/images/1/tags"},
		{method: http.MethodPost, path: "/api/v1/viewer/window"},
		{method: http.MethodPost, path: "/api/v1/images/1/ai-tags"},
		{method: http.MethodPost, path: "/api/v1/images/batch-ai-tags/regenerate"},
		{method: http.MethodPost, path: "/api/v1/image-moves/preview"},
		{method: http.MethodPost, path: "/api/v1/image-moves/execute"},
		{method: http.MethodPost, path: "/api/v1/image-moves/jobs"},
		{method: http.MethodGet, path: "/api/v1/image-moves/jobs/1"},
		{method: http.MethodPost, path: "/api/v1/image-moves/jobs/1/cancel"},
		{method: http.MethodGet, path: "/api/v1/image-moves/history"},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		router.ServeHTTP(w, req)
		if w.Code == http.StatusNotFound {
			t.Fatalf("%s %s returned 404, route not registered", tc.method, tc.path)
		}
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /health status = %d, want %d", w.Code, http.StatusOK)
	}
	var healthBody map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &healthBody); err != nil {
		t.Fatalf("json.Unmarshal(/health) error = %v", err)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/ready", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /ready status = %d, want %d", w.Code, http.StatusOK)
	}
	var readyBody map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &readyBody); err != nil {
		t.Fatalf("json.Unmarshal(/ready) error = %v", err)
	}
}
