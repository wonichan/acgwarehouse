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

func TestAliasRepositorySavePersistsAlias(t *testing.T) {
	t.Parallel()

	aliasRepo, _ := newAliasRepositoryForTest(t)
	alias := &domain.TagAlias{TagID: 1, Label: " Blue Sky ", AliasType: "synonym"}

	if err := aliasRepo.Save(context.Background(), alias); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if alias.ID == 0 {
		t.Fatal("expected Save() to assign alias ID")
	}
	if alias.NormalizedLabel != "blue sky" {
		t.Fatalf("NormalizedLabel = %q, want blue sky", alias.NormalizedLabel)
	}
}

func TestAliasRepositoryFindByTagIDReturnsAliases(t *testing.T) {
	t.Parallel()

	aliasRepo, _ := newAliasRepositoryForTest(t)
	mustSaveAlias(t, aliasRepo, &domain.TagAlias{TagID: 2, Label: "rainy night"})
	mustSaveAlias(t, aliasRepo, &domain.TagAlias{TagID: 2, Label: "night rain"})
	mustSaveAlias(t, aliasRepo, &domain.TagAlias{TagID: 3, Label: "sunrise"})

	aliases, err := aliasRepo.FindByTagID(context.Background(), 2)
	if err != nil {
		t.Fatalf("FindByTagID() error = %v", err)
	}
	if len(aliases) != 2 {
		t.Fatalf("len(aliases) = %d, want 2", len(aliases))
	}
}

func TestAliasRepositoryFindByNormalizedLabelMatchesNormalizedInput(t *testing.T) {
	t.Parallel()

	aliasRepo, _ := newAliasRepositoryForTest(t)
	mustSaveAlias(t, aliasRepo, &domain.TagAlias{TagID: 4, Label: "Rain Night"})

	alias, err := aliasRepo.FindByNormalizedLabel(context.Background(), " rain night ")
	if err != nil {
		t.Fatalf("FindByNormalizedLabel() error = %v", err)
	}
	if alias.TagID != 4 {
		t.Fatalf("TagID = %d, want 4", alias.TagID)
	}
}

func TestAliasRepositoryDeleteRemovesAlias(t *testing.T) {
	t.Parallel()

	aliasRepo, _ := newAliasRepositoryForTest(t)
	alias := mustSaveAlias(t, aliasRepo, &domain.TagAlias{TagID: 5, Label: "soft light"})

	if err := aliasRepo.Delete(context.Background(), alias.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := aliasRepo.FindByID(context.Background(), alias.ID); err == nil {
		t.Fatal("expected deleted alias lookup to fail")
	}
}

func newAliasRepositoryForTest(t *testing.T) (TagAliasRepository, TagRepository) {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "tag-alias-repo.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	tagRepo := NewTagRepository(db)
	if err := insertSeedTag(db, &domain.Tag{ID: 1, PreferredLabel: "blue sky", Slug: "blue-sky"}); err != nil {
		t.Fatalf("seed tag 1: %v", err)
	}
	if err := insertSeedTag(db, &domain.Tag{ID: 2, PreferredLabel: "rain night", Slug: "rain-night"}); err != nil {
		t.Fatalf("seed tag 2: %v", err)
	}
	if err := insertSeedTag(db, &domain.Tag{ID: 3, PreferredLabel: "sunrise", Slug: "sunrise"}); err != nil {
		t.Fatalf("seed tag 3: %v", err)
	}
	if err := insertSeedTag(db, &domain.Tag{ID: 4, PreferredLabel: "night rain", Slug: "night-rain"}); err != nil {
		t.Fatalf("seed tag 4: %v", err)
	}
	if err := insertSeedTag(db, &domain.Tag{ID: 5, PreferredLabel: "soft light", Slug: "soft-light"}); err != nil {
		t.Fatalf("seed tag 5: %v", err)
	}

	return NewTagAliasRepository(db), tagRepo
}

func mustSaveAlias(t *testing.T, repo TagAliasRepository, alias *domain.TagAlias) *domain.TagAlias {
	t.Helper()

	if err := repo.Save(context.Background(), alias); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	return alias
}

func insertSeedTag(db *sql.DB, tag *domain.Tag) error {
	if tag.Level == "" {
		tag.Level = domain.TagLevelChild
	}
	if tag.CreatedAt.IsZero() {
		tag.CreatedAt = time.Now()
	}
	_, err := db.Exec(`
		INSERT INTO tags (id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tag.ID, tag.PreferredLabel, tag.Slug, tag.Level, tag.ParentID, tag.PrimaryCategory, tag.ReviewState, tag.TrustScore, tag.UsageCount, tag.CreatedAt)
	return err
}
