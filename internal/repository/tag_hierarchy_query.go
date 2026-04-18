package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func expandHierarchicalTagClauses(ctx context.Context, db *sql.DB, tagIDs []int64) ([][]int64, error) {
	if len(tagIDs) == 0 {
		return nil, nil
	}

	resolved, err := NewTagRepository(db).ResolveDescendantIDs(ctx, tagIDs)
	if err != nil {
		return nil, err
	}

	clauses := make([][]int64, 0, len(tagIDs))
	for _, tagID := range tagIDs {
		ids := resolved[tagID]
		if len(ids) == 0 {
			ids = []int64{tagID}
		}
		clauses = append(clauses, ids)
	}

	return clauses, nil
}

func buildImageTagClauseFilters(clauses [][]int64, imageColumn string) (string, []any) {
	if len(clauses) == 0 {
		return "", nil
	}

	parts := make([]string, 0, len(clauses))
	args := make([]any, 0)
	for _, clause := range clauses {
		placeholders := make([]string, len(clause))
		for i, id := range clause {
			placeholders[i] = "?"
			args = append(args, id)
		}
		parts = append(parts, fmt.Sprintf(`%s IN (
			SELECT it.image_id
			FROM image_tags it
			WHERE it.tag_id IN (%s) AND it.review_state != 'rejected'
		)`, imageColumn, strings.Join(placeholders, ", ")))
	}

	return strings.Join(parts, " AND "), args
}

// buildGalleryFilterClauses builds a WHERE clause that combines exact tag matching
// with subtree tag matching using AND semantics.
// - exactTagIDs: each ID matches if the image has that specific tag (no descendant expansion)
// - subtreeRootTagIDs: each ID matches if the image has ANY tag in the subtree (root + descendants)
// Both sets are AND-connected: the image must match ALL exact tags AND ALL subtree root tags.
func buildGalleryFilterClauses(ctx context.Context, db *sql.DB, exactTagIDs, subtreeRootTagIDs []int64, imageColumn string) (string, []any, error) {
	var parts []string
	var args []any

	// Exact clauses: each tag matches precisely (no descendant expansion)
	for _, tagID := range exactTagIDs {
		parts = append(parts, fmt.Sprintf(`%s IN (
			SELECT it.image_id
			FROM image_tags it
			WHERE it.tag_id = ? AND it.review_state != 'rejected'
		)`, imageColumn))
		args = append(args, tagID)
	}

	// Subtree clauses: expand each root into its descendants
	if len(subtreeRootTagIDs) > 0 {
		clauses, err := expandHierarchicalTagClauses(ctx, db, subtreeRootTagIDs)
		if err != nil {
			return "", nil, err
		}
		subtreeWhere, subtreeArgs := buildImageTagClauseFilters(clauses, imageColumn)
		if subtreeWhere != "" {
			parts = append(parts, subtreeWhere)
			args = append(args, subtreeArgs...)
		}
	}

	if len(parts) == 0 {
		return "", nil, nil
	}

	return strings.Join(parts, " AND "), args, nil
}
