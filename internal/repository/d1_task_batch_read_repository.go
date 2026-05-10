package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/d1client"
)

type d1TaskBatchReadRepository struct {
	client *d1client.Client
}

func NewD1TaskBatchReadRepository(client *d1client.Client) TaskBatchReadRepository {
	return &d1TaskBatchReadRepository{client: client}
}

func (r *d1TaskBatchReadRepository) ListBatches(ctx context.Context, filter TaskBatchReadFilter) ([]TaskBatchReadRecord, error) {
	clauses := make([]string, 0, 3)
	args := make([]any, 0, 6)
	if filter.BatchID != nil {
		clauses = append(clauses, "tb.id = ?")
		args = append(args, *filter.BatchID)
	}
	if filter.SourceType != "" {
		clauses = append(clauses, "tb.source_type = ?")
		args = append(args, filter.SourceType)
	}
	if filter.Status != "" {
		clauses = append(clauses, "tb.status = ?")
		args = append(args, filter.Status)
	}

	query := `
		SELECT tb.id, tb.source_type, tb.summary_label, tb.status,
			tb.total_images, tb.new_images, tb.skipped_images, tb.skipped_unchanged,
			tb.skipped_duplicate_tasks, COALESCE(tb.latest_error_summary, '') AS latest_error_summary,
			tb.created_at, tb.finished_at,
			COALESCE((
				SELECT GROUP_CONCAT(source_label, ', ')
				FROM (
					SELECT CASE
						WHEN TRIM(source_label) <> '' THEN source_label
						ELSE source_root
					END AS source_label
					FROM task_batch_sources
					WHERE batch_id = tb.id
					ORDER BY id
				)
			), '') AS source_summary
		FROM task_batches tb`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	query += " ORDER BY tb.created_at DESC, tb.id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, max(filter.Offset, 0))

	rows, err := r.client.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	records := make([]TaskBatchReadRecord, 0, len(rows))
	for _, row := range rows {
		record, err := mapTaskBatchReadRecordFromD1(row)
		if err != nil {
			return nil, err
		}
		record.StatusCounts, err = r.loadBatchCounts(ctx, record.ID, "status")
		if err != nil {
			return nil, err
		}
		record.TaskTypeCounts, err = r.loadBatchCounts(ctx, record.ID, "task_type")
		if err != nil {
			return nil, err
		}
		records = append(records, *record)
	}
	return records, nil
}

func (r *d1TaskBatchReadRepository) ListTasks(ctx context.Context, filter TaskReadFilter) ([]TaskReadRecord, error) {
	clauses := make([]string, 0, 3)
	args := make([]any, 0, 6)
	if filter.BatchID != nil {
		clauses = append(clauses, "pt.batch_id = ?")
		args = append(args, *filter.BatchID)
	}
	if filter.TaskType != "" {
		clauses = append(clauses, "pt.task_type = ?")
		args = append(args, filter.TaskType)
	}
	if filter.Status != "" {
		clauses = append(clauses, "pt.status = ?")
		args = append(args, filter.Status)
	}

	query := `
		SELECT pt.id, pt.batch_id, pt.image_id, i.path, i.filename,
			pt.task_type, pt.status, COALESCE(pt.skip_reason, '') AS skip_reason,
			COALESCE(pt.error_summary, '') AS error_summary,
			pt.latest_async_job_id, pt.created_at, pt.finished_at
		FROM platform_tasks pt
		JOIN images i ON i.id = pt.image_id`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	query += " ORDER BY pt.created_at DESC, pt.id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, max(filter.Offset, 0))

	rows, err := r.client.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	records := make([]TaskReadRecord, 0, len(rows))
	for _, row := range rows {
		record, err := mapTaskReadRecordFromD1(row)
		if err != nil {
			return nil, err
		}
		records = append(records, *record)
	}
	return records, nil
}

func (r *d1TaskBatchReadRepository) LoadFailureGroups(ctx context.Context, batchID int64) ([]FailureGroupRecord, error) {
	rows, err := r.client.Query(ctx, `
		SELECT
			CASE
				WHEN error_summary IS NULL OR error_summary = '' THEN 'unknown'
				WHEN INSTR(error_summary, ': ') > 0 THEN SUBSTR(error_summary, 1, INSTR(error_summary, ': ') - 1)
				ELSE 'unknown'
			END AS reason_key,
			COUNT(*) AS cnt,
			MIN(error_summary) AS sample_error
		FROM platform_tasks
		WHERE batch_id = ? AND status = 'failed'
		GROUP BY reason_key
		ORDER BY cnt DESC`, batchID)
	if err != nil {
		return nil, err
	}
	groups := make([]FailureGroupRecord, 0, len(rows))
	for _, row := range rows {
		count, err := toInt64(row["cnt"])
		if err != nil {
			return nil, err
		}
		groups = append(groups, FailureGroupRecord{
			ReasonKey:   toStringDefault(row["reason_key"], "unknown"),
			Count:       count,
			SampleError: toStringDefault(row["sample_error"], ""),
		})
	}
	return groups, nil
}

func (r *d1TaskBatchReadRepository) loadBatchCounts(ctx context.Context, batchID int64, column string) (map[string]int64, error) {
	if column != "status" && column != "task_type" {
		return nil, fmt.Errorf("unsupported task batch count column %q", column)
	}
	rows, err := r.client.Query(ctx, "SELECT "+column+" AS key, COUNT(*) AS count FROM platform_tasks WHERE batch_id = ? GROUP BY "+column, batchID)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int64, len(rows))
	for _, row := range rows {
		count, err := toInt64(row["count"])
		if err != nil {
			return nil, err
		}
		counts[toStringDefault(row["key"], "")] = count
	}
	return counts, nil
}

func mapTaskBatchReadRecordFromD1(row map[string]any) (*TaskBatchReadRecord, error) {
	id, err := toInt64(row["id"])
	if err != nil {
		return nil, err
	}
	createdAt, _ := toTime(row["created_at"])
	var finishedAt *time.Time
	if t, err := toTime(row["finished_at"]); err == nil && !t.IsZero() {
		finishedAt = &t
	}
	totalImages, _ := toInt64(row["total_images"])
	newImages, _ := toInt64(row["new_images"])
	skippedImages, _ := toInt64(row["skipped_images"])
	skippedUnchanged, _ := toInt64(row["skipped_unchanged"])
	skippedDuplicateTasks, _ := toInt64(row["skipped_duplicate_tasks"])
	return &TaskBatchReadRecord{
		ID:                    id,
		SourceType:            toStringDefault(row["source_type"], ""),
		SummaryLabel:          toStringDefault(row["summary_label"], ""),
		Status:                toStringDefault(row["status"], ""),
		TotalImages:           totalImages,
		NewImages:             newImages,
		SkippedImages:         skippedImages,
		SkippedUnchanged:      skippedUnchanged,
		SkippedDuplicateTasks: skippedDuplicateTasks,
		LatestErrorSummary:    toStringDefault(row["latest_error_summary"], ""),
		CreatedAt:             createdAt,
		FinishedAt:            finishedAt,
		SourceSummary:         toStringDefault(row["source_summary"], ""),
	}, nil
}

func mapTaskReadRecordFromD1(row map[string]any) (*TaskReadRecord, error) {
	id, err := toInt64(row["id"])
	if err != nil {
		return nil, err
	}
	batchID, err := toInt64(row["batch_id"])
	if err != nil {
		return nil, err
	}
	imageID, err := toInt64(row["image_id"])
	if err != nil {
		return nil, err
	}
	createdAt, _ := toTime(row["created_at"])
	var latestAsyncJobID *int64
	if id, err := toInt64(row["latest_async_job_id"]); err == nil && id != 0 {
		latestAsyncJobID = &id
	}
	var finishedAt *time.Time
	if t, err := toTime(row["finished_at"]); err == nil && !t.IsZero() {
		finishedAt = &t
	}
	return &TaskReadRecord{
		ID:               id,
		BatchID:          batchID,
		ImageID:          imageID,
		ImagePath:        toStringDefault(row["path"], ""),
		ImageFilename:    toStringDefault(row["filename"], ""),
		TaskType:         toStringDefault(row["task_type"], ""),
		Status:           toStringDefault(row["status"], ""),
		SkipReason:       toStringDefault(row["skip_reason"], ""),
		ErrorSummary:     toStringDefault(row["error_summary"], ""),
		LatestAsyncJobID: latestAsyncJobID,
		CreatedAt:        createdAt,
		FinishedAt:       finishedAt,
	}, nil
}
