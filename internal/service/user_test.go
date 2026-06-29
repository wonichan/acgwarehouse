package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/service"
	jwtpkg "github.com/yachiyo/acgwarehouse/pkg/jwt"
)

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

func Test_UserService_UpdateCurrentUserProfile_persists_trimmed_profile_when_input_valid(t *testing.T) {
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
	input := do.User{
		Nickname:           " Alice Atelier ",
		FavoriteTags:       "雨景, 制服",
		Bio:                "  收藏高评分角色参考。  ",
		PublicProfile:      true,
		EmailNotifications: false,
		SyncCollections:    true,
	}

	// When
	updated, err := svc.UpdateCurrentUserProfile(context.Background(), created.ID, input)

	// Then
	if err != nil {
		t.Fatalf("update current user profile: %v", err)
	}
	if updated.Nickname != "Alice Atelier" || updated.Bio != "收藏高评分角色参考。" {
		t.Fatalf("updated profile = %#v, want trimmed profile", updated)
	}
	stored, err := repo.FindByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("find updated user: %v", err)
	}
	if stored.FavoriteTags != "雨景, 制服" || stored.EmailNotifications {
		t.Fatalf("stored preferences = %#v, want persisted preferences", stored)
	}
}

func Test_UserService_UpdateCurrentUserProfile_rejects_invalid_profile_fields(t *testing.T) {
	tests := []struct {
		name   string
		input  do.User
		wantOK bool
	}{
		{name: "empty nickname", input: do.User{Nickname: "   ", FavoriteTags: "雨景", Bio: "简介"}},
		{name: "tags too long", input: do.User{Nickname: "Alice", FavoriteTags: strings.Repeat("标", 121), Bio: "简介"}},
		{name: "bio too long", input: do.User{Nickname: "Alice", FavoriteTags: "雨景", Bio: strings.Repeat("介", 201)}},
		{name: "nickname counts unicode characters", input: do.User{Nickname: strings.Repeat("爱", 20), FavoriteTags: strings.Repeat("标", 120), Bio: strings.Repeat("介", 200)}, wantOK: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			_, err = svc.UpdateCurrentUserProfile(context.Background(), created.ID, tt.input)

			// Then
			if tt.wantOK {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, service.ErrInvalidUserInput) {
				t.Fatalf("error = %v, want invalid user input", err)
			}
		})
	}
}

func Test_UserService_ChangePassword_updates_hash_when_old_password_matches(t *testing.T) {
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
	err = svc.ChangePassword(context.Background(), created.ID, "secret1", "secret2")

	// Then
	if err != nil {
		t.Fatalf("change password: %v", err)
	}
	stored, err := repo.FindByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("find updated user: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("secret2")); err != nil {
		t.Fatalf("compare new bcrypt hash: %v", err)
	}
}

func Test_UserService_ChangePassword_rejects_invalid_password_change(t *testing.T) {
	tests := []struct {
		name        string
		oldPassword string
		newPassword string
		wantErr     error
	}{
		{name: "old password mismatch", oldPassword: "wrong1", newPassword: "secret2", wantErr: service.ErrInvalidCredential},
		{name: "new password too short", oldPassword: "secret1", newPassword: "short", wantErr: service.ErrInvalidUserInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			err = svc.ChangePassword(context.Background(), created.ID, tt.oldPassword, tt.newPassword)

			// Then
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
