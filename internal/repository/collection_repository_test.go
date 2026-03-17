package repository

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

// newCollectionRepositoryForTest creates a test database and repository
func newCollectionRepositoryForTest(t *testing.T) (*sql.DB, CollectionRepository) {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "collection-repo.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	return db, NewCollectionRepository(db)
}

// mustSaveCollection saves a collection and fails the test on error
func mustSaveCollection(t *testing.T, repo CollectionRepository, collection *domain.Collection) {
	t.Helper()
	ctx := context.Background()
	if err := repo.Save(ctx, collection); err != nil {
		t.Fatalf("Save collection: %v", err)
	}
}

// mustSaveImage saves an image and fails the test on error
func mustSaveImage(t *testing.T, db *sql.DB, image *domain.Image) {
	t.Helper()
	imageRepo := NewImageRepository(db)
	if err := imageRepo.SaveImage(image); err != nil {
		t.Fatalf("Save image: %v", err)
	}
}

// TestCollectionRepositorySave tests the Save method
func TestCollectionRepositorySave(t *testing.T) {
	t.Parallel()

	_, repo := newCollectionRepositoryForTest(t)
	ctx := context.Background()

	collection := &domain.Collection{
		Name:        "My Favorites",
		Description: "Best images",
	}

	err := repo.Save(ctx, collection)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if collection.ID == 0 {
		t.Error("Save() did not set collection ID")
	}
	if collection.CreatedAt.IsZero() {
		t.Error("Save() did not set CreatedAt")
	}
	if collection.UpdatedAt.IsZero() {
		t.Error("Save() did not set UpdatedAt")
	}
}

// TestCollectionRepositoryFindByID tests FindByID method
func TestCollectionRepositoryFindByID(t *testing.T) {
	t.Parallel()

	_, repo := newCollectionRepositoryForTest(t)
	ctx := context.Background()

	// Test not found
	_, err := repo.FindByID(ctx, 999)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("FindByID(999) error = %v, want sql.ErrNoRows", err)
	}

	// Test found
	collection := &domain.Collection{Name: "Test Collection"}
	mustSaveCollection(t, repo, collection)

	found, err := repo.FindByID(ctx, collection.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.Name != collection.Name {
		t.Errorf("Name = %q, want %q", found.Name, collection.Name)
	}
}

// TestCollectionRepositoryFindAll tests FindAll with pagination
func TestCollectionRepositoryFindAll(t *testing.T) {
	t.Parallel()

	_, repo := newCollectionRepositoryForTest(t)
	ctx := context.Background()

	// Test empty
	empty, err := repo.FindAll(ctx, 10, 0)
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("FindAll() returned %d items, want 0", len(empty))
	}

	// Create collections
	for i := 0; i < 5; i++ {
		collection := &domain.Collection{Name: string(rune('A' + i))}
		mustSaveCollection(t, repo, collection)
	}

	// Test pagination
	page1, err := repo.FindAll(ctx, 3, 0)
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}
	if len(page1) != 3 {
		t.Errorf("len(page1) = %d, want 3", len(page1))
	}

	page2, err := repo.FindAll(ctx, 3, 3)
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}
	if len(page2) != 2 {
		t.Errorf("len(page2) = %d, want 2", len(page2))
	}
}

// TestCollectionRepositoryFindByName tests FindByName method
func TestCollectionRepositoryFindByName(t *testing.T) {
	t.Parallel()

	_, repo := newCollectionRepositoryForTest(t)
	ctx := context.Background()

	// Test not found
	_, err := repo.FindByName(ctx, "Nonexistent")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("FindByName() error = %v, want sql.ErrNoRows", err)
	}

	// Test found
	collection := &domain.Collection{Name: "Unique Name"}
	mustSaveCollection(t, repo, collection)

	found, err := repo.FindByName(ctx, "Unique Name")
	if err != nil {
		t.Fatalf("FindByName() error = %v", err)
	}
	if found.ID != collection.ID {
		t.Errorf("ID = %d, want %d", found.ID, collection.ID)
	}
}

// TestCollectionRepositoryUpdate tests Update method
func TestCollectionRepositoryUpdate(t *testing.T) {
	t.Parallel()

	_, repo := newCollectionRepositoryForTest(t)
	ctx := context.Background()

	collection := &domain.Collection{Name: "Original", Description: "Old desc"}
	mustSaveCollection(t, repo, collection)
	originalUpdatedAt := collection.UpdatedAt

	// Update
	time.Sleep(10 * time.Millisecond) // Ensure time difference
	collection.Name = "Updated"
	collection.Description = "New desc"
	err := repo.Update(ctx, collection)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify
	found, err := repo.FindByID(ctx, collection.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.Name != "Updated" {
		t.Errorf("Name = %q, want %q", found.Name, "Updated")
	}
	if found.Description != "New desc" {
		t.Errorf("Description = %q, want %q", found.Description, "New desc")
	}
	if !found.UpdatedAt.After(originalUpdatedAt) {
		t.Error("UpdatedAt was not updated")
	}
}

// TestCollectionRepositoryDelete tests Delete method
func TestCollectionRepositoryDelete(t *testing.T) {
	t.Parallel()

	_, repo := newCollectionRepositoryForTest(t)
	ctx := context.Background()

	collection := &domain.Collection{Name: "To Delete"}
	mustSaveCollection(t, repo, collection)

	// Delete
	err := repo.Delete(ctx, collection.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deleted
	_, err = repo.FindByID(ctx, collection.ID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("FindByID() error = %v, want sql.ErrNoRows", err)
	}
}

// TestCollectionRepositoryAddImage tests AddImage method
func TestCollectionRepositoryAddImage(t *testing.T) {
	t.Parallel()

	db, repo := newCollectionRepositoryForTest(t)
	ctx := context.Background()

	collection := &domain.Collection{Name: "Test"}
	mustSaveCollection(t, repo, collection)

	image := &domain.Image{
		Path:       "/test.png",
		Filename:   "test.png",
		SourceRoot: "/",
		Format:     "png",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	mustSaveImage(t, db, image)

	// Add image
	err := repo.AddImage(ctx, collection.ID, image.ID)
	if err != nil {
		t.Fatalf("AddImage() error = %v", err)
	}

	// Verify image count
	count, err := repo.CountImages(ctx, collection.ID)
	if err != nil {
		t.Fatalf("CountImages() error = %v", err)
	}
	if count != 1 {
		t.Errorf("CountImages() = %d, want 1", count)
	}

	// Verify collection image_count updated
	found, err := repo.FindByID(ctx, collection.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.ImageCount != 1 {
		t.Errorf("ImageCount = %d, want 1", found.ImageCount)
	}
}

// TestCollectionRepositoryAddImageIdempotent tests that adding same image twice is idempotent
func TestCollectionRepositoryAddImageIdempotent(t *testing.T) {
	t.Parallel()

	db, repo := newCollectionRepositoryForTest(t)
	ctx := context.Background()

	collection := &domain.Collection{Name: "Test"}
	mustSaveCollection(t, repo, collection)

	image := &domain.Image{
		Path:       "/test.png",
		Filename:   "test.png",
		SourceRoot: "/",
		Format:     "png",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	mustSaveImage(t, db, image)

	// Add image twice
	_ = repo.AddImage(ctx, collection.ID, image.ID)
	err := repo.AddImage(ctx, collection.ID, image.ID)
	if err != nil {
		t.Fatalf("AddImage() second call error = %v", err)
	}

	// Verify image count is still 1
	count, err := repo.CountImages(ctx, collection.ID)
	if err != nil {
		t.Fatalf("CountImages() error = %v", err)
	}
	if count != 1 {
		t.Errorf("CountImages() = %d, want 1", count)
	}
}

// TestCollectionRepositoryRemoveImage tests RemoveImage method
func TestCollectionRepositoryRemoveImage(t *testing.T) {
	t.Parallel()

	db, repo := newCollectionRepositoryForTest(t)
	ctx := context.Background()

	collection := &domain.Collection{Name: "Test"}
	mustSaveCollection(t, repo, collection)

	image := &domain.Image{
		Path:       "/test.png",
		Filename:   "test.png",
		SourceRoot: "/",
		Format:     "png",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	mustSaveImage(t, db, image)

	// Add then remove
	_ = repo.AddImage(ctx, collection.ID, image.ID)
	err := repo.RemoveImage(ctx, collection.ID, image.ID)
	if err != nil {
		t.Fatalf("RemoveImage() error = %v", err)
	}

	// Verify image count
	count, err := repo.CountImages(ctx, collection.ID)
	if err != nil {
		t.Fatalf("CountImages() error = %v", err)
	}
	if count != 0 {
		t.Errorf("CountImages() = %d, want 0", count)
	}
}

// TestCollectionRepositoryFindImagesByCollection tests FindImagesByCollection with pagination
func TestCollectionRepositoryFindImagesByCollection(t *testing.T) {
	t.Parallel()

	db, repo := newCollectionRepositoryForTest(t)
	ctx := context.Background()

	collection := &domain.Collection{Name: "Test"}
	mustSaveCollection(t, repo, collection)

	// Create 3 images
	for i := 0; i < 3; i++ {
		image := &domain.Image{
			Path:       string(rune('a'+i)) + ".png",
			Filename:   string(rune('a'+i)) + ".png",
			SourceRoot: "/",
			Format:     "png",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		mustSaveImage(t, db, image)
		_ = repo.AddImage(ctx, collection.ID, image.ID)
	}

	// Test pagination
	page1, err := repo.FindImagesByCollection(ctx, collection.ID, 2, 0)
	if err != nil {
		t.Fatalf("FindImagesByCollection() error = %v", err)
	}
	if len(page1) != 2 {
		t.Errorf("len(page1) = %d, want 2", len(page1))
	}

	page2, err := repo.FindImagesByCollection(ctx, collection.ID, 2, 2)
	if err != nil {
		t.Fatalf("FindImagesByCollection() error = %v", err)
	}
	if len(page2) != 1 {
		t.Errorf("len(page2) = %d, want 1", len(page2))
	}
}

// TestCollectionRepositoryUpdateCover tests UpdateCover method
func TestCollectionRepositoryUpdateCover(t *testing.T) {
	t.Parallel()

	db, repo := newCollectionRepositoryForTest(t)
	ctx := context.Background()

	collection := &domain.Collection{Name: "Test"}
	mustSaveCollection(t, repo, collection)

	image := &domain.Image{
		Path:       "/test.png",
		Filename:   "test.png",
		SourceRoot: "/",
		Format:     "png",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	mustSaveImage(t, db, image)

	// Update cover
	err := repo.UpdateCover(ctx, collection.ID, image.ID)
	if err != nil {
		t.Fatalf("UpdateCover() error = %v", err)
	}

	// Verify
	found, err := repo.FindByID(ctx, collection.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.CoverImageID == nil || *found.CoverImageID != image.ID {
		t.Errorf("CoverImageID = %v, want %d", found.CoverImageID, image.ID)
	}
}

// TestCollectionRepositoryGetLatestImageID tests GetLatestImageID method
func TestCollectionRepositoryGetLatestImageID(t *testing.T) {
	t.Parallel()

	db, repo := newCollectionRepositoryForTest(t)
	ctx := context.Background()

	collection := &domain.Collection{Name: "Test"}
	mustSaveCollection(t, repo, collection)

	// Empty collection
	latestID, err := repo.GetLatestImageID(ctx, collection.ID)
	if err != nil {
		t.Fatalf("GetLatestImageID() error = %v", err)
	}
	if latestID != nil {
		t.Errorf("GetLatestImageID() = %d, want nil", *latestID)
	}

	// Add images with delay to ensure different added_at times
	var lastImageID int64
	for i := 0; i < 3; i++ {
		image := &domain.Image{
			Path:       string(rune('a'+i)) + ".png",
			Filename:   string(rune('a'+i)) + ".png",
			SourceRoot: "/",
			Format:     "png",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		mustSaveImage(t, db, image)
		_ = repo.AddImage(ctx, collection.ID, image.ID)
		lastImageID = image.ID
		time.Sleep(10 * time.Millisecond) // Ensure different added_at
	}

	// Get latest (should be the last added)
	latestID, err = repo.GetLatestImageID(ctx, collection.ID)
	if err != nil {
		t.Fatalf("GetLatestImageID() error = %v", err)
	}
	if latestID == nil || *latestID != lastImageID {
		t.Errorf("GetLatestImageID() = %v, want %d", latestID, lastImageID)
	}
}

// TestCollectionRepositoryCount tests Count method
func TestCollectionRepositoryCount(t *testing.T) {
	t.Parallel()

	_, repo := newCollectionRepositoryForTest(t)
	ctx := context.Background()

	// Empty
	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 0 {
		t.Errorf("Count() = %d, want 0", count)
	}

	// Add collections
	for i := 0; i < 3; i++ {
		collection := &domain.Collection{Name: string(rune('A' + i))}
		mustSaveCollection(t, repo, collection)
	}

	count, err = repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 3 {
		t.Errorf("Count() = %d, want 3", count)
	}
}

// TestCollectionRepositoryDeleteCascades tests that deleting a collection removes collection_images
func TestCollectionRepositoryDeleteCascades(t *testing.T) {
	t.Parallel()

	db, repo := newCollectionRepositoryForTest(t)
	ctx := context.Background()

	collection := &domain.Collection{Name: "Test"}
	mustSaveCollection(t, repo, collection)

	image := &domain.Image{
		Path:       "/test.png",
		Filename:   "test.png",
		SourceRoot: "/",
		Format:     "png",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	mustSaveImage(t, db, image)

	// Add image to collection
	_ = repo.AddImage(ctx, collection.ID, image.ID)

	// Delete collection
	err := repo.Delete(ctx, collection.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify cascade: check image still exists but collection_images gone
	images, err := repo.FindImagesByCollection(ctx, collection.ID, 10, 0)
	if err != nil {
		t.Fatalf("FindImagesByCollection() error = %v", err)
	}
	if len(images) != 0 {
		t.Errorf("FindImagesByCollection() = %d images, want 0 (cascade delete)", len(images))
	}
}
