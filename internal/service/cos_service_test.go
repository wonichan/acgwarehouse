package service

import (
	"context"
	"hash/crc64"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/wonichan/acgwarehouse-backend/internal/config"
)

func TestCOSServiceUploadReturnsURLAndUsesKeyFormat(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "image/jpeg" {
			t.Fatalf("content-type = %q, want image/jpeg", ct)
		}
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		checksum := crc64.Checksum(payload, crc64.MakeTable(crc64.ECMA))
		w.Header().Set("x-cos-hash-crc64ecma", strconv.FormatUint(checksum, 10))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<ok/>"))
	}))
	defer srv.Close()

	t.Setenv("COS_SECRET_ID", "")
	t.Setenv("COS_SECRET_KEY", "")

	svc, err := NewCOSService(&config.COSConfig{BucketURL: srv.URL, SecretID: "sid", SecretKey: "skey"})
	if err != nil {
		t.Fatalf("NewCOSService() error = %v", err)
	}

	url, err := svc.Upload(context.Background(), 123, "small", []byte("jpg-bytes"))
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	if gotPath != "/thumbnails/123_small.jpg" {
		t.Fatalf("request path = %q, want /thumbnails/123_small.jpg", gotPath)
	}
	if url != srv.URL+"/thumbnails/123_small.jpg" {
		t.Fatalf("returned url = %q, want %q", url, srv.URL+"/thumbnails/123_small.jpg")
	}
}

func TestCOSServiceUploadReturnsErrorOnFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	}))
	defer srv.Close()

	t.Setenv("COS_SECRET_ID", "")
	t.Setenv("COS_SECRET_KEY", "")

	svc, err := NewCOSService(&config.COSConfig{BucketURL: srv.URL, SecretID: "sid", SecretKey: "skey"})
	if err != nil {
		t.Fatalf("NewCOSService() error = %v", err)
	}

	_, err = svc.Upload(context.Background(), 1, "large", []byte("jpg-bytes"))
	if err == nil {
		t.Fatal("Upload() expected error on non-2xx response")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "500") {
		t.Fatalf("Upload() error = %v, want 500 in error", err)
	}
}
