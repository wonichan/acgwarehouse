package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/service"
	jwtpkg "github.com/yachiyo/acgwarehouse/pkg/jwt"
)

type memoryUserRepository struct {
	nextID int64
	byID   map[int64]do.User
	byName map[string]do.User
}

func newMemoryUserRepository() *memoryUserRepository {
	return &memoryUserRepository{
		nextID: 1,
		byID:   make(map[int64]do.User),
		byName: make(map[string]do.User),
	}
}

func (r *memoryUserRepository) FindByUsername(_ context.Context, username string) (do.User, error) {
	user, ok := r.byName[username]
	if !ok {
		return do.User{}, service.ErrUserNotFound
	}
	return user, nil
}

func (r *memoryUserRepository) FindByID(_ context.Context, id int64) (do.User, error) {
	user, ok := r.byID[id]
	if !ok {
		return do.User{}, service.ErrUserNotFound
	}
	return user, nil
}

func (r *memoryUserRepository) Create(_ context.Context, user do.User) (do.User, error) {
	if _, ok := r.byName[user.Username]; ok {
		return do.User{}, service.ErrUsernameExists
	}
	user.ID = r.nextID
	r.nextID++
	r.byID[user.ID] = user
	r.byName[user.Username] = user
	return user, nil
}

func Test_UserService_Register_hashes_password_when_input_valid(t *testing.T) {
	// Given
	repo := newMemoryUserRepository()
	svc := service.NewUserService(repo, jwtpkg.NewManager("test-secret", time.Hour))
	input := do.User{Username: "alice", Password: "secret1", Role: do.UserRoleUser}

	// When
	created, err := svc.Register(context.Background(), input)

	// Then
	if err != nil {
		t.Fatalf("register user: %v", err)
	}
	stored, err := repo.FindByUsername(context.Background(), "alice")
	if err != nil {
		t.Fatalf("find created user: %v", err)
	}
	if created.PasswordHash != "" {
		t.Fatalf("created user exposed password hash")
	}
	if stored.PasswordHash == "secret1" || stored.PasswordHash == "" {
		t.Fatalf("stored password hash = %q, want bcrypt hash", stored.PasswordHash)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("secret1")); err != nil {
		t.Fatalf("compare bcrypt hash: %v", err)
	}
}

func Test_UserService_Login_returns_token_when_password_matches(t *testing.T) {
	// Given
	repo := newMemoryUserRepository()
	svc := service.NewUserService(repo, jwtpkg.NewManager("test-secret", time.Hour))
	_, err := svc.Register(context.Background(), do.User{
		Username: "alice",
		Password: "secret1",
		Role:     do.UserRoleUser,
	})
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	// When
	result, err := svc.Login(context.Background(), do.User{Username: "alice", Password: "secret1"})

	// Then
	if err != nil {
		t.Fatalf("login user: %v", err)
	}
	if result.Token == "" {
		t.Fatalf("empty login token")
	}
	claims, err := jwtpkg.NewManager("test-secret", time.Hour).Parse(result.Token, time.Now().UTC())
	if err != nil {
		t.Fatalf("parse login token: %v", err)
	}
	if claims.Username != "alice" || claims.Role != string(do.UserRoleUser) {
		t.Fatalf("claims = %#v, want alice user", claims)
	}
}

func Test_UserService_Login_rejects_wrong_password(t *testing.T) {
	// Given
	repo := newMemoryUserRepository()
	svc := service.NewUserService(repo, jwtpkg.NewManager("test-secret", time.Hour))
	_, err := svc.Register(context.Background(), do.User{
		Username: "alice",
		Password: "secret1",
		Role:     do.UserRoleUser,
	})
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	// When
	_, err = svc.Login(context.Background(), do.User{Username: "alice", Password: "wrong1"})

	// Then
	if !errors.Is(err, service.ErrInvalidCredential) {
		t.Fatalf("error = %v, want invalid credential", err)
	}
}

func Test_UserService_CurrentUser_returns_public_user_when_id_exists(t *testing.T) {
	// Given
	repo := newMemoryUserRepository()
	svc := service.NewUserService(repo, jwtpkg.NewManager("test-secret", time.Hour))
	created, err := svc.Register(context.Background(), do.User{
		Username: "alice",
		Password: "secret1",
		Role:     do.UserRoleUser,
	})
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	// When
	got, err := svc.CurrentUser(context.Background(), created.ID)

	// Then
	if err != nil {
		t.Fatalf("current user: %v", err)
	}
	if got.Username != "alice" || got.PasswordHash != "" || got.Password != "" {
		t.Fatalf("current user = %#v, want public alice", got)
	}
}
