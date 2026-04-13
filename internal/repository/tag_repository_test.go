package repository

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

func TestTagRepositorySaveCreatesTagAndReturnsID(t *testing.T) {
	t.Parallel()

	repo := newTagRepositoryForTest(t)
	tag := &domain.Tag{PreferredLabel: "blue sky", Slug: "blue-sky", ReviewState: "pending", TrustScore: 0.6, Level: domain.TagLevelChild}

	if err := repo.Save(context.Background(), tag); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if tag.ID == 0 {
		t.Fatal("expected Save() to assign tag ID")
	}
}

func TestTagRepositorySavePersistsHierarchyFields(t *testing.T) {
	t.Parallel()

	repo := newTagRepositoryForTest(t)
	root := mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "character", Slug: "character", Level: domain.TagLevelRoot})
	tag := &domain.Tag{
		PreferredLabel: "heroine",
		Slug:           "heroine",
		ReviewState:    "pending",
		TrustScore:     0.6,
		Level:          domain.TagLevelChild,
		ParentID:       &root.ID,
	}

	if err := repo.Save(context.Background(), tag); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	stored, err := repo.FindByID(context.Background(), tag.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if stored.Level != domain.TagLevelChild {
		t.Fatalf("Level = %q, want %q", stored.Level, domain.TagLevelChild)
	}
	if stored.ParentID == nil || *stored.ParentID != root.ID {
		t.Fatalf("ParentID = %v, want %d", stored.ParentID, root.ID)
	}
	if tag.Level != domain.TagLevelChild {
		t.Fatalf("tag.Level = %q, want %q", tag.Level, domain.TagLevelChild)
	}
}

func TestTagRepositorySaveUpdatesExistingTagWithoutReplacingAssociations(t *testing.T) {
	t.Parallel()

	repo, db := newTagRepositoryWithDBForTest(t)
	ctx := context.Background()
	now := time.Now()

	if _, err := db.Exec(`
		INSERT INTO images (id, path, filename, source_root, file_size, width, height, format, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, 1, "/images/1.png", "1.png", "/images", 100, 100, 100, "png", now, now); err != nil {
		t.Fatalf("seed image: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO tags (id, preferred_label, slug, level, primary_category, review_state, trust_score, usage_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, 10, "blue sky", "blue-sky", domain.TagLevelChild, "", "confirmed", 0.9, 1, now); err != nil {
		t.Fatalf("seed tag: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO tag_aliases (tag_id, label, normalized_label, alias_type)
		VALUES (?, ?, ?, ?)
	`, 10, "azure sky", "azure sky", "synonym"); err != nil {
		t.Fatalf("seed alias: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO image_tags (image_id, tag_id, review_state)
		VALUES (?, ?, ?)
	`, 1, 10, "confirmed"); err != nil {
		t.Fatalf("seed image tag: %v", err)
	}

	tag, err := repo.FindByID(ctx, 10)
	if err != nil {
		t.Fatalf("FindByID() before save error = %v", err)
	}
	tag.PreferredLabel = "blue sky updated"
	tag.Slug = "blue-sky-updated"

	if err := repo.Save(ctx, tag); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	stored, err := repo.FindByID(ctx, 10)
	if err != nil {
		t.Fatalf("FindByID() after save error = %v", err)
	}
	if stored.PreferredLabel != "blue sky updated" {
		t.Fatalf("PreferredLabel = %q, want %q", stored.PreferredLabel, "blue sky updated")
	}
	if stored.Slug != "blue-sky-updated" {
		t.Fatalf("Slug = %q, want %q", stored.Slug, "blue-sky-updated")
	}
	if stored.UsageCount != 2 {
		t.Fatalf("UsageCount = %d, want 2", stored.UsageCount)
	}

	var aliasCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tag_aliases WHERE tag_id = ?`, 10).Scan(&aliasCount); err != nil {
		t.Fatalf("count aliases: %v", err)
	}
	if aliasCount != 1 {
		t.Fatalf("alias count = %d, want 1", aliasCount)
	}

	var imageTagCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM image_tags WHERE image_id = ? AND tag_id = ?`, 1, 10).Scan(&imageTagCount); err != nil {
		t.Fatalf("count image tags: %v", err)
	}
	if imageTagCount != 1 {
		t.Fatalf("image_tag count = %d, want 1", imageTagCount)
	}
}

func TestTagRepositoryFindByLabelReturnsExactMatch(t *testing.T) {
	t.Parallel()

	repo := newTagRepositoryForTest(t)
	mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "night rain", Slug: "night-rain", ReviewState: "confirmed"})
	mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "rain night", Slug: "rain-night", ReviewState: "pending"})

	tag, err := repo.FindByLabel(context.Background(), "night rain")
	if err != nil {
		t.Fatalf("FindByLabel() error = %v", err)
	}
	if tag.PreferredLabel != "night rain" {
		t.Fatalf("PreferredLabel = %q, want %q", tag.PreferredLabel, "night rain")
	}
}

func TestTagRepositoryFindByLabelLikeReturnsSortedMatches(t *testing.T) {
	t.Parallel()

	repo := newTagRepositoryForTest(t)
	mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "blue hair", Slug: "blue-hair", UsageCount: 5})
	mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "blue sky", Slug: "blue-sky", UsageCount: 12})
	mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "red sky", Slug: "red-sky", UsageCount: 8})

	tags, err := repo.FindByLabelLike(context.Background(), "blue", 10)
	if err != nil {
		t.Fatalf("FindByLabelLike() error = %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("len(tags) = %d, want 2", len(tags))
	}
	if tags[0].PreferredLabel != "blue sky" || tags[1].PreferredLabel != "blue hair" {
		t.Fatalf("unexpected order: got [%q, %q]", tags[0].PreferredLabel, tags[1].PreferredLabel)
	}
}

func TestTagRepositoryUpdateReviewStateUpdatesTag(t *testing.T) {
	t.Parallel()

	repo := newTagRepositoryForTest(t)
	tag := mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "sunset", Slug: "sunset", ReviewState: "pending"})

	if err := repo.UpdateReviewState(context.Background(), tag.ID, "confirmed"); err != nil {
		t.Fatalf("UpdateReviewState() error = %v", err)
	}

	updated, err := repo.FindByID(context.Background(), tag.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if updated.ReviewState != "confirmed" {
		t.Fatalf("ReviewState = %q, want confirmed", updated.ReviewState)
	}
}

func TestTagRepositoryIncrementUsageCountIncreasesCounter(t *testing.T) {
	t.Parallel()

	repo := newTagRepositoryForTest(t)
	tag := mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "forest", Slug: "forest", UsageCount: 2})

	if err := repo.IncrementUsageCount(context.Background(), tag.ID); err != nil {
		t.Fatalf("IncrementUsageCount() error = %v", err)
	}

	updated, err := repo.FindByID(context.Background(), tag.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if updated.UsageCount != 3 {
		t.Fatalf("UsageCount = %d, want 3", updated.UsageCount)
	}
}

func TestTagRepositoryFindRootsReturnsOnlyRootTags(t *testing.T) {
	t.Parallel()

	repo := newTagRepositoryForTest(t)
	mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "series", Slug: "series", Level: domain.TagLevelRoot, UsageCount: 9})
	root2 := mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "artist", Slug: "artist", Level: domain.TagLevelRoot, UsageCount: 4})
	mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "lead", Slug: "lead", Level: domain.TagLevelParent, ParentID: &root2.ID})
	mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "solo", Slug: "solo", Level: domain.TagLevelChild})

	roots, err := repo.FindRoots(context.Background())
	if err != nil {
		t.Fatalf("FindRoots() error = %v", err)
	}
	if len(roots) != 2 {
		t.Fatalf("len(roots) = %d, want 2", len(roots))
	}
	for _, root := range roots {
		if root.Level != domain.TagLevelRoot {
			t.Fatalf("root.Level = %q, want %q", root.Level, domain.TagLevelRoot)
		}
		if root.ParentID != nil {
			t.Fatalf("root.ParentID = %v, want nil", root.ParentID)
		}
	}
}

func TestTagRepositoryFindChildrenByParentReturnsDirectChildren(t *testing.T) {
	t.Parallel()

	repo := newTagRepositoryForTest(t)
	root := mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "franchise", Slug: "franchise", Level: domain.TagLevelRoot})
	child1 := mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "cast", Slug: "cast", Level: domain.TagLevelParent, ParentID: &root.ID, UsageCount: 7})
	child2 := mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "setting", Slug: "setting", Level: domain.TagLevelParent, ParentID: &root.ID, UsageCount: 2})
	mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "other", Slug: "other", Level: domain.TagLevelChild})

	children, err := repo.FindChildrenByParent(context.Background(), root.ID)
	if err != nil {
		t.Fatalf("FindChildrenByParent() error = %v", err)
	}
	if len(children) != 2 {
		t.Fatalf("len(children) = %d, want 2", len(children))
	}
	if children[0].ID != child1.ID || children[1].ID != child2.ID {
		t.Fatalf("unexpected child order: got [%d, %d], want [%d, %d]", children[0].ID, children[1].ID, child1.ID, child2.ID)
	}
}

func TestTagRepositoryFindValidParentCandidatesUsesTargetLevel(t *testing.T) {
	t.Parallel()

	repo := newTagRepositoryForTest(t)
	root1 := mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "copyright", Slug: "copyright", Level: domain.TagLevelRoot, UsageCount: 8})
	root2 := mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "meta", Slug: "meta", Level: domain.TagLevelRoot, UsageCount: 3})
	parent1 := mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "protagonist", Slug: "protagonist", Level: domain.TagLevelParent, ParentID: &root1.ID, UsageCount: 5})
	parent2 := mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "antagonist", Slug: "antagonist", Level: domain.TagLevelParent, ParentID: &root2.ID, UsageCount: 2})
	mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "blue hair", Slug: "blue-hair", Level: domain.TagLevelChild})

	forParent, err := repo.FindValidParentCandidates(context.Background(), domain.TagLevelParent)
	if err != nil {
		t.Fatalf("FindValidParentCandidates(parent) error = %v", err)
	}
	if len(forParent) != 2 || forParent[0].ID != root1.ID || forParent[1].ID != root2.ID {
		t.Fatalf("unexpected parent candidates for parent level: %+v", forParent)
	}

	forChild, err := repo.FindValidParentCandidates(context.Background(), domain.TagLevelChild)
	if err != nil {
		t.Fatalf("FindValidParentCandidates(child) error = %v", err)
	}
	if len(forChild) != 2 || forChild[0].ID != parent1.ID || forChild[1].ID != parent2.ID {
		t.Fatalf("unexpected parent candidates for child level: %+v", forChild)
	}

	forRoot, err := repo.FindValidParentCandidates(context.Background(), domain.TagLevelRoot)
	if err != nil {
		t.Fatalf("FindValidParentCandidates(root) error = %v", err)
	}
	if len(forRoot) != 0 {
		t.Fatalf("len(forRoot) = %d, want 0", len(forRoot))
	}
}

func TestTagRepositoryResolveDescendantIDsReturnsSelfAndDescendants(t *testing.T) {
	t.Parallel()

	repo := newTagRepositoryForTest(t)
	root := mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "root", Slug: "root", Level: domain.TagLevelRoot})
	parent := mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "parent", Slug: "parent", Level: domain.TagLevelParent, ParentID: &root.ID})
	child := mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "child", Slug: "child", Level: domain.TagLevelChild, ParentID: &parent.ID})
	orphan := mustSaveTag(t, repo, &domain.Tag{PreferredLabel: "orphan", Slug: "orphan", Level: domain.TagLevelChild})

	resolved, err := repo.ResolveDescendantIDs(context.Background(), []int64{root.ID, parent.ID, orphan.ID})
	if err != nil {
		t.Fatalf("ResolveDescendantIDs() error = %v", err)
	}
	if len(resolved) != 3 {
		t.Fatalf("len(resolved) = %d, want 3", len(resolved))
	}
	assertResolvedIDs(t, resolved, root.ID, []int64{root.ID, parent.ID, child.ID})
	assertResolvedIDs(t, resolved, parent.ID, []int64{parent.ID, child.ID})
	assertResolvedIDs(t, resolved, orphan.ID, []int64{orphan.ID})

	all, err := repo.ResolveAllDescendantIDs(context.Background(), []int64{root.ID, parent.ID})
	if err != nil {
		t.Fatalf("ResolveAllDescendantIDs() error = %v", err)
	}
	assertIDsEqual(t, all, []int64{root.ID, parent.ID, child.ID})
}

func newTagRepositoryForTest(t *testing.T) TagRepository {
	t.Helper()

	repo, _ := newTagRepositoryWithDBForTest(t)
	return repo
}

func newTagRepositoryWithDBForTest(t *testing.T) (TagRepository, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "tag-repo.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	return NewTagRepository(db), db
}

func mustSaveTag(t *testing.T, repo TagRepository, tag *domain.Tag) *domain.Tag {
	t.Helper()

	if tag.Level == "" {
		tag.Level = domain.TagLevelChild
	}

	if err := repo.Save(context.Background(), tag); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	return tag
}

func assertResolvedIDs(t *testing.T, resolved map[int64][]int64, key int64, want []int64) {
	t.Helper()

	got, ok := resolved[key]
	if !ok {
		t.Fatalf("missing resolved key %d", key)
	}
	assertIDsEqual(t, got, want)
}

func assertIDsEqual(t *testing.T, got, want []int64) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(ids) = %d, want %d (got=%v want=%v)", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ids[%d] = %d, want %d (got=%v want=%v)", i, got[i], want[i], got, want)
		}
	}
}
