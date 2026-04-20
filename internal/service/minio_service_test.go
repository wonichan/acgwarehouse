package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestMinioServiceUploadReturnsRelativePathAndUsesBucketScopedKey(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.RawQuery == "location=" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/"></LocationConstraint>`))
			return
		}
		gotPath = r.URL.Path
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		if _, err := io.ReadAll(r.Body); err != nil {
			t.Fatalf("read request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	parsed, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	svc, err := NewMinioService(parsed.Host, "access", "secret", "acg", false)
	if err != nil {
		t.Fatalf("NewMinioService() error = %v", err)
	}

	path, err := svc.Upload(context.Background(), "test-image", "large", []byte("jpg-bytes"))
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	datePrefix := time.Now().Format("20060102")
	wantPath := "/acg/thumbnails/" + datePrefix + "/test-image-large.jpg"
	if gotPath != wantPath {
		t.Fatalf("request path = %q, want %q", gotPath, wantPath)
	}
	wantStoredPath := "acg/thumbnails/" + datePrefix + "/test-image-large.jpg"
	if path != wantStoredPath {
		t.Fatalf("stored path = %q, want %q", path, wantStoredPath)
	}
}
