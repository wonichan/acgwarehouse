package service

import (
	"context"
	"sort"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

func (s *TagAdminService) computeHierarchyStats(ctx context.Context, tagIDs []int64) (*hierarchyStats, error) {
	stats, err := s.govQuery.ComputeHierarchyStats(ctx, tagIDs)
	if err != nil {
		return nil, err
	}
	return &hierarchyStats{
		UsageCount:     stats.UsageCount,
		PendingCount:   stats.PendingCount,
		ConfirmedCount: stats.ConfirmedCount,
		RejectedCount:  stats.RejectedCount,
		AICount:        stats.AICount,
		ManualCount:    stats.ManualCount,
	}, nil
}

func (s *TagAdminService) batchResolveDescendants(ctx context.Context, tagIDs []int64) (map[int64][]int64, error) {
	return s.govQuery.BatchResolveDescendants(ctx, tagIDs)
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
	return s.govQuery.BatchFindChildren(ctx, tagIDs)
}

func (s *TagAdminService) batchLoadDirectHierarchyStats(ctx context.Context, tagIDs []int64, result map[int64]*hierarchyStatsResult) error {
	if len(tagIDs) == 0 {
		return nil
	}

	statsMap, err := s.govQuery.BatchDirectHierarchyStats(ctx, tagIDs)
	if err != nil {
		return err
	}
	for tagID, stats := range statsMap {
		entry := result[tagID]
		entry.DirectUsageCount = stats.UsageCount
		entry.DirectPendingCount = stats.PendingCount
		entry.DirectConfirmedCount = stats.ConfirmedCount
		entry.DirectRejectedCount = stats.RejectedCount
		entry.DirectAICount = stats.AICount
		entry.DirectManualCount = stats.ManualCount
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

	rows, err := s.govQuery.BatchTreeImageTagRows(ctx, descendantIDs)
	if err != nil {
		return err
	}
	for _, row := range rows {
		for _, ancestorID := range descendantToAncestors[row.TagID] {
			agg := getAggregate(ancestorID)
			if row.ReviewState != domain.ReviewStateRejected {
				agg.usage[row.ImageID] = struct{}{}
				if row.Source == domain.ImageTagSourceAI {
					agg.ai[row.ImageID] = struct{}{}
				} else {
					agg.manual[row.ImageID] = struct{}{}
				}
			}
			switch row.ReviewState {
			case domain.ReviewStatePending:
				agg.pending[row.ImageID] = struct{}{}
			case domain.ReviewStateConfirmed:
				agg.confirmed[row.ImageID] = struct{}{}
			}
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
	return s.govQuery.BatchCountDirectAssociations(ctx, tagIDs)
}
