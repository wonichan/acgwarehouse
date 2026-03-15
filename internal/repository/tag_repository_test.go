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

func TestTagRepositorySaveCreatesTagAndReturnsID(t *testing.T) {
	t.Parallel()

	repo := newTagRepositoryForTest(t)
	tag := &domain.Tag{PreferredLabel: "blue sky", Slug: "blue-sky", ReviewState: "pending", TrustScore: 0.6}

	if err := repo.Save(context.Background(), tag); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if tag.ID == 0 {
		t.Fatal("expected Save() to assign tag ID")
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

func newTagRepositoryForTest(t *testing.T) TagRepository {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "tag-repo.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	return NewTagRepository(db)
}

func mustSaveTag(t *testing.T, repo TagRepository, tag *domain.Tag) *domain.Tag {
	t.Helper()

	if err := repo.Save(context.Background(), tag); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	return tag
}
