package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

type batchHandlerTestEnv struct {
	router         *gin.Engine
	imageRepo      repository.ImageRepository
	tagRepo        repository.TagRepository
	collectionRepo repository.CollectionRepository
	imageTagRepo   repository.ImageTagRepository
}

func setupBatchHandlerTest(t *testing.T) *batchHandlerTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	tmpFile, err := os.CreateTemp("", "batch_handler_test_*.db")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
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

	imageRepo := repository.NewImageRepository(db)
	tagRepo := repository.NewTagRepository(db)
	imageTagRepo := repository.NewImageTagRepository(db)
	collectionRepo := repository.NewCollectionRepository(db)

	batchSvc := service.NewBatchService(imageRepo, tagRepo, imageTagRepo, collectionRepo)
	h := NewBatchHandler(batchSvc)

	r := gin.New()
	api := r.Group("/api/v1/batch")
	api.POST("/tags/add", h.BatchAddTags)
	api.POST("/tags/remove", h.BatchRemoveTags)
	api.POST("/collections/move", h.BatchMoveToCollection)
	api.POST("/collections/remove", h.BatchRemoveFromCollection)
	api.POST("/images/delete", h.BatchDeleteImages)

	return &batchHandlerTestEnv{
		router:         r,
		imageRepo:      imageRepo,
		tagRepo:        tagRepo,
		collectionRepo: collectionRepo,
		imageTagRepo:   imageTagRepo,
	}
}

func saveBatchHandlerImage(t *testing.T, imageRepo repository.ImageRepository, filename string) *domain.Image {
	t.Helper()
	now := time.Now()
	image := &domain.Image{
		Path:       "/batch-handler/" + filename,
		Filename:   filename,
		SourceRoot: "/batch-handler",
		FileSize:   120,
		Width:      100,
		Height:     100,
		Format:     "png",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if _, err := imageRepo.SaveImage(image); err != nil {
		t.Fatalf("save image: %v", err)
	}
	return image
}

func saveBatchHandlerTag(t *testing.T, tagRepo repository.TagRepository, label string) *domain.Tag {
	t.Helper()
	tag := &domain.Tag{PreferredLabel: label, Slug: label, ReviewState: "confirmed"}
	if err := tagRepo.Save(context.Background(), tag); err != nil {
		t.Fatalf("save tag: %v", err)
	}
	return tag
}

func TestBatchHandler_TagEndpoints(t *testing.T) {
	t.Parallel()
	env := setupBatchHandlerTest(t)
	ctx := context.Background()

	img1 := saveBatchHandlerImage(t, env.imageRepo, "a.png")
	img2 := saveBatchHandlerImage(t, env.imageRepo, "b.png")
	tag := saveBatchHandlerTag(t, env.tagRepo, "hero")

	addReq := bytes.NewBufferString(`{"image_ids":[` + int64ToString(img1.ID) + `,` + int64ToString(img2.ID) + `],"tag_ids":[` + int64ToString(tag.ID) + `]}`)
	addResp := httptest.NewRecorder()
	env.router.ServeHTTP(addResp, httptest.NewRequest(http.MethodPost, "/api/v1/batch/tags/add", addReq))
	if addResp.Code != http.StatusOK {
		t.Fatalf("POST /batch/tags/add status = %d, body=%s", addResp.Code, addResp.Body.String())
	}

	img1Tags, err := env.imageTagRepo.FindByImageID(ctx, img1.ID)
	if err != nil {
		t.Fatalf("FindByImageID error: %v", err)
	}
	if len(img1Tags) != 1 {
		t.Fatalf("img1 tag count = %d, want 1", len(img1Tags))
	}

	removeReq := bytes.NewBufferString(`{"image_ids":[` + int64ToString(img1.ID) + `],"tag_ids":[` + int64ToString(tag.ID) + `]}`)
	removeResp := httptest.NewRecorder()
	env.router.ServeHTTP(removeResp, httptest.NewRequest(http.MethodPost, "/api/v1/batch/tags/remove", removeReq))
	if removeResp.Code != http.StatusOK {
		t.Fatalf("POST /batch/tags/remove status = %d, body=%s", removeResp.Code, removeResp.Body.String())
	}

	img1Tags, err = env.imageTagRepo.FindByImageID(ctx, img1.ID)
	if err != nil {
		t.Fatalf("FindByImageID after remove error: %v", err)
	}
	if len(img1Tags) != 0 {
		t.Fatalf("img1 tags after remove = %d, want 0", len(img1Tags))
	}
}

func TestBatchHandler_MoveAndDeleteEndpoints(t *testing.T) {
	t.Parallel()
	env := setupBatchHandlerTest(t)
	ctx := context.Background()

	collection := &domain.Collection{Name: "batch-target"}
	if err := env.collectionRepo.Save(ctx, collection); err != nil {
		t.Fatalf("save collection: %v", err)
	}

	img1 := saveBatchHandlerImage(t, env.imageRepo, "a.png")
	img2 := saveBatchHandlerImage(t, env.imageRepo, "b.png")

	moveReq := bytes.NewBufferString(`{"image_ids":[` + int64ToString(img1.ID) + `,` + int64ToString(img2.ID) + `],"collection_id":` + int64ToString(collection.ID) + `}`)
	moveResp := httptest.NewRecorder()
	env.router.ServeHTTP(moveResp, httptest.NewRequest(http.MethodPost, "/api/v1/batch/collections/move", moveReq))
	if moveResp.Code != http.StatusOK {
		t.Fatalf("POST /batch/collections/move status = %d, body=%s", moveResp.Code, moveResp.Body.String())
	}

	updated, err := env.collectionRepo.FindByID(ctx, collection.ID)
	if err != nil {
		t.Fatalf("FindByID collection error: %v", err)
	}
	if updated.ImageCount != 2 {
		t.Fatalf("ImageCount after move = %d, want 2", updated.ImageCount)
	}

	deleteReq := bytes.NewBufferString(`{"image_ids":[` + int64ToString(img1.ID) + `]}`)
	deleteResp := httptest.NewRecorder()
	env.router.ServeHTTP(deleteResp, httptest.NewRequest(http.MethodPost, "/api/v1/batch/images/delete", deleteReq))
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("POST /batch/images/delete status = %d, body=%s", deleteResp.Code, deleteResp.Body.String())
	}

	var deleteBody map[string]interface{}
	if err := json.Unmarshal(deleteResp.Body.Bytes(), &deleteBody); err != nil {
		t.Fatalf("unmarshal delete response: %v", err)
	}
	if deleteBody["images_deleted"].(float64) != 1 {
		t.Fatalf("images_deleted = %v, want 1", deleteBody["images_deleted"])
	}
}
