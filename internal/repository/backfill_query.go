package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

var activeAIBackfillTaskStatuses = []string{
	domain.PlatformTaskStatusPending,
	domain.PlatformTaskStatusQueued,
	domain.PlatformTaskStatusRunning,
}

type backfillQuery struct {
	baseWhere string
	baseArgs  []any
}

type backfillCondition struct {
	sql  string
	args []any
}

func newBackfillQuery(ctx context.Context, db *sql.DB, filter BackfillCandidateFilter) (*backfillQuery, error) {
	baseWhere, baseArgs, err := buildBackfillBaseWhere(ctx, db, filter)
	if err != nil {
		return nil, err
	}
	return &backfillQuery{baseWhere: baseWhere, baseArgs: baseArgs}, nil
}

func buildBackfillBaseWhere(ctx context.Context, db *sql.DB, filter BackfillCandidateFilter) (string, []any, error) {
	var conds []string
	var args []any

	if len(filter.TagIDs) > 0 {
		clauses, err := expandHierarchicalTagClauses(ctx, db, filter.TagIDs)
		if err != nil {
			return "", nil, err
		}
		cond, clauseArgs := buildImageTagClauseFilters(clauses, "i.id")
		conds = append(conds, cond)
		args = append(args, clauseArgs...)
	}

	if filter.HasTags != nil {
		if !*filter.HasTags {
			conds = append(conds, `NOT EXISTS (SELECT 1 FROM image_tags it2 WHERE it2.image_id = i.id AND it2.review_state != ?)`)
		} else {
			conds = append(conds, `EXISTS (SELECT 1 FROM image_tags it2 WHERE it2.image_id = i.id AND it2.review_state != ?)`)
		}
		args = append(args, domain.ReviewStateRejected)
	}

	if len(conds) > 0 {
		return " AND " + strings.Join(conds, " AND "), args, nil
	}
	return "", args, nil
}

func (q *backfillQuery) hitWhere() (string, []any) {
	return q.baseWhere, append([]any(nil), q.baseArgs...)
}

func (q *backfillQuery) eligibleWhere() (string, []any) {
	aiWhere, aiArgs := backfillAcceptedAITagExists()
	taskWhere, taskArgs := backfillActiveAITagTaskExists()
	return q.combine(
		backfillCondition{sql: "AND NOT EXISTS (" + aiWhere + ")", args: aiArgs},
		backfillCondition{sql: "AND NOT EXISTS (" + taskWhere + ")", args: taskArgs},
	)
}

func (q *backfillQuery) skippedWithAITagWhere() (string, []any) {
	aiWhere, aiArgs := backfillAcceptedAITagExists()
	return q.combine(backfillCondition{sql: "AND EXISTS (" + aiWhere + ")", args: aiArgs})
}

func (q *backfillQuery) skippedWithActiveTaskWhere() (string, []any) {
	taskWhere, taskArgs := backfillActiveAITagTaskExists()
	return q.combine(backfillCondition{sql: "AND EXISTS (" + taskWhere + ")", args: taskArgs})
}

func (q *backfillQuery) combine(conditions ...backfillCondition) (string, []any) {
	where := q.baseWhere
	args := append([]any(nil), q.baseArgs...)
	for _, condition := range conditions {
		where += "\n\t\t" + condition.sql
		args = append(args, condition.args...)
	}
	return where, args
}

func backfillAcceptedAITagExists() (string, []any) {
	return `
			SELECT 1 FROM image_tags it3
			WHERE it3.image_id = i.id
			  AND it3.source = ?
			  AND it3.review_state != ?
		`, []any{domain.ImageTagSourceAI, domain.ReviewStateRejected}
}

func backfillActiveAITagTaskExists() (string, []any) {
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(activeAIBackfillTaskStatuses)), ",")
	args := make([]any, 0, 1+len(activeAIBackfillTaskStatuses))
	args = append(args, domain.PlatformTaskTypeAITagGeneration)
	for _, status := range activeAIBackfillTaskStatuses {
		args = append(args, status)
	}

	return fmt.Sprintf(`
			SELECT 1 FROM platform_tasks pt
			WHERE pt.image_id = i.id
			  AND pt.task_type = ?
			  AND pt.status IN (%s)
		`, placeholders), args
}
