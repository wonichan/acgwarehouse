package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/logger"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

var (
	ErrTagNotFound            = errors.New("tag not found")
	ErrMergeSameSourceTarget  = errors.New("source and target tags must be different")
	ErrCrossLevelMerge        = errors.New("merge requires tags at the same level")
	ErrMergeSourceHasChildren = errors.New("merge source has child tags")
	ErrInvalidHierarchy       = errors.New("invalid tag hierarchy")
)

const sqliteBulkQueryChunkSize = 900

type TagGovernanceRow struct {
	TagID                int64    `json:"tag_id"`
	PreferredLabel       string   `json:"preferred_label"`
	Level                string   `json:"level"`
	ParentID             *int64   `json:"parent_id,omitempty"`
	PrimaryCategory      string   `json:"primary_category"`
	Aliases              []string `json:"aliases"`
	UsageCount           int64    `json:"usage_count"`
	DirectUsageCount     int64    `json:"direct_usage_count"`
	TreeUsageCount       int64    `json:"tree_usage_count"`
	PendingCount         int64    `json:"pending_count"`
	DirectPendingCount   int64    `json:"direct_pending_count"`
	TreePendingCount     int64    `json:"tree_pending_count"`
	ConfirmedCount       int64    `json:"confirmed_count"`
	DirectConfirmedCount int64    `json:"direct_confirmed_count"`
	TreeConfirmedCount   int64    `json:"tree_confirmed_count"`
	RejectedCount        int64    `json:"rejected_count"`
	AICount              int64    `json:"ai_count"`
	DirectAICount        int64    `json:"direct_ai_count"`
	TreeAICount          int64    `json:"tree_ai_count"`
	ManualCount          int64    `json:"manual_count"`
	DirectManualCount    int64    `json:"direct_manual_count"`
	TreeManualCount      int64    `json:"tree_manual_count"`
	AffectedImageCount   int64    `json:"affected_image_count"`
	CanDelete            bool     `json:"can_delete"`
}

type TagMergeResult struct {
	SourceTagID               int64 `json:"source_tag_id"`
	TargetTagID               int64 `json:"target_tag_id"`
	MigratedImageAssociations int   `json:"migrated_image_associations"`
	MigratedAliases           int   `json:"migrated_aliases"`
}

type TagDeletePreview struct {
	TagID              int64  `json:"tag_id"`
	PreferredLabel     string `json:"preferred_label"`
	AffectedImageCount int64  `json:"affected_image_count"`
	ChildCount         int64  `json:"child_count"`
	CanDelete          bool   `json:"can_delete"`
	BlockingReason     string `json:"blocking_reason"`
}

type TagDeleteResult struct {
	DeletedTagID       int64 `json:"deleted_tag_id"`
	AffectedImageCount int64 `json:"affected_image_count"`
	DetachedChildCount int64 `json:"detached_child_count"`
}

type TagCleanupEntry struct {
	TagID              int64  `json:"tag_id"`
	PreferredLabel     string `json:"preferred_label"`
	AffectedImageCount int64  `json:"affected_image_count,omitempty"`
	BlockingReason     string `json:"blocking_reason,omitempty"`
	Error              string `json:"error,omitempty"`
}

type TagCleanupResult struct {
	Deleted []TagCleanupEntry `json:"deleted"`
	Blocked []TagCleanupEntry `json:"blocked"`
	Failed  []TagCleanupEntry `json:"failed"`
}

type TagTreeNode struct {
	TagID          int64         `json:"tag_id"`
	PreferredLabel string        `json:"preferred_label"`
	Level          string        `json:"level"`
	ParentID       *int64        `json:"parent_id,omitempty"`
	UsageCount     int64         `json:"usage_count"`
	TreeUsageCount int64         `json:"tree_usage_count"`
	Children       []TagTreeNode `json:"children"`
}

type TagAdminService struct {
	db           *sql.DB
	tagRepo      repository.TagRepository
	aliasRepo    repository.TagAliasRepository
	imageTagRepo repository.ImageTagRepository
}

type hierarchyStats struct {
	UsageCount     int64
	PendingCount   int64
	ConfirmedCount int64
	RejectedCount  int64
	AICount        int64
	ManualCount    int64
}

type hierarchyStatsResult struct {
	DirectUsageCount     int64
	DirectPendingCount   int64
	DirectConfirmedCount int64
	DirectAICount        int64
	DirectManualCount    int64
	TreeUsageCount       int64
	TreePendingCount     int64
	TreeConfirmedCount   int64
	TreeAICount          int64
	TreeManualCount      int64
	DirectRejectedCount  int64
}

func NewTagAdminService(db *sql.DB, tagRepo repository.TagRepository, aliasRepo repository.TagAliasRepository, imageTagRepo repository.ImageTagRepository) *TagAdminService {
	return &TagAdminService{
		db:           db,
		tagRepo:      tagRepo,
		aliasRepo:    aliasRepo,
		imageTagRepo: imageTagRepo,
	}
}

func (s *TagAdminService) ListGovernanceTags(ctx context.Context, search string, limit, offset int) ([]TagGovernanceRow, int, error) {
	search = strings.TrimSpace(search)
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	tags, total, err := s.resolveTagSlice(ctx, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	if len(tags) == 0 {
		return []TagGovernanceRow{}, total, nil
	}

	tagIDs := make([]int64, len(tags))
	for i, t := range tags {
		tagIDs[i] = t.ID
	}

	descMap, err := s.batchResolveDescendants(ctx, tagIDs)
	if err != nil {
		return nil, 0, err
	}

	statsMap, err := s.batchComputeHierarchyStats(ctx, descMap)
	if err != nil {
		return nil, 0, err
	}

	directAssocMap, err := s.batchCountDirectAssociations(ctx, tagIDs)
	if err != nil {
		return nil, 0, err
	}

	rows := make([]TagGovernanceRow, 0, len(tags))
	for _, tag := range tags {
		aliases, aliasErr := s.aliasRepo.FindByTagID(ctx, tag.ID)
		if aliasErr != nil {
			return nil, 0, aliasErr
		}
		aliasLabels := make([]string, 0, len(aliases))
		for _, alias := range aliases {
			aliasLabels = append(aliasLabels, alias.Label)
		}

		stats := statsMap[tag.ID]
		directAssoc := directAssocMap[tag.ID]

		row := TagGovernanceRow{
			TagID:                tag.ID,
			PreferredLabel:       tag.PreferredLabel,
			Level:                tag.Level,
			ParentID:             tag.ParentID,
			PrimaryCategory:      tag.PrimaryCategory,
			Aliases:              aliasLabels,
			UsageCount:           stats.DirectUsageCount,
			DirectUsageCount:     stats.DirectUsageCount,
			TreeUsageCount:       stats.TreeUsageCount,
			PendingCount:         stats.DirectPendingCount,
			DirectPendingCount:   stats.DirectPendingCount,
			TreePendingCount:     stats.TreePendingCount,
			ConfirmedCount:       stats.DirectConfirmedCount,
			DirectConfirmedCount: stats.DirectConfirmedCount,
			TreeConfirmedCount:   stats.TreeConfirmedCount,
			RejectedCount:        stats.DirectRejectedCount,
			AICount:              stats.DirectAICount,
			DirectAICount:        stats.DirectAICount,
			TreeAICount:          stats.TreeAICount,
			ManualCount:          stats.DirectManualCount,
			DirectManualCount:    stats.DirectManualCount,
			TreeManualCount:      stats.TreeManualCount,
			AffectedImageCount:   directAssoc,
			CanDelete:            true,
		}
		rows = append(rows, row)
	}

	return rows, total, nil
}

func (s *TagAdminService) ListGovernanceTagsFiltered(ctx context.Context, filter domain.GovernanceTagFilter) ([]TagGovernanceRow, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	// 1. Resolve ALL candidates without pagination (search + level + orphan)
	candidates, err := s.resolveFilteredCandidates(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	if len(candidates) == 0 {
		return []TagGovernanceRow{}, 0, nil
	}

	// 2. Compute direct stats for ALL candidates
	tagIDs := make([]int64, len(candidates))
	for i, t := range candidates {
		tagIDs[i] = t.ID
	}

	directStats, err := s.computeHierarchyStats(ctx, tagIDs)
	if err != nil {
		return nil, 0, err
	}

	// 3. Apply in-memory filters (usage range, source AI/manual)
	filtered := s.applyMemoryFilters(candidates, directStats, filter)

	// 4. Total from filtered set (before pagination)
	total := len(filtered)

	// 5. Paginate
	offset := filter.Offset
	if offset >= total {
		return []TagGovernanceRow{}, total, nil
	}
	end := total
	if offset+filter.Limit < end {
		end = offset + filter.Limit
	}
	page := filtered[offset:end]

	// 6. Build governance rows with full stats for the paginated page only
	pageIDs := make([]int64, len(page))
	for i, t := range page {
		pageIDs[i] = t.ID
	}

	descMap, err := s.batchResolveDescendants(ctx, pageIDs)
	if err != nil {
		return nil, 0, err
	}

	statsMap, err := s.batchComputeHierarchyStats(ctx, descMap)
	if err != nil {
		return nil, 0, err
	}

	directAssocMap, err := s.batchCountDirectAssociations(ctx, pageIDs)
	if err != nil {
		return nil, 0, err
	}

	rows := make([]TagGovernanceRow, 0, len(page))
	for _, tag := range page {
		aliases, aliasErr := s.aliasRepo.FindByTagID(ctx, tag.ID)
		if aliasErr != nil {
			return nil, 0, aliasErr
		}
		aliasLabels := make([]string, 0, len(aliases))
		for _, alias := range aliases {
			aliasLabels = append(aliasLabels, alias.Label)
		}

		stats := statsMap[tag.ID]
		directAssoc := directAssocMap[tag.ID]

		row := TagGovernanceRow{
			TagID:                tag.ID,
			PreferredLabel:       tag.PreferredLabel,
			Level:                tag.Level,
			ParentID:             tag.ParentID,
			PrimaryCategory:      tag.PrimaryCategory,
			Aliases:              aliasLabels,
			UsageCount:           stats.DirectUsageCount,
			DirectUsageCount:     stats.DirectUsageCount,
			TreeUsageCount:       stats.TreeUsageCount,
			PendingCount:         stats.DirectPendingCount,
			DirectPendingCount:   stats.DirectPendingCount,
			TreePendingCount:     stats.TreePendingCount,
			ConfirmedCount:       stats.DirectConfirmedCount,
			DirectConfirmedCount: stats.DirectConfirmedCount,
			TreeConfirmedCount:   stats.TreeConfirmedCount,
			RejectedCount:        stats.DirectRejectedCount,
			AICount:              stats.DirectAICount,
			DirectAICount:        stats.DirectAICount,
			TreeAICount:          stats.TreeAICount,
			ManualCount:          stats.DirectManualCount,
			DirectManualCount:    stats.DirectManualCount,
			TreeManualCount:      stats.TreeManualCount,
			AffectedImageCount:   directAssoc,
			CanDelete:            true,
		}
		rows = append(rows, row)
	}

	return rows, total, nil
}

// resolveFilteredCandidates resolves all tags matching search, level, and orphan filters
// WITHOUT applying pagination. This ensures total count and pagination are correct.
func (s *TagAdminService) resolveFilteredCandidates(ctx context.Context, filter domain.GovernanceTagFilter) ([]*domain.Tag, error) {
	// Resolve search candidates (full set, no pagination)
	var candidates []*domain.Tag
	if filter.Search != "" {
		results := make(map[int64]*domain.Tag)

		totalCount, _ := s.tagRepo.Count(ctx)
		if totalCount > 0 {
			byLabel, err := s.tagRepo.FindByLabelLike(ctx, filter.Search, totalCount)
			if err != nil {
				return nil, err
			}
			for _, tag := range byLabel {
				results[tag.ID] = tag
			}
		}

		aliases, err := s.aliasRepo.FindByLabelLike(ctx, filter.Search)
		if err != nil {
			return nil, err
		}
		for _, alias := range aliases {
			if _, exists := results[alias.TagID]; exists {
				continue
			}
			tag, err := s.tagRepo.FindByID(ctx, alias.TagID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					continue
				}
				return nil, err
			}
			results[tag.ID] = tag
		}

		candidates = make([]*domain.Tag, 0, len(results))
		for _, tag := range results {
			candidates = append(candidates, tag)
		}
	} else {
		// Count first, then load all with count as limit
		totalCount, err := s.tagRepo.Count(ctx)
		if err != nil {
			return nil, err
		}
		allTags, err := s.tagRepo.FindAll(ctx, totalCount, 0)
		if err != nil {
			return nil, err
		}
		candidates = allTags
	}

	// Apply level filter (OR within levels)
	if len(filter.Levels) > 0 {
		levelSet := make(map[string]bool, len(filter.Levels))
		for _, l := range filter.Levels {
			levelSet[l] = true
		}
		filtered := make([]*domain.Tag, 0, len(candidates))
		for _, tag := range candidates {
			if levelSet[tag.Level] {
				filtered = append(filtered, tag)
			}
		}
		candidates = filtered
	}

	// Apply orphan filter: parent_id IS NULL AND level != 'root'
	if filter.OrphanOnly {
		filtered := make([]*domain.Tag, 0, len(candidates))
		for _, tag := range candidates {
			if tag.ParentID == nil && tag.Level != domain.TagLevelRoot {
				filtered = append(filtered, tag)
			}
		}
		candidates = filtered
	}

	// Sort consistently: UsageCount DESC, ID ASC
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].UsageCount == candidates[j].UsageCount {
			return candidates[i].ID < candidates[j].ID
		}
		return candidates[i].UsageCount > candidates[j].UsageCount
	})

	return candidates, nil
}

// applyMemoryFilters applies usage range and source filters in-memory on the full candidate set.
// These filters depend on computed stats that aren't available at the tag-struct level.
func (s *TagAdminService) applyMemoryFilters(candidates []*domain.Tag, stats *hierarchyStats, filter domain.GovernanceTagFilter) []*domain.Tag {
	if filter.MinUsageCount == nil && filter.MaxUsageCount == nil && !filter.SourceAI && !filter.SourceManual {
		return candidates
	}

	// For source and usage filters, we need per-tag stats.
	// computeHierarchyStats returns aggregate; we need per-tag batch query.
	// Use batchComputeHierarchyStats with self-only descendant map.
	tagIDs := make([]int64, len(candidates))
	for i, t := range candidates {
		tagIDs[i] = t.ID
	}

	// Build self-only descendant map for per-tag direct stats
	selfMap := make(map[int64][]int64, len(tagIDs))
	for _, id := range tagIDs {
		selfMap[id] = []int64{id}
	}

	perTagStats, err := s.batchComputeHierarchyStats(context.Background(), selfMap)
	if err != nil {
		return candidates
	}

	filtered := make([]*domain.Tag, 0, len(candidates))
	for _, tag := range candidates {
		stats, ok := perTagStats[tag.ID]
		if !ok {
			continue
		}

		usageCount := stats.DirectUsageCount
		if filter.MinUsageCount != nil && usageCount < int64(*filter.MinUsageCount) {
			continue
		}
		if filter.MaxUsageCount != nil && usageCount > int64(*filter.MaxUsageCount) {
			continue
		}
		if filter.SourceAI && stats.DirectAICount <= 0 {
			continue
		}
		if filter.SourceManual && stats.DirectManualCount <= 0 {
			continue
		}

		filtered = append(filtered, tag)
	}

	return filtered
}

func (s *TagAdminService) MergeTags(ctx context.Context, sourceTagID, targetTagID int64) (*TagMergeResult, error) {
	logger.Infof("[service] TagAdmin MergeTags started: source_id=%d target_id=%d", sourceTagID, targetTagID)
	if sourceTagID <= 0 || targetTagID <= 0 {
		return nil, ErrTagNotFound
	}
	if sourceTagID == targetTagID {
		return nil, ErrMergeSameSourceTarget
	}
	if s.db == nil {
		return nil, errors.New("database is required")
	}

	mergeResult := &TagMergeResult{
		SourceTagID: sourceTagID,
		TargetTagID: targetTagID,
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	sourceTag, err := queryTagByIDTx(ctx, tx, sourceTagID)
	if err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}
	targetTag, err := queryTagByIDTx(ctx, tx, targetTagID)
	if err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}
	if sourceTag.Level != targetTag.Level {
		return nil, ErrCrossLevelMerge
	}
	children, err := s.tagRepo.FindChildrenByParent(ctx, sourceTagID)
	if err != nil {
		return nil, err
	}
	if len(children) > 0 {
		return nil, ErrMergeSourceHasChildren
	}

	if err := s.mergeImageAssociationsTx(ctx, tx, sourceTagID, targetTagID, mergeResult); err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		return nil, err
	}

	if err := s.mergeAliasesTx(ctx, tx, sourceTag, targetTag, mergeResult); err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM tag_aliases WHERE tag_id = ?`, sourceTagID); err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, sourceTagID); err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		return nil, err
	}
	tx = nil

	if err := s.imageTagRepo.SyncFTSForTag(ctx, targetTagID); err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		return nil, err
	}

	logger.Infof("[service] TagAdmin MergeTags completed: source_id=%d target_id=%d migrated_images=%d migrated_aliases=%d", sourceTagID, targetTagID, mergeResult.MigratedImageAssociations, mergeResult.MigratedAliases)
	return mergeResult, nil
}

func (s *TagAdminService) GetDeletePreview(ctx context.Context, tagID int64) (*TagDeletePreview, error) {
	if tagID <= 0 {
		return nil, ErrTagNotFound
	}

	tag, err := s.tagRepo.FindByID(ctx, tagID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}

	children, err := s.tagRepo.FindChildrenByParent(ctx, tagID)
	if err != nil {
		return nil, err
	}
	directAssociationCount, err := s.countDirectAssociations(ctx, tagID)
	if err != nil {
		return nil, err
	}

	preview := &TagDeletePreview{
		TagID:              tag.ID,
		PreferredLabel:     tag.PreferredLabel,
		AffectedImageCount: directAssociationCount,
		ChildCount:         int64(len(children)),
		CanDelete:          true,
	}

	return preview, nil
}

func (s *TagAdminService) DeleteTag(ctx context.Context, tagID int64) (*TagDeleteResult, error) {
	if tagID <= 0 {
		return nil, ErrTagNotFound
	}
	if s.db == nil {
		return nil, errors.New("database is required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err := queryTagByIDTx(ctx, tx, tagID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}

	children, err := queryChildrenByParentTx(ctx, tx, tagID)
	if err != nil {
		return nil, err
	}
	affectedImageCount, err := countDirectAssociationsTx(ctx, tx, tagID)
	if err != nil {
		return nil, err
	}
	imageIDs, err := listImageIDsByTagTx(ctx, tx, tagID)
	if err != nil {
		return nil, err
	}

	for _, child := range children {
		if _, err := tx.ExecContext(ctx, `UPDATE tags SET parent_id = NULL WHERE id = ?`, child.ID); err != nil {
			return nil, err
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM image_tags WHERE tag_id = ?`, tagID); err != nil {
		return nil, err
	}
	for _, imageID := range imageIDs {
		if err := syncImageFTSForTx(ctx, tx, imageID); err != nil {
			return nil, err
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM tag_aliases WHERE tag_id = ?`, tagID); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, tagID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil

	return &TagDeleteResult{
		DeletedTagID:       tagID,
		AffectedImageCount: affectedImageCount,
		DetachedChildCount: int64(len(children)),
	}, nil
}

func (s *TagAdminService) CleanupUnusedTags(ctx context.Context, tagIDs []int64) (*TagCleanupResult, error) {
	logger.Infof("[service] TagAdmin CleanupUnusedTags started: tag_count=%d", len(tagIDs))
	result := &TagCleanupResult{
		Deleted: make([]TagCleanupEntry, 0),
		Blocked: make([]TagCleanupEntry, 0),
		Failed:  make([]TagCleanupEntry, 0),
	}

	for _, tagID := range tagIDs {
		preview, err := s.GetDeletePreview(ctx, tagID)
		if err != nil {
			logger.Errorf("[service] TagAdmin CleanupUnusedTags failed GetDeletePreview for tag_id=%d: %v", tagID, err)
			entry := TagCleanupEntry{TagID: tagID}
			if errors.Is(err, ErrTagNotFound) {
				entry.Error = "tag not found"
			} else {
				entry.Error = err.Error()
			}
			result.Failed = append(result.Failed, entry)
			continue
		}

		if preview.AffectedImageCount > 0 || preview.ChildCount > 0 {
			blockingReason := "merge_or_reclassify_required"
			if preview.ChildCount > 0 {
				blockingReason = "child_tags_exist"
			}
			result.Blocked = append(result.Blocked, TagCleanupEntry{
				TagID:              preview.TagID,
				PreferredLabel:     preview.PreferredLabel,
				AffectedImageCount: preview.AffectedImageCount,
				BlockingReason:     blockingReason,
			})
			continue
		}

		if err := s.deleteUnusedTag(ctx, preview.TagID); err != nil {
			logger.Errorf("[service] TagAdmin CleanupUnusedTags failed deleteUnusedTag for tag_id=%d: %v", preview.TagID, err)
			result.Failed = append(result.Failed, TagCleanupEntry{
				TagID:          preview.TagID,
				PreferredLabel: preview.PreferredLabel,
				Error:          err.Error(),
			})
			continue
		}

		result.Deleted = append(result.Deleted, TagCleanupEntry{
			TagID:          preview.TagID,
			PreferredLabel: preview.PreferredLabel,
		})
	}

	logger.Infof("[service] TagAdmin CleanupUnusedTags completed: deleted=%d blocked=%d failed=%d", len(result.Deleted), len(result.Blocked), len(result.Failed))
	return result, nil
}

func (s *TagAdminService) GetParentCandidates(ctx context.Context, targetLevel string) ([]*domain.Tag, error) {
	return s.tagRepo.FindValidParentCandidates(ctx, targetLevel)
}

func (s *TagAdminService) GetTagTree(ctx context.Context) ([]TagTreeNode, error) {
	total, err := s.tagRepo.Count(ctx)
	if err != nil {
		return nil, err
	}
	tags, err := s.tagRepo.FindAll(ctx, total, 0)
	if err != nil {
		return nil, err
	}

	if len(tags) == 0 {
		return []TagTreeNode{}, nil
	}

	tagIDs := make([]int64, len(tags))
	for i, t := range tags {
		tagIDs[i] = t.ID
	}

	descMap, err := s.batchResolveDescendants(ctx, tagIDs)
	if err != nil {
		return nil, err
	}

	statsMap, err := s.batchComputeHierarchyStats(ctx, descMap)
	if err != nil {
		return nil, err
	}

	nodes := make(map[int64]*TagTreeNode, len(tags))
	childrenByParent := make(map[int64][]*TagTreeNode)
	roots := make([]*TagTreeNode, 0)
	for _, tag := range tags {
		treeStats := statsMap[tag.ID]
		node := &TagTreeNode{
			TagID:          tag.ID,
			PreferredLabel: tag.PreferredLabel,
			Level:          tag.Level,
			ParentID:       tag.ParentID,
			UsageCount:     int64(tag.UsageCount),
			TreeUsageCount: treeStats.TreeUsageCount,
			Children:       []TagTreeNode{},
		}
		nodes[tag.ID] = node
		if tag.ParentID == nil {
			roots = append(roots, node)
			continue
		}
		childrenByParent[*tag.ParentID] = append(childrenByParent[*tag.ParentID], node)
	}

	for id := range nodes {
		children := childrenByParent[id]
		sort.Slice(children, func(i, j int) bool {
			if children[i].UsageCount == children[j].UsageCount {
				return children[i].TagID < children[j].TagID
			}
			return children[i].UsageCount > children[j].UsageCount
		})
	}

	for _, node := range nodes {
		if node.ParentID != nil {
			if _, ok := nodes[*node.ParentID]; !ok {
				roots = append(roots, node)
			}
		}
	}

	sort.Slice(roots, func(i, j int) bool {
		if roots[i].UsageCount == roots[j].UsageCount {
			return roots[i].TagID < roots[j].TagID
		}
		return roots[i].UsageCount > roots[j].UsageCount
	})

	result := make([]TagTreeNode, 0, len(roots))
	for _, root := range roots {
		result = append(result, buildTagTreeNode(root, childrenByParent))
	}
	return result, nil
}

func (s *TagAdminService) ChangeLevel(ctx context.Context, tagID int64, targetLevel string, parentID *int64) (*domain.Tag, error) {
	tag, err := s.tagRepo.FindByID(ctx, tagID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}

	children, err := s.tagRepo.FindChildrenByParent(ctx, tag.ID)
	if err != nil {
		return nil, err
	}
	descendants, err := s.tagRepo.ResolveAllDescendantIDs(ctx, []int64{tag.ID})
	if err != nil {
		return nil, err
	}
	hasChildren := len(children) > 0
	hasDescendants := len(descendants) > 1

	switch {
	case tag.Level == domain.TagLevelParent && targetLevel == domain.TagLevelChild && hasChildren:
		return nil, ErrInvalidHierarchy
	case tag.Level == domain.TagLevelRoot && targetLevel != domain.TagLevelRoot && hasDescendants:
		return nil, ErrInvalidHierarchy
	}

	if err := s.validateHierarchyAssignment(ctx, tag.ID, targetLevel, parentID); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	tag.Level = targetLevel
	tag.ParentID = parentID
	if _, err := tx.ExecContext(ctx, `
		UPDATE tags
		SET preferred_label = ?, slug = ?, level = ?, parent_id = ?, primary_category = ?, review_state = ?, trust_score = ?, usage_count = ?
		WHERE id = ?
	`, tag.PreferredLabel, tag.Slug, tag.Level, tag.ParentID, tag.PrimaryCategory, tag.ReviewState, tag.TrustScore, tag.UsageCount, tag.ID); err != nil {
		return nil, err
	}
	// When a parent-level tag is promoted to root, its direct children must be detached.
	// In the 3-level hierarchy model (root → parent → child), a parent tag can only have
	// child-level tags as direct children. Child-level tags cannot have their own children,
	// so detaching the direct children is sufficient — there are no grandchildren to handle.
	if tag.Level == domain.TagLevelRoot && len(children) > 0 {
		for _, child := range children {
			child.ParentID = nil
			if _, err := tx.ExecContext(ctx, `
				UPDATE tags
				SET preferred_label = ?, slug = ?, level = ?, parent_id = ?, primary_category = ?, review_state = ?, trust_score = ?, usage_count = ?
				WHERE id = ?
			`, child.PreferredLabel, child.Slug, child.Level, child.ParentID, child.PrimaryCategory, child.ReviewState, child.TrustScore, child.UsageCount, child.ID); err != nil {
				return nil, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return tag, nil
}

func (s *TagAdminService) ReparentTag(ctx context.Context, tagID int64, parentID *int64) (*domain.Tag, error) {
	tag, err := s.tagRepo.FindByID(ctx, tagID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}
	if tag.Level == domain.TagLevelRoot {
		return nil, ErrInvalidHierarchy
	}
	if err := s.validateHierarchyAssignment(ctx, tag.ID, tag.Level, parentID); err != nil {
		return nil, err
	}
	// Defensive cycle detection: reject if the proposed parent is a descendant of this tag.
	if parentID != nil {
		isDesc, err := s.isDescendantOf(ctx, *parentID, tag.ID)
		if err != nil {
			return nil, err
		}
		if isDesc {
			return nil, ErrInvalidHierarchy
		}
	}
	tag.ParentID = parentID

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		UPDATE tags
		SET preferred_label = ?, slug = ?, level = ?, parent_id = ?, primary_category = ?, review_state = ?, trust_score = ?, usage_count = ?
		WHERE id = ?
	`, tag.PreferredLabel, tag.Slug, tag.Level, tag.ParentID, tag.PrimaryCategory, tag.ReviewState, tag.TrustScore, tag.UsageCount, tag.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return tag, nil
}

// isDescendantOf checks whether potentialDescendantID is in the descendant tree of ancestorID.
// Returns true if potentialDescendantID is a descendant of (or equal to) ancestorID.
func (s *TagAdminService) isDescendantOf(ctx context.Context, potentialDescendantID, ancestorID int64) (bool, error) {
	descendants, err := s.tagRepo.ResolveAllDescendantIDs(ctx, []int64{ancestorID})
	if err != nil {
		return false, err
	}
	for _, id := range descendants {
		if id == potentialDescendantID {
			return true, nil
		}
	}
	return false, nil
}

func (s *TagAdminService) deleteUnusedTag(ctx context.Context, tagID int64) error {
	aliases, err := s.aliasRepo.FindByTagID(ctx, tagID)
	if err != nil {
		return err
	}
	for _, alias := range aliases {
		if err := s.aliasRepo.Delete(ctx, alias.ID); err != nil {
			return err
		}
	}
	if err := s.tagRepo.Delete(ctx, tagID); err != nil {
		return err
	}
	return nil
}

func (s *TagAdminService) resolveTagSlice(ctx context.Context, search string, limit, offset int) ([]*domain.Tag, int, error) {
	if search == "" {
		tags, err := s.tagRepo.FindAll(ctx, limit, offset)
		if err != nil {
			return nil, 0, err
		}
		total, err := s.tagRepo.Count(ctx)
		if err != nil {
			return nil, 0, err
		}
		return tags, total, nil
	}

	results := make(map[int64]*domain.Tag)

	byLabel, err := s.tagRepo.FindByLabelLike(ctx, search, limit+offset)
	if err != nil {
		return nil, 0, err
	}
	for _, tag := range byLabel {
		results[tag.ID] = tag
	}

	aliases, err := s.aliasRepo.FindByLabelLike(ctx, search)
	if err != nil {
		return nil, 0, err
	}
	for _, alias := range aliases {
		if _, exists := results[alias.TagID]; exists {
			continue
		}
		tag, err := s.tagRepo.FindByID(ctx, alias.TagID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, 0, err
		}
		results[tag.ID] = tag
	}

	merged := make([]*domain.Tag, 0, len(results))
	for _, tag := range results {
		merged = append(merged, tag)
	}
	sort.Slice(merged, func(i, j int) bool {
		if merged[i].UsageCount == merged[j].UsageCount {
			return merged[i].ID < merged[j].ID
		}
		return merged[i].UsageCount > merged[j].UsageCount
	})

	total := len(merged)
	if offset >= total {
		return []*domain.Tag{}, total, nil
	}
	end := total
	if offset+limit < end {
		end = offset + limit
	}
	return merged[offset:end], total, nil
}

func (s *TagAdminService) computeHierarchyStats(ctx context.Context, tagIDs []int64) (*hierarchyStats, error) {
	stats := &hierarchyStats{}
	if len(tagIDs) == 0 {
		return stats, nil
	}

	placeholders := make([]string, len(tagIDs))
	args := make([]any, 0, len(tagIDs))
	for i, id := range tagIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf(`
		SELECT
			COUNT(DISTINCT CASE WHEN review_state != 'rejected' THEN image_id END) as usage_count,
			COUNT(DISTINCT CASE WHEN review_state = 'pending' THEN image_id END) as pending_count,
			COUNT(DISTINCT CASE WHEN review_state = 'confirmed' THEN image_id END) as confirmed_count,
			COUNT(DISTINCT CASE WHEN review_state = 'rejected' THEN image_id END) as rejected_count,
			COUNT(DISTINCT CASE WHEN source = 'ai' AND review_state != 'rejected' THEN image_id END) as ai_count,
			COUNT(DISTINCT CASE WHEN COALESCE(source, 'manual') != 'ai' AND review_state != 'rejected' THEN image_id END) as manual_count
		FROM image_tags
		WHERE tag_id IN (%s)
	`, strings.Join(placeholders, ", "))

	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&stats.UsageCount,
		&stats.PendingCount,
		&stats.ConfirmedCount,
		&stats.RejectedCount,
		&stats.AICount,
		&stats.ManualCount,
	)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func (s *TagAdminService) batchResolveDescendants(ctx context.Context, tagIDs []int64) (map[int64][]int64, error) {
	result := make(map[int64][]int64, len(tagIDs))
	if len(tagIDs) == 0 {
		return result, nil
	}
	for _, id := range tagIDs {
		result[id] = []int64{}
	}

	for _, chunk := range chunkInt64IDs(tagIDs, sqliteBulkQueryChunkSize) {
		placeholders := make([]string, len(chunk))
		args := make([]any, len(chunk))
		for i, id := range chunk {
			placeholders[i] = "?"
			args[i] = id
		}

		query := fmt.Sprintf(`
			WITH RECURSIVE descs(ancestor_id, descendant_id) AS (
				SELECT id AS ancestor_id, id AS descendant_id
				FROM tags
				WHERE id IN (%s)
				UNION ALL
				SELECT d.ancestor_id, t.id
				FROM descs d
				JOIN tags t ON t.parent_id = d.descendant_id
			)
			SELECT ancestor_id, descendant_id
			FROM descs
			ORDER BY ancestor_id, descendant_id
		`, strings.Join(placeholders, ", "))

		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var ancestorID, descendantID int64
			if err := rows.Scan(&ancestorID, &descendantID); err != nil {
				rows.Close()
				return nil, err
			}
			result[ancestorID] = append(result[ancestorID], descendantID)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (s *TagAdminService) batchComputeHierarchyStats(ctx context.Context, descMap map[int64][]int64) (map[int64]*hierarchyStatsResult, error) {
	result := make(map[int64]*hierarchyStatsResult, len(descMap))
	if len(descMap) == 0 {
		return result, nil
	}

	ancestorIDs := make([]int64, 0, len(descMap))
	uniqueDescendants := make(map[int64]struct{})
	descendantToAncestors := make(map[int64][]int64)
	for ancestorID, descendants := range descMap {
		result[ancestorID] = &hierarchyStatsResult{}
		ancestorIDs = append(ancestorIDs, ancestorID)
		for _, descendantID := range descendants {
			uniqueDescendants[descendantID] = struct{}{}
			descendantToAncestors[descendantID] = append(descendantToAncestors[descendantID], ancestorID)
		}
	}
	sort.Slice(ancestorIDs, func(i, j int) bool { return ancestorIDs[i] < ancestorIDs[j] })

	if err := s.batchLoadDirectHierarchyStats(ctx, ancestorIDs, result); err != nil {
		return nil, err
	}
	if err := s.batchLoadTreeHierarchyStats(ctx, uniqueDescendants, descendantToAncestors, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *TagAdminService) batchFindChildren(ctx context.Context, tagIDs []int64) (map[int64][]*domain.Tag, error) {
	result := make(map[int64][]*domain.Tag, len(tagIDs))
	if len(tagIDs) == 0 {
		return result, nil
	}
	for _, id := range tagIDs {
		result[id] = []*domain.Tag{}
	}

	for _, chunk := range chunkInt64IDs(tagIDs, sqliteBulkQueryChunkSize) {
		placeholders := make([]string, len(chunk))
		args := make([]any, len(chunk))
		for i, id := range chunk {
			placeholders[i] = "?"
			args[i] = id
		}
		query := fmt.Sprintf(`
			SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
			FROM tags
			WHERE parent_id IN (%s)
			ORDER BY usage_count DESC, id ASC
		`, strings.Join(placeholders, ", "))

		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			tag := &domain.Tag{}
			if err := scanServiceTag(rows, tag); err != nil {
				rows.Close()
				return nil, err
			}
			if tag.ParentID != nil {
				result[*tag.ParentID] = append(result[*tag.ParentID], tag)
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *TagAdminService) batchLoadDirectHierarchyStats(ctx context.Context, tagIDs []int64, result map[int64]*hierarchyStatsResult) error {
	if len(tagIDs) == 0 {
		return nil
	}

	for _, chunk := range chunkInt64IDs(tagIDs, sqliteBulkQueryChunkSize) {
		placeholders := make([]string, len(chunk))
		args := make([]any, len(chunk))
		for i, id := range chunk {
			placeholders[i] = "?"
			args[i] = id
		}

		query := fmt.Sprintf(`
			SELECT
				tag_id,
				COUNT(DISTINCT CASE WHEN review_state != 'rejected' THEN image_id END) AS usage_count,
				COUNT(DISTINCT CASE WHEN review_state = 'pending' THEN image_id END) AS pending_count,
				COUNT(DISTINCT CASE WHEN review_state = 'confirmed' THEN image_id END) AS confirmed_count,
				COUNT(DISTINCT CASE WHEN review_state = 'rejected' THEN image_id END) AS rejected_count,
				COUNT(DISTINCT CASE WHEN source = 'ai' AND review_state != 'rejected' THEN image_id END) AS ai_count,
				COUNT(DISTINCT CASE WHEN COALESCE(source, 'manual') != 'ai' AND review_state != 'rejected' THEN image_id END) AS manual_count
			FROM image_tags
			WHERE tag_id IN (%s)
			GROUP BY tag_id
		`, strings.Join(placeholders, ", "))

		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}

		for rows.Next() {
			var tagID int64
			var directUsage, directPending, directConfirmed, directRejected, directAI, directManual int64
			if err := rows.Scan(&tagID, &directUsage, &directPending, &directConfirmed, &directRejected, &directAI, &directManual); err != nil {
				rows.Close()
				return err
			}
			entry := result[tagID]
			entry.DirectUsageCount = directUsage
			entry.DirectPendingCount = directPending
			entry.DirectConfirmedCount = directConfirmed
			entry.DirectRejectedCount = directRejected
			entry.DirectAICount = directAI
			entry.DirectManualCount = directManual
		}

		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (s *TagAdminService) batchLoadTreeHierarchyStats(ctx context.Context, uniqueDescendants map[int64]struct{}, descendantToAncestors map[int64][]int64, result map[int64]*hierarchyStatsResult) error {
	if len(uniqueDescendants) == 0 {
		return nil
	}

	descendantIDs := make([]int64, 0, len(uniqueDescendants))
	for id := range uniqueDescendants {
		descendantIDs = append(descendantIDs, id)
	}
	sort.Slice(descendantIDs, func(i, j int) bool { return descendantIDs[i] < descendantIDs[j] })

	type imageSet map[int64]struct{}
	type aggregateSets struct {
		usage     imageSet
		pending   imageSet
		confirmed imageSet
		ai        imageSet
		manual    imageSet
	}

	aggregates := make(map[int64]*aggregateSets, len(result))
	getAggregate := func(ancestorID int64) *aggregateSets {
		if aggregates[ancestorID] == nil {
			aggregates[ancestorID] = &aggregateSets{
				usage:     imageSet{},
				pending:   imageSet{},
				confirmed: imageSet{},
				ai:        imageSet{},
				manual:    imageSet{},
			}
		}
		return aggregates[ancestorID]
	}

	for _, chunk := range chunkInt64IDs(descendantIDs, sqliteBulkQueryChunkSize) {
		placeholders := make([]string, len(chunk))
		args := make([]any, len(chunk))
		for i, id := range chunk {
			placeholders[i] = "?"
			args[i] = id
		}

		query := fmt.Sprintf(`
			SELECT tag_id, image_id, review_state, COALESCE(source, 'manual')
			FROM image_tags
			WHERE tag_id IN (%s)
		`, strings.Join(placeholders, ", "))

		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}

		for rows.Next() {
			var tagID, imageID int64
			var reviewState, source string
			if err := rows.Scan(&tagID, &imageID, &reviewState, &source); err != nil {
				rows.Close()
				return err
			}

			for _, ancestorID := range descendantToAncestors[tagID] {
				agg := getAggregate(ancestorID)
				if reviewState != "rejected" {
					agg.usage[imageID] = struct{}{}
					if source == "ai" {
						agg.ai[imageID] = struct{}{}
					} else {
						agg.manual[imageID] = struct{}{}
					}
				}
				switch reviewState {
				case "pending":
					agg.pending[imageID] = struct{}{}
				case "confirmed":
					agg.confirmed[imageID] = struct{}{}
				}
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
	}

	for ancestorID, agg := range aggregates {
		entry := result[ancestorID]
		entry.TreeUsageCount = int64(len(agg.usage))
		entry.TreePendingCount = int64(len(agg.pending))
		entry.TreeConfirmedCount = int64(len(agg.confirmed))
		entry.TreeAICount = int64(len(agg.ai))
		entry.TreeManualCount = int64(len(agg.manual))
	}

	return nil
}

func (s *TagAdminService) batchCountDirectAssociations(ctx context.Context, tagIDs []int64) (map[int64]int64, error) {
	result := make(map[int64]int64, len(tagIDs))
	if len(tagIDs) == 0 {
		return result, nil
	}
	for _, id := range tagIDs {
		result[id] = 0
	}
	for _, chunk := range chunkInt64IDs(tagIDs, sqliteBulkQueryChunkSize) {
		placeholders := make([]string, len(chunk))
		args := make([]any, len(chunk))
		for i, id := range chunk {
			placeholders[i] = "?"
			args[i] = id
		}
		query := fmt.Sprintf(`
			SELECT tag_id, COUNT(DISTINCT image_id)
			FROM image_tags
			WHERE tag_id IN (%s)
			GROUP BY tag_id
		`, strings.Join(placeholders, ", "))

		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var tagID, count int64
			if err := rows.Scan(&tagID, &count); err != nil {
				rows.Close()
				return nil, err
			}
			result[tagID] = count
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func chunkInt64IDs(ids []int64, chunkSize int) [][]int64 {
	if len(ids) == 0 {
		return nil
	}
	if chunkSize <= 0 || len(ids) <= chunkSize {
		return [][]int64{ids}
	}

	chunks := make([][]int64, 0, (len(ids)+chunkSize-1)/chunkSize)
	for start := 0; start < len(ids); start += chunkSize {
		end := start + chunkSize
		if end > len(ids) {
			end = len(ids)
		}
		chunks = append(chunks, ids[start:end])
	}
	return chunks
}

func (s *TagAdminService) countDirectAssociations(ctx context.Context, tagID int64) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT image_id)
		FROM image_tags
		WHERE tag_id = ?
	`, tagID).Scan(&count)
	return count, err
}

func (s *TagAdminService) validateHierarchyAssignment(ctx context.Context, tagID int64, level string, parentID *int64) error {
	switch level {
	case domain.TagLevelRoot:
		if parentID != nil {
			return ErrInvalidHierarchy
		}
		return nil
	case domain.TagLevelParent:
		if parentID == nil || *parentID == tagID {
			return ErrInvalidHierarchy
		}
		parentTag, err := s.tagRepo.FindByID(ctx, *parentID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrInvalidHierarchy
			}
			return err
		}
		if parentTag.Level != domain.TagLevelRoot {
			return ErrInvalidHierarchy
		}
		return nil
	case domain.TagLevelChild:
		if parentID == nil {
			return nil
		}
		if *parentID == tagID {
			return ErrInvalidHierarchy
		}
		parentTag, err := s.tagRepo.FindByID(ctx, *parentID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrInvalidHierarchy
			}
			return err
		}
		if parentTag.Level != domain.TagLevelParent {
			return ErrInvalidHierarchy
		}
		return nil
	default:
		return ErrInvalidHierarchy
	}
}

func buildTagTreeNode(node *TagTreeNode, childrenByParent map[int64][]*TagTreeNode) TagTreeNode {
	result := *node
	children := childrenByParent[node.TagID]
	result.Children = make([]TagTreeNode, 0, len(children))
	for _, child := range children {
		result.Children = append(result.Children, buildTagTreeNode(child, childrenByParent))
	}
	return result
}

func (s *TagAdminService) mergeImageAssociationsTx(ctx context.Context, tx *sql.Tx, sourceTagID, targetTagID int64, result *TagMergeResult) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT image_id, source, source_observation_id, confidence, review_state
		FROM image_tags
		WHERE tag_id = ?
		ORDER BY image_id ASC
	`, sourceTagID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type mergeRow struct {
		imageID             int64
		source              string
		sourceObservationID *int64
		confidence          float64
		reviewState         string
	}
	mergeRows := make([]mergeRow, 0)
	for rows.Next() {
		var row mergeRow
		if err := rows.Scan(&row.imageID, &row.source, &row.sourceObservationID, &row.confidence, &row.reviewState); err != nil {
			return err
		}
		if row.source == "" {
			row.source = domain.ImageTagSourceManual
		}
		mergeRows = append(mergeRows, row)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, row := range mergeRows {
		if _, err := tx.ExecContext(ctx, `DELETE FROM image_tags WHERE image_id = ? AND tag_id = ?`, row.imageID, sourceTagID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO image_tags (image_id, tag_id, source, source_observation_id, confidence, review_state)
			VALUES (?, ?, ?, ?, ?, ?)
		`, row.imageID, targetTagID, row.source, row.sourceObservationID, row.confidence, row.reviewState); err != nil {
			return err
		}
		if err := syncImageFTSForTx(ctx, tx, row.imageID); err != nil {
			return err
		}
		result.MigratedImageAssociations++
	}

	return nil
}

func (s *TagAdminService) mergeAliasesTx(ctx context.Context, tx *sql.Tx, sourceTag, targetTag *domain.Tag, result *TagMergeResult) error {
	targetAliases, err := queryAliasesByTagIDTx(ctx, tx, targetTag.ID)
	if err != nil {
		return err
	}
	sourceAliases, err := queryAliasesByTagIDTx(ctx, tx, sourceTag.ID)
	if err != nil {
		return err
	}

	existingNormalized := map[string]struct{}{
		repository.NormalizeLabel(targetTag.PreferredLabel): {},
	}
	for _, alias := range targetAliases {
		existingNormalized[repository.NormalizeLabel(alias.Label)] = struct{}{}
	}

	appendAlias := func(alias *domain.TagAlias) error {
		normalized := repository.NormalizeLabel(alias.Label)
		if normalized == "" {
			return nil
		}
		if _, exists := existingNormalized[normalized]; exists {
			return nil
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO tag_aliases (tag_id, label, normalized_label, locale, alias_type, is_preferred)
			VALUES (?, ?, ?, ?, ?, ?)
		`, targetTag.ID, alias.Label, normalized, alias.Locale, alias.AliasType, alias.IsPreferred); err != nil {
			return err
		}
		existingNormalized[normalized] = struct{}{}
		result.MigratedAliases++
		return nil
	}

	for _, sourceAlias := range sourceAliases {
		if err := appendAlias(sourceAlias); err != nil {
			return err
		}
	}

	if repository.NormalizeLabel(sourceTag.PreferredLabel) != repository.NormalizeLabel(targetTag.PreferredLabel) {
		if err := appendAlias(&domain.TagAlias{Label: sourceTag.PreferredLabel, AliasType: "synonym"}); err != nil {
			return err
		}
	}

	return nil
}

func queryTagByIDTx(ctx context.Context, tx *sql.Tx, id int64) (*domain.Tag, error) {
	tag := &domain.Tag{}
	err := scanServiceTag(tx.QueryRowContext(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags
		WHERE id = ?
	`, id), tag)
	if err != nil {
		return nil, err
	}
	return tag, nil
}

func queryAliasesByTagIDTx(ctx context.Context, tx *sql.Tx, tagID int64) ([]*domain.TagAlias, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, tag_id, label, normalized_label, locale, alias_type, is_preferred
		FROM tag_aliases
		WHERE tag_id = ?
		ORDER BY id ASC
	`, tagID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	aliases := make([]*domain.TagAlias, 0)
	for rows.Next() {
		alias := &domain.TagAlias{}
		if err := rows.Scan(&alias.ID, &alias.TagID, &alias.Label, &alias.NormalizedLabel, &alias.Locale, &alias.AliasType, &alias.IsPreferred); err != nil {
			return nil, err
		}
		aliases = append(aliases, alias)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return aliases, nil
}

func queryChildrenByParentTx(ctx context.Context, tx *sql.Tx, parentID int64) ([]*domain.Tag, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags
		WHERE parent_id = ?
		ORDER BY usage_count DESC, id ASC
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	children := make([]*domain.Tag, 0)
	for rows.Next() {
		child := &domain.Tag{}
		if err := scanServiceTag(rows, child); err != nil {
			return nil, err
		}
		children = append(children, child)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return children, nil
}

func scanServiceTag(scanner interface{ Scan(dest ...any) error }, tag *domain.Tag) error {
	var primaryCategory sql.NullString

	if err := scanner.Scan(
		&tag.ID,
		&tag.PreferredLabel,
		&tag.Slug,
		&tag.Level,
		&tag.ParentID,
		&primaryCategory,
		&tag.ReviewState,
		&tag.TrustScore,
		&tag.UsageCount,
		&tag.CreatedAt,
	); err != nil {
		return err
	}

	if primaryCategory.Valid {
		tag.PrimaryCategory = primaryCategory.String
	} else {
		tag.PrimaryCategory = ""
	}

	return nil
}

func countDirectAssociationsTx(ctx context.Context, tx *sql.Tx, tagID int64) (int64, error) {
	var count int64
	err := tx.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT image_id)
		FROM image_tags
		WHERE tag_id = ?
	`, tagID).Scan(&count)
	return count, err
}

func listImageIDsByTagTx(ctx context.Context, tx *sql.Tx, tagID int64) ([]int64, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT DISTINCT image_id
		FROM image_tags
		WHERE tag_id = ?
		ORDER BY image_id ASC
	`, tagID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	imageIDs := make([]int64, 0)
	for rows.Next() {
		var imageID int64
		if err := rows.Scan(&imageID); err != nil {
			return nil, err
		}
		imageIDs = append(imageIDs, imageID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return imageIDs, nil
}

func syncImageFTSForTx(ctx context.Context, tx *sql.Tx, imageID int64) error {
	var tagsText string
	err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(GROUP_CONCAT(t.preferred_label, ' '), '')
		FROM image_tags it
		JOIN tags t ON t.id = it.tag_id
		WHERE it.image_id = ?
	`, imageID).Scan(&tagsText)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE images_fts SET tags = ? WHERE image_id = ?`, tagsText, imageID); err != nil {
		return fmt.Errorf("sync image fts: %w", err)
	}
	return nil
}
