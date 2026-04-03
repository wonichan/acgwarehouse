package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthEndpointRemainsGoScopedWhenSidecarDegraded(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/health?sidecar_state=degraded", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got := body["scope"]; got != "go" {
		t.Fatalf("scope = %v, want go", got)
	}
	if _, exists := body["sidecar"]; exists {
		t.Fatal("health response should not include sidecar details")
	}
}

func TestReadyEndpointRemainsGoScopedWhenSidecarDegraded(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/ready?sidecar_state=degraded", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got := body["scope"]; got != "go" {
		t.Fatalf("scope = %v, want go", got)
	}
	if _, exists := body["sidecar"]; exists {
		t.Fatal("ready response should not include sidecar details")
	}
}
