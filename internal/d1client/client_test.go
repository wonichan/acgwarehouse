package d1client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestQuerySendsAPIKey(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/query" {
			t.Fatalf("path = %s, want /api/query", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"results":[{"cnt":1}]}`))
	}))
	defer server.Close()

	client := NewClientWithAPIKey(server.URL, "test-key")
	rows, err := client.Query(context.Background(), "SELECT 1 AS cnt")
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
}

func TestQueryUnauthorizedMentionsAPIKey(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClientWithAPIKey(server.URL, "bad-key")
	_, err := client.Query(context.Background(), "SELECT 1")
	if err == nil {
		t.Fatal("Query() error = nil, want unauthorized error")
	}
	if !strings.Contains(err.Error(), "check API key") {
		t.Fatalf("Query() error = %q, want API key hint", err.Error())
	}
}
