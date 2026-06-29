package frontendserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Handler_proxies_api_requests_before_spa_fallback(t *testing.T) {
	// Given
	var upstreamMethod string
	var upstreamPath string
	var upstreamBody string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read upstream request body: %v", err)
		}
		upstreamMethod = r.Method
		upstreamPath = r.URL.Path
		upstreamBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"ok":true},"msg":""}`))
	}))
	defer backend.Close()

	distDir := writeTestDist(t)
	handler, err := NewHandler(Config{DistDir: distDir, BackendURL: backend.URL})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/users/me", strings.NewReader(`{"nickname":"Alice"}`))

	// When
	handler.ServeHTTP(recorder, request)

	// Then
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "<html") {
		t.Fatalf("api response returned SPA HTML: %s", recorder.Body.String())
	}
	if upstreamMethod != http.MethodPut || upstreamPath != "/api/v1/users/me" || upstreamBody != `{"nickname":"Alice"}` {
		t.Fatalf("upstream request = %s %s %s, want proxied PUT profile body", upstreamMethod, upstreamPath, upstreamBody)
	}
}

func Test_Handler_rejects_api_request_when_body_exceeds_limit(t *testing.T) {
	// Given
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer backend.Close()
	distDir := writeTestDist(t)
	handler, err := NewHandler(Config{DistDir: distDir, BackendURL: backend.URL, MaxRequestBodyBytes: 4})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/users/me", strings.NewReader(`{"nickname":"Alice"}`))

	// When
	handler.ServeHTTP(recorder, request)

	// Then
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d body=%s, want 413", recorder.Code, recorder.Body.String())
	}
}

func Test_Handler_serves_spa_index_for_history_route(t *testing.T) {
	// Given
	backend := httptest.NewServer(http.NotFoundHandler())
	defer backend.Close()
	distDir := writeTestDist(t)
	handler, err := NewHandler(Config{DistDir: distDir, BackendURL: backend.URL})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/account", nil)

	// When
	handler.ServeHTTP(recorder, request)

	// Then
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "ACGWarehouse test app") {
		t.Fatalf("body = %q, want SPA index", recorder.Body.String())
	}
}

func writeTestDist(t *testing.T) string {
	t.Helper()
	distDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(distDir, "index.html"), []byte("<html>ACGWarehouse test app</html>"), 0o600); err != nil {
		t.Fatalf("write test index: %v", err)
	}
	return distDir
}
