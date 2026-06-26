package service_test

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/repository"
	"github.com/yachiyo/acgwarehouse/internal/service"
)

type memoryCollectionRepository struct {
	collections map[int64]do.Collection
	items       map[int64]map[int64]do.CollectionItem
	nextID      int64
}

func newMemoryCollectionRepository() *memoryCollectionRepository {
	return &memoryCollectionRepository{
		collections: make(map[int64]do.Collection),
		items:       make(map[int64]map[int64]do.CollectionItem),
		nextID:      1,
	}
}

func (r *memoryCollectionRepository) Create(_ context.Context, collection do.Collection) (do.Collection, error) {
	collection.ID = r.nextID
	r.nextID++
	collection.CreatedAt = time.Now().UTC()
	r.collections[collection.ID] = collection
	return collection, nil
}

func (r *memoryCollectionRepository) ListByOwner(_ context.Context, userID int64) ([]do.Collection, error) {
	collections := make([]do.Collection, 0)
	for _, collection := range r.collections {
		if collection.UserID == userID {
			collections = append(collections, collection)
		}
	}
	return collections, nil
}

func (r *memoryCollectionRepository) FindVisible(_ context.Context, collectionID int64, viewerID int64) (do.Collection, error) {
	collection, ok := r.collections[collectionID]
	if !ok {
		return do.Collection{}, repository.ErrCollectionNotFound
	}
	if collection.UserID != viewerID && collection.Visibility != do.CollectionVisibilityPublic {
		return do.Collection{}, repository.ErrCollectionForbidden
	}
	return collection, nil
}

func (r *memoryCollectionRepository) Update(_ context.Context, collection do.Collection) (do.Collection, error) {
	stored, ok := r.collections[collection.ID]
	if !ok {
		return do.Collection{}, repository.ErrCollectionNotFound
	}
	if stored.UserID != collection.UserID {
		return do.Collection{}, repository.ErrCollectionForbidden
	}
	stored.Name = collection.Name
	stored.Visibility = collection.Visibility
	r.collections[collection.ID] = stored
	return stored, nil
}

func (r *memoryCollectionRepository) Delete(_ context.Context, collectionID int64, userID int64) error {
	stored, ok := r.collections[collectionID]
	if !ok {
		return repository.ErrCollectionNotFound
	}
	if stored.UserID != userID {
		return repository.ErrCollectionForbidden
	}
	delete(r.collections, collectionID)
	return nil
}

func (r *memoryCollectionRepository) AddItem(_ context.Context, collectionID int64, userID int64, imageID int64) (do.CollectionItem, error) {
	stored, ok := r.collections[collectionID]
	if !ok {
		return do.CollectionItem{}, repository.ErrCollectionNotFound
	}
	if stored.UserID != userID {
		return do.CollectionItem{}, repository.ErrCollectionForbidden
	}
	if r.items[collectionID] == nil {
		r.items[collectionID] = make(map[int64]do.CollectionItem)
	}
	item := do.CollectionItem{CollectionID: collectionID, ImageID: imageID, CreatedAt: time.Now().UTC()}
	r.items[collectionID][imageID] = item
	return item, nil
}

func (r *memoryCollectionRepository) RemoveItem(_ context.Context, collectionID int64, userID int64, imageID int64) error {
	stored, ok := r.collections[collectionID]
	if !ok {
		return repository.ErrCollectionNotFound
	}
	if stored.UserID != userID {
		return repository.ErrCollectionForbidden
	}
	delete(r.items[collectionID], imageID)
	return nil
}

func Test_CollectionService_Create_accepts_multiple_named_collections_for_user(t *testing.T) {
	// Given
	repo := newMemoryCollectionRepository()
	svc := service.NewCollectionService(repo)

	// When
	first, firstErr := svc.Create(context.Background(), do.Collection{UserID: 7, Name: "miku"})
	second, secondErr := svc.Create(context.Background(), do.Collection{
		UserID:     7,
		Name:       "luka",
		Visibility: do.CollectionVisibilityPublic,
	})

	// Then
	if firstErr != nil || secondErr != nil {
		t.Fatalf("create collections: %v %v", firstErr, secondErr)
	}
	if first.ID == second.ID || first.Visibility != do.CollectionVisibilityPrivate {
		t.Fatalf("first=%#v second=%#v, want distinct collections with private default", first, second)
	}
}

func Test_CollectionService_Update_returns_forbidden_when_user_is_not_owner(t *testing.T) {
	// Given
	repo := newMemoryCollectionRepository()
	svc := service.NewCollectionService(repo)
	created, err := svc.Create(context.Background(), do.Collection{UserID: 7, Name: "owner"})
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}

	// When
	_, updateErr := svc.Update(context.Background(), do.Collection{
		ID:         created.ID,
		UserID:     8,
		Name:       "stolen",
		Visibility: do.CollectionVisibilityPublic,
	})

	// Then
	if !stderrors.Is(updateErr, service.ErrForbidden) {
		t.Fatalf("error = %v, want forbidden", updateErr)
	}
}

func Test_CollectionService_View_allows_public_guest_and_rejects_private_guest(t *testing.T) {
	// Given
	repo := newMemoryCollectionRepository()
	svc := service.NewCollectionService(repo)
	publicCollection, err := svc.Create(context.Background(), do.Collection{
		UserID:     7,
		Name:       "public",
		Visibility: do.CollectionVisibilityPublic,
	})
	if err != nil {
		t.Fatalf("create public collection: %v", err)
	}
	privateCollection, err := svc.Create(context.Background(), do.Collection{UserID: 7, Name: "private"})
	if err != nil {
		t.Fatalf("create private collection: %v", err)
	}

	// When
	visible, visibleErr := svc.FindVisible(context.Background(), publicCollection.ID, 0)
	_, privateErr := svc.FindVisible(context.Background(), privateCollection.ID, 0)

	// Then
	if visibleErr != nil || visible.ID != publicCollection.ID {
		t.Fatalf("visible=%#v err=%v, want public collection", visible, visibleErr)
	}
	if !stderrors.Is(privateErr, service.ErrForbidden) {
		t.Fatalf("private error = %v, want forbidden", privateErr)
	}
}

func Test_CollectionService_AddItem_returns_forbidden_when_user_is_not_owner(t *testing.T) {
	// Given
	repo := newMemoryCollectionRepository()
	svc := service.NewCollectionService(repo)
	collection, err := svc.Create(context.Background(), do.Collection{UserID: 7, Name: "owner"})
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}

	// When
	_, addErr := svc.AddItem(context.Background(), collection.ID, 8, 99)

	// Then
	if !stderrors.Is(addErr, service.ErrForbidden) {
		t.Fatalf("error = %v, want forbidden", addErr)
	}
}
