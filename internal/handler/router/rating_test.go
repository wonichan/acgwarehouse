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

func (r *memoryRouterRatingRepository) Upsert(_ context.Context, rating do.Rating) (do.Rating, error) {
	r.ratings = append(r.ratings, rating)
	return rating, nil
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
	engine := routerTestEngineWithRating(t, service.NewRatingService(ratingRepo))
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

func routerTestEngine(t *testing.T) *routerTestHarness {
	t.Helper()
	return routerTestEngineWithRating(t, service.NewRatingService(&memoryRouterRatingRepository{}))
}

func routerTestEngineWithRating(t *testing.T, ratingService *service.RatingService) *routerTestHarness {
	t.Helper()
	jwtManager := jwtpkg.NewManager("test-secret", time.Hour)
	engine := server.Default()
	router.Register(engine, router.Services{Rating: ratingService}, jwtManager)
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
