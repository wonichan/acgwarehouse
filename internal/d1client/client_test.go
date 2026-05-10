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

func TestQueryRejectsWritableCTE(t *testing.T) {
	t.Parallel()

	client := NewClientWithAPIKey("http://127.0.0.1:1", "test-key")
	_, err := client.Query(context.Background(), "WITH deleted AS (DELETE FROM images RETURNING id) SELECT id FROM deleted")
	if err == nil || !strings.Contains(err.Error(), "WITH") {
		t.Fatalf("Query() error = %v, want WITH rejection", err)
	}
}

func TestReadOnlyClientRejectsMutationsWithoutHTTPCall(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("readonly mutation should not call HTTP endpoint %s", r.URL.Path)
	}))
	defer server.Close()

	client := NewClientWithAPIKeyAndReadOnly(server.URL, "test-key", true)
	ctx := context.Background()

	if err := client.Exec(ctx, "INSERT INTO images(id) VALUES (?)", 1); err == nil || !strings.Contains(err.Error(), "readonly") {
		t.Fatalf("Exec() error = %v, want readonly rejection", err)
	}
	if _, err := client.ExecReturningID(ctx, "INSERT INTO images(path) VALUES (?)", "x"); err == nil || !strings.Contains(err.Error(), "readonly") {
		t.Fatalf("ExecReturningID() error = %v, want readonly rejection", err)
	}
	if err := client.ExecBatch(ctx, []MutateStatement{{SQL: "UPDATE images SET path = ? WHERE id = ?", Params: []any{"x", 1}}}); err == nil || !strings.Contains(err.Error(), "readonly") {
		t.Fatalf("ExecBatch() error = %v, want readonly rejection", err)
	}
}
