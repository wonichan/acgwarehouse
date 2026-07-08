package router_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"golang.org/x/crypto/bcrypt"

	"github.com/yachiyo/acgwarehouse/internal/handler"
	"github.com/yachiyo/acgwarehouse/internal/handler/router"
	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/ports"
	"github.com/yachiyo/acgwarehouse/internal/service"
	jwtpkg "github.com/yachiyo/acgwarehouse/pkg/jwt"
)

type memoryRouterUserRepository struct {
	nextID int64
	byID   map[int64]do.User
	byName map[string]do.User
}

func newMemoryRouterUserRepository() *memoryRouterUserRepository {
	return &memoryRouterUserRepository{nextID: 1, byID: make(map[int64]do.User), byName: make(map[string]do.User)}
}

func (r *memoryRouterUserRepository) FindByUsername(_ context.Context, username string) (do.User, error) {
	user, ok := r.byName[username]
	if !ok {
		return do.User{}, service.ErrUserNotFound
	}
	return user, nil
}

func (r *memoryRouterUserRepository) FindByID(_ context.Context, id int64) (do.User, error) {
	user, ok := r.byID[id]
	if !ok {
		return do.User{}, service.ErrUserNotFound
	}
	return user, nil
}

func (r *memoryRouterUserRepository) Create(_ context.Context, user do.User) (do.User, error) {
	if _, ok := r.byName[user.Username]; ok {
		return do.User{}, service.ErrUsernameExists
	}
	user.ID = r.nextID
	r.nextID++
	r.byID[user.ID] = user
	r.byName[user.Username] = user
	return user, nil
}

func (r *memoryRouterUserRepository) UpdateProfile(_ context.Context, user do.User) (do.User, error) {
	stored, ok := r.byID[user.ID]
	if !ok {
		return do.User{}, service.ErrUserNotFound
	}
	stored.Nickname = user.Nickname
	stored.FavoriteTags = user.FavoriteTags
	stored.Bio = user.Bio
	stored.PublicProfile = user.PublicProfile
	stored.EmailNotifications = user.EmailNotifications
	stored.SyncCollections = user.SyncCollections
	r.byID[stored.ID] = stored
	r.byName[stored.Username] = stored
	return stored, nil
}

func (r *memoryRouterUserRepository) UpdatePasswordHash(_ context.Context, userID int64, passwordHash string) error {
	stored, ok := r.byID[userID]
	if !ok {
		return service.ErrUserNotFound
	}
	stored.PasswordHash = passwordHash
	r.byID[stored.ID] = stored
	r.byName[stored.Username] = stored
	return nil
}

type memoryRouterCheckInRepository struct {
	records map[string]bool
}

func newMemoryRouterCheckInRepository() *memoryRouterCheckInRepository {
	return &memoryRouterCheckInRepository{records: make(map[string]bool)}
}

func (r *memoryRouterCheckInRepository) CheckInToday(_ context.Context, userID int64, date string, _ int) (bool, error) {
	key := fmt.Sprintf("%d|%s", userID, date)
	if r.records[key] {
		return false, nil
	}
	r.records[key] = true
	return true, nil
}

func (r *memoryRouterCheckInRepository) ListByMonth(_ context.Context, _ int64, _ int, _ int) ([]do.CheckIn, error) {
	return nil, nil
}

var _ ports.CheckInRepository = (*memoryRouterCheckInRepository)(nil)

func newRouterCheckInService(userRepo *memoryRouterUserRepository) *service.CheckInService {
	return service.NewCheckInService(newMemoryRouterCheckInRepository(), userRepo)
}

func Test_UserRoute_returns_current_user_profile_when_authenticated(t *testing.T) {
	// Given
	repo := newMemoryRouterUserRepository()
	repo.nextID = 42
	svc := service.NewUserService(repo, jwtpkg.NewManager("test-secret", time.Hour))
	if _, err := svc.Register(context.Background(), do.User{Username: "alice", Password: "secret1"}); err != nil {
		t.Fatalf("register user: %v", err)
	}
	engine := routerTestEngineWithServices(t, router.Services{User: svc, CheckIn: newRouterCheckInService(repo)})

	// When
	recorder := ut.PerformRequest(
		engine.Engine.Engine,
		consts.MethodGet,
		"/api/v1/users/me",
		nil,
		authHeader(signRouterToken(t)),
	)

	// Then
	if recorder.Code != consts.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", recorder.Code, recorder.Body.String())
	}
	var response handler.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data, ok := response.Data.(map[string]interface{})
	if !ok || data["nickname"] != "alice" || data["public_profile"] != true {
		t.Fatalf("response data = %#v, want expanded user profile", response.Data)
	}
}

func Test_UserRoute_updates_profile_when_authenticated(t *testing.T) {
	// Given
	repo := newMemoryRouterUserRepository()
	repo.nextID = 42
	svc := service.NewUserService(repo, jwtpkg.NewManager("test-secret", time.Hour))
	if _, err := svc.Register(context.Background(), do.User{Username: "alice", Password: "secret1"}); err != nil {
		t.Fatalf("register user: %v", err)
	}
	engine := routerTestEngineWithServices(t, router.Services{User: svc, CheckIn: newRouterCheckInService(repo)})
	bodyText := `{"nickname":"Alice","favorite_tags":"雨景","bio":"收藏整理","public_profile":true,"email_notifications":false,"sync_collections":true}`
	body := &ut.Body{Body: strings.NewReader(bodyText), Len: len(bodyText)}

	// When
	recorder := ut.PerformRequest(
		engine.Engine.Engine,
		consts.MethodPut,
		"/api/v1/users/me",
		body,
		jsonHeader(),
		authHeader(signRouterToken(t)),
	)

	// Then
	if recorder.Code != consts.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", recorder.Code, recorder.Body.String())
	}
	stored, err := repo.FindByID(context.Background(), 42)
	if err != nil {
		t.Fatalf("find updated user: %v", err)
	}
	if stored.Nickname != "Alice" || stored.EmailNotifications {
		t.Fatalf("stored user = %#v, want profile update", stored)
	}
}

func Test_UserRoute_updates_profile_when_cjk_values_reach_character_limit(t *testing.T) {
	// Given
	repo := newMemoryRouterUserRepository()
	repo.nextID = 42
	svc := service.NewUserService(repo, jwtpkg.NewManager("test-secret", time.Hour))
	if _, err := svc.Register(context.Background(), do.User{Username: "alice", Password: "secret1"}); err != nil {
		t.Fatalf("register user: %v", err)
	}
	engine := routerTestEngineWithServices(t, router.Services{User: svc, CheckIn: newRouterCheckInService(repo)})
	bodyText := `{"nickname":"画师收藏观察记录员甲乙丙丁戊己庚辛壬癸","favorite_tags":"雨景, 制服","bio":"收藏整理","public_profile":true,"email_notifications":false,"sync_collections":true}`
	body := &ut.Body{Body: strings.NewReader(bodyText), Len: len(bodyText)}

	// When
	recorder := ut.PerformRequest(
		engine.Engine.Engine,
		consts.MethodPut,
		"/api/v1/users/me",
		body,
		jsonHeader(),
		authHeader(signRouterToken(t)),
	)

	// Then
	if recorder.Code != consts.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", recorder.Code, recorder.Body.String())
	}
}

func Test_UserRoute_changes_password_when_old_password_matches(t *testing.T) {
	// Given
	repo := newMemoryRouterUserRepository()
	repo.nextID = 42
	svc := service.NewUserService(repo, jwtpkg.NewManager("test-secret", time.Hour))
	if _, err := svc.Register(context.Background(), do.User{Username: "alice", Password: "secret1"}); err != nil {
		t.Fatalf("register user: %v", err)
	}
	engine := routerTestEngineWithServices(t, router.Services{User: svc, CheckIn: newRouterCheckInService(repo)})
	bodyText := `{"old_password":"secret1","new_password":"secret2"}`
	body := &ut.Body{Body: strings.NewReader(bodyText), Len: len(bodyText)}

	// When
	recorder := ut.PerformRequest(
		engine.Engine.Engine,
		consts.MethodPut,
		"/api/v1/users/password",
		body,
		jsonHeader(),
		authHeader(signRouterToken(t)),
	)

	// Then
	if recorder.Code != consts.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", recorder.Code, recorder.Body.String())
	}
	stored, err := repo.FindByID(context.Background(), 42)
	if err != nil {
		t.Fatalf("find updated user: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("secret2")); err != nil {
		t.Fatalf("compare new password hash: %v", err)
	}
}

func Test_UserRoute_returns_unauthorized_when_password_old_password_mismatches(t *testing.T) {
	// Given
	repo := newMemoryRouterUserRepository()
	repo.nextID = 42
	svc := service.NewUserService(repo, jwtpkg.NewManager("test-secret", time.Hour))
	if _, err := svc.Register(context.Background(), do.User{Username: "alice", Password: "secret1"}); err != nil {
		t.Fatalf("register user: %v", err)
	}
	engine := routerTestEngineWithServices(t, router.Services{User: svc, CheckIn: newRouterCheckInService(repo)})
	bodyText := `{"old_password":"wrong1","new_password":"secret2"}`
	body := &ut.Body{Body: strings.NewReader(bodyText), Len: len(bodyText)}

	// When
	recorder := ut.PerformRequest(
		engine.Engine.Engine,
		consts.MethodPut,
		"/api/v1/users/password",
		body,
		jsonHeader(),
		authHeader(signRouterToken(t)),
	)

	// Then
	if recorder.Code != consts.StatusUnauthorized {
		t.Fatalf("status = %d body=%s, want 401", recorder.Code, recorder.Body.String())
	}
}

func Test_UserRoute_returns_bad_request_when_new_password_too_short(t *testing.T) {
	// Given
	repo := newMemoryRouterUserRepository()
	repo.nextID = 42
	svc := service.NewUserService(repo, jwtpkg.NewManager("test-secret", time.Hour))
	if _, err := svc.Register(context.Background(), do.User{Username: "alice", Password: "secret1"}); err != nil {
		t.Fatalf("register user: %v", err)
	}
	engine := routerTestEngineWithServices(t, router.Services{User: svc, CheckIn: newRouterCheckInService(repo)})
	bodyText := `{"old_password":"secret1","new_password":"short"}`
	body := &ut.Body{Body: strings.NewReader(bodyText), Len: len(bodyText)}

	// When
	recorder := ut.PerformRequest(
		engine.Engine.Engine,
		consts.MethodPut,
		"/api/v1/users/password",
		body,
		jsonHeader(),
		authHeader(signRouterToken(t)),
	)

	// Then
	if recorder.Code != consts.StatusBadRequest {
		t.Fatalf("status = %d body=%s, want 400", recorder.Code, recorder.Body.String())
	}
}
