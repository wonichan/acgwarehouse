package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/d1client"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type d1TagGovernanceQuery struct {
	client *d1client.Client
}

func NewD1TagGovernanceQuery(client *d1client.Client) TagGovernanceQuery {
	return &d1TagGovernanceQuery{client: client}
}

func (q *d1TagGovernanceQuery) ComputeHierarchyStats(ctx context.Context, tagIDs []int64) (*TagHierarchyStats, error) {
	stats := &TagHierarchyStats{}
	if len(tagIDs) == 0 {
		return stats, nil
	}
	where, args := int64InClause("tag_id", tagIDs)
	row, err := q.client.QueryOne(ctx, fmt.Sprintf(`
		SELECT
			COUNT(DISTINCT CASE WHEN review_state != 'rejected' THEN image_id END) as usage_count,
			COUNT(DISTINCT CASE WHEN review_state = 'pending' THEN image_id END) as pending_count,
			COUNT(DISTINCT CASE WHEN review_state = 'confirmed' THEN image_id END) as confirmed_count,
			COUNT(DISTINCT CASE WHEN review_state = 'rejected' THEN image_id END) as rejected_count,
			COUNT(DISTINCT CASE WHEN source = 'ai' AND review_state != 'rejected' THEN image_id END) as ai_count,
			COUNT(DISTINCT CASE WHEN COALESCE(source, 'manual') != 'ai' AND review_state != 'rejected' THEN image_id END) as manual_count
		FROM image_tags
		WHERE %s
	`, where), args...)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return stats, nil
	}
	stats.UsageCount, _ = toInt64(row["usage_count"])
	stats.PendingCount, _ = toInt64(row["pending_count"])
	stats.ConfirmedCount, _ = toInt64(row["confirmed_count"])
	stats.RejectedCount, _ = toInt64(row["rejected_count"])
	stats.AICount, _ = toInt64(row["ai_count"])
	stats.ManualCount, _ = toInt64(row["manual_count"])
	return stats, nil
}

func (q *d1TagGovernanceQuery) BatchResolveDescendants(ctx context.Context, tagIDs []int64) (map[int64][]int64, error) {
	result := make(map[int64][]int64, len(tagIDs))
	for _, id := range tagIDs {
		result[id] = []int64{}
	}
	if len(tagIDs) == 0 {
		return result, nil
	}

	for _, chunk := range ChunkInt64IDs(tagIDs, sqliteBulkQueryChunkSize) {
		where, args := int64InClause("id", chunk)
		rows, err := q.client.Query(ctx, fmt.Sprintf(`
			WITH RECURSIVE descs(ancestor_id, descendant_id) AS (
				SELECT id AS ancestor_id, id AS descendant_id
				FROM tags
				WHERE %s
				UNION ALL
				SELECT d.ancestor_id, t.id
				FROM descs d
				JOIN tags t ON t.parent_id = d.descendant_id
			)
			SELECT ancestor_id, descendant_id
			FROM descs
			ORDER BY ancestor_id, descendant_id
		`, where), args...)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			ancestorID, err := toInt64(row["ancestor_id"])
			if err != nil {
				return nil, err
			}
			descendantID, err := toInt64(row["descendant_id"])
			if err != nil {
				return nil, err
			}
			result[ancestorID] = append(result[ancestorID], descendantID)
		}
	}
	return result, nil
}

func (q *d1TagGovernanceQuery) BatchFindChildren(ctx context.Context, tagIDs []int64) (map[int64][]*domain.Tag, error) {
	result := make(map[int64][]*domain.Tag, len(tagIDs))
	for _, id := range tagIDs {
		result[id] = []*domain.Tag{}
	}
	if len(tagIDs) == 0 {
		return result, nil
	}
	for _, chunk := range ChunkInt64IDs(tagIDs, sqliteBulkQueryChunkSize) {
		where, args := int64InClause("parent_id", chunk)
		rows, err := q.client.Query(ctx, fmt.Sprintf(`
			SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
			FROM tags
			WHERE %s
			ORDER BY usage_count DESC, id ASC
		`, where), args...)
		if err != nil {
			return nil, err
		}
		tags, err := mapTagsFromD1(rows)
		if err != nil {
			return nil, err
		}
		for _, tag := range tags {
			if tag.ParentID != nil {
				result[*tag.ParentID] = append(result[*tag.ParentID], tag)
			}
		}
	}
	return result, nil
}

func (q *d1TagGovernanceQuery) BatchDirectHierarchyStats(ctx context.Context, tagIDs []int64) (map[int64]TagDirectHierarchyStats, error) {
	result := make(map[int64]TagDirectHierarchyStats, len(tagIDs))
	if len(tagIDs) == 0 {
		return result, nil
	}
	for _, chunk := range ChunkInt64IDs(tagIDs, sqliteBulkQueryChunkSize) {
		where, args := int64InClause("tag_id", chunk)
		rows, err := q.client.Query(ctx, fmt.Sprintf(`
			SELECT
				tag_id,
				COUNT(DISTINCT CASE WHEN review_state != 'rejected' THEN image_id END) AS usage_count,
				COUNT(DISTINCT CASE WHEN review_state = 'pending' THEN image_id END) AS pending_count,
				COUNT(DISTINCT CASE WHEN review_state = 'confirmed' THEN image_id END) AS confirmed_count,
				COUNT(DISTINCT CASE WHEN review_state = 'rejected' THEN image_id END) AS rejected_count,
				COUNT(DISTINCT CASE WHEN source = 'ai' AND review_state != 'rejected' THEN image_id END) AS ai_count,
				COUNT(DISTINCT CASE WHEN COALESCE(source, 'manual') != 'ai' AND review_state != 'rejected' THEN image_id END) AS manual_count
			FROM image_tags
			WHERE %s
			GROUP BY tag_id
		`, where), args...)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			tagID, err := toInt64(row["tag_id"])
			if err != nil {
				return nil, err
			}
			result[tagID] = tagStatsFromD1Row(row)
		}
	}
	return result, nil
}

func (q *d1TagGovernanceQuery) BatchTreeImageTagRows(ctx context.Context, tagIDs []int64) ([]TagTreeImageTagRow, error) {
	if len(tagIDs) == 0 {
		return []TagTreeImageTagRow{}, nil
	}
	result := make([]TagTreeImageTagRow, 0)
	for _, chunk := range ChunkInt64IDs(tagIDs, sqliteBulkQueryChunkSize) {
		where, args := int64InClause("tag_id", chunk)
		rows, err := q.client.Query(ctx, fmt.Sprintf(`
			SELECT tag_id, image_id, review_state, COALESCE(source, 'manual') as source
			FROM image_tags
			WHERE %s
		`, where), args...)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			tagID, err := toInt64(row["tag_id"])
			if err != nil {
				return nil, err
			}
			imageID, err := toInt64(row["image_id"])
			if err != nil {
				return nil, err
			}
			result = append(result, TagTreeImageTagRow{
				TagID:       tagID,
				ImageID:     imageID,
				ReviewState: toStringDefault(row["review_state"], ""),
				Source:      toStringDefault(row["source"], domain.ImageTagSourceManual),
			})
		}
	}
	return result, nil
}

func (q *d1TagGovernanceQuery) BatchCountDirectAssociations(ctx context.Context, tagIDs []int64) (map[int64]int64, error) {
	result := make(map[int64]int64, len(tagIDs))
	for _, id := range tagIDs {
		result[id] = 0
	}
	if len(tagIDs) == 0 {
		return result, nil
	}
	for _, chunk := range ChunkInt64IDs(tagIDs, sqliteBulkQueryChunkSize) {
		where, args := int64InClause("tag_id", chunk)
		rows, err := q.client.Query(ctx, fmt.Sprintf(`
			SELECT tag_id, COUNT(DISTINCT image_id) as cnt
			FROM image_tags
			WHERE %s
			GROUP BY tag_id
		`, where), args...)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			tagID, err := toInt64(row["tag_id"])
			if err != nil {
				return nil, err
			}
			count, err := toInt64(row["cnt"])
			if err != nil {
				return nil, err
			}
			result[tagID] = count
		}
	}
	return result, nil
}

func int64InClause(column string, ids []int64) (string, []any) {
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	return column + " IN (" + strings.Join(placeholders, ", ") + ")", args
}

func tagStatsFromD1Row(row map[string]any) TagDirectHierarchyStats {
	var stats TagDirectHierarchyStats
	stats.UsageCount, _ = toInt64(row["usage_count"])
	stats.PendingCount, _ = toInt64(row["pending_count"])
	stats.ConfirmedCount, _ = toInt64(row["confirmed_count"])
	stats.RejectedCount, _ = toInt64(row["rejected_count"])
	stats.AICount, _ = toInt64(row["ai_count"])
	stats.ManualCount, _ = toInt64(row["manual_count"])
	return stats
}
