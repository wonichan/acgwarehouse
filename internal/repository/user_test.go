package repository_test

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"github.com/yachiyo/acgwarehouse/internal/conf"
	"github.com/yachiyo/acgwarehouse/internal/infra/db"
	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/repository"
)

func Test_UserRepository_Create_persists_user_when_username_unique(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	repo := repository.NewUserRepository(database.Read, database.Write)
	input := do.User{Username: "alice", PasswordHash: "hash", Role: do.UserRoleUser}

	// When
	created, err := repo.Create(context.Background(), input)

	// Then
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("created id = 0, want generated id")
	}
	got, err := repo.FindByUsername(context.Background(), "alice")
	if err != nil {
		t.Fatalf("find user by username: %v", err)
	}
	if got.ID != created.ID || got.PasswordHash != "hash" || got.Role != do.UserRoleUser {
		t.Fatalf("stored user = %#v, want created user", got)
	}
	if got.Nickname != "" || !got.PublicProfile || !got.EmailNotifications || !got.SyncCollections {
		t.Fatalf("stored profile defaults = %#v, want database defaults", got)
	}
}

func Test_UserRepository_FindByID_returns_not_found_when_missing(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	repo := repository.NewUserRepository(database.Read, database.Write)

	// When
	_, err := repo.FindByID(context.Background(), 404)

	// Then
	if !stderrors.Is(err, repository.ErrUserNotFound) {
		t.Fatalf("error = %v, want user not found", err)
	}
}

func Test_UserRepository_Create_rejects_duplicate_username(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	repo := repository.NewUserRepository(database.Read, database.Write)
	input := do.User{Username: "alice", PasswordHash: "hash", Role: do.UserRoleUser}
	if _, err := repo.Create(context.Background(), input); err != nil {
		t.Fatalf("create first user: %v", err)
	}

	// When
	_, err := repo.Create(context.Background(), input)

	// Then
	if !stderrors.Is(err, repository.ErrUsernameExists) {
		t.Fatalf("error = %v, want username exists", err)
	}
}

func Test_UserRepository_UpdateProfile_persists_profile_fields_when_user_exists(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	repo := repository.NewUserRepository(database.Read, database.Write)
	created, err := repo.Create(context.Background(), do.User{
		Username:     "alice",
		PasswordHash: "hash",
		Role:         do.UserRoleUser,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	input := do.User{
		ID:                 created.ID,
		Nickname:           "Alice Atelier",
		FavoriteTags:       "雨景, 制服",
		Bio:                "收藏高评分角色参考。",
		PublicProfile:      true,
		EmailNotifications: false,
		SyncCollections:    true,
	}

	// When
	updated, err := repo.UpdateProfile(context.Background(), input)

	// Then
	if err != nil {
		t.Fatalf("update user profile: %v", err)
	}
	if updated.Nickname != "Alice Atelier" || updated.FavoriteTags != "雨景, 制服" {
		t.Fatalf("updated user = %#v, want persisted profile", updated)
	}
	if updated.EmailNotifications {
		t.Fatalf("email notifications = true, want false")
	}
}

func Test_UserRepository_UpdatePasswordHash_persists_hash_when_user_exists(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	repo := repository.NewUserRepository(database.Read, database.Write)
	created, err := repo.Create(context.Background(), do.User{
		Username:     "alice",
		PasswordHash: "old-hash",
		Role:         do.UserRoleUser,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// When
	err = repo.UpdatePasswordHash(context.Background(), created.ID, "new-hash")

	// Then
	if err != nil {
		t.Fatalf("update password hash: %v", err)
	}
	got, err := repo.FindByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("find user by id: %v", err)
	}
	if got.PasswordHash != "new-hash" {
		t.Fatalf("password hash = %q, want new-hash", got.PasswordHash)
	}
}

func openTestDatabase(t *testing.T) *db.SQLite {
	t.Helper()
	database, err := db.NewSQLite(testDatabaseConfig(t))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close test database: %v", err)
		}
	})
	return database
}

func testDatabaseConfig(t *testing.T) conf.DatabaseConfig {
	t.Helper()
	return conf.DatabaseConfig{
		Path:              t.TempDir() + "/test.db",
		BusyTimeoutMS:     int((5 * time.Second).Milliseconds()),
		ReadMaxOpenConns:  2,
		WriteMaxOpenConns: 1,
	}
}
