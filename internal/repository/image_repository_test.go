package repository

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

func TestImageRepositoryFindByTagIDsFiltersByStandardTags(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	// Create tags
	tagRepo := NewTagRepository(db)
	tags := []*domain.Tag{
		{PreferredLabel: "blue sky", Slug: "blue-sky", ReviewState: "confirmed", UsageCount: 5},
		{PreferredLabel: "sunset", Slug: "sunset", ReviewState: "confirmed", UsageCount: 3},
		{PreferredLabel: "ocean", Slug: "ocean", ReviewState: "confirmed", UsageCount: 1},
	}
	for _, tag := range tags {
		if err := tagRepo.Save(ctx, tag); err != nil {
			t.Fatalf("save tag: %v", err)
		}
	}

	// Create images
	images := []*domain.Image{
		{Path: "/img1.png", Filename: "img1.png", SourceRoot: "/", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Path: "/img2.png", Filename: "img2.png", SourceRoot: "/", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Path: "/img3.png", Filename: "img3.png", SourceRoot: "/", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	for _, img := range images {
		if _, err := repo.SaveImage(img); err != nil {
			t.Fatalf("save image: %v", err)
		}
	}

	// Create image-tag associations (AND semantics: img1 has both tag1 and tag2)
	imageTagRepo := NewImageTagRepository(db)
	associations := []*domain.ImageTag{
		{ImageID: images[0].ID, TagID: tags[0].ID, ReviewState: "confirmed"},
		{ImageID: images[0].ID, TagID: tags[1].ID, ReviewState: "confirmed"},
		{ImageID: images[1].ID, TagID: tags[0].ID, ReviewState: "confirmed"}, // only tag1
		{ImageID: images[2].ID, TagID: tags[2].ID, ReviewState: "confirmed"}, // tag3 only
	}
	for _, assoc := range associations {
		if err := imageTagRepo.Save(ctx, assoc); err != nil {
			t.Fatalf("save image-tag: %v", err)
		}
	}

	// Test 1: Filter by tag_ids=1,2 should return only img1 (has both tags - AND semantics)
	filtered, err := repo.FindByTagIDs(ctx, []int64{tags[0].ID, tags[1].ID}, 10, 0, "id", "asc")
	if err != nil {
		t.Fatalf("FindByTagIDs() error = %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("len(filtered) = %d, want 1 (only images with ALL requested tags)", len(filtered))
	}
	if filtered[0].ID != images[0].ID {
		t.Fatalf("filtered[0].ID = %d, want %d", filtered[0].ID, images[0].ID)
	}

	// Test 2: Filter by single tag should return images with that tag
	filteredSingle, err := repo.FindByTagIDs(ctx, []int64{tags[0].ID}, 10, 0, "id", "asc")
	if err != nil {
		t.Fatalf("FindByTagIDs() error = %v", err)
	}
	if len(filteredSingle) != 2 {
		t.Fatalf("len(filteredSingle) = %d, want 2 (images with tag1)", len(filteredSingle))
	}

	// Test 3: Empty tag_ids should return empty (no filter = no results per API contract)
	filteredEmpty, err := repo.FindByTagIDs(ctx, []int64{}, 10, 0, "id", "asc")
	if err != nil {
		t.Fatalf("FindByTagIDs() error = %v", err)
	}
	if len(filteredEmpty) != 0 {
		t.Fatalf("len(filteredEmpty) = %d, want 0", len(filteredEmpty))
	}
}

func TestImageRepositoryFindByTagIDSSupportsPagination(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	// Create tag
	tagRepo := NewTagRepository(db)
	tag := &domain.Tag{PreferredLabel: "test", Slug: "test", ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, tag); err != nil {
		t.Fatalf("save tag: %v", err)
	}

	// Create 3 images with the tag
	imageTagRepo := NewImageTagRepository(db)
	for i := 0; i < 3; i++ {
		img := &domain.Image{
			Path:       filepath.Join("/img", string(rune('a'+i))),
			Filename:   string(rune('a' + i)),
			SourceRoot: "/img",
			Format:     "png",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if _, err := repo.SaveImage(img); err != nil {
			t.Fatalf("save image: %v", err)
		}
		if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: img.ID, TagID: tag.ID, ReviewState: "confirmed"}); err != nil {
			t.Fatalf("save image-tag: %v", err)
		}
	}

	// Test pagination: limit 2, offset 0
	page1, err := repo.FindByTagIDs(ctx, []int64{tag.ID}, 2, 0, "id", "asc")
	if err != nil {
		t.Fatalf("FindByTagIDs() error = %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("len(page1) = %d, want 2", len(page1))
	}

	// Test pagination: limit 2, offset 2
	page2, err := repo.FindByTagIDs(ctx, []int64{tag.ID}, 2, 2, "id", "asc")
	if err != nil {
		t.Fatalf("FindByTagIDs() error = %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("len(page2) = %d, want 1", len(page2))
	}
}

func TestImageRepositoryCountByTagIDsReturnsCorrectCount(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	// Create tags
	tagRepo := NewTagRepository(db)
	tag1 := &domain.Tag{PreferredLabel: "tag1", Slug: "tag1", ReviewState: "confirmed"}
	tag2 := &domain.Tag{PreferredLabel: "tag2", Slug: "tag2", ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, tag1); err != nil {
		t.Fatalf("save tag1: %v", err)
	}
	if err := tagRepo.Save(ctx, tag2); err != nil {
		t.Fatalf("save tag2: %v", err)
	}

	// Create images with tags
	imageTagRepo := NewImageTagRepository(db)
	for i := 0; i < 3; i++ {
		img := &domain.Image{
			Path:       filepath.Join("/img", string(rune('a'+i))),
			Filename:   string(rune('a' + i)),
			SourceRoot: "/img",
			Format:     "png",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if _, err := repo.SaveImage(img); err != nil {
			t.Fatalf("save image: %v", err)
		}
		// img0 and img1 have tag1; img2 does not
		if i < 2 {
			if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: img.ID, TagID: tag1.ID, ReviewState: "confirmed"}); err != nil {
				t.Fatalf("save image-tag: %v", err)
			}
		}
	}

	// Count images with tag1
	count, err := repo.CountByTagIDs(ctx, []int64{tag1.ID})
	if err != nil {
		t.Fatalf("CountByTagIDs() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}

	// Count with empty tag_ids
	countEmpty, err := repo.CountByTagIDs(ctx, []int64{})
	if err != nil {
		t.Fatalf("CountByTagIDs() error = %v", err)
	}
	if countEmpty != 0 {
		t.Fatalf("countEmpty = %d, want 0", countEmpty)
	}
}

func TestImageRepositoryFindByTagIDsExpandsSelectedAncestors(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()
	tagRepo := NewTagRepository(db)
	imageTagRepo := NewImageTagRepository(db)

	root := &domain.Tag{PreferredLabel: "character", Slug: "character", Level: domain.TagLevelRoot}
	if err := tagRepo.Save(ctx, root); err != nil {
		t.Fatalf("save root: %v", err)
	}
	parent := &domain.Tag{PreferredLabel: "protagonist", Slug: "protagonist", Level: domain.TagLevelParent, ParentID: &root.ID}
	if err := tagRepo.Save(ctx, parent); err != nil {
		t.Fatalf("save parent: %v", err)
	}
	child := &domain.Tag{PreferredLabel: "heroine", Slug: "heroine", Level: domain.TagLevelChild, ParentID: &parent.ID}
	if err := tagRepo.Save(ctx, child); err != nil {
		t.Fatalf("save child: %v", err)
	}

	images := []*domain.Image{
		{Path: "/hierarchy/1.png", Filename: "1.png", SourceRoot: "/hierarchy", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Path: "/hierarchy/2.png", Filename: "2.png", SourceRoot: "/hierarchy", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	for _, img := range images {
		if _, err := repo.SaveImage(img); err != nil {
			t.Fatalf("save image: %v", err)
		}
	}

	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: images[0].ID, TagID: child.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("save child tag image: %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: images[1].ID, TagID: parent.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("save parent tag image: %v", err)
	}

	filtered, err := repo.FindByTagIDs(ctx, []int64{root.ID}, 10, 0, "id", "asc")
	if err != nil {
		t.Fatalf("FindByTagIDs(root) error = %v", err)
	}
	if len(filtered) != 2 {
		t.Fatalf("len(filtered) = %d, want 2", len(filtered))
	}

	count, err := repo.CountByTagIDs(ctx, []int64{root.ID})
	if err != nil {
		t.Fatalf("CountByTagIDs(root) error = %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
}

func TestImageRepositoryFindByTagIDsKeepsExpandedAndSemantics(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()
	tagRepo := NewTagRepository(db)
	imageTagRepo := NewImageTagRepository(db)

	rootA := &domain.Tag{PreferredLabel: "series", Slug: "series", Level: domain.TagLevelRoot}
	rootB := &domain.Tag{PreferredLabel: "mood", Slug: "mood", Level: domain.TagLevelRoot}
	if err := tagRepo.Save(ctx, rootA); err != nil {
		t.Fatalf("save rootA: %v", err)
	}
	if err := tagRepo.Save(ctx, rootB); err != nil {
		t.Fatalf("save rootB: %v", err)
	}
	parentA := &domain.Tag{PreferredLabel: "cast", Slug: "cast", Level: domain.TagLevelParent, ParentID: &rootA.ID}
	parentB := &domain.Tag{PreferredLabel: "tone", Slug: "tone", Level: domain.TagLevelParent, ParentID: &rootB.ID}
	childA := &domain.Tag{PreferredLabel: "lead", Slug: "lead", Level: domain.TagLevelChild, ParentID: &parentA.ID}
	childB := &domain.Tag{PreferredLabel: "calm", Slug: "calm", Level: domain.TagLevelChild, ParentID: &parentB.ID}
	if err := tagRepo.Save(ctx, parentA); err != nil {
		t.Fatalf("save parentA: %v", err)
	}
	if err := tagRepo.Save(ctx, parentB); err != nil {
		t.Fatalf("save parentB: %v", err)
	}
	if err := tagRepo.Save(ctx, childA); err != nil {
		t.Fatalf("save childA: %v", err)
	}
	if err := tagRepo.Save(ctx, childB); err != nil {
		t.Fatalf("save childB: %v", err)
	}

	images := []*domain.Image{
		{Path: "/expanded/1.png", Filename: "1.png", SourceRoot: "/expanded", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Path: "/expanded/2.png", Filename: "2.png", SourceRoot: "/expanded", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Path: "/expanded/3.png", Filename: "3.png", SourceRoot: "/expanded", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	for _, img := range images {
		if _, err := repo.SaveImage(img); err != nil {
			t.Fatalf("save image: %v", err)
		}
	}

	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: images[0].ID, TagID: childA.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("save image0 childA: %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: images[0].ID, TagID: childB.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("save image0 childB: %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: images[1].ID, TagID: childA.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("save image1 childA: %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: images[2].ID, TagID: childB.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("save image2 childB: %v", err)
	}

	filtered, err := repo.FindByTagIDs(ctx, []int64{rootA.ID, rootB.ID}, 10, 0, "id", "asc")
	if err != nil {
		t.Fatalf("FindByTagIDs(expanded AND) error = %v", err)
	}
	if len(filtered) != 1 || filtered[0].ID != images[0].ID {
		t.Fatalf("unexpected filtered images: %+v", filtered)
	}

	count, err := repo.CountByTagIDs(ctx, []int64{rootA.ID, rootB.ID})
	if err != nil {
		t.Fatalf("CountByTagIDs(expanded AND) error = %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func newImageRepositoryTestDB(t *testing.T) (*sql.DB, ImageRepository) {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "image-repo.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	return db, NewImageRepository(db)
}

func TestFindUntaggedReturnsOnlyImagesWithoutTags(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	// Create tags
	tagRepo := NewTagRepository(db)
	tag := &domain.Tag{PreferredLabel: "test", Slug: "test", ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, tag); err != nil {
		t.Fatalf("save tag: %v", err)
	}

	// Create 3 images
	images := []*domain.Image{
		{Path: "/img1.png", Filename: "img1.png", SourceRoot: "/", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Path: "/img2.png", Filename: "img2.png", SourceRoot: "/", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Path: "/img3.png", Filename: "img3.png", SourceRoot: "/", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	for _, img := range images {
		if _, err := repo.SaveImage(img); err != nil {
			t.Fatalf("save image: %v", err)
		}
	}

	// Tag img1 and img2, leave img3 untagged
	imageTagRepo := NewImageTagRepository(db)
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: images[0].ID, TagID: tag.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("save image-tag: %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: images[1].ID, TagID: tag.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("save image-tag: %v", err)
	}

	// Test: FindUntagged should return only img3
	untagged, err := repo.FindUntagged(ctx, 10, 0, "id", "asc")
	if err != nil {
		t.Fatalf("FindUntagged() error = %v", err)
	}
	if len(untagged) != 1 {
		t.Fatalf("len(untagged) = %d, want 1", len(untagged))
	}
	if untagged[0].ID != images[2].ID {
		t.Fatalf("untagged[0].ID = %d, want %d", untagged[0].ID, images[2].ID)
	}
}

func TestFindUntaggedIgnoresRejectedTags(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	rejectedOnly := saveImageForAITagSelectionTest(t, repo, "/untagged-rejected.png", "/thumb/untagged-rejected.jpg")
	confirmed := saveImageForAITagSelectionTest(t, repo, "/untagged-confirmed.png", "/thumb/untagged-confirmed.jpg")

	addManualTagForSelectionTest(t, db, rejectedOnly.ID, "rejected")
	addManualTagForSelectionTest(t, db, confirmed.ID, "confirmed")

	untagged, err := repo.FindUntagged(ctx, 10, 0, "id", "asc")
	if err != nil {
		t.Fatalf("FindUntagged() error = %v", err)
	}
	if !containsImageID(untagged, rejectedOnly.ID) {
		t.Fatalf("expected image %d with only rejected tags to be treated as untagged, got %+v", rejectedOnly.ID, imageIDs(untagged))
	}
	if containsImageID(untagged, confirmed.ID) {
		t.Fatalf("image %d with confirmed tag should not be untagged, got %+v", confirmed.ID, imageIDs(untagged))
	}

	count, err := repo.CountUntagged(ctx)
	if err != nil {
		t.Fatalf("CountUntagged() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("CountUntagged() = %d, want 1", count)
	}
}

func TestFindImagesWithoutAITagsTreatsRejectedAITagsAsAbsent(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	rejectedOnly := saveImageForAITagSelectionTest(t, repo, "/rejected-only.png", "/thumb/rejected-only.jpg")
	pending := saveImageForAITagSelectionTest(t, repo, "/pending.png", "/thumb/pending.jpg")
	confirmed := saveImageForAITagSelectionTest(t, repo, "/confirmed.png", "/thumb/confirmed.jpg")

	addAITagForSelectionTest(t, db, rejectedOnly.ID, "rejected")
	addAITagForSelectionTest(t, db, pending.ID, "pending")
	addAITagForSelectionTest(t, db, confirmed.ID, "confirmed")

	images, err := repo.FindImagesWithoutAITags(ctx, 10)
	if err != nil {
		t.Fatalf("FindImagesWithoutAITags() error = %v", err)
	}

	if !containsImageID(images, rejectedOnly.ID) {
		t.Fatalf("expected rejected-only image %d to remain eligible, got %+v", rejectedOnly.ID, imageIDs(images))
	}
	if containsImageID(images, pending.ID) {
		t.Fatalf("pending AI-tagged image %d should not be eligible, got %+v", pending.ID, imageIDs(images))
	}
	if containsImageID(images, confirmed.ID) {
		t.Fatalf("confirmed AI-tagged image %d should not be eligible, got %+v", confirmed.ID, imageIDs(images))
	}
}

func TestBackfillCandidateQueriesIgnoreRejectedAITags(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()
	filter := BackfillCandidateFilter{}

	rejectedOnly := saveImageForAITagSelectionTest(t, repo, "/backfill-rejected.png", "/thumb/backfill-rejected.jpg")
	pending := saveImageForAITagSelectionTest(t, repo, "/backfill-pending.png", "/thumb/backfill-pending.jpg")

	addAITagForSelectionTest(t, db, rejectedOnly.ID, "rejected")
	addAITagForSelectionTest(t, db, pending.ID, "pending")

	candidates, err := repo.FindBackfillCandidates(ctx, filter)
	if err != nil {
		t.Fatalf("FindBackfillCandidates() error = %v", err)
	}
	if !containsImageID(candidates, rejectedOnly.ID) {
		t.Fatalf("expected rejected-only image %d to be backfill candidate, got %+v", rejectedOnly.ID, imageIDs(candidates))
	}
	if containsImageID(candidates, pending.ID) {
		t.Fatalf("pending AI-tagged image %d should not be backfill candidate, got %+v", pending.ID, imageIDs(candidates))
	}

	skippedWithAITag, err := repo.CountBackfillSkippedWithAITag(ctx, filter)
	if err != nil {
		t.Fatalf("CountBackfillSkippedWithAITag() error = %v", err)
	}
	if skippedWithAITag != 1 {
		t.Fatalf("skippedWithAITag = %d, want 1 (pending only)", skippedWithAITag)
	}
}

func TestImage_PHashHexColumn(t *testing.T) {
	t.Parallel()

	db, _ := newImageRepositoryTestDB(t)
	now := time.Now()
	phashHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	result, err := db.Exec(`
		INSERT INTO images (
			path, filename, source_root, file_size, width, height, format, phash, phash_hex, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "/tmp/phash-hex-test.png", "phash-hex-test.png", "/tmp", 2048, 1920, 1080, "png", 12345, phashHex, now, now)
	if err != nil {
		t.Fatalf("insert image with phash_hex: %v", err)
	}

	imageID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}

	var got string
	if err := db.QueryRow(`SELECT phash_hex FROM images WHERE id = ?`, imageID).Scan(&got); err != nil {
		t.Fatalf("select phash_hex: %v", err)
	}

	if got != phashHex {
		t.Fatalf("phash_hex = %q, want %q", got, phashHex)
	}
}

func TestImage_HashCacheColumns(t *testing.T) {
	t.Parallel()

	db, _ := newImageRepositoryTestDB(t)
	now := time.Now()
	sha256 := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	phashHex := "bbbbbbbbbbbbbbbb"
	sourceMTimeUnix := int64(1710000000123456789)

	result, err := db.Exec(`
		INSERT INTO images (
			path, filename, source_root, file_size, width, height, format, phash, phash_hex, sha256, source_mtime_unix, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "/tmp/hash-cache-test.png", "hash-cache-test.png", "/tmp", 2048, 1920, 1080, "png", 12345, phashHex, sha256, sourceMTimeUnix, now, now)
	if err != nil {
		t.Fatalf("insert image with hash cache columns: %v", err)
	}

	imageID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}

	var gotSHA256 string
	var gotSourceMTime int64
	if err := db.QueryRow(`SELECT sha256, source_mtime_unix FROM images WHERE id = ?`, imageID).Scan(&gotSHA256, &gotSourceMTime); err != nil {
		t.Fatalf("select hash cache columns: %v", err)
	}

	if gotSHA256 != sha256 {
		t.Fatalf("sha256 = %q, want %q", gotSHA256, sha256)
	}
	if gotSourceMTime != sourceMTimeUnix {
		t.Fatalf("source_mtime_unix = %d, want %d", gotSourceMTime, sourceMTimeUnix)
	}
}

func TestEnsureScanSchema_ImageHashCacheMigrationKeepsLegacyRowsReadable(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "legacy-image-repo.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`
		CREATE TABLE images (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT UNIQUE NOT NULL,
			filename TEXT NOT NULL,
			source_root TEXT NOT NULL,
			file_size INTEGER,
			width INTEGER,
			height INTEGER,
			format TEXT,
			phash INTEGER,
			phash_hex TEXT,
			thumbnail_small_url TEXT,
			thumbnail_large_url TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		t.Fatalf("create legacy images table: %v", err)
	}

	legacyNow := time.Now()
	_, err = db.Exec(`
		INSERT INTO images(path, filename, source_root, file_size, width, height, format, phash, phash_hex, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "/legacy.png", "legacy.png", "/", 100, 10, 10, "png", 0, "", legacyNow, legacyNow)
	if err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() migration error = %v", err)
	}

	repo := NewImageRepository(db)
	found, err := repo.FindByPath("/legacy.png")
	if err != nil {
		t.Fatalf("FindByPath after migration: %v", err)
	}

	if found.SHA256 != "" {
		t.Fatalf("SHA256 = %q, want empty default", found.SHA256)
	}
	if found.SourceMTimeUnix != 0 {
		t.Fatalf("SourceMTimeUnix = %d, want 0 default", found.SourceMTimeUnix)
	}
	if found.PHashHex != "" {
		t.Fatalf("PHashHex = %q, want empty string", found.PHashHex)
	}
}

func saveImageForAITagSelectionTest(t *testing.T, repo ImageRepository, path, thumbnailSmallURL string) *domain.Image {
	t.Helper()

	image := &domain.Image{
		Path:              path,
		Filename:          filepath.Base(path),
		SourceRoot:        "/",
		Format:            "png",
		ThumbnailSmallUrl: thumbnailSmallURL,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	if _, err := repo.SaveImage(image); err != nil {
		t.Fatalf("save image: %v", err)
	}
	return image
}

func addAITagForSelectionTest(t *testing.T, db *sql.DB, imageID int64, reviewState string) {
	t.Helper()
	addTagForSelectionTest(t, db, imageID, domain.ImageTagSourceAI, reviewState)
}

func addManualTagForSelectionTest(t *testing.T, db *sql.DB, imageID int64, reviewState string) {
	t.Helper()
	addTagForSelectionTest(t, db, imageID, domain.ImageTagSourceManual, reviewState)
}

func addTagForSelectionTest(t *testing.T, db *sql.DB, imageID int64, source, reviewState string) {
	t.Helper()

	ctx := context.Background()
	tagRepo := NewTagRepository(db)
	tag := &domain.Tag{
		PreferredLabel: fmt.Sprintf("ai-%s-%d", reviewState, time.Now().UnixNano()),
		Slug:           fmt.Sprintf("ai-%s-%d", reviewState, time.Now().UnixNano()),
		ReviewState:    "confirmed",
	}
	if err := tagRepo.Save(ctx, tag); err != nil {
		t.Fatalf("save tag: %v", err)
	}

	imageTagRepo := NewImageTagRepository(db)
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{
		ImageID:     imageID,
		TagID:       tag.ID,
		Source:      source,
		ReviewState: reviewState,
		Confidence:  0.9,
	}); err != nil {
		t.Fatalf("save image-tag: %v", err)
	}
}

func containsImageID(images []domain.Image, target int64) bool {
	for _, image := range images {
		if image.ID == target {
			return true
		}
	}
	return false
}

func imageIDs(images []domain.Image) []int64 {
	ids := make([]int64, 0, len(images))
	for _, image := range images {
		ids = append(ids, image.ID)
	}
	return ids
}

func TestUpdateImagePHashHex(t *testing.T) {
	t.Parallel()

	_, repo := newImageRepositoryTestDB(t)

	image := &domain.Image{
		Path:       "/tmp/update-phash-hex.png",
		Filename:   "update-phash-hex.png",
		SourceRoot: "/tmp",
		Format:     "png",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if _, err := repo.SaveImage(image); err != nil {
		t.Fatalf("save image: %v", err)
	}

	phashHex := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	if err := repo.UpdateImagePHashHex(image.ID, phashHex); err != nil {
		t.Fatalf("UpdateImagePHashHex failed: %v", err)
	}

	found, err := repo.FindByID(image.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}

	if found.PHashHex != phashHex {
		t.Fatalf("PHashHex = %q, want %q", found.PHashHex, phashHex)
	}

	all, err := repo.FindAll(10, 0, "id", "asc")
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("FindAll returned no images")
	}
	if all[0].PHashHex != phashHex {
		t.Fatalf("FindAll[0].PHashHex = %q, want %q", all[0].PHashHex, phashHex)
	}
}

func TestUpdateImageDuplicateHashCache(t *testing.T) {
	t.Parallel()

	_, repo := newImageRepositoryTestDB(t)

	image := &domain.Image{
		Path:       "/tmp/update-hash-cache.png",
		Filename:   "update-hash-cache.png",
		SourceRoot: "/tmp",
		Format:     "png",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if _, err := repo.SaveImage(image); err != nil {
		t.Fatalf("save image: %v", err)
	}

	sha256 := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	phashHex := "dddddddddddddddd"
	sourceMTimeUnix := int64(1710000000765432100)

	if err := repo.UpdateImageDuplicateHashCache(image.ID, sha256, phashHex, sourceMTimeUnix); err != nil {
		t.Fatalf("UpdateImageDuplicateHashCache failed: %v", err)
	}

	found, err := repo.FindByID(image.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}

	if found.SHA256 != sha256 {
		t.Fatalf("SHA256 = %q, want %q", found.SHA256, sha256)
	}
	if found.PHashHex != phashHex {
		t.Fatalf("PHashHex = %q, want %q", found.PHashHex, phashHex)
	}
	if found.SourceMTimeUnix != sourceMTimeUnix {
		t.Fatalf("SourceMTimeUnix = %d, want %d", found.SourceMTimeUnix, sourceMTimeUnix)
	}
}

func TestFindByIDRange_ReturnsPagedAscendingResults(t *testing.T) {
	t.Parallel()

	_, repo := newImageRepositoryTestDB(t)

	inserted := make([]*domain.Image, 0, 5)
	for i := 0; i < 5; i++ {
		img := &domain.Image{
			Path:       filepath.Join("/range", fmt.Sprintf("img-%d.png", i)),
			Filename:   fmt.Sprintf("img-%d.png", i),
			SourceRoot: "/range",
			Format:     "png",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if _, err := repo.SaveImage(img); err != nil {
			t.Fatalf("SaveImage(%d): %v", i, err)
		}
		inserted = append(inserted, img)
	}

	page1, err := repo.FindByIDRange(2, 0)
	if err != nil {
		t.Fatalf("FindByIDRange page1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("len(page1) = %d, want 2", len(page1))
	}
	if page1[0].ID != inserted[0].ID || page1[1].ID != inserted[1].ID {
		t.Fatalf("unexpected page1 IDs: got [%d,%d], want [%d,%d]", page1[0].ID, page1[1].ID, inserted[0].ID, inserted[1].ID)
	}

	page2, err := repo.FindByIDRange(2, page1[len(page1)-1].ID)
	if err != nil {
		t.Fatalf("FindByIDRange page2: %v", err)
	}
	if len(page2) != 2 {
		t.Fatalf("len(page2) = %d, want 2", len(page2))
	}
	if page2[0].ID != inserted[2].ID || page2[1].ID != inserted[3].ID {
		t.Fatalf("unexpected page2 IDs: got [%d,%d], want [%d,%d]", page2[0].ID, page2[1].ID, inserted[2].ID, inserted[3].ID)
	}

	page3, err := repo.FindByIDRange(2, page2[len(page2)-1].ID)
	if err != nil {
		t.Fatalf("FindByIDRange page3: %v", err)
	}
	if len(page3) != 1 {
		t.Fatalf("len(page3) = %d, want 1", len(page3))
	}
	if page3[0].ID != inserted[4].ID {
		t.Fatalf("page3[0].ID = %d, want %d", page3[0].ID, inserted[4].ID)
	}
}

func TestFindBySourceRootsAfterID_ReturnsFilteredPagedResults(t *testing.T) {
	t.Parallel()

	_, repo := newImageRepositoryTestDB(t)

	insert := func(path, root string) *domain.Image {
		img := &domain.Image{
			Path:       path,
			Filename:   filepath.Base(path),
			SourceRoot: root,
			Format:     "png",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if _, err := repo.SaveImage(img); err != nil {
			t.Fatalf("SaveImage(%s): %v", path, err)
		}
		return img
	}

	a1 := insert(filepath.Join("/a", "1.png"), "/a")
	a2 := insert(filepath.Join("/a", "2.png"), "/a")
	_ = insert(filepath.Join("/b", "1.png"), "/b")
	a3 := insert(filepath.Join("/a", "3.png"), "/a")

	page1, err := repo.FindBySourceRootsAfterID(2, 0, []string{"/a"})
	if err != nil {
		t.Fatalf("FindBySourceRootsAfterID page1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("len(page1) = %d, want 2", len(page1))
	}
	if page1[0].ID != a1.ID || page1[1].ID != a2.ID {
		t.Fatalf("unexpected page1 IDs: got [%d,%d], want [%d,%d]", page1[0].ID, page1[1].ID, a1.ID, a2.ID)
	}

	page2, err := repo.FindBySourceRootsAfterID(2, page1[len(page1)-1].ID, []string{"/a"})
	if err != nil {
		t.Fatalf("FindBySourceRootsAfterID page2: %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("len(page2) = %d, want 1", len(page2))
	}
	if page2[0].ID != a3.ID {
		t.Fatalf("page2[0].ID = %d, want %d", page2[0].ID, a3.ID)
	}
}

func TestFindUntaggedSupportsPagination(t *testing.T) {
	t.Parallel()

	_, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	// Create 5 untagged images
	for i := 0; i < 5; i++ {
		img := &domain.Image{
			Path:       filepath.Join("/img", string(rune('a'+i))),
			Filename:   string(rune('a'+i)) + ".png",
			SourceRoot: "/img",
			Format:     "png",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if _, err := repo.SaveImage(img); err != nil {
			t.Fatalf("save image: %v", err)
		}
	}

	// Test pagination: limit 2, offset 0
	page1, err := repo.FindUntagged(ctx, 2, 0, "id", "asc")
	if err != nil {
		t.Fatalf("FindUntagged() error = %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("len(page1) = %d, want 2", len(page1))
	}

	// Test pagination: limit 2, offset 2
	page2, err := repo.FindUntagged(ctx, 2, 2, "id", "asc")
	if err != nil {
		t.Fatalf("FindUntagged() error = %v", err)
	}
	if len(page2) != 2 {
		t.Fatalf("len(page2) = %d, want 2", len(page2))
	}

	// Test pagination: limit 2, offset 4
	page3, err := repo.FindUntagged(ctx, 2, 4, "id", "asc")
	if err != nil {
		t.Fatalf("FindUntagged() error = %v", err)
	}
	if len(page3) != 1 {
		t.Fatalf("len(page3) = %d, want 1", len(page3))
	}
}

func TestFindUntaggedSupportsSorting(t *testing.T) {
	t.Parallel()

	_, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	// Create images with different file sizes
	sizes := []int{100, 200, 300}
	for i, size := range sizes {
		img := &domain.Image{
			Path:       filepath.Join("/img", string(rune('a'+i))),
			Filename:   string(rune('a'+i)) + ".png",
			SourceRoot: "/img",
			FileSize:   int64(size),
			Format:     "png",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if _, err := repo.SaveImage(img); err != nil {
			t.Fatalf("save image: %v", err)
		}
	}

	// Test sorting by file_size descending
	desc, err := repo.FindUntagged(ctx, 10, 0, "file_size", "desc")
	if err != nil {
		t.Fatalf("FindUntagged() error = %v", err)
	}
	if len(desc) != 3 {
		t.Fatalf("len(desc) = %d, want 3", len(desc))
	}
	if desc[0].FileSize != 300 {
		t.Fatalf("desc[0].FileSize = %d, want 300 (largest first)", desc[0].FileSize)
	}

	// Test sorting by file_size ascending
	asc, err := repo.FindUntagged(ctx, 10, 0, "file_size", "asc")
	if err != nil {
		t.Fatalf("FindUntagged() error = %v", err)
	}
	if asc[0].FileSize != 100 {
		t.Fatalf("asc[0].FileSize = %d, want 100 (smallest first)", asc[0].FileSize)
	}
}

func TestImageRepositorySortingUsesIDTieBreaker(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()
	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)

	images := []*domain.Image{
		{Path: "/img/a.png", Filename: "same.png", SourceRoot: "/img", FileSize: 100, Format: "png", CreatedAt: now, UpdatedAt: now},
		{Path: "/img/b.png", Filename: "same.png", SourceRoot: "/img", FileSize: 100, Format: "png", CreatedAt: now, UpdatedAt: now},
		{Path: "/img/c.png", Filename: "same.png", SourceRoot: "/img", FileSize: 100, Format: "png", CreatedAt: now, UpdatedAt: now},
	}
	for _, img := range images {
		if _, err := repo.SaveImage(img); err != nil {
			t.Fatalf("save image: %v", err)
		}
	}

	tagRepo := NewTagRepository(db)
	tag := &domain.Tag{PreferredLabel: "viewer", Slug: "viewer", ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, tag); err != nil {
		t.Fatalf("save tag: %v", err)
	}
	imageTagRepo := NewImageTagRepository(db)
	for _, img := range images[:2] {
		if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: img.ID, TagID: tag.ID, ReviewState: "confirmed"}); err != nil {
			t.Fatalf("save image tag: %v", err)
		}
	}

	assertIDs := func(label string, got []domain.Image, want []int64) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s len = %d, want %d", label, len(got), len(want))
		}
		for i, image := range got {
			if image.ID != want[i] {
				t.Fatalf("%s[%d].ID = %d, want %d", label, i, image.ID, want[i])
			}
		}
	}

	allAsc, err := repo.FindAll(10, 0, "created_at", "asc")
	if err != nil {
		t.Fatalf("FindAll asc: %v", err)
	}
	assertIDs("FindAll asc", allAsc, []int64{images[0].ID, images[1].ID, images[2].ID})

	allDesc, err := repo.FindAll(10, 0, "created_at", "desc")
	if err != nil {
		t.Fatalf("FindAll desc: %v", err)
	}
	assertIDs("FindAll desc", allDesc, []int64{images[2].ID, images[1].ID, images[0].ID})

	tagAsc, err := repo.FindByTagIDs(ctx, []int64{tag.ID}, 10, 0, "file_size", "asc")
	if err != nil {
		t.Fatalf("FindByTagIDs asc: %v", err)
	}
	assertIDs("FindByTagIDs asc", tagAsc, []int64{images[0].ID, images[1].ID})

	tagDesc, err := repo.FindByTagIDs(ctx, []int64{tag.ID}, 10, 0, "file_size", "desc")
	if err != nil {
		t.Fatalf("FindByTagIDs desc: %v", err)
	}
	assertIDs("FindByTagIDs desc", tagDesc, []int64{images[1].ID, images[0].ID})

	untaggedAsc, err := repo.FindUntagged(ctx, 10, 0, "filename", "asc")
	if err != nil {
		t.Fatalf("FindUntagged asc: %v", err)
	}
	assertIDs("FindUntagged asc", untaggedAsc, []int64{images[2].ID})
}

func TestCountUntaggedReturnsCorrectCount(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	// Create tags
	tagRepo := NewTagRepository(db)
	tag := &domain.Tag{PreferredLabel: "test", Slug: "test", ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, tag); err != nil {
		t.Fatalf("save tag: %v", err)
	}

	// Create 5 images
	for i := 0; i < 5; i++ {
		img := &domain.Image{
			Path:       filepath.Join("/img", string(rune('a'+i))),
			Filename:   string(rune('a'+i)) + ".png",
			SourceRoot: "/img",
			Format:     "png",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if _, err := repo.SaveImage(img); err != nil {
			t.Fatalf("save image: %v", err)
		}
	}

	// Tag first 2 images
	imageTagRepo := NewImageTagRepository(db)
	imgs, _ := repo.FindAll(10, 0, "id", "asc")
	for i := 0; i < 2; i++ {
		if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: imgs[i].ID, TagID: tag.ID, ReviewState: "confirmed"}); err != nil {
			t.Fatalf("save image-tag: %v", err)
		}
	}

	// Count untagged: should be 3
	count, err := repo.CountUntagged(ctx)
	if err != nil {
		t.Fatalf("CountUntagged() error = %v", err)
	}
	if count != 3 {
		t.Fatalf("count = %d, want 3", count)
	}
}

func TestFindUntaggedReturnsEmptyWhenAllImagesHaveTags(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	// Create tag
	tagRepo := NewTagRepository(db)
	tag := &domain.Tag{PreferredLabel: "test", Slug: "test", ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, tag); err != nil {
		t.Fatalf("save tag: %v", err)
	}

	// Create 3 images and tag all of them
	imageTagRepo := NewImageTagRepository(db)
	for i := 0; i < 3; i++ {
		img := &domain.Image{
			Path:       filepath.Join("/img", string(rune('a'+i))),
			Filename:   string(rune('a'+i)) + ".png",
			SourceRoot: "/img",
			Format:     "png",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if _, err := repo.SaveImage(img); err != nil {
			t.Fatalf("save image: %v", err)
		}
		if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: img.ID, TagID: tag.ID, ReviewState: "confirmed"}); err != nil {
			t.Fatalf("save image-tag: %v", err)
		}
	}

	// Test: should return empty
	untagged, err := repo.FindUntagged(ctx, 10, 0, "id", "asc")
	if err != nil {
		t.Fatalf("FindUntagged() error = %v", err)
	}
	if len(untagged) != 0 {
		t.Fatalf("len(untagged) = %d, want 0", len(untagged))
	}

	// Count should also be 0
	count, err := repo.CountUntagged(ctx)
	if err != nil {
		t.Fatalf("CountUntagged() error = %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
}

func TestFindImagesWithoutAITagsReturnsThumbnailReadyImagesWithoutAISource(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	imageIDs := seedImagesForFindImagesWithoutAITags(t, repo)
	seedAITagSourcesForFindImagesWithoutAITags(t, db, imageIDs)

	images, err := repo.FindImagesWithoutAITags(ctx, 10)
	if err != nil {
		t.Fatalf("FindImagesWithoutAITags() error = %v", err)
	}

	if len(images) != 2 {
		t.Fatalf("len(images) = %d, want 2", len(images))
	}
	if images[0].ID != imageIDs[0] {
		t.Fatalf("images[0].ID = %d, want %d", images[0].ID, imageIDs[0])
	}
	if images[1].ID != imageIDs[1] {
		t.Fatalf("images[1].ID = %d, want %d", images[1].ID, imageIDs[1])
	}
}

func TestFindImagesWithoutAITagsExcludesImagesWithAISourceTag(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	imageIDs := seedImagesForFindImagesWithoutAITags(t, repo)
	seedAITagSourcesForFindImagesWithoutAITags(t, db, imageIDs)

	images, err := repo.FindImagesWithoutAITags(ctx, 10)
	if err != nil {
		t.Fatalf("FindImagesWithoutAITags() error = %v", err)
	}

	for _, image := range images {
		if image.ID == imageIDs[2] {
			t.Fatalf("image %d has AI source tag and should be excluded", imageIDs[2])
		}
	}
}

func TestFindImagesWithoutAITagsRespectsLimit(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	imageIDs := seedImagesForFindImagesWithoutAITags(t, repo)
	seedAITagSourcesForFindImagesWithoutAITags(t, db, imageIDs)

	images, err := repo.FindImagesWithoutAITags(ctx, 1)
	if err != nil {
		t.Fatalf("FindImagesWithoutAITags() error = %v", err)
	}

	if len(images) != 1 {
		t.Fatalf("len(images) = %d, want 1", len(images))
	}
	if images[0].ID != imageIDs[0] {
		t.Fatalf("images[0].ID = %d, want %d", images[0].ID, imageIDs[0])
	}
}

func TestFindImagesWithoutAITagsExcludesImagesWithoutThumbnail(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	imageIDs := seedImagesForFindImagesWithoutAITags(t, repo)
	seedAITagSourcesForFindImagesWithoutAITags(t, db, imageIDs)

	images, err := repo.FindImagesWithoutAITags(ctx, 10)
	if err != nil {
		t.Fatalf("FindImagesWithoutAITags() error = %v", err)
	}

	for _, image := range images {
		if image.ID == imageIDs[3] {
			t.Fatalf("image %d has no thumbnail and should be excluded", imageIDs[3])
		}
	}
}

func seedImagesForFindImagesWithoutAITags(t *testing.T, repo ImageRepository) []int64 {
	t.Helper()

	images := []*domain.Image{
		{Path: "/eligible-no-tags.png", Filename: "eligible-no-tags.png", SourceRoot: "/", Format: "png", ThumbnailSmallUrl: "/thumb/s1.jpg", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Path: "/eligible-manual-only.png", Filename: "eligible-manual-only.png", SourceRoot: "/", Format: "png", ThumbnailSmallUrl: "/thumb/s2.jpg", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Path: "/exclude-has-ai-tag.png", Filename: "exclude-has-ai-tag.png", SourceRoot: "/", Format: "png", ThumbnailSmallUrl: "/thumb/s3.jpg", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Path: "/exclude-no-thumbnail.png", Filename: "exclude-no-thumbnail.png", SourceRoot: "/", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	ids := make([]int64, 0, len(images))
	for _, image := range images {
		if _, err := repo.SaveImage(image); err != nil {
			t.Fatalf("save image: %v", err)
		}
		ids = append(ids, image.ID)
	}

	return ids
}

// --- Backfill candidate query tests ---

func TestCountBackfillHitCountWithHasTagsFilter(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	// Seed: 5 images total
	// img1, img2 have tags; img3, img4, img5 have no tags
	tagRepo := NewTagRepository(db)
	tag := &domain.Tag{PreferredLabel: "test", Slug: "test", ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, tag); err != nil {
		t.Fatalf("save tag: %v", err)
	}

	imageTagRepo := NewImageTagRepository(db)
	for i := 0; i < 5; i++ {
		img := &domain.Image{
			Path:       filepath.Join("/img", string(rune('a'+i))+".png"),
			Filename:   string(rune('a'+i)) + ".png",
			SourceRoot: "/img",
			Format:     "png",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if _, err := repo.SaveImage(img); err != nil {
			t.Fatalf("save image: %v", err)
		}
		// Tag first 2 images
		if i < 2 {
			if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: img.ID, TagID: tag.ID, ReviewState: "confirmed"}); err != nil {
				t.Fatalf("save image-tag: %v", err)
			}
		}
	}

	// has_tags=false: should count 3 untagged images
	hasTagsFalse := false
	hitCount, err := repo.CountBackfillHitCount(ctx, BackfillCandidateFilter{HasTags: &hasTagsFalse})
	if err != nil {
		t.Fatalf("CountBackfillHitCount() error = %v", err)
	}
	if hitCount != 3 {
		t.Errorf("hitCount with has_tags=false = %d, want 3", hitCount)
	}

	// has_tags=true: should count 2 tagged images
	hasTagsTrue := true
	hitCount, err = repo.CountBackfillHitCount(ctx, BackfillCandidateFilter{HasTags: &hasTagsTrue})
	if err != nil {
		t.Fatalf("CountBackfillHitCount() error = %v", err)
	}
	if hitCount != 2 {
		t.Errorf("hitCount with has_tags=true = %d, want 2", hitCount)
	}
}

func TestCountBackfillHitCountWithTagIDsFilter(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	tagRepo := NewTagRepository(db)
	tag1 := &domain.Tag{PreferredLabel: "tag1", Slug: "tag1", ReviewState: "confirmed"}
	tag2 := &domain.Tag{PreferredLabel: "tag2", Slug: "tag2", ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, tag1); err != nil {
		t.Fatalf("save tag1: %v", err)
	}
	if err := tagRepo.Save(ctx, tag2); err != nil {
		t.Fatalf("save tag2: %v", err)
	}

	// img1 has tag1, img2 has tag1+tag2, img3 untagged
	images := []*domain.Image{
		{Path: "/a.png", Filename: "a.png", SourceRoot: "/", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Path: "/b.png", Filename: "b.png", SourceRoot: "/", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Path: "/c.png", Filename: "c.png", SourceRoot: "/", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	for _, img := range images {
		if _, err := repo.SaveImage(img); err != nil {
			t.Fatalf("save image: %v", err)
		}
	}

	imageTagRepo := NewImageTagRepository(db)
	// img1 has tag1
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: images[0].ID, TagID: tag1.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	// img2 has tag1 and tag2
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: images[1].ID, TagID: tag1.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: images[1].ID, TagID: tag2.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Filter by tag1: 2 images
	hitCount, err := repo.CountBackfillHitCount(ctx, BackfillCandidateFilter{TagIDs: []int64{tag1.ID}})
	if err != nil {
		t.Fatalf("CountBackfillHitCount() error = %v", err)
	}
	if hitCount != 2 {
		t.Errorf("hitCount with tag1 = %d, want 2", hitCount)
	}

	// Filter by tag1 AND tag2: only img2
	hitCount, err = repo.CountBackfillHitCount(ctx, BackfillCandidateFilter{TagIDs: []int64{tag1.ID, tag2.ID}})
	if err != nil {
		t.Fatalf("CountBackfillHitCount() error = %v", err)
	}
	if hitCount != 1 {
		t.Errorf("hitCount with tag1+tag2 = %d, want 1", hitCount)
	}
}

func TestBackfillCandidateCountClassifiesSkipReasons(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	// Create tags
	tagRepo := NewTagRepository(db)
	aiTag := &domain.Tag{PreferredLabel: "ai-tag", Slug: "ai-tag", ReviewState: "confirmed"}
	manualTag := &domain.Tag{PreferredLabel: "manual-tag", Slug: "manual-tag", ReviewState: "confirmed"}
	filterTag := &domain.Tag{PreferredLabel: "filter-tag", Slug: "filter-tag", ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, aiTag); err != nil {
		t.Fatalf("save ai tag: %v", err)
	}
	if err := tagRepo.Save(ctx, manualTag); err != nil {
		t.Fatalf("save manual tag: %v", err)
	}
	if err := tagRepo.Save(ctx, filterTag); err != nil {
		t.Fatalf("save filter tag: %v", err)
	}

	// Create 5 images, all with filter-tag for narrowing
	images := make([]*domain.Image, 5)
	for i := range images {
		images[i] = &domain.Image{
			Path:       fmt.Sprintf("/img%d.png", i),
			Filename:   fmt.Sprintf("img%d.png", i),
			SourceRoot: "/",
			Format:     "png",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if _, err := repo.SaveImage(images[i]); err != nil {
			t.Fatalf("save image %d: %v", i, err)
		}
	}

	imageTagRepo := NewImageTagRepository(db)
	for _, img := range images {
		// All images have filter-tag to satisfy narrowing
		if err := imageTagRepo.Save(ctx, &domain.ImageTag{
			ImageID: img.ID, TagID: filterTag.ID, ReviewState: "confirmed",
		}); err != nil {
			t.Fatalf("save filter-tag: %v", err)
		}
	}

	// img0: eligible (no AI tag, no active task)
	// img1: has AI tag → skipped_with_ai_tag
	if _, err := db.Exec(`INSERT INTO image_tags (image_id, tag_id, source, confidence, review_state) VALUES (?, ?, 'ai', 0.95, 'confirmed')`, images[1].ID, aiTag.ID); err != nil {
		t.Fatalf("insert ai tag: %v", err)
	}
	// Create the batch BEFORE the task (FK constraint)
	if _, err := db.Exec(`INSERT INTO task_batches (id, source_type, summary_label, status) VALUES (1, 'manual_batch', 'test', 'running')`); err != nil {
		t.Fatalf("insert batch: %v", err)
	}
	// img2: has active task → skipped_with_active_task
	if _, err := db.Exec(`INSERT INTO platform_tasks (batch_id, image_id, task_type, source_type, status, dedupe_key, image_version_key) VALUES (1, ?, 'ai_tag_generation', 'manual_batch', 'queued', 'dk1', 'vk1')`, images[2].ID); err != nil {
		t.Fatalf("insert active task: %v", err)
	}
	// img3: has both AI tag AND active task → counted in both skip reasons
	if _, err := db.Exec(`INSERT INTO image_tags (image_id, tag_id, source, confidence, review_state) VALUES (?, ?, 'ai', 0.9, 'confirmed')`, images[3].ID, aiTag.ID); err != nil {
		t.Fatalf("insert ai tag: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO platform_tasks (batch_id, image_id, task_type, source_type, status, dedupe_key, image_version_key) VALUES (1, ?, 'ai_tag_generation', 'manual_batch', 'running', 'dk2', 'vk2')`, images[3].ID); err != nil {
		t.Fatalf("insert active task: %v", err)
	}
	// img4: eligible (no AI tag, no active task)

	filter := BackfillCandidateFilter{TagIDs: []int64{filterTag.ID}}

	hitCount, err := repo.CountBackfillHitCount(ctx, filter)
	if err != nil {
		t.Fatalf("CountBackfillHitCount() error = %v", err)
	}
	if hitCount != 5 {
		t.Errorf("hitCount = %d, want 5", hitCount)
	}

	enqueueable, err := repo.CountBackfillCandidates(ctx, filter)
	if err != nil {
		t.Fatalf("CountBackfillCandidates() error = %v", err)
	}
	if enqueueable != 2 {
		t.Errorf("enqueueable = %d, want 2 (img0 + img4)", enqueueable)
	}

	skippedAITag, err := repo.CountBackfillSkippedWithAITag(ctx, filter)
	if err != nil {
		t.Fatalf("CountBackfillSkippedWithAITag() error = %v", err)
	}
	if skippedAITag != 2 {
		t.Errorf("skippedWithAITag = %d, want 2 (img1 + img3)", skippedAITag)
	}

	skippedActiveTask, err := repo.CountBackfillSkippedWithActiveTask(ctx, filter)
	if err != nil {
		t.Fatalf("CountBackfillSkippedWithActiveTask() error = %v", err)
	}
	if skippedActiveTask != 2 {
		t.Errorf("skippedWithActiveTask = %d, want 2 (img2 + img3)", skippedActiveTask)
	}
}

func TestFindBackfillCandidatesReturnsOnlyEligible(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	tagRepo := NewTagRepository(db)
	aiTag := &domain.Tag{PreferredLabel: "ai-tag", Slug: "ai-tag", ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, aiTag); err != nil {
		t.Fatalf("save ai tag: %v", err)
	}

	// img1: eligible, img2: has AI tag, img3: has active task, img4: eligible
	images := []*domain.Image{
		{Path: "/eligible1.png", Filename: "eligible1.png", SourceRoot: "/", Format: "png", ThumbnailSmallUrl: "/thumb/s1.jpg", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Path: "/has-ai.png", Filename: "has-ai.png", SourceRoot: "/", Format: "png", ThumbnailSmallUrl: "/thumb/s2.jpg", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Path: "/has-task.png", Filename: "has-task.png", SourceRoot: "/", Format: "png", ThumbnailSmallUrl: "/thumb/s3.jpg", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Path: "/eligible2.png", Filename: "eligible2.png", SourceRoot: "/", Format: "png", ThumbnailSmallUrl: "/thumb/s4.jpg", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	for _, img := range images {
		if _, err := repo.SaveImage(img); err != nil {
			t.Fatalf("save image: %v", err)
		}
	}

	// img2 has AI tag
	if _, err := db.Exec(`INSERT INTO image_tags (image_id, tag_id, source, confidence, review_state) VALUES (?, ?, 'ai', 0.9, 'confirmed')`, images[1].ID, aiTag.ID); err != nil {
		t.Fatalf("insert ai tag: %v", err)
	}

	// img3 has active task
	if _, err := db.Exec(`INSERT INTO task_batches (id, source_type, summary_label, status) VALUES (1, 'manual_batch', 'test', 'running')`); err != nil {
		t.Fatalf("insert batch: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO platform_tasks (batch_id, image_id, task_type, source_type, status, dedupe_key, image_version_key) VALUES (1, ?, 'ai_tag_generation', 'manual_batch', 'queued', 'dk1', 'vk1')`, images[2].ID); err != nil {
		t.Fatalf("insert active task: %v", err)
	}

	// Use has_tags=false filter (no tags = untagged)
	hasTagsFalse := false
	candidates, err := repo.FindBackfillCandidates(ctx, BackfillCandidateFilter{HasTags: &hasTagsFalse})
	if err != nil {
		t.Fatalf("FindBackfillCandidates() error = %v", err)
	}

	if len(candidates) != 2 {
		t.Fatalf("len(candidates) = %d, want 2 (img1 + img4)", len(candidates))
	}
	// Should be ordered by id ASC
	if candidates[0].ID != images[0].ID {
		t.Errorf("candidates[0].ID = %d, want %d", candidates[0].ID, images[0].ID)
	}
	if candidates[1].ID != images[3].ID {
		t.Errorf("candidates[1].ID = %d, want %d", candidates[1].ID, images[3].ID)
	}
}

func TestBackfillCountsWithHasTagsFalseFilter(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	tagRepo := NewTagRepository(db)
	aiTag := &domain.Tag{PreferredLabel: "ai-tag", Slug: "ai-tag", ReviewState: "confirmed"}
	manualTag := &domain.Tag{PreferredLabel: "manual-tag", Slug: "manual-tag", ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, aiTag); err != nil {
		t.Fatalf("save ai tag: %v", err)
	}
	if err := tagRepo.Save(ctx, manualTag); err != nil {
		t.Fatalf("save manual tag: %v", err)
	}

	// 4 images, all untagged (no entries in image_tags)
	images := make([]*domain.Image, 4)
	for i := range images {
		images[i] = &domain.Image{
			Path:       fmt.Sprintf("/img%d.png", i),
			Filename:   fmt.Sprintf("img%d.png", i),
			SourceRoot: "/",
			Format:     "png",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if _, err := repo.SaveImage(images[i]); err != nil {
			t.Fatalf("save image %d: %v", i, err)
		}
	}

	// img0, img1: eligible (no tags, no tasks)
	// img2: has active task (but no AI tag since untagged)
	// Create batch BEFORE task (FK constraint)
	if _, err := db.Exec(`INSERT INTO task_batches (id, source_type, summary_label, status) VALUES (1, 'manual_batch', 'test', 'running')`); err != nil {
		t.Fatalf("insert batch: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO platform_tasks (batch_id, image_id, task_type, source_type, status, dedupe_key, image_version_key) VALUES (1, ?, 'ai_tag_generation', 'manual_batch', 'queued', 'dk1', 'vk1')`, images[2].ID); err != nil {
		t.Fatalf("insert active task: %v", err)
	}
	// img3: eligible

	hasTagsFalse := false
	filter := BackfillCandidateFilter{HasTags: &hasTagsFalse}

	hitCount, _ := repo.CountBackfillHitCount(ctx, filter)
	if hitCount != 4 {
		t.Errorf("hitCount = %d, want 4", hitCount)
	}

	enqueueable, _ := repo.CountBackfillCandidates(ctx, filter)
	if enqueueable != 3 {
		t.Errorf("enqueueable = %d, want 3 (img0, img1, img3)", enqueueable)
	}

	skippedAITag, _ := repo.CountBackfillSkippedWithAITag(ctx, filter)
	if skippedAITag != 0 {
		t.Errorf("skippedAITag = %d, want 0 (none have AI tags)", skippedAITag)
	}

	skippedActive, _ := repo.CountBackfillSkippedWithActiveTask(ctx, filter)
	if skippedActive != 1 {
		t.Errorf("skippedActive = %d, want 1 (img2)", skippedActive)
	}
}

func TestBackfillActiveTaskStatusesDriveEligibility(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	if _, err := db.Exec(`INSERT INTO task_batches (id, source_type, summary_label, status) VALUES (1, 'manual_batch', 'test', 'running')`); err != nil {
		t.Fatalf("insert batch: %v", err)
	}

	cases := []struct {
		name   string
		status string
		active bool
	}{
		{name: "pending", status: domain.PlatformTaskStatusPending, active: true},
		{name: "queued", status: domain.PlatformTaskStatusQueued, active: true},
		{name: "running", status: domain.PlatformTaskStatusRunning, active: true},
		{name: "completed", status: domain.PlatformTaskStatusCompleted},
		{name: "failed", status: domain.PlatformTaskStatusFailed},
		{name: "cancelled", status: domain.PlatformTaskStatusCancelled},
		{name: "skipped", status: domain.PlatformTaskStatusSkipped},
	}

	images := make([]*domain.Image, 0, len(cases))
	for i, tc := range cases {
		img := &domain.Image{
			Path:       fmt.Sprintf("/active-status/%d.png", i),
			Filename:   fmt.Sprintf("%d.png", i),
			SourceRoot: "/active-status",
			Format:     "png",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if _, err := repo.SaveImage(img); err != nil {
			t.Fatalf("save image %s: %v", tc.name, err)
		}
		images = append(images, img)
		if _, err := db.Exec(`
			INSERT INTO platform_tasks (batch_id, image_id, task_type, source_type, status, dedupe_key, image_version_key)
			VALUES (1, ?, ?, 'manual_batch', ?, ?, ?)
		`, img.ID, domain.PlatformTaskTypeAITagGeneration, tc.status, "dk-"+tc.name, "vk-"+tc.name); err != nil {
			t.Fatalf("insert task %s: %v", tc.name, err)
		}
	}

	hasTagsFalse := false
	filter := BackfillCandidateFilter{HasTags: &hasTagsFalse}

	skippedActive, err := repo.CountBackfillSkippedWithActiveTask(ctx, filter)
	if err != nil {
		t.Fatalf("CountBackfillSkippedWithActiveTask() error = %v", err)
	}
	if skippedActive != 3 {
		t.Fatalf("skippedActive = %d, want 3", skippedActive)
	}

	candidates, err := repo.FindBackfillCandidates(ctx, filter)
	if err != nil {
		t.Fatalf("FindBackfillCandidates() error = %v", err)
	}
	for i, tc := range cases {
		contains := containsImageID(candidates, images[i].ID)
		if tc.active && contains {
			t.Fatalf("%s task image should not be eligible, got %+v", tc.name, imageIDs(candidates))
		}
		if !tc.active && !contains {
			t.Fatalf("%s task image should be eligible, got %+v", tc.name, imageIDs(candidates))
		}
	}
}

func seedAITagSourcesForFindImagesWithoutAITags(t *testing.T, db *sql.DB, imageIDs []int64) {
	t.Helper()

	tagRepo := NewTagRepository(db)
	aiTag := &domain.Tag{PreferredLabel: "ai-tag", Slug: "ai-tag", ReviewState: "confirmed"}
	manualTag := &domain.Tag{PreferredLabel: "manual-tag", Slug: "manual-tag", ReviewState: "confirmed"}
	if err := tagRepo.Save(context.Background(), aiTag); err != nil {
		t.Fatalf("save ai tag: %v", err)
	}
	if err := tagRepo.Save(context.Background(), manualTag); err != nil {
		t.Fatalf("save manual tag: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO image_tags (image_id, tag_id, source, confidence, review_state)
		VALUES (?, ?, ?, ?, ?)
	`, imageIDs[1], manualTag.ID, "manual", 1.0, "confirmed"); err != nil {
		t.Fatalf("insert manual image tag: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO image_tags (image_id, tag_id, source, confidence, review_state)
		VALUES (?, ?, ?, ?, ?)
	`, imageIDs[2], aiTag.ID, "ai", 0.98, "confirmed"); err != nil {
		t.Fatalf("insert ai image tag: %v", err)
	}
}

func TestFindByGalleryFilterExactTagDoesNotExpandDescendants(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()
	tagRepo := NewTagRepository(db)
	imageTagRepo := NewImageTagRepository(db)

	root := &domain.Tag{PreferredLabel: "character", Slug: "character", Level: domain.TagLevelRoot}
	if err := tagRepo.Save(ctx, root); err != nil {
		t.Fatalf("save root: %v", err)
	}
	parent := &domain.Tag{PreferredLabel: "protagonist", Slug: "protagonist", Level: domain.TagLevelParent, ParentID: &root.ID}
	if err := tagRepo.Save(ctx, parent); err != nil {
		t.Fatalf("save parent: %v", err)
	}
	child := &domain.Tag{PreferredLabel: "heroine", Slug: "heroine", Level: domain.TagLevelChild, ParentID: &parent.ID}
	if err := tagRepo.Save(ctx, child); err != nil {
		t.Fatalf("save child: %v", err)
	}

	imgWithTag := &domain.Image{Path: "/exact/1.png", Filename: "1.png", SourceRoot: "/exact", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	imgWithChild := &domain.Image{Path: "/exact/2.png", Filename: "2.png", SourceRoot: "/exact", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	imgWithRoot := &domain.Image{Path: "/exact/3.png", Filename: "3.png", SourceRoot: "/exact", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if _, err := repo.SaveImage(imgWithTag); err != nil {
		t.Fatalf("save img1: %v", err)
	}
	if _, err := repo.SaveImage(imgWithChild); err != nil {
		t.Fatalf("save img2: %v", err)
	}
	if _, err := repo.SaveImage(imgWithRoot); err != nil {
		t.Fatalf("save img3: %v", err)
	}

	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: imgWithTag.ID, TagID: parent.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("tag img1 with parent: %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: imgWithChild.ID, TagID: child.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("tag img2 with child: %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: imgWithRoot.ID, TagID: root.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("tag img3 with root: %v", err)
	}

	filtered, err := repo.FindByGalleryFilter(ctx, []int64{parent.ID}, nil, 10, 0, "id", "asc")
	if err != nil {
		t.Fatalf("FindByGalleryFilter(exact=parent): %v", err)
	}
	if len(filtered) != 1 || filtered[0].ID != imgWithTag.ID {
		t.Fatalf("exact parent should match only imgWithTag (not child's image), got %d images: %+v", len(filtered), imageIDs(filtered))
	}

	count, err := repo.CountByGalleryFilter(ctx, []int64{parent.ID}, nil)
	if err != nil {
		t.Fatalf("CountByGalleryFilter(exact=parent): %v", err)
	}
	if count != 1 {
		t.Fatalf("exact parent count = %d, want 1", count)
	}
}

func TestFindByGalleryFilterSubtreeRootExpandsDescendants(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()
	tagRepo := NewTagRepository(db)
	imageTagRepo := NewImageTagRepository(db)

	root := &domain.Tag{PreferredLabel: "character", Slug: "character", Level: domain.TagLevelRoot}
	if err := tagRepo.Save(ctx, root); err != nil {
		t.Fatalf("save root: %v", err)
	}
	parent := &domain.Tag{PreferredLabel: "protagonist", Slug: "protagonist", Level: domain.TagLevelParent, ParentID: &root.ID}
	if err := tagRepo.Save(ctx, parent); err != nil {
		t.Fatalf("save parent: %v", err)
	}
	child := &domain.Tag{PreferredLabel: "heroine", Slug: "heroine", Level: domain.TagLevelChild, ParentID: &parent.ID}
	if err := tagRepo.Save(ctx, child); err != nil {
		t.Fatalf("save child: %v", err)
	}

	imgWithRoot := &domain.Image{Path: "/sub/1.png", Filename: "1.png", SourceRoot: "/sub", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	imgWithParent := &domain.Image{Path: "/sub/2.png", Filename: "2.png", SourceRoot: "/sub", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	imgWithChild := &domain.Image{Path: "/sub/3.png", Filename: "3.png", SourceRoot: "/sub", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if _, err := repo.SaveImage(imgWithRoot); err != nil {
		t.Fatalf("save img1: %v", err)
	}
	if _, err := repo.SaveImage(imgWithParent); err != nil {
		t.Fatalf("save img2: %v", err)
	}
	if _, err := repo.SaveImage(imgWithChild); err != nil {
		t.Fatalf("save img3: %v", err)
	}

	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: imgWithRoot.ID, TagID: root.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("tag img1: %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: imgWithParent.ID, TagID: parent.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("tag img2: %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: imgWithChild.ID, TagID: child.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("tag img3: %v", err)
	}

	filtered, err := repo.FindByGalleryFilter(ctx, nil, []int64{root.ID}, 10, 0, "id", "asc")
	if err != nil {
		t.Fatalf("FindByGalleryFilter(subtree=root): %v", err)
	}
	if len(filtered) != 3 {
		t.Fatalf("subtree root should match all 3 images (root + parent + child), got %d: %+v", len(filtered), imageIDs(filtered))
	}

	count, err := repo.CountByGalleryFilter(ctx, nil, []int64{root.ID})
	if err != nil {
		t.Fatalf("CountByGalleryFilter(subtree=root): %v", err)
	}
	if count != 3 {
		t.Fatalf("subtree root count = %d, want 3", count)
	}
}

func TestFindByGalleryFilterMixedExactAndSubtreeKeepsAndSemantics(t *testing.T) {
	t.Parallel()

	db, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()
	tagRepo := NewTagRepository(db)
	imageTagRepo := NewImageTagRepository(db)

	seriesRoot := &domain.Tag{PreferredLabel: "series", Slug: "series", Level: domain.TagLevelRoot}
	moodRoot := &domain.Tag{PreferredLabel: "mood", Slug: "mood", Level: domain.TagLevelRoot}
	orphanTag := &domain.Tag{PreferredLabel: "solo", Slug: "solo", Level: domain.TagLevelChild}
	if err := tagRepo.Save(ctx, seriesRoot); err != nil {
		t.Fatalf("save seriesRoot: %v", err)
	}
	if err := tagRepo.Save(ctx, moodRoot); err != nil {
		t.Fatalf("save moodRoot: %v", err)
	}
	if err := tagRepo.Save(ctx, orphanTag); err != nil {
		t.Fatalf("save orphanTag: %v", err)
	}
	seriesChild := &domain.Tag{PreferredLabel: "lead", Slug: "lead", Level: domain.TagLevelChild, ParentID: &seriesRoot.ID}
	if err := tagRepo.Save(ctx, seriesChild); err != nil {
		t.Fatalf("save seriesChild: %v", err)
	}

	images := []*domain.Image{
		{Path: "/mix/1.png", Filename: "1.png", SourceRoot: "/mix", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Path: "/mix/2.png", Filename: "2.png", SourceRoot: "/mix", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Path: "/mix/3.png", Filename: "3.png", SourceRoot: "/mix", Format: "png", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	for _, img := range images {
		if _, err := repo.SaveImage(img); err != nil {
			t.Fatalf("save image: %v", err)
		}
	}

	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: images[0].ID, TagID: seriesChild.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("tag: %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: images[0].ID, TagID: orphanTag.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("tag: %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: images[1].ID, TagID: seriesChild.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("tag: %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: images[2].ID, TagID: orphanTag.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("tag: %v", err)
	}

	filtered, err := repo.FindByGalleryFilter(ctx, []int64{orphanTag.ID}, []int64{seriesRoot.ID}, 10, 0, "id", "asc")
	if err != nil {
		t.Fatalf("FindByGalleryFilter(exact=orphan, subtree=series): %v", err)
	}
	if len(filtered) != 1 || filtered[0].ID != images[0].ID {
		t.Fatalf("exact orphan + subtree series should match only img0, got %d: %+v", len(filtered), imageIDs(filtered))
	}

	count, err := repo.CountByGalleryFilter(ctx, []int64{orphanTag.ID}, []int64{seriesRoot.ID})
	if err != nil {
		t.Fatalf("CountByGalleryFilter: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func TestFindByGalleryFilterReturnsEmptyForEmptyInputs(t *testing.T) {
	t.Parallel()

	_, repo := newImageRepositoryTestDB(t)
	ctx := context.Background()

	filtered, err := repo.FindByGalleryFilter(ctx, nil, nil, 10, 0, "id", "asc")
	if err != nil {
		t.Fatalf("FindByGalleryFilter(): %v", err)
	}
	if len(filtered) != 0 {
		t.Fatalf("expected empty result for empty inputs, got %d", len(filtered))
	}

	count, err := repo.CountByGalleryFilter(ctx, nil, nil)
	if err != nil {
		t.Fatalf("CountByGalleryFilter(): %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 count for empty inputs, got %d", count)
	}
}
