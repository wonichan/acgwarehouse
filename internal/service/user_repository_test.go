package service_test

import (
	"context"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/service"
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

func (r *memoryUserRepository) UpdateProfile(_ context.Context, user do.User) (do.User, error) {
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

func (r *memoryUserRepository) UpdatePasswordHash(_ context.Context, userID int64, passwordHash string) error {
	stored, ok := r.byID[userID]
	if !ok {
		return service.ErrUserNotFound
	}
	stored.PasswordHash = passwordHash
	r.byID[stored.ID] = stored
	r.byName[stored.Username] = stored
	return nil
}
