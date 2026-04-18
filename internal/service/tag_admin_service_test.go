package service

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strconv"
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
	if total != 7 {
		t.Fatalf("total = %d, want 7", total)
	}
	if len(rows) != 7 {
		t.Fatalf("len(rows) = %d, want 7", len(rows))
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
	if rows[0].Level == "" {
		t.Fatal("expected level to be populated")
	}
	if rows[0].DirectUsageCount == 0 {
		t.Fatal("expected direct_usage_count to be populated")
	}
	if rows[0].TreeUsageCount == 0 {
		t.Fatal("expected tree_usage_count to be populated")
	}
	if rows[0].PrimaryCategory == "" {
		t.Fatal("expected primary_category to be populated")
	}
	if len(rows[0].Aliases) == 0 {
		t.Fatal("expected aliases to be populated")
	}
	var unusedRow *TagGovernanceRow
	for i := range rows {
		if rows[i].TagID == 3 {
			unusedRow = &rows[i]
			break
		}
	}
	if unusedRow == nil {
		t.Fatal("expected to find unused tag row")
	}
	if !unusedRow.CanDelete {
		t.Fatal("expected unused tag to be deletable")
	}
}

func TestTagAdminServiceMergeTagsRejectsCrossLevelMerge(t *testing.T) {
	t.Parallel()

	service, _, _, _ := newTagAdminServiceForTest(t)

	_, err := service.MergeTags(context.Background(), 1, 4)
	if err == nil {
		t.Fatal("expected cross-level merge to fail")
	}
}

func TestTagAdminServiceMergeTagsRejectsSourceWithChildren(t *testing.T) {
	t.Parallel()

	service, _, _, _ := newTagAdminServiceForTest(t)

	_, err := service.MergeTags(context.Background(), 4, 7)
	if !errors.Is(err, ErrMergeSourceHasChildren) {
		t.Fatalf("MergeTags() error = %v, want %v", err, ErrMergeSourceHasChildren)
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

func TestTagAdminServiceGetDeletePreviewBlocksUsedTag(t *testing.T) {
	t.Parallel()

	service, _, _, _ := newTagAdminServiceForTest(t)

	preview, err := service.GetDeletePreview(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetDeletePreview() error = %v", err)
	}
	if preview.TagID != 1 {
		t.Fatalf("tag_id = %d, want 1", preview.TagID)
	}
	if preview.AffectedImageCount == 0 {
		t.Fatal("expected affected_image_count > 0 for used tag")
	}
	if preview.CanDelete {
		t.Fatal("expected used tag to be blocked from delete")
	}
	if preview.BlockingReason != "merge_or_reclassify_required" {
		t.Fatalf("blocking_reason = %q, want %q", preview.BlockingReason, "merge_or_reclassify_required")
	}
}

func TestTagAdminServiceGetDeletePreviewAllowsUnusedTag(t *testing.T) {
	t.Parallel()

	service, _, _, _ := newTagAdminServiceForTest(t)

	preview, err := service.GetDeletePreview(context.Background(), 3)
	if err != nil {
		t.Fatalf("GetDeletePreview() error = %v", err)
	}
	if !preview.CanDelete {
		t.Fatal("expected unused tag to be deletable")
	}
	if preview.AffectedImageCount != 0 {
		t.Fatalf("affected_image_count = %d, want 0", preview.AffectedImageCount)
	}
}

func TestTagAdminServiceGetDeletePreviewBlocksTagWithChildren(t *testing.T) {
	t.Parallel()

	service, _, _, _ := newTagAdminServiceForTest(t)

	preview, err := service.GetDeletePreview(context.Background(), 4)
	if err != nil {
		t.Fatalf("GetDeletePreview() error = %v", err)
	}
	if preview.CanDelete {
		t.Fatal("expected parent tag with children to be blocked")
	}
	if preview.BlockingReason != "child_tags_exist" {
		t.Fatalf("blocking_reason = %q, want %q", preview.BlockingReason, "child_tags_exist")
	}
}

func TestTagAdminServiceGetDeletePreviewBlocksRejectedOnlyAssociation(t *testing.T) {
	t.Parallel()

	service, tagRepo, _, imageTagRepo := newTagAdminServiceForTest(t)
	rejectedOnly := mustSaveAdminTag(t, tagRepo, &domain.Tag{PreferredLabel: "rejected-only", Slug: "rejected-only", Level: domain.TagLevelChild})
	if err := imageTagRepo.Save(context.Background(), &domain.ImageTag{ImageID: 2, TagID: rejectedOnly.ID, ReviewState: "rejected", Source: domain.ImageTagSourceAI}); err != nil {
		t.Fatalf("seed rejected image tag: %v", err)
	}

	preview, err := service.GetDeletePreview(context.Background(), rejectedOnly.ID)
	if err != nil {
		t.Fatalf("GetDeletePreview() error = %v", err)
	}
	if preview.CanDelete {
		t.Fatal("expected rejected-only direct association to still block delete")
	}
	if preview.AffectedImageCount != 1 {
		t.Fatalf("affected_image_count = %d, want 1", preview.AffectedImageCount)
	}
}

func TestTagAdminServiceGetParentCandidatesReturnsExpectedLevels(t *testing.T) {
	t.Parallel()

	service, _, _, _ := newTagAdminServiceForTest(t)

	parents, err := service.GetParentCandidates(context.Background(), domain.TagLevelParent)
	if err != nil {
		t.Fatalf("GetParentCandidates(parent) error = %v", err)
	}
	if len(parents) != 1 || parents[0].ID != 6 {
		t.Fatalf("unexpected parent candidates for parent level: %+v", parents)
	}

	children, err := service.GetParentCandidates(context.Background(), domain.TagLevelChild)
	if err != nil {
		t.Fatalf("GetParentCandidates(child) error = %v", err)
	}
	if len(children) != 2 || children[0].ID != 4 || children[1].ID != 7 {
		t.Fatalf("unexpected parent candidates for child level: %+v", children)
	}
}

func TestTagAdminServiceGetTagTreeBuildsHierarchy(t *testing.T) {
	t.Parallel()

	service, _, _, _ := newTagAdminServiceForTest(t)

	tree, err := service.GetTagTree(context.Background())
	if err != nil {
		t.Fatalf("GetTagTree() error = %v", err)
	}
	if len(tree) == 0 {
		t.Fatal("expected non-empty tree")
	}
	var root *TagTreeNode
	var parent *TagTreeNode
	for i := range tree {
		if tree[i].TagID == 6 {
			root = &tree[i]
			break
		}
	}
	if root == nil {
		t.Fatal("expected root tag 6 in tree")
	}
	if root.Level != domain.TagLevelRoot {
		t.Fatalf("root.Level = %q, want %q", root.Level, domain.TagLevelRoot)
	}
	if len(root.Children) != 2 || root.Children[0].TagID != 4 || root.Children[1].TagID != 7 {
		t.Fatalf("unexpected children for root node: %+v", root.Children)
	}
	parent = &root.Children[0]
	if parent.Level != domain.TagLevelParent {
		t.Fatalf("parent.Level = %q, want %q", parent.Level, domain.TagLevelParent)
	}
	if len(parent.Children) != 1 || parent.Children[0].TagID != 5 {
		t.Fatalf("unexpected children for parent node: %+v", parent.Children)
	}
	if root.TreeUsageCount < root.UsageCount {
		t.Fatalf("tree usage should be >= direct usage, got tree=%d direct=%d", root.TreeUsageCount, root.UsageCount)
	}
}

func TestTagAdminServiceGetTagTreeHandlesLargeOrphanPopulation(t *testing.T) {
	t.Parallel()

	service, tagRepo, _, imageTagRepo := newTagAdminServiceForTest(t)

	const extraOrphans = 33005
	for i := 0; i < extraOrphans; i++ {
		label := "orphan-large-" + strconv.Itoa(i)
		tag := mustSaveAdminTag(t, tagRepo, &domain.Tag{
			PreferredLabel:  label,
			Slug:            label,
			Level:           domain.TagLevelChild,
			PrimaryCategory: "meta",
			ReviewState:     "confirmed",
		})
		if i < 3 {
			if err := imageTagRepo.Save(context.Background(), &domain.ImageTag{ImageID: 1, TagID: tag.ID, Source: domain.ImageTagSourceManual, ReviewState: "confirmed"}); err != nil {
				t.Fatalf("seed orphan image tag %d: %v", i, err)
			}
		}
	}

	tree, err := service.GetTagTree(context.Background())
	if err != nil {
		t.Fatalf("GetTagTree() error = %v", err)
	}

	wantRoots := 4 + extraOrphans
	if len(tree) != wantRoots {
		t.Fatalf("len(tree) = %d, want %d", len(tree), wantRoots)
	}

	var orphan *TagTreeNode
	for i := range tree {
		if tree[i].PreferredLabel == "orphan-large-0" {
			orphan = &tree[i]
			break
		}
	}
	if orphan == nil {
		t.Fatal("expected orphan-large-0 in root tree")
	}
	if len(orphan.Children) != 0 {
		t.Fatalf("orphan-large-0 children = %+v, want none", orphan.Children)
	}
	if orphan.TreeUsageCount != 1 {
		t.Fatalf("orphan-large-0 tree usage = %d, want 1", orphan.TreeUsageCount)
	}
}

func TestTagAdminServiceBatchResolveDescendantsReturnsSelfAndDescendants(t *testing.T) {
	t.Parallel()

	service, _, _, _ := newTagAdminServiceForTest(t)

	result, err := service.batchResolveDescendants(context.Background(), []int64{4, 6})
	if err != nil {
		t.Fatalf("batchResolveDescendants() error = %v", err)
	}

	assertSameInt64Set(t, result[4], []int64{4, 5})
	assertSameInt64Set(t, result[6], []int64{4, 5, 6, 7})
}

func TestTagAdminServiceBatchComputeHierarchyStatsPreservesDistinctImageCounts(t *testing.T) {
	t.Parallel()

	service, _, _, imageTagRepo := newTagAdminServiceForTest(t)
	if err := imageTagRepo.Save(context.Background(), &domain.ImageTag{ImageID: 1, TagID: 4, Source: domain.ImageTagSourceAI, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("seed overlapping hierarchy image tag: %v", err)
	}

	descendants, err := service.batchResolveDescendants(context.Background(), []int64{1, 4, 6})
	if err != nil {
		t.Fatalf("batchResolveDescendants() error = %v", err)
	}

	stats, err := service.batchComputeHierarchyStats(context.Background(), descendants)
	if err != nil {
		t.Fatalf("batchComputeHierarchyStats() error = %v", err)
	}

	if stats[1].DirectUsageCount != 2 || stats[1].TreeUsageCount != 2 {
		t.Fatalf("tag 1 usage counts = %+v, want direct=2 tree=2", *stats[1])
	}
	if stats[4].DirectUsageCount != 1 || stats[4].TreeUsageCount != 1 {
		t.Fatalf("tag 4 usage counts = %+v, want direct=1 tree=1", *stats[4])
	}
	if stats[6].DirectUsageCount != 0 || stats[6].TreeUsageCount != 1 {
		t.Fatalf("tag 6 usage counts = %+v, want direct=0 tree=1", *stats[6])
	}
	if stats[6].TreeConfirmedCount != 1 || stats[6].TreeAICount != 1 || stats[6].TreeManualCount != 1 {
		t.Fatalf("tag 6 tree counts = %+v, want confirmed=1 ai=1 manual=1", *stats[6])
	}
}

func TestTagAdminServiceBatchFindChildrenGroupsChildrenByParent(t *testing.T) {
	t.Parallel()

	service, _, _, _ := newTagAdminServiceForTest(t)

	childrenMap, err := service.batchFindChildren(context.Background(), []int64{4, 6})
	if err != nil {
		t.Fatalf("batchFindChildren() error = %v", err)
	}

	if len(childrenMap[4]) != 1 || childrenMap[4][0].ID != 5 {
		t.Fatalf("childrenMap[4] = %+v, want only child 5", childrenMap[4])
	}
	if len(childrenMap[6]) != 2 || childrenMap[6][0].ID != 4 || childrenMap[6][1].ID != 7 {
		t.Fatalf("childrenMap[6] = %+v, want children 4 and 7", childrenMap[6])
	}
}

func TestTagAdminServiceChangeLevelUpdatesHierarchy(t *testing.T) {
	t.Parallel()

	service, tagRepo, _, _ := newTagAdminServiceForTest(t)
	newRoot := mustSaveAdminTag(t, tagRepo, &domain.Tag{PreferredLabel: "meta-root", Slug: "meta-root", Level: domain.TagLevelRoot})

	updated, err := service.ChangeLevel(context.Background(), 3, domain.TagLevelParent, &newRoot.ID)
	if err != nil {
		t.Fatalf("ChangeLevel() error = %v", err)
	}
	if updated.Level != domain.TagLevelParent {
		t.Fatalf("Level = %q, want %q", updated.Level, domain.TagLevelParent)
	}
	if updated.ParentID == nil || *updated.ParentID != newRoot.ID {
		t.Fatalf("ParentID = %v, want %d", updated.ParentID, newRoot.ID)
	}
}

func TestTagAdminServiceChangeLevelParentToRootDetachesChildren(t *testing.T) {
	t.Parallel()

	service, tagRepo, _, _ := newTagAdminServiceForTest(t)

	updated, err := service.ChangeLevel(context.Background(), 4, domain.TagLevelRoot, nil)
	if err != nil {
		t.Fatalf("ChangeLevel() error = %v", err)
	}
	if updated.Level != domain.TagLevelRoot {
		t.Fatalf("Level = %q, want %q", updated.Level, domain.TagLevelRoot)
	}
	if updated.ParentID != nil {
		t.Fatalf("ParentID = %v, want nil", updated.ParentID)
	}

	child, err := tagRepo.FindByID(context.Background(), 5)
	if err != nil {
		t.Fatalf("FindByID(child) error = %v", err)
	}
	if child.ParentID != nil {
		t.Fatalf("child.ParentID = %v, want nil", child.ParentID)
	}
}

func TestTagAdminServiceChangeLevelParentToRootRollsBackOnChildDetachFailure(t *testing.T) {
	t.Parallel()

	service, tagRepo, _, _ := newTagAdminServiceForTest(t)

	if _, err := service.db.Exec(`
		CREATE TRIGGER tags_block_child_detach
		BEFORE UPDATE ON tags
		WHEN OLD.id = 5 AND NEW.parent_id IS NULL
		BEGIN
			SELECT RAISE(FAIL, 'detach blocked');
		END;
	`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	_, err := service.ChangeLevel(context.Background(), 4, domain.TagLevelRoot, nil)
	if err == nil {
		t.Fatal("expected ChangeLevel() to fail when child detach update fails")
	}

	parent, err := tagRepo.FindByID(context.Background(), 4)
	if err != nil {
		t.Fatalf("FindByID(parent) error = %v", err)
	}
	if parent.Level != domain.TagLevelParent {
		t.Fatalf("parent.Level = %q, want %q", parent.Level, domain.TagLevelParent)
	}

	child, err := tagRepo.FindByID(context.Background(), 5)
	if err != nil {
		t.Fatalf("FindByID(child) error = %v", err)
	}
	if child.ParentID == nil || *child.ParentID != 4 {
		t.Fatalf("child.ParentID = %v, want 4", child.ParentID)
	}
}

func TestTagAdminServiceReparentTagAllowsChildDetach(t *testing.T) {
	t.Parallel()

	service, _, _, _ := newTagAdminServiceForTest(t)

	updated, err := service.ReparentTag(context.Background(), 5, nil)
	if err != nil {
		t.Fatalf("ReparentTag() error = %v", err)
	}
	if updated.ParentID != nil {
		t.Fatalf("ParentID = %v, want nil", updated.ParentID)
	}
}

func TestTagAdminServiceCleanupUnusedTagsProcessesSelectedIDsOnly(t *testing.T) {
	t.Parallel()

	service, tagRepo, _, _ := newTagAdminServiceForTest(t)

	result, err := service.CleanupUnusedTags(context.Background(), []int64{1, 3, 999})
	if err != nil {
		t.Fatalf("CleanupUnusedTags() error = %v", err)
	}

	if len(result.Deleted) != 1 || result.Deleted[0].TagID != 3 {
		t.Fatalf("deleted = %+v, want only tag_id=3", result.Deleted)
	}
	if len(result.Blocked) != 1 || result.Blocked[0].TagID != 1 {
		t.Fatalf("blocked = %+v, want only tag_id=1", result.Blocked)
	}
	if len(result.Failed) != 1 || result.Failed[0].TagID != 999 {
		t.Fatalf("failed = %+v, want only tag_id=999", result.Failed)
	}

	if _, err := tagRepo.FindByID(context.Background(), 3); err == nil {
		t.Fatal("expected selected unused tag to be deleted")
	}
	if _, err := tagRepo.FindByID(context.Background(), 2); err != nil {
		t.Fatalf("non-selected tag should not be deleted: %v", err)
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

	return NewTagAdminService(db, tagRepo, aliasRepo, imageTagRepo), tagRepo, aliasRepo, imageTagRepo
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

	for _, tag := range []*domain.Tag{
		{ID: 1, PreferredLabel: "alpha-source", Slug: "alpha-source", Level: domain.TagLevelChild, PrimaryCategory: "artist", ReviewState: "confirmed", UsageCount: 7},
		{ID: 2, PreferredLabel: "alpha-target", Slug: "alpha-target", Level: domain.TagLevelChild, PrimaryCategory: "artist", ReviewState: "confirmed", UsageCount: 4},
		{ID: 3, PreferredLabel: "unused", Slug: "unused", Level: domain.TagLevelChild, PrimaryCategory: "meta", ReviewState: "pending", UsageCount: 0},
		{ID: 6, PreferredLabel: "franchise", Slug: "franchise", Level: domain.TagLevelRoot, PrimaryCategory: "copyright", ReviewState: "confirmed", UsageCount: 1},
		{ID: 4, PreferredLabel: "characters", Slug: "characters", Level: domain.TagLevelParent, ParentID: int64Ptr(6), PrimaryCategory: "artist", ReviewState: "confirmed", UsageCount: 1},
		{ID: 5, PreferredLabel: "heroine", Slug: "heroine", Level: domain.TagLevelChild, ParentID: int64Ptr(4), PrimaryCategory: "artist", ReviewState: "confirmed", UsageCount: 1},
		{ID: 7, PreferredLabel: "outfit", Slug: "outfit", Level: domain.TagLevelParent, ParentID: int64Ptr(6), PrimaryCategory: "artist", ReviewState: "confirmed", UsageCount: 0},
	} {
		tag.CreatedAt = now
		if err := insertSeedTag(db, tag); err != nil {
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
	if err := imageTagRepo.Save(context.Background(), &domain.ImageTag{ImageID: 1, TagID: 5, Source: domain.ImageTagSourceManual, ReviewState: "confirmed"}); err != nil {
		t.Fatalf("seed image_tag 3: %v", err)
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}

func assertSameInt64Set(t *testing.T, got, want []int64) {
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

func mustSaveAdminTag(t *testing.T, repo repository.TagRepository, tag *domain.Tag) *domain.Tag {
	t.Helper()
	if err := repo.Save(context.Background(), tag); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	return tag
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
