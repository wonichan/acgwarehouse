package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

type imageMoveHandlerTestEnv struct {
	router       *gin.Engine
	imageRepo    repository.ImageRepository
	tagRepo      repository.TagRepository
	imageTagRepo repository.ImageTagRepository
	sourceDir    string
	targetDir    string
	tag          *domain.Tag
}

func setupImageMoveHandlerTest(t *testing.T) *imageMoveHandlerTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "image-move-handler.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	sourceDir := filepath.Join(t.TempDir(), "source")
	targetDir := filepath.Join(t.TempDir(), "target")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}

	imageRepo := repository.NewImageRepository(db)
	historyRepo := repository.NewImageMoveHistoryRepository(db)
	tagRepo := repository.NewTagRepository(db)
	imageTagRepo := repository.NewImageTagRepository(db)
	tag := &domain.Tag{PreferredLabel: "target", Slug: "target", ReviewState: "confirmed"}
	if err := tagRepo.Save(context.Background(), tag); err != nil {
		t.Fatalf("save tag: %v", err)
	}

	svc := service.NewImageMoveService(imageRepo, tagRepo, historyRepo, func() *config.Config {
		return &config.Config{Storage: config.StorageConfig{ScanRoots: []string{targetDir}}}
	})
	handler := NewImageMoveHandler(svc)
	router := gin.New()
	api := router.Group("/api/v1")
	api.POST("/image-moves/preview", handler.PreviewMove)
	api.POST("/image-moves/execute", handler.ExecuteMove)
	api.POST("/image-moves/jobs", handler.CreateMoveJob)
	api.GET("/image-moves/jobs/:id", handler.GetMoveJob)
	api.POST("/image-moves/jobs/:id/cancel", handler.CancelMoveJob)
	api.GET("/image-moves/history", handler.ListHistory)

	return &imageMoveHandlerTestEnv{
		router:       router,
		imageRepo:    imageRepo,
		tagRepo:      tagRepo,
		imageTagRepo: imageTagRepo,
		sourceDir:    sourceDir,
		targetDir:    targetDir,
		tag:          tag,
	}
}

func TestImageMoveHandlerPreviewAndExecute(t *testing.T) {
	t.Parallel()
	env := setupImageMoveHandlerTest(t)
	image := env.saveImageWithFile(t, "move.png", "payload")
	env.saveImageTag(t, image.ID)

	reqBody := bytes.NewBufferString(`{"source_dirs":[` + quoteJSON(t, env.sourceDir) + `],"tag_id":` + int64ToString(env.tag.ID) + `,"target_dir":` + quoteJSON(t, env.targetDir) + `}`)
	previewResp := httptest.NewRecorder()
	env.router.ServeHTTP(previewResp, httptest.NewRequest(http.MethodPost, "/api/v1/image-moves/preview", reqBody))
	if previewResp.Code != http.StatusOK {
		t.Fatalf("preview status = %d, body=%s", previewResp.Code, previewResp.Body.String())
	}
	var preview domain.ImageMovePreview
	if err := json.Unmarshal(previewResp.Body.Bytes(), &preview); err != nil {
		t.Fatalf("unmarshal preview: %v", err)
	}
	if preview.Movable != 1 || preview.Skipped != 0 {
		t.Fatalf("preview movable/skipped = %d/%d, want 1/0", preview.Movable, preview.Skipped)
	}

	reqBody = bytes.NewBufferString(`{"source_dirs":[` + quoteJSON(t, env.sourceDir) + `],"tag_id":` + int64ToString(env.tag.ID) + `,"target_dir":` + quoteJSON(t, env.targetDir) + `}`)
	executeResp := httptest.NewRecorder()
	env.router.ServeHTTP(executeResp, httptest.NewRequest(http.MethodPost, "/api/v1/image-moves/execute", reqBody))
	if executeResp.Code != http.StatusOK {
		t.Fatalf("execute status = %d, body=%s", executeResp.Code, executeResp.Body.String())
	}
	var result domain.ImageMoveResult
	if err := json.Unmarshal(executeResp.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.Moved != 1 || result.Failed != 0 {
		t.Fatalf("result moved/failed = %d/%d, want 1/0", result.Moved, result.Failed)
	}
}

func TestImageMoveHandlerValidationAndMissingTag(t *testing.T) {
	t.Parallel()
	env := setupImageMoveHandlerTest(t)

	badResp := httptest.NewRecorder()
	env.router.ServeHTTP(badResp, httptest.NewRequest(http.MethodPost, "/api/v1/image-moves/preview", bytes.NewBufferString(`{"tag_id":0}`)))
	if badResp.Code != http.StatusBadRequest {
		t.Fatalf("bad request status = %d, want %d", badResp.Code, http.StatusBadRequest)
	}

	missingTagBody := bytes.NewBufferString(`{"source_dirs":[` + quoteJSON(t, env.sourceDir) + `],"tag_id":999999,"target_dir":` + quoteJSON(t, env.targetDir) + `}`)
	missingTagResp := httptest.NewRecorder()
	env.router.ServeHTTP(missingTagResp, httptest.NewRequest(http.MethodPost, "/api/v1/image-moves/preview", missingTagBody))
	if missingTagResp.Code != http.StatusNotFound {
		t.Fatalf("missing tag status = %d, want %d", missingTagResp.Code, http.StatusNotFound)
	}
}

func (env *imageMoveHandlerTestEnv) saveImageWithFile(t *testing.T, filename, payload string) *domain.Image {
	t.Helper()
	path := filepath.Join(env.sourceDir, filename)
	if err := os.WriteFile(path, []byte(payload), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	image := &domain.Image{
		Path:       path,
		Filename:   filename,
		SourceRoot: env.sourceDir,
		FileSize:   int64(len(payload)),
		Format:     "png",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if _, err := env.imageRepo.SaveImage(image); err != nil {
		t.Fatalf("save image: %v", err)
	}
	return image
}

func (env *imageMoveHandlerTestEnv) saveImageTag(t *testing.T, imageID int64) {
	t.Helper()
	if err := env.imageTagRepo.Save(context.Background(), &domain.ImageTag{ImageID: imageID, TagID: env.tag.ID, ReviewState: domain.ReviewStateConfirmed}); err != nil {
		t.Fatalf("save image-tag: %v", err)
	}
}

func quoteJSON(t *testing.T, value string) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}
	return string(data)
}
