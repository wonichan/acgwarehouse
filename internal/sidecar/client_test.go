package sidecar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSidecarClient_NewSidecarClient(t *testing.T) {
	t.Parallel()

	client := NewSidecarClient("http://127.0.0.1:9090")
	if client == nil {
		t.Fatal("NewSidecarClient() returned nil")
	}
	if got, want := client.baseURL, "http://127.0.0.1:9090"; got != want {
		t.Fatalf("baseURL = %q, want %q", got, want)
	}
	if client.httpClient == nil {
		t.Fatal("httpClient is nil")
	}
}

func TestSidecarClient_SubmitDetectionSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Method, http.MethodPost; got != want {
			t.Fatalf("method = %q, want %q", got, want)
		}
		if got, want := r.URL.Path, "/compute/duplicates/detect"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}

		var req DetectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if req.Threshold != 40 {
			t.Fatalf("threshold = %d, want 40", req.Threshold)
		}
		if len(req.Images) != 1 || req.Images[0].ID != 1 {
			t.Fatalf("images unexpected: %+v", req.Images)
		}
		if req.Images[0].SHA256 != "sha-a" {
			t.Fatalf("sha256 = %q, want %q", req.Images[0].SHA256, "sha-a")
		}
		if req.Images[0].PHashHex != "0123456789abcdef" {
			t.Fatalf("phash_hex = %q, want %q", req.Images[0].PHashHex, "0123456789abcdef")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(DetectionTaskStatus{TaskID: "task-123", Status: "pending", Progress: 0})
	}))
	defer server.Close()

	client := NewSidecarClient(server.URL)
	taskID, err := client.SubmitDetection(context.Background(), DetectionRequest{
		Threshold: 40,
		Images: []DetectionImageInput{{
			ID:       1,
			Path:     "C:/img/a.jpg",
			Width:    100,
			Height:   200,
			FileSize: 12345,
			Format:   "jpg",
			SHA256:   "sha-a",
			PHashHex: "0123456789abcdef",
		}},
	})
	if err != nil {
		t.Fatalf("SubmitDetection() error = %v", err)
	}
	if got, want := taskID, "task-123"; got != want {
		t.Fatalf("taskID = %q, want %q", got, want)
	}
}

func TestSidecarClient_SubmitDetectionErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    string
	}{
		{name: "conflict", statusCode: http.StatusConflict, body: `{"detail":"busy"}`, wantErr: "HTTP 409"},
		{name: "internal", statusCode: http.StatusInternalServerError, body: `{"detail":"boom"}`, wantErr: "HTTP 500"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			client := NewSidecarClient(server.URL)
			_, err := client.SubmitDetection(context.Background(), DetectionRequest{Threshold: 40})
			if err == nil {
				t.Fatal("SubmitDetection() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %q, want contains %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestSidecarClient_PollProgress(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Method, http.MethodGet; got != want {
			t.Fatalf("method = %q, want %q", got, want)
		}
		if got, want := r.URL.Path, "/compute/duplicates/tasks/task-xyz"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
		_ = json.NewEncoder(w).Encode(DetectionTaskStatus{TaskID: "task-xyz", Status: "running", Progress: 55.5, Message: "working"})
	}))
	defer server.Close()

	client := NewSidecarClient(server.URL)
	status, err := client.PollProgress(context.Background(), "task-xyz")
	if err != nil {
		t.Fatalf("PollProgress() error = %v", err)
	}
	if status.TaskID != "task-xyz" || status.Status != "running" {
		t.Fatalf("status unexpected: %+v", status)
	}
}

func TestSidecarClient_PollProgressNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"Task not found"}`))
	}))
	defer server.Close()

	client := NewSidecarClient(server.URL)
	_, err := client.PollProgress(context.Background(), "missing")
	if err == nil {
		t.Fatal("PollProgress() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("error = %q, want contains HTTP 404", err.Error())
	}
}

func TestSidecarClient_FetchResults(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/compute/duplicates/tasks/task-done/result"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
		_ = json.NewEncoder(w).Encode(DetectionResult{
			Groups: []DetectionResultGroup{{
				GroupID:       0,
				RecommendedID: 2,
				Members: []DetectionResultMember{{
					ImageID:             2,
					SHA256:              "sha",
					PHash:               "phash",
					Distance:            0,
					IsRecommended:       true,
					RecommendationScore: 99,
				}},
			}},
			TotalImages:       2,
			TotalGroups:       1,
			SkippedImages:     []map[string]interface{}{},
			ComputationTimeMs: 120,
		})
	}))
	defer server.Close()

	client := NewSidecarClient(server.URL)
	result, err := client.FetchResults(context.Background(), "task-done")
	if err != nil {
		t.Fatalf("FetchResults() error = %v", err)
	}
	if result.TotalGroups != 1 || len(result.Groups) != 1 {
		t.Fatalf("result unexpected: %+v", result)
	}
}

func TestSidecarClient_SubmitDetection_OmitsOptionalCacheFieldsWhenEmpty(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		imagesRaw, ok := req["images"].([]interface{})
		if !ok || len(imagesRaw) != 1 {
			t.Fatalf("images unexpected: %#v", req["images"])
		}
		img, ok := imagesRaw[0].(map[string]interface{})
		if !ok {
			t.Fatalf("image item unexpected: %#v", imagesRaw[0])
		}

		if _, exists := img["sha256"]; exists {
			t.Fatalf("expected sha256 to be omitted when empty, got %v", img["sha256"])
		}
		if _, exists := img["phash"]; exists {
			t.Fatalf("expected phash to be omitted when empty, got %v", img["phash"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(DetectionTaskStatus{TaskID: "task-omit", Status: "pending", Progress: 0})
	}))
	defer server.Close()

	client := NewSidecarClient(server.URL)
	_, err := client.SubmitDetection(context.Background(), DetectionRequest{
		Threshold: 10,
		Images: []DetectionImageInput{{
			ID:       1,
			Path:     "C:/img/no-cache.jpg",
			Width:    100,
			Height:   200,
			FileSize: 12345,
			Format:   "jpg",
		}},
	})
	if err != nil {
		t.Fatalf("SubmitDetection() error = %v", err)
	}
}

func TestSidecarClient_FetchResults_AllowsMissingOptionalHashFields(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/compute/duplicates/tasks/task-done/result"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
		_, _ = w.Write([]byte(`{
			"groups": [{
				"group_id": 0,
				"recommended_id": 2,
				"members": [{
					"image_id": 2,
					"distance": 0,
					"is_recommended": true,
					"recommendation_score": 99,
					"recommendation_reasons": []
				}]
			}],
			"total_images": 2,
			"total_groups": 1,
			"skipped_images": [],
			"computation_time_ms": 120
		}`))
	}))
	defer server.Close()

	client := NewSidecarClient(server.URL)
	result, err := client.FetchResults(context.Background(), "task-done")
	if err != nil {
		t.Fatalf("FetchResults() error = %v", err)
	}

	if len(result.Groups) != 1 || len(result.Groups[0].Members) != 1 {
		t.Fatalf("result unexpected: %+v", result)
	}
	member := result.Groups[0].Members[0]
	if member.SHA256 != "" {
		t.Fatalf("SHA256 = %q, want empty string for missing field", member.SHA256)
	}
	if member.PHash != "" {
		t.Fatalf("PHash = %q, want empty string for missing field", member.PHash)
	}
}

func TestSidecarClient_FetchResultsBadRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"detail":"Task not completed"}`))
	}))
	defer server.Close()

	client := NewSidecarClient(server.URL)
	_, err := client.FetchResults(context.Background(), "task-not-complete")
	if err == nil {
		t.Fatal("FetchResults() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "HTTP 400") {
		t.Fatalf("error = %q, want contains HTTP 400", err.Error())
	}
}
