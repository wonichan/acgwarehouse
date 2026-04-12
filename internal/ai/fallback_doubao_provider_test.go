package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFallbackDoubaoProvider_UsesFirstModelWhenSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(doubaoResponse{Choices: []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}{{Message: struct {
			Content string `json:"content"`
		}{Content: "tag1"}}}})
	}))
	defer server.Close()

	provider := NewFallbackDoubaoProvider("auto", &DoubaoProvider{apiKey: "key", model: "doubao-pro", endpoint: server.URL, httpClient: server.Client()})
	result, err := provider.GenerateTags(context.Background(), "https://example.com/img.jpg", "prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ModelName != "doubao-pro" {
		t.Fatalf("expected model doubao-pro, got %s", result.ModelName)
	}
}

func TestFallbackDoubaoProvider_FallsBackOnRetryableFailure(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer primary.Close()
	secondary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(doubaoResponse{Choices: []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}{{Message: struct {
			Content string `json:"content"`
		}{Content: "fallback-tag"}}}})
	}))
	defer secondary.Close()

	provider := NewFallbackDoubaoProvider("auto",
		&DoubaoProvider{apiKey: "key", model: "doubao-pro", endpoint: primary.URL, httpClient: primary.Client()},
		&DoubaoProvider{apiKey: "key", model: "doubao-lite", endpoint: secondary.URL, httpClient: secondary.Client()},
	)

	result, err := provider.GenerateTags(context.Background(), "https://example.com/img.jpg", "prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ModelName != "doubao-lite" {
		t.Fatalf("expected fallback model doubao-lite, got %s", result.ModelName)
	}
}

func TestFallbackDoubaoProvider_StopsOnContextCancellation(t *testing.T) {
	provider := NewFallbackDoubaoProvider("auto", &DoubaoProvider{apiKey: "key", model: "doubao-pro", endpoint: "https://example.com", httpClient: http.DefaultClient})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := provider.GenerateTags(ctx, "https://example.com/img.jpg", "prompt")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestFallbackDoubaoProvider_Name(t *testing.T) {
	provider := NewFallbackDoubaoProvider("auto", &DoubaoProvider{model: "doubao-pro"})
	if provider.Name() != "doubao" {
		t.Fatalf("expected name doubao, got %s", provider.Name())
	}
}

func TestFallbackDoubaoProvider_EmptyClients(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic with empty clients")
		}
	}()
	NewFallbackDoubaoProvider("auto")
}
