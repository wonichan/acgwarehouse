package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

type TaskBatchReadFilter struct {
	BatchID    *int64
	SourceType string
	Status     string
	Limit      int
	Offset     int
}

type TaskReadFilter struct {
	BatchID  *int64
	TaskType string
	Status   string
	Limit    int
	Offset   int
}

type TaskBatchReadRecord struct {
	ID                    int64
	SourceType            string
	SummaryLabel          string
	Status                string
	TotalImages           int64
	NewImages             int64
	SkippedImages         int64
	SkippedUnchanged      int64
	SkippedDuplicateTasks int64
	LatestErrorSummary    string
	CreatedAt             time.Time
	FinishedAt            *time.Time
	SourceSummary         string
	StatusCounts          map[string]int64
	TaskTypeCounts        map[string]int64
}

type TaskReadRecord struct {
	ID               int64
	BatchID          int64
	ImageID          int64
	ImagePath        string
	ImageFilename    string
	TaskType         string
	Status           string
	SkipReason       string
	ErrorSummary     string
	LatestAsyncJobID *int64
	CreatedAt        time.Time
	FinishedAt       *time.Time
}

// FailureGroupRecord holds aggregated failure data grouped by reason.
type FailureGroupRecord struct {
	ReasonKey   string
	Count       int64
	SampleError string
}

type TaskBatchReadRepository interface {
	ListBatches(ctx context.Context, filter TaskBatchReadFilter) ([]TaskBatchReadRecord, error)
	ListTasks(ctx context.Context, filter TaskReadFilter) ([]TaskReadRecord, error)
	LoadFailureGroups(ctx context.Context, batchID int64) ([]FailureGroupRecord, error)
}

type sqliteTaskBatchReadRepository struct {
	db *sql.DB
}

func NewTaskBatchReadRepository(db *sql.DB) TaskBatchReadRepository {
	return &sqliteTaskBatchReadRepository{db: db}
}

func (r *sqliteTaskBatchReadRepository) ListBatches(ctx context.Context, filter TaskBatchReadFilter) ([]TaskBatchReadRecord, error) {
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
			tb.skipped_duplicate_tasks, COALESCE(tb.latest_error_summary, ''), tb.created_at, tb.finished_at,
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

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]TaskBatchReadRecord, 0)
	for rows.Next() {
		var record TaskBatchReadRecord
		var latestError string
		var finishedAt sql.NullTime
		if err := rows.Scan(
			&record.ID,
			&record.SourceType,
			&record.SummaryLabel,
			&record.Status,
			&record.TotalImages,
			&record.NewImages,
			&record.SkippedImages,
			&record.SkippedUnchanged,
			&record.SkippedDuplicateTasks,
			&latestError,
			&record.CreatedAt,
			&finishedAt,
			&record.SourceSummary,
		); err != nil {
			return nil, err
		}
		if finishedAt.Valid {
			record.FinishedAt = &finishedAt.Time
		}
		record.LatestErrorSummary = latestError
		record.StatusCounts, err = r.loadBatchCounts(ctx, record.ID, "status")
		if err != nil {
			return nil, err
		}
		record.TaskTypeCounts, err = r.loadBatchCounts(ctx, record.ID, "task_type")
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (r *sqliteTaskBatchReadRepository) ListTasks(ctx context.Context, filter TaskReadFilter) ([]TaskReadRecord, error) {
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
			pt.task_type, pt.status, COALESCE(pt.skip_reason, ''), COALESCE(pt.error_summary, ''),
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

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]TaskReadRecord, 0)
	for rows.Next() {
		var record TaskReadRecord
		var latestAsyncJobID sql.NullInt64
		var finishedAt sql.NullTime
		if err := rows.Scan(
			&record.ID,
			&record.BatchID,
			&record.ImageID,
			&record.ImagePath,
			&record.ImageFilename,
			&record.TaskType,
			&record.Status,
			&record.SkipReason,
			&record.ErrorSummary,
			&latestAsyncJobID,
			&record.CreatedAt,
			&finishedAt,
		); err != nil {
			return nil, err
		}
		if latestAsyncJobID.Valid {
			record.LatestAsyncJobID = &latestAsyncJobID.Int64
		}
		if finishedAt.Valid {
			record.FinishedAt = &finishedAt.Time
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (r *sqliteTaskBatchReadRepository) loadBatchCounts(ctx context.Context, batchID int64, column string) (map[string]int64, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT "+column+", COUNT(*) FROM platform_tasks WHERE batch_id = ? GROUP BY "+column, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int64)
	for rows.Next() {
		var key string
		var count int64
		if err := rows.Scan(&key, &count); err != nil {
			return nil, err
		}
		counts[key] = count
	}
	return counts, rows.Err()
}

// LoadFailureGroups aggregates failed task error_summaries by reason key for a batch.
// The reason key is extracted as the prefix before the first ": " in error_summary.
// Tasks with empty error_summary are grouped under "unknown".
func (r *sqliteTaskBatchReadRepository) LoadFailureGroups(ctx context.Context, batchID int64) ([]FailureGroupRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
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
	defer rows.Close()

	groups := make([]FailureGroupRecord, 0)
	for rows.Next() {
		var g FailureGroupRecord
		var sampleError sql.NullString
		if err := rows.Scan(&g.ReasonKey, &g.Count, &sampleError); err != nil {
			return nil, err
		}
		if sampleError.Valid {
			g.SampleError = sampleError.String
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}
