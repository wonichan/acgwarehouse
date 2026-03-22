package repository

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

func TestFTS5VirtualTableCreated(t *testing.T) {
	t.Parallel()

	db := newFTSTestDB(t)

	// Verify FTS5 table exists
	var name string
	err := db.QueryRow(`
		SELECT name FROM sqlite_master 
		WHERE type='table' AND name='images_fts'
	`).Scan(&name)
	if err != nil {
		t.Fatalf("FTS5 table not found: %v", err)
	}
	if name != "images_fts" {
		t.Fatalf("table name = %s, want images_fts", name)
	}
}

func TestFTSInsertSyncsOnImageInsert(t *testing.T) {
	t.Parallel()

	db := newFTSTestDB(t)
	imageRepo := NewImageRepository(db)

	// Create image
	img := &domain.Image{
		Path:       "/test/image.png",
		Filename:   "test_image.png",
		SourceRoot: "/test",
		Format:     "png",
	}
	if _, err := imageRepo.SaveImage(img); err != nil {
		t.Fatalf("SaveImage() error = %v", err)
	}

	// Verify FTS index has the image
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM images_fts WHERE image_id = ?`, img.ID).Scan(&count)
	if err != nil {
		t.Fatalf("query FTS count: %v", err)
	}
	if count != 1 {
		t.Fatalf("FTS count = %d, want 1 (image should be indexed)", count)
	}
}

func TestFTSDeleteCleansIndex(t *testing.T) {
	t.Parallel()

	db := newFTSTestDB(t)
	imageRepo := NewImageRepository(db)

	// Create image
	img := &domain.Image{
		Path:       "/test/delete_me.png",
		Filename:   "delete_me.png",
		SourceRoot: "/test",
		Format:     "png",
	}
	if _, err := imageRepo.SaveImage(img); err != nil {
		t.Fatalf("SaveImage() error = %v", err)
	}

	// Verify it's in FTS
	var countBefore int
	db.QueryRow(`SELECT COUNT(*) FROM images_fts WHERE image_id = ?`, img.ID).Scan(&countBefore)

	// Delete image
	_, err := db.Exec(`DELETE FROM images WHERE id = ?`, img.ID)
	if err != nil {
		t.Fatalf("delete image: %v", err)
	}

	// Verify FTS index is cleaned
	var countAfter int
	err = db.QueryRow(`SELECT COUNT(*) FROM images_fts WHERE image_id = ?`, img.ID).Scan(&countAfter)
	if err != nil {
		t.Fatalf("query FTS count after delete: %v", err)
	}
	if countAfter != 0 {
		t.Fatalf("FTS count after delete = %d, want 0 (index should be cleaned)", countAfter)
	}
}

func TestFTSSearchReturnsMatchingImageIDs(t *testing.T) {
	t.Parallel()

	db := newFTSTestDB(t)
	imageRepo := NewImageRepository(db)

	// Create images with different names
	images := []*domain.Image{
		{Path: "/cat1.png", Filename: "cute_cat.png", SourceRoot: "/", Format: "png"},
		{Path: "/dog1.png", Filename: "happy_dog.png", SourceRoot: "/", Format: "png"},
		{Path: "/cat2.png", Filename: "sleeping_cat.png", SourceRoot: "/", Format: "png"},
	}
	for _, img := range images {
		if _, err := imageRepo.SaveImage(img); err != nil {
			t.Fatalf("SaveImage() error = %v", err)
		}
	}

	// Update tags for first image
	err := UpdateImageTagsInFTS(db, images[0].ID, "animal pet cute")
	if err != nil {
		t.Fatalf("UpdateImageTagsInFTS() error = %v", err)
	}

	// Search for "cat" in filename
	rows, err := db.Query(`
		SELECT image_id FROM images_fts 
		WHERE images_fts MATCH 'filename:cat*'
	`)
	if err != nil {
		t.Fatalf("FTS search: %v", err)
	}
	defer rows.Close()

	var foundIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan image_id: %v", err)
		}
		foundIDs = append(foundIDs, id)
	}

	if len(foundIDs) != 2 {
		t.Fatalf("len(foundIDs) = %d, want 2 (two cat images)", len(foundIDs))
	}

	// Search for "cute" in tags
	rows2, err := db.Query(`
		SELECT image_id FROM images_fts 
		WHERE images_fts MATCH 'tags:cute'
	`)
	if err != nil {
		t.Fatalf("FTS tag search: %v", err)
	}
	defer rows2.Close()

	var tagFoundIDs []int64
	for rows2.Next() {
		var id int64
		if err := rows2.Scan(&id); err != nil {
			t.Fatalf("scan image_id: %v", err)
		}
		tagFoundIDs = append(tagFoundIDs, id)
	}

	if len(tagFoundIDs) != 1 {
		t.Fatalf("len(tagFoundIDs) = %d, want 1 (one image with 'cute' tag)", len(tagFoundIDs))
	}
	if tagFoundIDs[0] != images[0].ID {
		t.Fatalf("tagFoundIDs[0] = %d, want %d", tagFoundIDs[0], images[0].ID)
	}
}

func TestRebuildFTSIndex(t *testing.T) {
	t.Parallel()

	db := newFTSTestDB(t)
	imageRepo := NewImageRepository(db)

	// Create images
	for i := 0; i < 3; i++ {
		img := &domain.Image{
			Path:       filepath.Join("/test", string(rune('a'+i))+".png"),
			Filename:   string(rune('a' + i)),
			SourceRoot: "/test",
			Format:     "png",
		}
		if _, err := imageRepo.SaveImage(img); err != nil {
			t.Fatalf("SaveImage() error = %v", err)
		}
	}

	// Clear FTS index manually
	_, err := db.Exec(`DELETE FROM images_fts`)
	if err != nil {
		t.Fatalf("clear FTS: %v", err)
	}

	// Rebuild
	if err := RebuildFTSIndex(db); err != nil {
		t.Fatalf("RebuildFTSIndex() error = %v", err)
	}

	// Verify all images are back in FTS
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM images_fts`).Scan(&count)
	if err != nil {
		t.Fatalf("query FTS count: %v", err)
	}
	if count != 3 {
		t.Fatalf("FTS count after rebuild = %d, want 3", count)
	}
}

func newFTSTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "fts-test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	return db
}

func TestImageTagSaveUpdatesFTS(t *testing.T) {
	t.Parallel()

	db := newFTSTestDB(t)
	imageRepo := NewImageRepository(db)
	tagRepo := NewTagRepository(db)
	imageTagRepo := NewImageTagRepository(db)

	// Create image
	img := &domain.Image{
		Path:       "/test/anime.png",
		Filename:   "anime.png",
		SourceRoot: "/test",
		Format:     "png",
	}
	if _, err := imageRepo.SaveImage(img); err != nil {
		t.Fatalf("SaveImage() error = %v", err)
	}

	// Create tag
	tag := &domain.Tag{
		PreferredLabel: "cute",
		Slug:           "cute",
		ReviewState:    "confirmed",
	}
	if err := tagRepo.Save(context.Background(), tag); err != nil {
		t.Fatalf("Save tag: %v", err)
	}

	// Verify FTS tags is empty before adding tag
	var tagsBefore string
	err := db.QueryRow(`SELECT tags FROM images_fts WHERE image_id = ?`, img.ID).Scan(&tagsBefore)
	if err != nil {
		t.Fatalf("query FTS tags: %v", err)
	}
	if tagsBefore != "" {
		t.Fatalf("tags before = %q, want empty", tagsBefore)
	}

	// Add image-tag association
	if err := imageTagRepo.Save(context.Background(), &domain.ImageTag{
		ImageID:     img.ID,
		TagID:       tag.ID,
		ReviewState: "confirmed",
	}); err != nil {
		t.Fatalf("Save image-tag: %v", err)
	}

	// Verify FTS tags is updated
	var tagsAfter string
	err = db.QueryRow(`SELECT tags FROM images_fts WHERE image_id = ?`, img.ID).Scan(&tagsAfter)
	if err != nil {
		t.Fatalf("query FTS tags after save: %v", err)
	}
	if tagsAfter != "cute" {
		t.Fatalf("tags after = %q, want 'cute'", tagsAfter)
	}

	// Search by tag name should find the image
	ids, err := NewSearchRepository(db).FTSFullTextSearch(context.Background(), "cute", 10, 0)
	if err != nil {
		t.Fatalf("FTS search: %v", err)
	}
	if len(ids) != 1 || ids[0] != img.ID {
		t.Fatalf("FTS search results = %v, want [%d]", ids, img.ID)
	}
}

func TestImageTagDeleteUpdatesFTS(t *testing.T) {
	t.Parallel()

	db := newFTSTestDB(t)
	imageRepo := NewImageRepository(db)
	tagRepo := NewTagRepository(db)
	imageTagRepo := NewImageTagRepository(db)

	// Create image
	img := &domain.Image{
		Path:       "/test/anime2.png",
		Filename:   "anime2.png",
		SourceRoot: "/test",
		Format:     "png",
	}
	if _, err := imageRepo.SaveImage(img); err != nil {
		t.Fatalf("SaveImage() error = %v", err)
	}

	// Create tag
	tag := &domain.Tag{
		PreferredLabel: "animal",
		Slug:           "animal",
		ReviewState:    "confirmed",
	}
	if err := tagRepo.Save(context.Background(), tag); err != nil {
		t.Fatalf("Save tag: %v", err)
	}

	// Add image-tag association
	if err := imageTagRepo.Save(context.Background(), &domain.ImageTag{
		ImageID:     img.ID,
		TagID:       tag.ID,
		ReviewState: "confirmed",
	}); err != nil {
		t.Fatalf("Save image-tag: %v", err)
	}

	// Verify tag is in FTS
	var tagsBefore string
	db.QueryRow(`SELECT tags FROM images_fts WHERE image_id = ?`, img.ID).Scan(&tagsBefore)
	if tagsBefore != "animal" {
		t.Fatalf("tags before delete = %q, want 'animal'", tagsBefore)
	}

	// Delete image-tag association
	if _, err := imageTagRepo.Delete(context.Background(), img.ID, tag.ID); err != nil {
		t.Fatalf("Delete image-tag: %v", err)
	}

	// Verify FTS tags is cleared
	var tagsAfter string
	err := db.QueryRow(`SELECT tags FROM images_fts WHERE image_id = ?`, img.ID).Scan(&tagsAfter)
	if err != nil {
		t.Fatalf("query FTS tags after delete: %v", err)
	}
	if tagsAfter != "" {
		t.Fatalf("tags after delete = %q, want empty", tagsAfter)
	}
}
