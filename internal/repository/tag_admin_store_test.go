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

func TestSQLTxRunnerCommitsAndRollsBack(t *testing.T) {
	t.Parallel()

	db := newTagAdminStoreTestDB(t)
	ctx := context.Background()
	runner := sqlTxRunner{db: db}

	if err := runner.WithinTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO tags (preferred_label, slug, level, review_state, trust_score, usage_count, created_at)
			VALUES ('committed', 'committed', ?, 'confirmed', 0, 0, ?)
		`, domain.TagLevelChild, time.Now())
		return err
	}); err != nil {
		t.Fatalf("WithinTx(commit) error = %v", err)
	}

	if _, err := NewTagRepository(db).FindByLabel(ctx, "committed"); err != nil {
		t.Fatalf("FindByLabel(committed) error = %v", err)
	}

	errRollback := errors.New("rollback")
	if err := runner.WithinTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO tags (preferred_label, slug, level, review_state, trust_score, usage_count, created_at)
			VALUES ('rolled back', 'rolled-back', ?, 'confirmed', 0, 0, ?)
		`, domain.TagLevelChild, time.Now())
		if err != nil {
			return err
		}
		return errRollback
	}); !errors.Is(err, errRollback) {
		t.Fatalf("WithinTx(rollback) error = %v, want %v", err, errRollback)
	}

	if _, err := NewTagRepository(db).FindByLabel(ctx, "rolled back"); err != sql.ErrNoRows {
		t.Fatalf("FindByLabel(rolled back) error = %v, want %v", err, sql.ErrNoRows)
	}
}

func TestTagAdminStoreDeleteTagRemovesAssociationsAliasesAndSyncsFTS(t *testing.T) {
	t.Parallel()

	db := newTagAdminStoreTestDB(t)
	ctx := context.Background()
	imageRepo := NewImageRepository(db)
	tagRepo := NewTagRepository(db)
	aliasRepo := NewTagAliasRepository(db)
	imageTagRepo := NewImageTagRepository(db)

	image := saveTagAdminStoreImage(t, imageRepo, "hero.png")
	source := &domain.Tag{PreferredLabel: "source tag", Slug: "source-tag", Level: domain.TagLevelChild, ReviewState: "confirmed"}
	target := &domain.Tag{PreferredLabel: "target tag", Slug: "target-tag", Level: domain.TagLevelChild, ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, source); err != nil {
		t.Fatalf("save source: %v", err)
	}
	if err := tagRepo.Save(ctx, target); err != nil {
		t.Fatalf("save target: %v", err)
	}
	if err := aliasRepo.Save(ctx, &domain.TagAlias{TagID: source.ID, Label: "source alias"}); err != nil {
		t.Fatalf("save alias: %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: image.ID, TagID: source.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("save source image tag: %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: image.ID, TagID: target.ID, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("save target image tag: %v", err)
	}

	result, err := NewTagAdminStore(db).DeleteTag(ctx, source.ID)
	if err != nil {
		t.Fatalf("DeleteTag() error = %v", err)
	}
	if result.AffectedImageCount != 1 || result.DetachedChildCount != 0 {
		t.Fatalf("DeleteTag() result = %+v", result)
	}
	if _, err := tagRepo.FindByID(ctx, source.ID); err != sql.ErrNoRows {
		t.Fatalf("FindByID(deleted source) error = %v, want %v", err, sql.ErrNoRows)
	}
	aliases, err := aliasRepo.FindByTagID(ctx, source.ID)
	if err != nil {
		t.Fatalf("FindByTagID(source aliases) error = %v", err)
	}
	if len(aliases) != 0 {
		t.Fatalf("source alias count = %d, want 0", len(aliases))
	}
	tags, err := imageTagRepo.FindByImageID(ctx, image.ID)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	if len(tags) != 1 || tags[0].TagID != target.ID {
		t.Fatalf("image tags after delete = %+v, want only target %d", tags, target.ID)
	}
	ids, err := NewSearchRepository(db).FTSFullTextSearch(ctx, "source", 10, 0)
	if err != nil {
		t.Fatalf("FTSFullTextSearch(source) error = %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("FTS source ids = %v, want none", ids)
	}
	ids, err = NewSearchRepository(db).FTSFullTextSearch(ctx, "target", 10, 0)
	if err != nil {
		t.Fatalf("FTSFullTextSearch(target) error = %v", err)
	}
	if len(ids) != 1 || ids[0] != image.ID {
		t.Fatalf("FTS target ids = %v, want [%d]", ids, image.ID)
	}
}

func TestTagAdminStoreDeleteTagDetachesChildren(t *testing.T) {
	t.Parallel()

	db := newTagAdminStoreTestDB(t)
	ctx := context.Background()
	tagRepo := NewTagRepository(db)

	root := &domain.Tag{PreferredLabel: "root", Slug: "root", Level: domain.TagLevelRoot, ReviewState: "confirmed"}
	parent := &domain.Tag{PreferredLabel: "parent", Slug: "parent", Level: domain.TagLevelParent, ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, root); err != nil {
		t.Fatalf("save root: %v", err)
	}
	parent.ParentID = &root.ID
	if err := tagRepo.Save(ctx, parent); err != nil {
		t.Fatalf("save parent: %v", err)
	}

	result, err := NewTagAdminStore(db).DeleteTag(ctx, root.ID)
	if err != nil {
		t.Fatalf("DeleteTag() error = %v", err)
	}
	if result.DetachedChildCount != 1 {
		t.Fatalf("DetachedChildCount = %d, want 1", result.DetachedChildCount)
	}
	reloaded, err := tagRepo.FindByID(ctx, parent.ID)
	if err != nil {
		t.Fatalf("FindByID(parent) error = %v", err)
	}
	if reloaded.ParentID != nil {
		t.Fatalf("parent ParentID = %v, want nil", *reloaded.ParentID)
	}
}

func TestTagAdminStoreChangeTagLevelDetachesChildrenInTransaction(t *testing.T) {
	t.Parallel()

	db := newTagAdminStoreTestDB(t)
	ctx := context.Background()
	tagRepo := NewTagRepository(db)

	root := &domain.Tag{PreferredLabel: "root", Slug: "root", Level: domain.TagLevelRoot, ReviewState: "confirmed"}
	parent := &domain.Tag{PreferredLabel: "parent", Slug: "parent", Level: domain.TagLevelParent, ReviewState: "confirmed"}
	child := &domain.Tag{PreferredLabel: "child", Slug: "child", Level: domain.TagLevelChild, ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, root); err != nil {
		t.Fatalf("save root: %v", err)
	}
	parent.ParentID = &root.ID
	if err := tagRepo.Save(ctx, parent); err != nil {
		t.Fatalf("save parent: %v", err)
	}
	child.ParentID = &parent.ID
	if err := tagRepo.Save(ctx, child); err != nil {
		t.Fatalf("save child: %v", err)
	}

	parent.Level = domain.TagLevelRoot
	parent.ParentID = nil
	updated, err := NewTagAdminStore(db).ChangeTagLevel(ctx, parent, []*domain.Tag{child})
	if err != nil {
		t.Fatalf("ChangeTagLevel() error = %v", err)
	}
	if updated.Level != domain.TagLevelRoot || updated.ParentID != nil {
		t.Fatalf("updated parent = %+v", updated)
	}
	reloadedChild, err := tagRepo.FindByID(ctx, child.ID)
	if err != nil {
		t.Fatalf("FindByID(child) error = %v", err)
	}
	if reloadedChild.ParentID != nil {
		t.Fatalf("child ParentID = %v, want nil", *reloadedChild.ParentID)
	}
}

func TestTagAdminStoreReparentTagUpdatesParent(t *testing.T) {
	t.Parallel()

	db := newTagAdminStoreTestDB(t)
	ctx := context.Background()
	tagRepo := NewTagRepository(db)

	root := &domain.Tag{PreferredLabel: "root", Slug: "root", Level: domain.TagLevelRoot, ReviewState: "confirmed"}
	parent := &domain.Tag{PreferredLabel: "parent", Slug: "parent", Level: domain.TagLevelParent, ReviewState: "confirmed"}
	child := &domain.Tag{PreferredLabel: "child", Slug: "child", Level: domain.TagLevelChild, ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, root); err != nil {
		t.Fatalf("save root: %v", err)
	}
	parent.ParentID = &root.ID
	if err := tagRepo.Save(ctx, parent); err != nil {
		t.Fatalf("save parent: %v", err)
	}
	if err := tagRepo.Save(ctx, child); err != nil {
		t.Fatalf("save child: %v", err)
	}

	child.ParentID = &parent.ID
	updated, err := NewTagAdminStore(db).ReparentTag(ctx, child)
	if err != nil {
		t.Fatalf("ReparentTag() error = %v", err)
	}
	if updated.ParentID == nil || *updated.ParentID != parent.ID {
		t.Fatalf("updated ParentID = %v, want %d", updated.ParentID, parent.ID)
	}
	reloadedChild, err := tagRepo.FindByID(ctx, child.ID)
	if err != nil {
		t.Fatalf("FindByID(child) error = %v", err)
	}
	if reloadedChild.ParentID == nil || *reloadedChild.ParentID != parent.ID {
		t.Fatalf("child ParentID = %v, want %d", reloadedChild.ParentID, parent.ID)
	}
}

func TestTagAdminStoreMergeTagsMovesAssociationsAliasesAndSyncsFTS(t *testing.T) {
	t.Parallel()

	db := newTagAdminStoreTestDB(t)
	ctx := context.Background()
	imageRepo := NewImageRepository(db)
	tagRepo := NewTagRepository(db)
	aliasRepo := NewTagAliasRepository(db)
	imageTagRepo := NewImageTagRepository(db)

	image := saveTagAdminStoreImage(t, imageRepo, "merge.png")
	source := &domain.Tag{PreferredLabel: "source merge", Slug: "source-merge", Level: domain.TagLevelChild, ReviewState: "confirmed"}
	target := &domain.Tag{PreferredLabel: "target merge", Slug: "target-merge", Level: domain.TagLevelChild, ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, source); err != nil {
		t.Fatalf("save source: %v", err)
	}
	if err := tagRepo.Save(ctx, target); err != nil {
		t.Fatalf("save target: %v", err)
	}
	if err := aliasRepo.Save(ctx, &domain.TagAlias{TagID: source.ID, Label: "source alias"}); err != nil {
		t.Fatalf("save source alias: %v", err)
	}
	if err := aliasRepo.Save(ctx, &domain.TagAlias{TagID: target.ID, Label: "target alias"}); err != nil {
		t.Fatalf("save target alias: %v", err)
	}
	if err := imageTagRepo.Save(ctx, &domain.ImageTag{ImageID: image.ID, TagID: source.ID, Source: domain.ImageTagSourceAI, ReviewState: "pending", Confidence: 0.7}); err != nil {
		t.Fatalf("save image tag: %v", err)
	}

	result, err := NewTagAdminStore(db).MergeTags(ctx, source.ID, target.ID)
	if err != nil {
		t.Fatalf("MergeTags() error = %v", err)
	}
	if result.MigratedImageAssociations != 1 || result.MigratedAliases != 2 {
		t.Fatalf("MergeTags() result = %+v, want 1 association and 2 aliases", result)
	}
	if _, err := tagRepo.FindByID(ctx, source.ID); err != sql.ErrNoRows {
		t.Fatalf("FindByID(source) error = %v, want %v", err, sql.ErrNoRows)
	}
	tags, err := imageTagRepo.FindByImageID(ctx, image.ID)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	if len(tags) != 1 || tags[0].TagID != target.ID || tags[0].Source != domain.ImageTagSourceAI || tags[0].ReviewState != "pending" {
		t.Fatalf("image tags after merge = %+v", tags)
	}
	aliases, err := aliasRepo.FindByTagID(ctx, target.ID)
	if err != nil {
		t.Fatalf("FindByTagID(target aliases) error = %v", err)
	}
	aliasLabels := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		aliasLabels = append(aliasLabels, alias.Label)
	}
	wantAliases := map[string]bool{
		"target alias": true,
		"source merge": true,
	}
	if len(aliasLabels) != len(wantAliases) {
		t.Fatalf("target aliases = %v, want %d aliases", aliasLabels, len(wantAliases))
	}
	for _, label := range aliasLabels {
		if !wantAliases[label] {
			t.Fatalf("target aliases = %v, unexpected alias %q", aliasLabels, label)
		}
		delete(wantAliases, label)
	}
	if len(wantAliases) != 0 {
		t.Fatalf("target aliases = %v, missing aliases %v", aliasLabels, wantAliases)
	}
	ids, err := NewSearchRepository(db).FTSFullTextSearch(ctx, "source", 10, 0)
	if err != nil {
		t.Fatalf("FTSFullTextSearch(source) error = %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("FTS source ids = %v, want none", ids)
	}
	ids, err = NewSearchRepository(db).FTSFullTextSearch(ctx, "target", 10, 0)
	if err != nil {
		t.Fatalf("FTSFullTextSearch(target) error = %v", err)
	}
	if len(ids) != 1 || ids[0] != image.ID {
		t.Fatalf("FTS target ids = %v, want [%d]", ids, image.ID)
	}
}

func newTagAdminStoreTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "tag-admin-store.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}
	return db
}

func saveTagAdminStoreImage(t *testing.T, repo ImageRepository, filename string) *domain.Image {
	t.Helper()

	now := time.Now()
	image := &domain.Image{
		Path:       "/library/" + filename,
		Filename:   filename,
		SourceRoot: "/library",
		FileSize:   100,
		Width:      100,
		Height:     100,
		Format:     "png",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if _, err := repo.SaveImage(image); err != nil {
		t.Fatalf("SaveImage() error = %v", err)
	}
	return image
}
