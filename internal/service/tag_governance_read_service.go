package service

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

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

	candidates, err := s.resolveFilteredCandidates(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	if len(candidates) == 0 {
		return []TagGovernanceRow{}, 0, nil
	}

	tagIDs := make([]int64, len(candidates))
	for i, t := range candidates {
		tagIDs[i] = t.ID
	}

	directStats, err := s.computeHierarchyStats(ctx, tagIDs)
	if err != nil {
		return nil, 0, err
	}

	filtered := s.applyMemoryFilters(ctx, candidates, directStats, filter)
	total := len(filtered)

	offset := filter.Offset
	if offset >= total {
		return []TagGovernanceRow{}, total, nil
	}
	end := total
	if offset+filter.Limit < end {
		end = offset + filter.Limit
	}
	page := filtered[offset:end]

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

func (s *TagAdminService) resolveFilteredCandidates(ctx context.Context, filter domain.GovernanceTagFilter) ([]*domain.Tag, error) {
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

	if filter.OrphanOnly {
		filtered := make([]*domain.Tag, 0, len(candidates))
		for _, tag := range candidates {
			if tag.ParentID == nil && tag.Level != domain.TagLevelRoot {
				filtered = append(filtered, tag)
			}
		}
		candidates = filtered
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].UsageCount == candidates[j].UsageCount {
			return candidates[i].ID < candidates[j].ID
		}
		return candidates[i].UsageCount > candidates[j].UsageCount
	})

	return candidates, nil
}

func (s *TagAdminService) applyMemoryFilters(ctx context.Context, candidates []*domain.Tag, stats *hierarchyStats, filter domain.GovernanceTagFilter) []*domain.Tag {
	if filter.MinUsageCount == nil && filter.MaxUsageCount == nil && !filter.SourceAI && !filter.SourceManual {
		return candidates
	}

	tagIDs := make([]int64, len(candidates))
	for i, t := range candidates {
		tagIDs[i] = t.ID
	}

	selfMap := make(map[int64][]int64, len(tagIDs))
	for _, id := range tagIDs {
		selfMap[id] = []int64{id}
	}

	perTagStats, err := s.batchComputeHierarchyStats(ctx, selfMap)
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
