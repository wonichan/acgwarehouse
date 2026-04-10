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

type collectionHandlerTestEnv struct {
	router         *gin.Engine
	handler        *CollectionHandler
	imageRepo      repository.ImageRepository
	collectionRepo repository.CollectionRepository
}

func setupCollectionHandlerTest(t *testing.T) *collectionHandlerTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	tmpFile, err := os.CreateTemp("", "collection_handler_test_*.db")
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
	collectionRepo := repository.NewCollectionRepository(db)
	collectionSvc := service.NewCollectionService(collectionRepo)
	h := NewCollectionHandler(collectionSvc)

	r := gin.New()
	api := r.Group("/api/v1")
	api.POST("/collections", h.CreateCollection)
	api.GET("/collections", h.ListCollections)
	api.GET("/collections/:id", h.GetCollection)
	api.PUT("/collections/:id", h.UpdateCollection)
	api.DELETE("/collections/:id", h.DeleteCollection)
	api.POST("/collections/:id/images", h.AddImageToCollection)
	api.DELETE("/collections/:id/images/:image_id", h.RemoveImageFromCollection)
	api.PUT("/collections/:id/cover", h.SetCoverImage)

	return &collectionHandlerTestEnv{
		router:         r,
		handler:        h,
		imageRepo:      imageRepo,
		collectionRepo: collectionRepo,
	}
}

func saveCollectionHandlerImage(t *testing.T, imageRepo repository.ImageRepository, filename string) *domain.Image {
	t.Helper()
	now := time.Now()
	image := &domain.Image{
		Path:       "/handler/" + filename,
		Filename:   filename,
		SourceRoot: "/handler",
		FileSize:   100,
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

func TestCollectionHandler_CRUDAndImageOperations(t *testing.T) {
	t.Parallel()
	env := setupCollectionHandlerTest(t)

	createReq := bytes.NewBufferString(`{"name":"Favorites","description":"my picks"}`)
	createResp := httptest.NewRecorder()
	env.router.ServeHTTP(createResp, httptest.NewRequest(http.MethodPost, "/api/v1/collections", createReq))
	if createResp.Code != http.StatusCreated {
		t.Fatalf("POST /collections status = %d, body=%s", createResp.Code, createResp.Body.String())
	}

	var created domain.Collection
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("created collection ID is zero")
	}

	listResp := httptest.NewRecorder()
	env.router.ServeHTTP(listResp, httptest.NewRequest(http.MethodGet, "/api/v1/collections?limit=10&offset=0", nil))
	if listResp.Code != http.StatusOK {
		t.Fatalf("GET /collections status = %d, body=%s", listResp.Code, listResp.Body.String())
	}

	var listBody struct {
		Collections []domain.Collection `json:"collections"`
		Total       int64               `json:"total"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	if listBody.Total != 1 || len(listBody.Collections) != 1 {
		t.Fatalf("list mismatch: total=%d len=%d", listBody.Total, len(listBody.Collections))
	}

	image := saveCollectionHandlerImage(t, env.imageRepo, "one.png")
	addReq := bytes.NewBufferString(`{"image_id":` + int64ToString(image.ID) + `}`)
	addResp := httptest.NewRecorder()
	env.router.ServeHTTP(addResp, httptest.NewRequest(http.MethodPost, "/api/v1/collections/"+int64ToString(created.ID)+"/images", addReq))
	if addResp.Code != http.StatusOK {
		t.Fatalf("POST /collections/:id/images status = %d, body=%s", addResp.Code, addResp.Body.String())
	}

	setCoverReq := bytes.NewBufferString(`{"image_id":` + int64ToString(image.ID) + `}`)
	setCoverResp := httptest.NewRecorder()
	env.router.ServeHTTP(setCoverResp, httptest.NewRequest(http.MethodPut, "/api/v1/collections/"+int64ToString(created.ID)+"/cover", setCoverReq))
	if setCoverResp.Code != http.StatusOK {
		t.Fatalf("PUT /collections/:id/cover status = %d, body=%s", setCoverResp.Code, setCoverResp.Body.String())
	}

	removeResp := httptest.NewRecorder()
	env.router.ServeHTTP(removeResp, httptest.NewRequest(http.MethodDelete, "/api/v1/collections/"+int64ToString(created.ID)+"/images/"+int64ToString(image.ID), nil))
	if removeResp.Code != http.StatusOK {
		t.Fatalf("DELETE /collections/:id/images/:image_id status = %d, body=%s", removeResp.Code, removeResp.Body.String())
	}

	updateReq := bytes.NewBufferString(`{"name":"Updated","description":"updated"}`)
	updateResp := httptest.NewRecorder()
	env.router.ServeHTTP(updateResp, httptest.NewRequest(http.MethodPut, "/api/v1/collections/"+int64ToString(created.ID), updateReq))
	if updateResp.Code != http.StatusOK {
		t.Fatalf("PUT /collections/:id status = %d, body=%s", updateResp.Code, updateResp.Body.String())
	}

	deleteResp := httptest.NewRecorder()
	env.router.ServeHTTP(deleteResp, httptest.NewRequest(http.MethodDelete, "/api/v1/collections/"+int64ToString(created.ID), nil))
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("DELETE /collections/:id status = %d, body=%s", deleteResp.Code, deleteResp.Body.String())
	}
}

func TestCollectionHandler_GetCollection_NotFound(t *testing.T) {
	t.Parallel()
	env := setupCollectionHandlerTest(t)

	resp := httptest.NewRecorder()
	env.router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/collections/999", nil))
	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body=%s", resp.Code, resp.Body.String())
	}
}

func int64ToString(value int64) string {
	if value == 0 {
		return "0"
	}
	digits := make([]byte, 0, 20)
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}

func TestCollectionHandler_ListIncludesImageCount(t *testing.T) {
	t.Parallel()
	env := setupCollectionHandlerTest(t)
	ctx := context.Background()

	collection := &domain.Collection{Name: "Count Test"}
	if err := env.collectionRepo.Save(ctx, collection); err != nil {
		t.Fatalf("save collection: %v", err)
	}

	image := saveCollectionHandlerImage(t, env.imageRepo, "count.png")
	if err := env.collectionRepo.AddImage(ctx, collection.ID, image.ID); err != nil {
		t.Fatalf("AddImage: %v", err)
	}

	resp := httptest.NewRecorder()
	env.router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/collections", nil))
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", resp.Code, resp.Body.String())
	}

	var body struct {
		Collections []domain.Collection `json:"collections"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body.Collections) == 0 || body.Collections[0].ImageCount < 1 {
		t.Fatalf("expected image_count >= 1, got %+v", body.Collections)
	}
}

func TestCollectionHandler_AddImageMovesImageToSingleCollection(t *testing.T) {
	t.Parallel()
	env := setupCollectionHandlerTest(t)
	ctx := context.Background()

	first := &domain.Collection{Name: "First"}
	second := &domain.Collection{Name: "Second"}
	if err := env.collectionRepo.Save(ctx, first); err != nil {
		t.Fatalf("save first collection: %v", err)
	}
	if err := env.collectionRepo.Save(ctx, second); err != nil {
		t.Fatalf("save second collection: %v", err)
	}

	image := saveCollectionHandlerImage(t, env.imageRepo, "move.png")

	addToFirstReq := bytes.NewBufferString(`{"image_id":` + int64ToString(image.ID) + `}`)
	addToFirstResp := httptest.NewRecorder()
	env.router.ServeHTTP(addToFirstResp, httptest.NewRequest(http.MethodPost, "/api/v1/collections/"+int64ToString(first.ID)+"/images", addToFirstReq))
	if addToFirstResp.Code != http.StatusOK {
		t.Fatalf("add to first status = %d, body=%s", addToFirstResp.Code, addToFirstResp.Body.String())
	}

	addToSecondReq := bytes.NewBufferString(`{"image_id":` + int64ToString(image.ID) + `}`)
	addToSecondResp := httptest.NewRecorder()
	env.router.ServeHTTP(addToSecondResp, httptest.NewRequest(http.MethodPost, "/api/v1/collections/"+int64ToString(second.ID)+"/images", addToSecondReq))
	if addToSecondResp.Code != http.StatusOK {
		t.Fatalf("add to second status = %d, body=%s", addToSecondResp.Code, addToSecondResp.Body.String())
	}

	firstReloaded, err := env.collectionRepo.FindByID(ctx, first.ID)
	if err != nil {
		t.Fatalf("reload first collection: %v", err)
	}
	secondReloaded, err := env.collectionRepo.FindByID(ctx, second.ID)
	if err != nil {
		t.Fatalf("reload second collection: %v", err)
	}
	if firstReloaded.ImageCount != 0 {
		t.Fatalf("first collection image_count = %d, want 0", firstReloaded.ImageCount)
	}
	if secondReloaded.ImageCount != 1 {
		t.Fatalf("second collection image_count = %d, want 1", secondReloaded.ImageCount)
	}

	reloadedImage, err := env.imageRepo.FindByID(image.ID)
	if err != nil {
		t.Fatalf("reload image: %v", err)
	}
	if reloadedImage.CollectionID == nil || *reloadedImage.CollectionID != second.ID {
		t.Fatalf("image collection_id = %v, want %d", reloadedImage.CollectionID, second.ID)
	}
}
