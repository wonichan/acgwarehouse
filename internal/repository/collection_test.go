package repository_test

import (
	"context"
	stderrors "errors"
	"testing"

	"gorm.io/gorm"

	"github.com/yachiyo/acgwarehouse/internal/infra/db"
	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/po"
	"github.com/yachiyo/acgwarehouse/internal/repository"
)

func Test_CollectionRepository_Create_persists_named_collection_with_visibility(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	repo := repository.NewCollectionRepository(database.Read, database.Write)
	input := do.Collection{UserID: 7, Name: "miku", Visibility: do.CollectionVisibilityPublic}

	// When
	created, err := repo.Create(context.Background(), input)

	// Then
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}
	if created.ID == 0 || created.UserID != 7 || created.Name != "miku" {
		t.Fatalf("created = %#v, want persisted collection", created)
	}
	if created.Visibility != do.CollectionVisibilityPublic || created.CreatedAt.IsZero() {
		t.Fatalf("created = %#v, want public collection with created time", created)
	}
}

func Test_CollectionRepository_FindVisible_allows_public_viewer_and_blocks_private_non_owner(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	repo := repository.NewCollectionRepository(database.Read, database.Write)
	publicCollection := mustCreateCollection(t, repo, do.CollectionVisibilityPublic, 7, "public")
	privateCollection := mustCreateCollection(t, repo, do.CollectionVisibilityPrivate, 7, "private")

	// When
	visible, visibleErr := repo.FindVisible(context.Background(), publicCollection.ID, 0)
	_, privateErr := repo.FindVisible(context.Background(), privateCollection.ID, 8)

	// Then
	if visibleErr != nil {
		t.Fatalf("find public collection: %v", visibleErr)
	}
	if visible.ID != publicCollection.ID {
		t.Fatalf("visible = %#v, want public collection", visible)
	}
	if !stderrors.Is(privateErr, repository.ErrCollectionForbidden) {
		t.Fatalf("private error = %v, want collection forbidden", privateErr)
	}
}

func Test_CollectionRepository_ListByOwner_returns_collection_items(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	repo := repository.NewCollectionRepository(database.Read, database.Write)
	image := mustCreateImage(t, imageRepo, "thumbnails/list-owner-item.png")
	collection := mustCreateCollection(t, repo, do.CollectionVisibilityPrivate, 7, "favorites")
	if _, err := repo.AddItem(context.Background(), collection.ID, 7, image.ID); err != nil {
		t.Fatalf("add collection item: %v", err)
	}

	// When
	collections, err := repo.ListByOwner(context.Background(), 7)

	// Then
	if err != nil {
		t.Fatalf("list owner collections: %v", err)
	}
	if len(collections) != 1 {
		t.Fatalf("collections = %#v, want one collection", collections)
	}
	if len(collections[0].Items) != 1 || collections[0].Items[0].ImageID != image.ID {
		t.Fatalf("items = %#v, want image %d", collections[0].Items, image.ID)
	}
}

func Test_CollectionRepository_Update_rejects_non_owner_management(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	repo := repository.NewCollectionRepository(database.Read, database.Write)
	collection := mustCreateCollection(t, repo, do.CollectionVisibilityPrivate, 7, "private")

	// When
	_, err := repo.Update(context.Background(), do.Collection{
		ID:         collection.ID,
		UserID:     8,
		Name:       "new-name",
		Visibility: do.CollectionVisibilityPublic,
	})

	// Then
	if !stderrors.Is(err, repository.ErrCollectionForbidden) {
		t.Fatalf("error = %v, want collection forbidden", err)
	}
}

func Test_CollectionRepository_AddItem_keeps_image_unique_within_collection(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	repo := repository.NewCollectionRepository(database.Read, database.Write)
	image := mustCreateImage(t, imageRepo, "thumbnails/unique-favorite.png")
	collection := mustCreateCollection(t, repo, do.CollectionVisibilityPrivate, 7, "favorites")
	if _, err := repo.AddItem(context.Background(), collection.ID, 7, image.ID); err != nil {
		t.Fatalf("add first item: %v", err)
	}

	// When
	_, err := repo.AddItem(context.Background(), collection.ID, 7, image.ID)

	// Then
	if err != nil {
		t.Fatalf("add duplicate item: %v", err)
	}
	assertCollectionImageState(t, database, image.ID, 1, 1, 1)
}

func Test_CollectionRepository_RemoveItem_decrements_favorite_count_only_after_user_last_copy(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	repo := repository.NewCollectionRepository(database.Read, database.Write)
	image := mustCreateImage(t, imageRepo, "thumbnails/dedup-favorite.png")
	first := mustCreateCollection(t, repo, do.CollectionVisibilityPrivate, 7, "first")
	second := mustCreateCollection(t, repo, do.CollectionVisibilityPrivate, 7, "second")
	if _, err := repo.AddItem(context.Background(), first.ID, 7, image.ID); err != nil {
		t.Fatalf("add first collection item: %v", err)
	}
	if _, err := repo.AddItem(context.Background(), second.ID, 7, image.ID); err != nil {
		t.Fatalf("add second collection item: %v", err)
	}

	// When
	firstErr := repo.RemoveItem(context.Background(), first.ID, 7, image.ID)
	firstImage := mustFindImage(t, imageRepo, image.ID)
	secondErr := repo.RemoveItem(context.Background(), second.ID, 7, image.ID)
	secondImage := mustFindImage(t, imageRepo, image.ID)

	// Then
	if firstErr != nil || secondErr != nil {
		t.Fatalf("remove items: %v %v", firstErr, secondErr)
	}
	if firstImage.FavoriteCount != 1 {
		t.Fatalf("favorite count after first remove = %d, want 1", firstImage.FavoriteCount)
	}
	if secondImage.FavoriteCount != 0 {
		t.Fatalf("favorite count after second remove = %d, want 0", secondImage.FavoriteCount)
	}
	assertFavoriteEventValues(t, database, image.ID, []int{1, -1})
}

func Test_CollectionRepository_Delete_recomputes_favorite_count_for_removed_items(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	repo := repository.NewCollectionRepository(database.Read, database.Write)
	image := mustCreateImage(t, imageRepo, "thumbnails/delete-favorite.png")
	first := mustCreateCollection(t, repo, do.CollectionVisibilityPrivate, 7, "first")
	second := mustCreateCollection(t, repo, do.CollectionVisibilityPrivate, 7, "second")
	if _, err := repo.AddItem(context.Background(), first.ID, 7, image.ID); err != nil {
		t.Fatalf("add first collection item: %v", err)
	}
	if _, err := repo.AddItem(context.Background(), second.ID, 7, image.ID); err != nil {
		t.Fatalf("add second collection item: %v", err)
	}

	// When
	firstErr := repo.Delete(context.Background(), first.ID, 7)
	firstImage := mustFindImage(t, imageRepo, image.ID)
	secondErr := repo.Delete(context.Background(), second.ID, 7)
	secondImage := mustFindImage(t, imageRepo, image.ID)

	// Then
	if firstErr != nil || secondErr != nil {
		t.Fatalf("delete collections: %v %v", firstErr, secondErr)
	}
	if firstImage.FavoriteCount != 1 {
		t.Fatalf("favorite count after first delete = %d, want 1", firstImage.FavoriteCount)
	}
	if secondImage.FavoriteCount != 0 {
		t.Fatalf("favorite count after second delete = %d, want 0", secondImage.FavoriteCount)
	}
	assertFavoriteEventValues(t, database, image.ID, []int{1, -1})
}

func mustCreateCollection(
	t *testing.T,
	repo *repository.CollectionRepository,
	visibility do.CollectionVisibility,
	userID int64,
	name string,
) do.Collection {
	t.Helper()
	collection, err := repo.Create(context.Background(), do.Collection{UserID: userID, Name: name, Visibility: visibility})
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}
	return collection
}

func mustFindImage(t *testing.T, repo *repository.ImageRepository, imageID int64) do.Image {
	t.Helper()
	image, err := repo.FindActiveByID(context.Background(), imageID)
	if err != nil {
		t.Fatalf("find image: %v", err)
	}
	return image
}

func assertCollectionImageState(t *testing.T, database *db.SQLite, imageID int64, items int64, events int64, favoriteCount int64) {
	t.Helper()
	itemCount := countRows(t, database.Read.Model(&po.CollectionItem{}).Where("image_id = ?", imageID))
	eventCount := countRows(
		t,
		database.Read.Model(&po.ImageEvent{}).Where("image_id = ? AND type = ?", imageID, string(do.ImageEventTypeFavorite)),
	)
	var image po.Image
	if err := database.Read.Where("id = ?", imageID).First(&image).Error; err != nil {
		t.Fatalf("find image row: %v", err)
	}
	if itemCount != items || eventCount != events || image.FavoriteCount != favoriteCount {
		t.Fatalf("items=%d events=%d favorite=%d, want %d/%d/%d", itemCount, eventCount, image.FavoriteCount, items, events, favoriteCount)
	}
}

func countRows(t *testing.T, query *gorm.DB) int64 {
	t.Helper()
	var count int64
	if err := query.Count(&count).Error; err != nil {
		t.Fatalf("count rows: %v", err)
	}
	return count
}

func assertFavoriteEventValues(t *testing.T, database *db.SQLite, imageID int64, want []int) {
	t.Helper()
	var events []po.ImageEvent
	if err := database.Read.Where("image_id = ? AND type = ?", imageID, string(do.ImageEventTypeFavorite)).
		Order("id asc").Find(&events).Error; err != nil {
		t.Fatalf("list favorite events: %v", err)
	}
	if len(events) != len(want) {
		t.Fatalf("events = %#v, want %d favorite events", events, len(want))
	}
	for index, event := range events {
		if event.Value != want[index] {
			t.Fatalf("event[%d] value = %d, want %d", index, event.Value, want[index])
		}
	}
}
