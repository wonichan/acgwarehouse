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
