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
