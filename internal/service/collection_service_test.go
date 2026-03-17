package service

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

func newCollectionServiceForTest(t *testing.T) (*CollectionService, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Initialize schema
	schema, err := os.ReadFile("../repository/schema.sql")
	if err != nil {
		// Use inline schema if file not found
		schema = []byte(`
			CREATE TABLE IF NOT EXISTS collections (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL UNIQUE,
				description TEXT,
				cover_image_id INTEGER,
				image_count INTEGER DEFAULT 0,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
			CREATE TABLE IF NOT EXISTS collection_images (
				collection_id INTEGER NOT NULL,
				image_id INTEGER NOT NULL,
				added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (collection_id, image_id),
				FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE,
				FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE
			);
			CREATE TABLE IF NOT EXISTS images (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				path TEXT NOT NULL UNIQUE,
				filename TEXT NOT NULL,
				source_root TEXT NOT NULL,
				file_size INTEGER DEFAULT 0,
				width INTEGER DEFAULT 0,
				height INTEGER DEFAULT 0,
				format TEXT,
				phash INTEGER,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
		`)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	repo := repository.NewCollectionRepository(db)
	return NewCollectionService(repo), db
}

func TestCollectionService_CreateCollection(t *testing.T) {
	svc, db := newCollectionServiceForTest(t)
	defer db.Close()

	ctx := context.Background()

	// Test create
	collection, err := svc.CreateCollection(ctx, "Test Collection", "Test Description")
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	if collection.ID == 0 {
		t.Error("Expected collection ID to be set")
	}
	if collection.Name != "Test Collection" {
		t.Errorf("Expected name 'Test Collection', got '%s'", collection.Name)
	}
}

func TestCollectionService_CreateCollection_DuplicateName(t *testing.T) {
	svc, db := newCollectionServiceForTest(t)
	defer db.Close()

	ctx := context.Background()

	// Create first collection
	_, err := svc.CreateCollection(ctx, "Test Collection", "Test Description")
	if err != nil {
		t.Fatalf("Failed to create first collection: %v", err)
	}

	// Try to create duplicate
	_, err = svc.CreateCollection(ctx, "Test Collection", "Another Description")
	if err == nil {
		t.Error("Expected error for duplicate name")
	}
}

func TestCollectionService_GetCollection(t *testing.T) {
	svc, db := newCollectionServiceForTest(t)
	defer db.Close()

	ctx := context.Background()

	// Create collection
	created, err := svc.CreateCollection(ctx, "Test Collection", "Test Description")
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	// Get collection
	collection, err := svc.GetCollection(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to get collection: %v", err)
	}

	if collection.Name != "Test Collection" {
		t.Errorf("Expected name 'Test Collection', got '%s'", collection.Name)
	}
}

func TestCollectionService_ListCollections(t *testing.T) {
	svc, db := newCollectionServiceForTest(t)
	defer db.Close()

	ctx := context.Background()

	// Create collections
	for i := 0; i < 3; i++ {
		_, err := svc.CreateCollection(ctx, "Collection "+string(rune('A'+i)), "")
		if err != nil {
			t.Fatalf("Failed to create collection: %v", err)
		}
	}

	// List collections
	collections, err := svc.ListCollections(ctx, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list collections: %v", err)
	}

	if len(collections) != 3 {
		t.Errorf("Expected 3 collections, got %d", len(collections))
	}
}

func TestCollectionService_UpdateCollection(t *testing.T) {
	svc, db := newCollectionServiceForTest(t)
	defer db.Close()

	ctx := context.Background()

	// Create collection
	created, err := svc.CreateCollection(ctx, "Test Collection", "Test Description")
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	// Update collection
	updated, err := svc.UpdateCollection(ctx, created.ID, "Updated Name", "Updated Description")
	if err != nil {
		t.Fatalf("Failed to update collection: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got '%s'", updated.Name)
	}
}

func TestCollectionService_DeleteCollection(t *testing.T) {
	svc, db := newCollectionServiceForTest(t)
	defer db.Close()

	ctx := context.Background()

	// Create collection
	created, err := svc.CreateCollection(ctx, "Test Collection", "Test Description")
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	// Delete collection
	err = svc.DeleteCollection(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to delete collection: %v", err)
	}

	// Verify deleted
	_, err = svc.GetCollection(ctx, created.ID)
	if err == nil {
		t.Error("Expected error when getting deleted collection")
	}
}
