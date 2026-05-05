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

func TestTagGovernanceQueryBatchResolveDescendants(t *testing.T) {
	t.Parallel()

	db := newTagGovernanceQueryTestDB(t)
	seedTagGovernanceQueryData(t, db)

	result, err := NewTagGovernanceQuery(db).BatchResolveDescendants(context.Background(), []int64{4, 6})
	if err != nil {
		t.Fatalf("BatchResolveDescendants() error = %v", err)
	}

	assertSameRepositoryInt64Set(t, result[4], []int64{4, 5})
	assertSameRepositoryInt64Set(t, result[6], []int64{4, 5, 6, 7})
}

func TestTagGovernanceQueryComputeHierarchyStats(t *testing.T) {
	t.Parallel()

	db := newTagGovernanceQueryTestDB(t)
	seedTagGovernanceQueryData(t, db)

	stats, err := NewTagGovernanceQuery(db).ComputeHierarchyStats(context.Background(), []int64{1, 4})
	if err != nil {
		t.Fatalf("ComputeHierarchyStats() error = %v", err)
	}

	if stats.UsageCount != 2 || stats.PendingCount != 1 || stats.ConfirmedCount != 1 || stats.AICount != 2 || stats.ManualCount != 1 {
		t.Fatalf("stats = %+v, want usage=2 pending=1 confirmed=1 ai=2 manual=1", stats)
	}
}

func TestTagGovernanceQueryBatchFindChildrenGroupsChildrenByParent(t *testing.T) {
	t.Parallel()

	db := newTagGovernanceQueryTestDB(t)
	seedTagGovernanceQueryData(t, db)

	childrenMap, err := NewTagGovernanceQuery(db).BatchFindChildren(context.Background(), []int64{4, 6})
	if err != nil {
		t.Fatalf("BatchFindChildren() error = %v", err)
	}

	if len(childrenMap[4]) != 1 || childrenMap[4][0].ID != 5 {
		t.Fatalf("childrenMap[4] = %+v, want only child 5", childrenMap[4])
	}
	if len(childrenMap[6]) != 2 || childrenMap[6][0].ID != 4 || childrenMap[6][1].ID != 7 {
		t.Fatalf("childrenMap[6] = %+v, want children 4 and 7", childrenMap[6])
	}
}

func TestTagGovernanceQueryBatchFindChildrenAllowsNullPrimaryCategory(t *testing.T) {
	t.Parallel()

	db := newTagGovernanceQueryTestDB(t)
	seedTagGovernanceQueryData(t, db)
	if _, err := db.Exec(`UPDATE tags SET primary_category = NULL WHERE id IN (?, ?)`, 4, 5); err != nil {
		t.Fatalf("set primary_category null: %v", err)
	}

	childrenMap, err := NewTagGovernanceQuery(db).BatchFindChildren(context.Background(), []int64{4, 6})
	if err != nil {
		t.Fatalf("BatchFindChildren() error = %v", err)
	}
	if childrenMap[4][0].PrimaryCategory != "" {
		t.Fatalf("childrenMap[4][0].PrimaryCategory = %q, want empty string", childrenMap[4][0].PrimaryCategory)
	}
	if childrenMap[6][0].PrimaryCategory != "" {
		t.Fatalf("childrenMap[6][0].PrimaryCategory = %q, want empty string", childrenMap[6][0].PrimaryCategory)
	}
}

func TestTagGovernanceQueryBatchCountDirectAssociations(t *testing.T) {
	t.Parallel()

	db := newTagGovernanceQueryTestDB(t)
	seedTagGovernanceQueryData(t, db)

	result, err := NewTagGovernanceQuery(db).BatchCountDirectAssociations(context.Background(), []int64{1, 4, 999})
	if err != nil {
		t.Fatalf("BatchCountDirectAssociations() error = %v", err)
	}

	if result[1] != 2 || result[4] != 1 || result[999] != 0 {
		t.Fatalf("direct association counts = %v, want tag1=2 tag4=1 tag999=0", result)
	}
}

func TestTagGovernanceQueryBatchDirectHierarchyStats(t *testing.T) {
	t.Parallel()

	db := newTagGovernanceQueryTestDB(t)
	seedTagGovernanceQueryData(t, db)

	result, err := NewTagGovernanceQuery(db).BatchDirectHierarchyStats(context.Background(), []int64{1, 4, 999})
	if err != nil {
		t.Fatalf("BatchDirectHierarchyStats() error = %v", err)
	}

	if result[1].UsageCount != 2 || result[1].PendingCount != 1 || result[1].ConfirmedCount != 1 || result[1].AICount != 1 || result[1].ManualCount != 1 {
		t.Fatalf("stats[1] = %+v, want usage=2 pending=1 confirmed=1 ai=1 manual=1", result[1])
	}
	if result[4].UsageCount != 1 || result[4].ConfirmedCount != 1 || result[4].AICount != 1 {
		t.Fatalf("stats[4] = %+v, want usage=1 confirmed=1 ai=1", result[4])
	}
	if _, ok := result[999]; ok {
		t.Fatalf("stats[999] = %+v, want missing zero row", result[999])
	}
}

func TestTagGovernanceQueryBatchTreeImageTagRows(t *testing.T) {
	t.Parallel()

	db := newTagGovernanceQueryTestDB(t)
	seedTagGovernanceQueryData(t, db)

	rows, err := NewTagGovernanceQuery(db).BatchTreeImageTagRows(context.Background(), []int64{1, 4})
	if err != nil {
		t.Fatalf("BatchTreeImageTagRows() error = %v", err)
	}

	if len(rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3: %+v", len(rows), rows)
	}
	countByTag := map[int64]int{}
	for _, row := range rows {
		countByTag[row.TagID]++
		if row.Source == "" {
			t.Fatalf("row source should be defaulted by SQL: %+v", row)
		}
	}
	if countByTag[1] != 2 || countByTag[4] != 1 {
		t.Fatalf("countByTag = %v, want tag1=2 tag4=1", countByTag)
	}
}

func newTagGovernanceQueryTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "tag-governance-query.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}
	return db
}

func seedTagGovernanceQueryData(t *testing.T, db *sql.DB) {
	t.Helper()

	now := time.Now()
	if _, err := db.Exec(`
		INSERT INTO images (id, path, filename, source_root, file_size, width, height, format, created_at, updated_at)
		VALUES
			(1, '/images/1.png', '1.png', '/images', 100, 100, 100, 'png', ?, ?),
			(2, '/images/2.png', '2.png', '/images', 100, 100, 100, 'png', ?, ?)
	`, now, now, now, now); err != nil {
		t.Fatalf("seed images: %v", err)
	}

	for _, tag := range []*domain.Tag{
		{ID: 1, PreferredLabel: "alpha-source", Slug: "alpha-source", Level: domain.TagLevelChild, PrimaryCategory: "artist", ReviewState: domain.ReviewStateConfirmed, UsageCount: 7},
		{ID: 6, PreferredLabel: "franchise", Slug: "franchise", Level: domain.TagLevelRoot, PrimaryCategory: "copyright", ReviewState: domain.ReviewStateConfirmed, UsageCount: 1},
		{ID: 4, PreferredLabel: "characters", Slug: "characters", Level: domain.TagLevelParent, ParentID: repositoryInt64Ptr(6), PrimaryCategory: "artist", ReviewState: domain.ReviewStateConfirmed, UsageCount: 1},
		{ID: 5, PreferredLabel: "heroine", Slug: "heroine", Level: domain.TagLevelChild, ParentID: repositoryInt64Ptr(4), PrimaryCategory: "artist", ReviewState: domain.ReviewStateConfirmed, UsageCount: 1},
		{ID: 7, PreferredLabel: "outfit", Slug: "outfit", Level: domain.TagLevelParent, ParentID: repositoryInt64Ptr(6), PrimaryCategory: "artist", ReviewState: domain.ReviewStateConfirmed, UsageCount: 0},
	} {
		tag.CreatedAt = now
		if err := insertTagGovernanceQuerySeedTag(db, tag); err != nil {
			t.Fatalf("seed tag %d: %v", tag.ID, err)
		}
	}

	imageTagRepo := NewImageTagRepository(db)
	for _, imageTag := range []*domain.ImageTag{
		{ImageID: 1, TagID: 1, Source: domain.ImageTagSourceManual, ReviewState: domain.ReviewStateConfirmed},
		{ImageID: 2, TagID: 1, Source: domain.ImageTagSourceAI, ReviewState: domain.ReviewStatePending},
		{ImageID: 1, TagID: 4, Source: domain.ImageTagSourceAI, ReviewState: domain.ReviewStateConfirmed},
	} {
		if err := imageTagRepo.Save(context.Background(), imageTag); err != nil {
			t.Fatalf("seed image_tag: %v", err)
		}
	}
}

func insertTagGovernanceQuerySeedTag(db *sql.DB, tag *domain.Tag) error {
	_, err := db.Exec(`
		INSERT INTO tags (id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tag.ID, tag.PreferredLabel, tag.Slug, tag.Level, tag.ParentID, tag.PrimaryCategory, tag.ReviewState, tag.TrustScore, tag.UsageCount, tag.CreatedAt)
	return err
}

func repositoryInt64Ptr(v int64) *int64 {
	return &v
}

func assertSameRepositoryInt64Set(t *testing.T, got, want []int64) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d (got=%v want=%v)", len(got), len(want), got, want)
	}

	seen := make(map[int64]int, len(got))
	for _, id := range got {
		seen[id]++
	}
	for _, id := range want {
		if seen[id] == 0 {
			t.Fatalf("missing id %d in %v", id, got)
		}
		seen[id]--
	}
	for id, remaining := range seen {
		if remaining != 0 {
			t.Fatalf("unexpected id %d in %v", id, got)
		}
	}
}
