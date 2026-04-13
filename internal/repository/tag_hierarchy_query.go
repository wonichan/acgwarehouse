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
