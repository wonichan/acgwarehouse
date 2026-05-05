package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

const sqliteBulkQueryChunkSize = 900

type TagGovernanceQuery interface {
	BatchCountDirectAssociations(ctx context.Context, tagIDs []int64) (map[int64]int64, error)
	BatchDirectHierarchyStats(ctx context.Context, tagIDs []int64) (map[int64]TagDirectHierarchyStats, error)
	BatchFindChildren(ctx context.Context, tagIDs []int64) (map[int64][]*domain.Tag, error)
	BatchResolveDescendants(ctx context.Context, tagIDs []int64) (map[int64][]int64, error)
	BatchTreeImageTagRows(ctx context.Context, tagIDs []int64) ([]TagTreeImageTagRow, error)
	ComputeHierarchyStats(ctx context.Context, tagIDs []int64) (*TagHierarchyStats, error)
}

type TagHierarchyStats struct {
	UsageCount     int64
	PendingCount   int64
	ConfirmedCount int64
	RejectedCount  int64
	AICount        int64
	ManualCount    int64
}

type TagDirectHierarchyStats = TagHierarchyStats

type TagTreeImageTagRow struct {
	TagID       int64
	ImageID     int64
	ReviewState string
	Source      string
}

type sqliteTagGovernanceQuery struct {
	db *sql.DB
}

func NewTagGovernanceQuery(db *sql.DB) TagGovernanceQuery {
	return &sqliteTagGovernanceQuery{db: db}
}

func (q *sqliteTagGovernanceQuery) ComputeHierarchyStats(ctx context.Context, tagIDs []int64) (*TagHierarchyStats, error) {
	stats := &TagHierarchyStats{}
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

	err := q.db.QueryRowContext(ctx, query, args...).Scan(
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

func (q *sqliteTagGovernanceQuery) BatchResolveDescendants(ctx context.Context, tagIDs []int64) (map[int64][]int64, error) {
	result := make(map[int64][]int64, len(tagIDs))
	if len(tagIDs) == 0 {
		return result, nil
	}
	for _, id := range tagIDs {
		result[id] = []int64{}
	}

	for _, chunk := range ChunkInt64IDs(tagIDs, sqliteBulkQueryChunkSize) {
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

		rows, err := q.db.QueryContext(ctx, query, args...)
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

func (q *sqliteTagGovernanceQuery) BatchFindChildren(ctx context.Context, tagIDs []int64) (map[int64][]*domain.Tag, error) {
	result := make(map[int64][]*domain.Tag, len(tagIDs))
	if len(tagIDs) == 0 {
		return result, nil
	}
	for _, id := range tagIDs {
		result[id] = []*domain.Tag{}
	}

	for _, chunk := range ChunkInt64IDs(tagIDs, sqliteBulkQueryChunkSize) {
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

		rows, err := q.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			tag := &domain.Tag{}
			if err := scanTag(rows, tag); err != nil {
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

func (q *sqliteTagGovernanceQuery) BatchDirectHierarchyStats(ctx context.Context, tagIDs []int64) (map[int64]TagDirectHierarchyStats, error) {
	result := make(map[int64]TagDirectHierarchyStats, len(tagIDs))
	if len(tagIDs) == 0 {
		return result, nil
	}

	for _, chunk := range ChunkInt64IDs(tagIDs, sqliteBulkQueryChunkSize) {
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

		rows, err := q.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var tagID int64
			var stats TagDirectHierarchyStats
			if err := rows.Scan(&tagID, &stats.UsageCount, &stats.PendingCount, &stats.ConfirmedCount, &stats.RejectedCount, &stats.AICount, &stats.ManualCount); err != nil {
				rows.Close()
				return nil, err
			}
			result[tagID] = stats
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

func (q *sqliteTagGovernanceQuery) BatchTreeImageTagRows(ctx context.Context, tagIDs []int64) ([]TagTreeImageTagRow, error) {
	if len(tagIDs) == 0 {
		return []TagTreeImageTagRow{}, nil
	}

	result := make([]TagTreeImageTagRow, 0)
	for _, chunk := range ChunkInt64IDs(tagIDs, sqliteBulkQueryChunkSize) {
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

		rows, err := q.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var row TagTreeImageTagRow
			if err := rows.Scan(&row.TagID, &row.ImageID, &row.ReviewState, &row.Source); err != nil {
				rows.Close()
				return nil, err
			}
			result = append(result, row)
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

func (q *sqliteTagGovernanceQuery) BatchCountDirectAssociations(ctx context.Context, tagIDs []int64) (map[int64]int64, error) {
	result := make(map[int64]int64, len(tagIDs))
	if len(tagIDs) == 0 {
		return result, nil
	}
	for _, id := range tagIDs {
		result[id] = 0
	}
	for _, chunk := range ChunkInt64IDs(tagIDs, sqliteBulkQueryChunkSize) {
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

		rows, err := q.db.QueryContext(ctx, query, args...)
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

func ChunkInt64IDs(ids []int64, chunkSize int) [][]int64 {
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
