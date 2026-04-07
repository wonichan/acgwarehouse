package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/sidecar"
)

func captureStandardLogs(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()

	buf := &bytes.Buffer{}
	originalWriter := log.Writer()
	originalFlags := log.Flags()
	originalPrefix := log.Prefix()
	log.SetOutput(buf)
	log.SetFlags(0)
	log.SetPrefix("")

	return buf, func() {
		log.SetOutput(originalWriter)
		log.SetFlags(originalFlags)
		log.SetPrefix(originalPrefix)
	}
}

func setupDuplicateTestDB(t *testing.T) *sql.DB {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "duplicate_service_test_*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	t.Cleanup(func() { _ = os.Remove(tmpPath) })

	db, err := sql.Open("sqlite3", tmpPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}

	return db
}

func insertTestImages(t *testing.T, db *sql.DB) []int64 {
	t.Helper()

	now := time.Now()
	ids := make([]int64, 3)
	paths := []string{"/test/a.jpg", "/test/b.jpg", "/test/c.jpg"}

	for i := range paths {
		result, err := db.Exec(`
			INSERT INTO images (path, filename, source_root, file_size, width, height, format, phash, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, paths[i], "img.jpg", "/test", 1024+int64(i), 100+i, 120+i, "jpg", int64(0), now, now)
		if err != nil {
			t.Fatalf("insert image: %v", err)
		}
		ids[i], _ = result.LastInsertId()
	}

	return ids
}

func insertOldDuplicateGroup(t *testing.T, db *sql.DB, imageID int64) {
	t.Helper()

	now := time.Now()
	res, err := db.Exec(`INSERT INTO duplicate_groups (recommended_image_id, similarity_threshold, created_at) VALUES (?, ?, ?)`, imageID, 10, now)
	if err != nil {
		t.Fatalf("insert duplicate_group: %v", err)
	}
	groupID, _ := res.LastInsertId()
	_, err = db.Exec(`INSERT INTO duplicate_relations (group_id, image_id, is_recommended, file_hash, phash_distance, recommendation_score, recommendation_rationale) VALUES (?, ?, 1, ?, 0, 0, ?)`, groupID, imageID, "legacy", "[]")
	if err != nil {
		t.Fatalf("insert duplicate_relation: %v", err)
	}
}

func TestDuplicateService_DetectDuplicates_SidecarFlowSuccess(t *testing.T) {
	db := setupDuplicateTestDB(t)
	ids := insertTestImages(t, db)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/compute/duplicates/detect":
			_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-1", "status": "pending", "progress": 0})
		case "/compute/duplicates/tasks/task-1":
			_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-1", "status": "completed", "progress": 100})
		case "/compute/duplicates/tasks/task-1/result":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"groups": []map[string]any{{
					"group_id":       0,
					"recommended_id": ids[1],
					"members": []map[string]any{
						{"image_id": ids[0], "sha256": "sha-a", "phash": "phash-a", "distance": 3, "is_recommended": false, "recommendation_score": 70.5, "recommendation_reasons": []map[string]any{{"factor": "resolution", "value": "100x120", "score": 10.0, "weight": 0.5}}},
						{"image_id": ids[1], "sha256": "sha-b", "phash": "phash-b", "distance": 0, "is_recommended": true, "recommendation_score": 90.0, "recommendation_reasons": []map[string]any{{"factor": "resolution", "value": "101x121", "score": 20.0, "weight": 0.5}}},
					},
				}},
				"total_images":        3,
				"total_groups":        1,
				"skipped_images":      []map[string]any{},
				"computation_time_ms": 123,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	imageRepo := repository.NewImageRepository(db)
	duplicateRepo := repository.NewDuplicateRepository(db)
	svc := NewDuplicateService(imageRepo, duplicateRepo, sidecar.NewSidecarClient(server.URL), nil)

	count, err := svc.DetectDuplicates(context.Background(), DetectOptions{})
	if err != nil {
		t.Fatalf("DetectDuplicates() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("groups count = %d, want 1", count)
	}

	img, err := imageRepo.FindByID(ids[0])
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if img.PHashHex != "phash-a" {
		t.Fatalf("PHashHex = %q, want phash-a", img.PHashHex)
	}

	groups, total, err := svc.GetDuplicateGroups(10, 0)
	if err != nil {
		t.Fatalf("GetDuplicateGroups() error = %v", err)
	}
	if total != 1 || len(groups) != 1 {
		t.Fatalf("groups total/list = %d/%d, want 1/1", total, len(groups))
	}
	if len(groups[0].Images) != 2 {
		t.Fatalf("images in group = %d, want 2", len(groups[0].Images))
	}
	if string(groups[0].Images[0].RecommendationRationale) == "" {
		t.Fatal("recommendation_rationale should be structured json")
	}
}

func TestDuplicateService_DetectDuplicates_LogsLifecycle(t *testing.T) {
	db := setupDuplicateTestDB(t)
	ids := insertTestImages(t, db)

	pollCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/compute/duplicates/detect":
			_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-log", "status": "pending", "progress": 0})
		case "/compute/duplicates/tasks/task-log":
			pollCount++
			if pollCount == 1 {
				_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-log", "status": "running", "progress": 25, "message": "hashing"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-log", "status": "completed", "progress": 100, "message": "completed"})
		case "/compute/duplicates/tasks/task-log/result":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"groups": []map[string]any{{
					"group_id":       0,
					"recommended_id": ids[1],
					"members": []map[string]any{{
						"image_id":               ids[1],
						"sha256":                 "sha-b",
						"phash":                  "phash-b",
						"distance":               0,
						"is_recommended":         true,
						"recommendation_score":   90.0,
						"recommendation_reasons": []map[string]any{},
					}},
				}},
				"total_images":        3,
				"total_groups":        1,
				"skipped_images":      []map[string]any{{"path": "/test/c.jpg", "error": "broken"}},
				"computation_time_ms": 123,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	buf, restoreLogs := captureStandardLogs(t)
	defer restoreLogs()

	svc := NewDuplicateService(repository.NewImageRepository(db), repository.NewDuplicateRepository(db), sidecar.NewSidecarClient(server.URL), nil)
	if _, err := svc.DetectDuplicates(context.Background(), DetectOptions{Threshold: 40}); err != nil {
		t.Fatalf("DetectDuplicates() error = %v", err)
	}

	output := buf.String()
	t.Logf("captured duplicate detection logs:\n%s", output)
	for _, want := range []string{
		"duplicate detection started: threshold=40 image_count=3",
		"duplicate detection task submitted: task_id=task-log",
		"duplicate detection status: task_id=task-log status=running progress=25.0 message=hashing",
		"duplicate detection status: task_id=task-log status=completed progress=100.0 message=completed",
		"duplicate detection result fetched: task_id=task-log total_images=3 total_groups=1 skipped_images=1 computation_time_ms=123",
		"duplicate detection persisted: task_id=task-log total_groups=1",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("log output = %q, want contains %q", output, want)
		}
	}
}

func TestDuplicateService_DetectDuplicates_LogsFailure(t *testing.T) {
	db := setupDuplicateTestDB(t)
	_ = insertTestImages(t, db)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/compute/duplicates/detect":
			_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-fail", "status": "pending", "progress": 0})
		case "/compute/duplicates/tasks/task-fail":
			_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-fail", "status": "failed", "progress": 60, "message": "python error"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	buf, restoreLogs := captureStandardLogs(t)
	defer restoreLogs()

	svc := NewDuplicateService(repository.NewImageRepository(db), repository.NewDuplicateRepository(db), sidecar.NewSidecarClient(server.URL), nil)
	if _, err := svc.DetectDuplicates(context.Background(), DetectOptions{Threshold: 40}); err == nil {
		t.Fatal("DetectDuplicates() error = nil, want error")
	}

	output := buf.String()
	for _, want := range []string{
		"duplicate detection task submitted: task_id=task-fail",
		"duplicate detection status: task_id=task-fail status=failed progress=60.0 message=python error",
		"duplicate detection failed: task_id=task-fail message=python error",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("log output = %q, want contains %q", output, want)
		}
	}
}

func TestDuplicateService_DetectDuplicates_DoesNotDeleteOldRowsOnSidecarFailures(t *testing.T) {
	tests := []struct {
		name       string
		statusPath string
		statusCode int
	}{
		{name: "submit-failed", statusPath: "/compute/duplicates/detect", statusCode: http.StatusInternalServerError},
		{name: "poll-failed", statusPath: "/compute/duplicates/tasks/task-1", statusCode: http.StatusNotFound},
		{name: "fetch-failed", statusPath: "/compute/duplicates/tasks/task-1/result", statusCode: http.StatusBadRequest},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db := setupDuplicateTestDB(t)
			ids := insertTestImages(t, db)
			insertOldDuplicateGroup(t, db, ids[0])

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == tc.statusPath {
					w.WriteHeader(tc.statusCode)
					_, _ = w.Write([]byte(`{"detail":"boom"}`))
					return
				}
				switch r.URL.Path {
				case "/compute/duplicates/detect":
					_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-1", "status": "pending", "progress": 0})
				case "/compute/duplicates/tasks/task-1":
					_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-1", "status": "completed", "progress": 100})
				case "/compute/duplicates/tasks/task-1/result":
					_ = json.NewEncoder(w).Encode(map[string]any{"groups": []any{}, "total_images": 3, "total_groups": 0, "skipped_images": []any{}, "computation_time_ms": 10})
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			svc := NewDuplicateService(repository.NewImageRepository(db), repository.NewDuplicateRepository(db), sidecar.NewSidecarClient(server.URL), nil)
			if _, err := svc.DetectDuplicates(context.Background(), DetectOptions{Threshold: 40}); err == nil {
				t.Fatal("DetectDuplicates() error = nil, want error")
			}

			var groupCount int64
			if err := db.QueryRow(`SELECT COUNT(*) FROM duplicate_groups`).Scan(&groupCount); err != nil {
				t.Fatalf("count duplicate_groups: %v", err)
			}
			if groupCount == 0 {
				t.Fatal("expected old duplicate rows to remain when sidecar flow fails")
			}
		})
	}
}

func TestDuplicateService_DetectDuplicates_SidecarTaskFailed(t *testing.T) {
	db := setupDuplicateTestDB(t)
	_ = insertTestImages(t, db)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/compute/duplicates/detect":
			_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-1", "status": "pending", "progress": 0})
		case "/compute/duplicates/tasks/task-1":
			_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-1", "status": "failed", "progress": 100, "message": "python error"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	svc := NewDuplicateService(repository.NewImageRepository(db), repository.NewDuplicateRepository(db), sidecar.NewSidecarClient(server.URL), nil)
	if _, err := svc.DetectDuplicates(context.Background(), DetectOptions{Threshold: 40}); err == nil {
		t.Fatal("DetectDuplicates() error = nil, want error")
	}
}

func TestDuplicateService_DeleteDuplicateGroup(t *testing.T) {
	db := setupDuplicateTestDB(t)
	ids := insertTestImages(t, db)
	insertOldDuplicateGroup(t, db, ids[0])

	duplicateRepo := repository.NewDuplicateRepository(db)
	svc := NewDuplicateService(repository.NewImageRepository(db), duplicateRepo, nil, nil)

	groups, err := duplicateRepo.FindDuplicateGroups(10, 0)
	if err != nil {
		t.Fatalf("FindDuplicateGroups() error = %v", err)
	}
	if len(groups) == 0 {
		t.Fatal("expected one group")
	}

	if err := svc.DeleteDuplicateGroup(groups[0].ID); err != nil {
		t.Fatalf("DeleteDuplicateGroup() error = %v", err)
	}
}
