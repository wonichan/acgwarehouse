package service

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

func TestTagAdminServiceListGovernanceTagsIncludesRequiredFields(t *testing.T) {
	t.Parallel()

	service, _, _, _ := newTagAdminServiceForTest(t)

	rows, total, err := service.ListGovernanceTags(context.Background(), "", 20, 0)
	if err != nil {
		t.Fatalf("ListGovernanceTags() error = %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if len(rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3", len(rows))
	}
	if rows[0].PreferredLabel != "alpha-source" {
		t.Fatalf("first preferred_label = %q, want %q", rows[0].PreferredLabel, "alpha-source")
	}
	if rows[0].UsageCount < rows[1].UsageCount {
		t.Fatalf("rows not sorted by usage_count desc: first=%d second=%d", rows[0].UsageCount, rows[1].UsageCount)
	}
	if rows[0].TagID == 0 {
		t.Fatal("expected tag_id to be populated")
	}
	if rows[0].PrimaryCategory == "" {
		t.Fatal("expected primary_category to be populated")
	}
	if len(rows[0].Aliases) == 0 {
		t.Fatal("expected aliases to be populated")
	}
	if !rows[2].CanDelete {
		t.Fatal("expected unused tag to be deletable")
	}
}

func TestTagAdminServiceMergeTagsRequiresDifferentSourceAndTarget(t *testing.T) {
	t.Parallel()

	service, _, _, _ := newTagAdminServiceForTest(t)

	_, err := service.MergeTags(context.Background(), 1, 1)
	if err == nil {
		t.Fatal("expected MergeTags() to reject source == target")
	}
}

func TestTagAdminServiceMergeTagsMovesAssociationsAliasesAndSyncsFTS(t *testing.T) {
	t.Parallel()

	service, tagRepo, aliasRepo, imageTagRepo := newTagAdminServiceForTest(t)

	result, err := service.MergeTags(context.Background(), 1, 2)
	if err != nil {
		t.Fatalf("MergeTags() error = %v", err)
	}
	if result.SourceTagID != 1 || result.TargetTagID != 2 {
		t.Fatalf("unexpected merge ids: %+v", result)
	}

	if _, err := tagRepo.FindByID(context.Background(), 1); err == nil {
		t.Fatal("expected source tag to be deleted")
	}

	targetImageTags, err := imageTagRepo.FindByTagID(context.Background(), 2, 10, 0)
	if err != nil {
		t.Fatalf("FindByTagID(target) error = %v", err)
	}
	if len(targetImageTags) == 0 {
		t.Fatal("expected source image-tag rows to move to target")
	}

	aliases, err := aliasRepo.FindByTagID(context.Background(), 2)
	if err != nil {
		t.Fatalf("FindByTagID(target aliases) error = %v", err)
	}
	if len(aliases) < 2 {
		t.Fatalf("expected merged aliases on target, got %d", len(aliases))
	}

	if err := imageTagRepo.SyncFTSForTag(context.Background(), 2); err != nil {
		t.Fatalf("SyncFTSForTag(target) error = %v", err)
	}
}

func TestTagAdminServiceMergeTagsUsesExplicitTargetWithoutFuzzySelection(t *testing.T) {
	t.Parallel()

	service, tagRepo, _, _ := newTagAdminServiceForTest(t)

	if _, err := service.MergeTags(context.Background(), 1, 3); err != nil {
		t.Fatalf("MergeTags() explicit target error = %v", err)
	}

	if _, err := tagRepo.FindByID(context.Background(), 3); err != nil {
		t.Fatalf("explicit target should remain after merge: %v", err)
	}
	if _, err := tagRepo.FindByID(context.Background(), 2); err != nil {
		t.Fatalf("non-selected candidate target should not be touched: %v", err)
	}
}

func newTagAdminServiceForTest(t *testing.T) (*TagAdminService, repository.TagRepository, repository.TagAliasRepository, repository.ImageTagRepository) {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "tag-admin.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	seedTagAdminData(t, db)

	tagRepo := repository.NewTagRepository(db)
	aliasRepo := repository.NewTagAliasRepository(db)
	imageTagRepo := repository.NewImageTagRepository(db)

	return NewTagAdminService(tagRepo, aliasRepo, imageTagRepo), tagRepo, aliasRepo, imageTagRepo
}

func seedTagAdminData(t *testing.T, db *sql.DB) {
	t.Helper()

	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO images (id, path, filename, source_root, file_size, width, height, format, created_at, updated_at)
		VALUES
			(1, '/images/1.png', '1.png', '/images', 100, 100, 100, 'png', ?, ?),
			(2, '/images/2.png', '2.png', '/images', 100, 100, 100, 'png', ?, ?)
	`, now, now, now, now)
	if err != nil {
		t.Fatalf("seed images: %v", err)
	}

	tagRepo := repository.NewTagRepository(db)
	for _, tag := range []*domain.Tag{
		{ID: 1, PreferredLabel: "alpha-source", Slug: "alpha-source", PrimaryCategory: "artist", ReviewState: "confirmed", UsageCount: 7},
		{ID: 2, PreferredLabel: "alpha-target", Slug: "alpha-target", PrimaryCategory: "artist", ReviewState: "confirmed", UsageCount: 4},
		{ID: 3, PreferredLabel: "unused", Slug: "unused", PrimaryCategory: "meta", ReviewState: "pending", UsageCount: 0},
	} {
		if err := tagRepo.Save(context.Background(), tag); err != nil {
			t.Fatalf("seed tag %d: %v", tag.ID, err)
		}
	}

	aliasRepo := repository.NewTagAliasRepository(db)
	for _, alias := range []*domain.TagAlias{
		{TagID: 1, Label: "alpha", AliasType: "synonym"},
		{TagID: 2, Label: "target alpha", AliasType: "synonym"},
	} {
		if err := aliasRepo.Save(context.Background(), alias); err != nil {
			t.Fatalf("seed alias %q: %v", alias.Label, err)
		}
	}

	imageTagRepo := repository.NewImageTagRepository(db)
	if err := imageTagRepo.Save(context.Background(), &domain.ImageTag{ImageID: 1, TagID: 1, Source: domain.ImageTagSourceManual, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("seed image_tag 1: %v", err)
	}
	if err := imageTagRepo.Save(context.Background(), &domain.ImageTag{ImageID: 2, TagID: 1, Source: domain.ImageTagSourceAI, ReviewState: "pending"}); err != nil {
		t.Fatalf("seed image_tag 2: %v", err)
	}
}
