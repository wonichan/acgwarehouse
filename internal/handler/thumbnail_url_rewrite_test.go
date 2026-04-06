package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestThumbnailURLRewriteImageHandlerListAndGet(t *testing.T) {
	t.Parallel()

	router, repos := newImageHandlerTestRouter(t)
	setImageThumbnailURLs(t, repos.db, 1,
		"http://localhost:3001/thumbs/image-1-small.jpg?size=small",
		"http://127.0.0.1:3002/thumbs/image-1-large.jpg",
	)

	listResp := performRequest(t, router, http.MethodGet, "http://127.0.0.1:4321/api/v1/images?sort_by=id&sort_dir=asc&limit=1", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body=%s", listResp.Code, http.StatusOK, listResp.Body.String())
	}

	var listBody struct {
		Images []map[string]any `json:"images"`
	}
	decodeThumbnailRewriteJSON(t, listResp, &listBody)
	if len(listBody.Images) != 1 {
		t.Fatalf("len(images) = %d, want 1", len(listBody.Images))
	}
	assertThumbnailURLs(t, listBody.Images[0],
		"http://127.0.0.1:4321/thumbs/image-1-small.jpg?size=small",
		"http://127.0.0.1:4321/thumbs/image-1-large.jpg",
	)

	getResp := performRequest(t, router, http.MethodGet, "http://127.0.0.1:4321/api/v1/images/1", nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d, body=%s", getResp.Code, http.StatusOK, getResp.Body.String())
	}

	var getBody map[string]any
	decodeThumbnailRewriteJSON(t, getResp, &getBody)
	assertThumbnailURLs(t, getBody,
		"http://127.0.0.1:4321/thumbs/image-1-small.jpg?size=small",
		"http://127.0.0.1:4321/thumbs/image-1-large.jpg",
	)
}

func TestThumbnailURLRewriteSearchHandlerLeavesExternalURLsUntouched(t *testing.T) {
	t.Parallel()

	router, repos := newViewerWindowTestRouter(t)
	seedViewerWindowSearchData(t, repos)
	setImageThumbnailURLs(t, repos.db, 1,
		"http://localhost:4101/thumbs/search-small.jpg",
		"https://cdn.example.com/thumbs/search-large.jpg",
	)

	resp := performRequest(t, router, http.MethodGet, "http://127.0.0.1:4321/api/v1/search?q=cat&sort_by=filename&sort_order=asc", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("search status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var body struct {
		Images []map[string]any `json:"images"`
	}
	decodeThumbnailRewriteJSON(t, resp, &body)
	if len(body.Images) == 0 {
		t.Fatal("search returned no images")
	}
	assertThumbnailURLs(t, body.Images[0],
		"http://127.0.0.1:4321/thumbs/search-small.jpg",
		"https://cdn.example.com/thumbs/search-large.jpg",
	)
}

func TestThumbnailURLRewriteViewerWindowItems(t *testing.T) {
	t.Parallel()

	router, repos := newViewerWindowTestRouter(t)
	setImageThumbnailURLs(t, repos.db, 1,
		"http://127.0.0.1:5101/thumbs/viewer-small.jpg",
		"http://localhost:5102/thumbs/viewer-large.jpg",
	)

	payload := map[string]any{
		"source":            "gallery",
		"selected_index":    0,
		"selected_image_id": 1,
		"limit":             10,
		"snapshot": map[string]any{
			"sort_by":  "id",
			"sort_dir": "asc",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	resp := performRequest(t, router, http.MethodPost, "http://localhost:4321/api/v1/viewer/window", body)
	if resp.Code != http.StatusOK {
		t.Fatalf("viewer status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var decoded struct {
		Items []map[string]any `json:"items"`
	}
	decodeThumbnailRewriteJSON(t, resp, &decoded)
	if len(decoded.Items) == 0 {
		t.Fatal("viewer returned no items")
	}
	assertThumbnailURLs(t, decoded.Items[0],
		"http://localhost:4321/thumbs/viewer-small.jpg",
		"http://localhost:4321/thumbs/viewer-large.jpg",
	)
}

func TestThumbnailURLRewriteDuplicateHandlerListAndGet(t *testing.T) {
	t.Parallel()

	mockSidecar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/compute/duplicates/detect":
			_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-rewrite", "status": "pending", "progress": 0})
		case "/compute/duplicates/tasks/task-rewrite":
			_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-rewrite", "status": "completed", "progress": 100})
		case "/compute/duplicates/tasks/task-rewrite/result":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"groups": []map[string]any{{
					"group_id":       0,
					"recommended_id": 1,
					"members": []map[string]any{
						{"image_id": 1, "sha256": "sha1", "phash": "phash1", "distance": 0, "is_recommended": true, "recommendation_score": 90.0, "recommendation_reasons": []map[string]any{{"factor": "resolution", "value": "100x120", "score": 10.0, "weight": 0.5}}},
						{"image_id": 2, "sha256": "sha2", "phash": "phash2", "distance": 3, "is_recommended": false, "recommendation_score": 70.0, "recommendation_reasons": []map[string]any{{"factor": "size", "value": "1025", "score": 8.0, "weight": 0.3}}},
					},
				}},
				"total_images":        3,
				"total_groups":        1,
				"skipped_images":      []any{},
				"computation_time_ms": 10,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockSidecar.Close()

	runtime := createReadyRuntime(t)
	router, db := setupDuplicateHandlerTest(t, runtime, mockSidecar.URL)
	setImageThumbnailURLs(t, db, 1,
		"http://localhost:6101/thumbs/dup-small.jpg",
		"http://127.0.0.1:6102/thumbs/dup-large.jpg",
	)

	detectResp := performRequest(t, router, http.MethodPost, "http://127.0.0.1:4321/api/v1/duplicates/detect", []byte(`{"threshold":40}`))
	if detectResp.Code != http.StatusOK {
		t.Fatalf("detect status = %d, want %d, body=%s", detectResp.Code, http.StatusOK, detectResp.Body.String())
	}

	listResp := performRequest(t, router, http.MethodGet, "http://127.0.0.1:4321/api/v1/duplicates", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body=%s", listResp.Code, http.StatusOK, listResp.Body.String())
	}
	var listBody map[string]any
	decodeThumbnailRewriteJSON(t, listResp, &listBody)
	listGroup := listBody["groups"].([]any)[0].(map[string]any)
	listImage := listGroup["images"].([]any)[0].(map[string]any)
	assertThumbnailURLs(t, listImage,
		"http://127.0.0.1:4321/thumbs/dup-small.jpg",
		"http://127.0.0.1:4321/thumbs/dup-large.jpg",
	)

	getResp := performRequest(t, router, http.MethodGet, "http://127.0.0.1:4321/api/v1/duplicates/1", nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d, body=%s", getResp.Code, http.StatusOK, getResp.Body.String())
	}
	var getBody map[string]any
	decodeThumbnailRewriteJSON(t, getResp, &getBody)
	getImage := getBody["images"].([]any)[0].(map[string]any)
	assertThumbnailURLs(t, getImage,
		"http://127.0.0.1:4321/thumbs/dup-small.jpg",
		"http://127.0.0.1:4321/thumbs/dup-large.jpg",
	)
}

func performRequest(t *testing.T, router http.Handler, method, target string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func setImageThumbnailURLs(t *testing.T, db *sql.DB, imageID int64, smallURL, largeURL string) {
	t.Helper()
	if _, err := db.Exec(`UPDATE images SET thumbnail_small_url = ?, thumbnail_large_url = ? WHERE id = ?`, smallURL, largeURL, imageID); err != nil {
		t.Fatalf("update thumbnail urls for image %d: %v", imageID, err)
	}
}

func decodeThumbnailRewriteJSON(t *testing.T, w *httptest.ResponseRecorder, dest any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), dest); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", w.Body.String(), err)
	}
}

func assertThumbnailURLs(t *testing.T, image map[string]any, wantSmall, wantLarge string) {
	t.Helper()
	if got := image["thumbnail_small_url"]; got != wantSmall {
		t.Fatalf("thumbnail_small_url = %v, want %q", got, wantSmall)
	}
	if got := image["thumbnail_large_url"]; got != wantLarge {
		t.Fatalf("thumbnail_large_url = %v, want %q", got, wantLarge)
	}
}
