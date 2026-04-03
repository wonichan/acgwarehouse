package handler

import (
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
		{method: http.MethodPost, path: "/api/v1/images/1/ai-tags"},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		router.ServeHTTP(w, req)
		if w.Code == http.StatusNotFound {
			t.Fatalf("%s %s returned 404, route not registered", tc.method, tc.path)
		}
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/python/health", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET /python/health status = %d, want 404 (no direct python route)", w.Code)
	}
}
