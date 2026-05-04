package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

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
