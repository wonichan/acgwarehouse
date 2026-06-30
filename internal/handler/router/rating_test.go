package router_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/yachiyo/acgwarehouse/internal/handler"
	"github.com/yachiyo/acgwarehouse/internal/handler/router"
	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/service"
	jwtpkg "github.com/yachiyo/acgwarehouse/pkg/jwt"
)

type memoryRouterRatingRepository struct {
	ratings []do.Rating
}

type memoryRouterCollectionRepository struct {
	collections map[int64]do.Collection
	items       []do.CollectionItem
	nextID      int64
}

func newMemoryRouterCollectionRepository() *memoryRouterCollectionRepository {
	return &memoryRouterCollectionRepository{collections: make(map[int64]do.Collection), nextID: 1}
}

func (r *memoryRouterRatingRepository) Upsert(_ context.Context, rating do.Rating) (do.Rating, error) {
	r.ratings = append(r.ratings, rating)
	return rating, nil
}

func (r *memoryRouterCollectionRepository) Create(_ context.Context, collection do.Collection) (do.Collection, error) {
	collection.ID = r.nextID
	r.nextID++
	collection.CreatedAt = time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	r.collections[collection.ID] = collection
	return collection, nil
}

func (r *memoryRouterCollectionRepository) ListByOwner(_ context.Context, userID int64) ([]do.Collection, error) {
	collections := make([]do.Collection, 0)
	for _, collection := range r.collections {
		if collection.UserID == userID {
			collections = append(collections, collection)
		}
	}
	return collections, nil
}

func (r *memoryRouterCollectionRepository) FindVisible(_ context.Context, id int64, viewerID int64) (do.Collection, error) {
	collection, ok := r.collections[id]
	if !ok {
		return do.Collection{}, service.ErrCollectionNotFound
	}
	if collection.UserID != viewerID && collection.Visibility != do.CollectionVisibilityPublic {
		return do.Collection{}, service.ErrForbidden
	}
	return collection, nil
}

func (r *memoryRouterCollectionRepository) Update(_ context.Context, collection do.Collection) (do.Collection, error) {
	stored, ok := r.collections[collection.ID]
	if !ok {
		return do.Collection{}, service.ErrCollectionNotFound
	}
	if stored.UserID != collection.UserID {
		return do.Collection{}, service.ErrForbidden
	}
	stored.Name = collection.Name
	stored.Visibility = collection.Visibility
	r.collections[collection.ID] = stored
	return stored, nil
}

func (r *memoryRouterCollectionRepository) Delete(_ context.Context, id int64, userID int64) error {
	stored, ok := r.collections[id]
	if !ok {
		return service.ErrCollectionNotFound
	}
	if stored.UserID != userID {
		return service.ErrForbidden
	}
	delete(r.collections, id)
	return nil
}

func (r *memoryRouterCollectionRepository) AddItem(
	_ context.Context,
	collectionID int64,
	userID int64,
	imageID int64,
) (do.CollectionItem, error) {
	stored, ok := r.collections[collectionID]
	if !ok {
		return do.CollectionItem{}, service.ErrCollectionNotFound
	}
	if stored.UserID != userID {
		return do.CollectionItem{}, service.ErrForbidden
	}
	item := do.CollectionItem{CollectionID: collectionID, ImageID: imageID, CreatedAt: time.Now().UTC()}
	r.items = append(r.items, item)
	return item, nil
}

func (r *memoryRouterCollectionRepository) RemoveItem(_ context.Context, collectionID int64, userID int64, _ int64) error {
	stored, ok := r.collections[collectionID]
	if !ok {
		return service.ErrCollectionNotFound
	}
	if stored.UserID != userID {
		return service.ErrForbidden
	}
	return nil
}

func Test_RatingRoute_requires_auth_when_token_missing(t *testing.T) {
	// Given
	engine := routerTestEngine(t)
	body := &ut.Body{Body: strings.NewReader(`{"score":80}`), Len: len(`{"score":80}`)}

	// When
	recorder := ut.PerformRequest(engine.Engine.Engine, consts.MethodPut, "/api/v1/images/7/rating", body, jsonHeader())

	// Then
	if recorder.Code != consts.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
}

func Test_RatingRoute_returns_bad_request_when_score_invalid(t *testing.T) {
	// Given
	engine := routerTestEngine(t)
	body := &ut.Body{Body: strings.NewReader(`{"score":101}`), Len: len(`{"score":101}`)}
	token := signRouterToken(t)

	// When
	recorder := ut.PerformRequest(
		engine.Engine.Engine,
		consts.MethodPut,
		"/api/v1/images/7/rating",
		body,
		jsonHeader(),
		authHeader(token),
	)

	// Then
	if recorder.Code != consts.StatusBadRequest {
		t.Fatalf("status = %d body=%s, want 400", recorder.Code, recorder.Body.String())
	}
}

func Test_RatingRoute_upserts_rating_when_authenticated(t *testing.T) {
	// Given
	ratingRepo := &memoryRouterRatingRepository{}
	engine := routerTestEngineWithServices(t, router.Services{Rating: service.NewRatingService(ratingRepo)})
	body := &ut.Body{Body: strings.NewReader(`{"score":80}`), Len: len(`{"score":80}`)}
	token := signRouterToken(t)

	// When
	recorder := ut.PerformRequest(
		engine.Engine.Engine,
		consts.MethodPut,
		"/api/v1/images/7/rating",
		body,
		jsonHeader(),
		authHeader(token),
	)

	// Then
	if recorder.Code != consts.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", recorder.Code, recorder.Body.String())
	}
	if len(ratingRepo.ratings) != 1 || ratingRepo.ratings[0].UserID != 42 || ratingRepo.ratings[0].ImageID != 7 {
		t.Fatalf("ratings = %#v, want one rating from user 42 for image 7", ratingRepo.ratings)
	}
	var response handler.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data, ok := response.Data.(map[string]interface{})
	if !ok || data["score"].(float64) != 80 {
		t.Fatalf("response = %#v, want score response", response)
	}
}

func Test_CollectionRoute_creates_collection_when_authenticated(t *testing.T) {
	// Given
	collectionRepo := newMemoryRouterCollectionRepository()
	engine := routerTestEngineWithServices(t, router.Services{Collection: service.NewCollectionService(collectionRepo, "")})
	body := &ut.Body{Body: strings.NewReader(`{"name":"miku","visibility":"public"}`), Len: len(`{"name":"miku","visibility":"public"}`)}

	// When
	recorder := ut.PerformRequest(
		engine.Engine.Engine,
		consts.MethodPost,
		"/api/v1/collections",
		body,
		jsonHeader(),
		authHeader(signRouterToken(t)),
	)

	// Then
	if recorder.Code != consts.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", recorder.Code, recorder.Body.String())
	}
	if len(collectionRepo.collections) != 1 || collectionRepo.collections[1].UserID != 42 {
		t.Fatalf("collections = %#v, want one collection owned by user 42", collectionRepo.collections)
	}
}

func Test_CollectionRoute_returns_forbidden_when_non_owner_adds_item(t *testing.T) {
	// Given
	collectionRepo := newMemoryRouterCollectionRepository()
	collectionRepo.collections[9] = do.Collection{ID: 9, UserID: 7, Name: "owner", Visibility: do.CollectionVisibilityPrivate}
	engine := routerTestEngineWithServices(t, router.Services{Collection: service.NewCollectionService(collectionRepo, "")})
	body := &ut.Body{Body: strings.NewReader(`{"image_id":5}`), Len: len(`{"image_id":5}`)}

	// When
	recorder := ut.PerformRequest(
		engine.Engine.Engine,
		consts.MethodPost,
		"/api/v1/collections/9/items",
		body,
		jsonHeader(),
		authHeader(signRouterToken(t)),
	)

	// Then
	if recorder.Code != consts.StatusForbidden {
		t.Fatalf("status = %d body=%s, want 403", recorder.Code, recorder.Body.String())
	}
}

func Test_CollectionRoute_allows_guest_to_view_public_collection(t *testing.T) {
	// Given
	collectionRepo := newMemoryRouterCollectionRepository()
	collectionRepo.collections[3] = do.Collection{ID: 3, UserID: 7, Name: "public", Visibility: do.CollectionVisibilityPublic}
	engine := routerTestEngineWithServices(t, router.Services{Collection: service.NewCollectionService(collectionRepo, "")})

	// When
	recorder := ut.PerformRequest(engine.Engine.Engine, consts.MethodGet, "/api/v1/collections/3", nil)

	// Then
	if recorder.Code != consts.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", recorder.Code, recorder.Body.String())
	}
}

func routerTestEngine(t *testing.T) *routerTestHarness {
	t.Helper()
	return routerTestEngineWithServices(t, router.Services{Rating: service.NewRatingService(&memoryRouterRatingRepository{})})
}

func routerTestEngineWithServices(t *testing.T, services router.Services) *routerTestHarness {
	t.Helper()
	jwtManager := jwtpkg.NewManager("test-secret", time.Hour)
	engine := server.Default()
	router.Register(engine, services, jwtManager)
	return &routerTestHarness{Engine: engine}
}

type routerTestHarness struct {
	Engine *server.Hertz
}

func signRouterToken(t *testing.T) string {
	t.Helper()
	token, err := jwtpkg.NewManager("test-secret", time.Hour).Sign(jwtpkg.Claims{
		UserID:   42,
		Username: "alice",
		Role:     string(do.UserRoleUser),
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

func jsonHeader() ut.Header {
	return ut.Header{Key: "Content-Type", Value: "application/json"}
}

func authHeader(token string) ut.Header {
	return ut.Header{Key: "Authorization", Value: "Bearer " + token}
}
