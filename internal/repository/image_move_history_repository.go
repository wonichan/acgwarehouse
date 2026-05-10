package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type ImageMoveHistoryRepository interface {
	CreateImageMoveBatch(ctx context.Context, batch *domain.ImageMoveBatch) error
	UpdateImageMoveBatch(ctx context.Context, batch *domain.ImageMoveBatch) error
	AddImageMoveItem(ctx context.Context, batchID int64, item domain.ImageMoveItem) error
	FindImageMoveBatch(ctx context.Context, id int64) (*domain.ImageMoveBatch, error)
	ListImageMoveBatches(ctx context.Context, limit int) ([]domain.ImageMoveBatch, error)
}

type sqliteImageMoveHistoryRepository struct {
	db *sql.DB
}

func NewImageMoveHistoryRepository(db *sql.DB) ImageMoveHistoryRepository {
	return &sqliteImageMoveHistoryRepository{db: db}
}

func (r *sqliteImageMoveHistoryRepository) CreateImageMoveBatch(ctx context.Context, batch *domain.ImageMoveBatch) error {
	sourceDirsJSON, err := json.Marshal(batch.SourceDirs)
	if err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO image_move_batches
		(tag_id, source_dirs_json, target_dir, conflict_strategy, total_matched, moved, skipped, failed, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, batch.TagID, string(sourceDirsJSON), batch.TargetDir, batch.ConflictStrategy, batch.TotalMatched, batch.Moved, batch.Skipped, batch.Failed, batch.Status)
	if err != nil {
		return err
	}
	batch.ID, err = result.LastInsertId()
	return err
}

func (r *sqliteImageMoveHistoryRepository) UpdateImageMoveBatch(ctx context.Context, batch *domain.ImageMoveBatch) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE image_move_batches
		SET total_matched = ?, moved = ?, skipped = ?, failed = ?, status = ?,
		    finished_at = CASE WHEN ? IN ('completed', 'failed', 'cancelled') THEN COALESCE(finished_at, CURRENT_TIMESTAMP) ELSE finished_at END
		WHERE id = ?
	`, batch.TotalMatched, batch.Moved, batch.Skipped, batch.Failed, batch.Status, batch.Status, batch.ID)
	return err
}

func (r *sqliteImageMoveHistoryRepository) AddImageMoveItem(ctx context.Context, batchID int64, item domain.ImageMoveItem) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO image_move_items
		(batch_id, image_id, source_path, target_path, status, reason)
		VALUES (?, ?, ?, ?, ?, ?)
	`, batchID, item.ImageID, item.SourcePath, item.TargetPath, item.Status, item.Reason)
	return err
}

func (r *sqliteImageMoveHistoryRepository) FindImageMoveBatch(ctx context.Context, id int64) (*domain.ImageMoveBatch, error) {
	batch, err := r.scanBatch(r.db.QueryRowContext(ctx, `
		SELECT id, tag_id, source_dirs_json, target_dir, conflict_strategy,
		       total_matched, moved, skipped, failed, status, created_at, finished_at
		FROM image_move_batches
		WHERE id = ?
	`, id))
	if err != nil {
		return nil, err
	}
	items, err := r.listBatchItems(ctx, id)
	if err != nil {
		return nil, err
	}
	batch.Items = items
	batch.Progress = imageMoveProgressFromBatch(batch)
	return batch, nil
}

func (r *sqliteImageMoveHistoryRepository) ListImageMoveBatches(ctx context.Context, limit int) ([]domain.ImageMoveBatch, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tag_id, source_dirs_json, target_dir, conflict_strategy,
		       total_matched, moved, skipped, failed, status, created_at, finished_at
		FROM image_move_batches
		ORDER BY id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	batches := make([]domain.ImageMoveBatch, 0, limit)
	for rows.Next() {
		batch, err := scanImageMoveBatch(rows)
		if err != nil {
			return nil, err
		}
		batch.Progress = imageMoveProgressFromBatch(batch)
		batches = append(batches, *batch)
	}
	return batches, rows.Err()
}

func (r *sqliteImageMoveHistoryRepository) listBatchItems(ctx context.Context, batchID int64) ([]domain.ImageMoveItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT image_id, source_path, target_path, status, reason
		FROM image_move_items
		WHERE batch_id = ?
		ORDER BY id ASC
	`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.ImageMoveItem, 0)
	for rows.Next() {
		var item domain.ImageMoveItem
		if err := rows.Scan(&item.ImageID, &item.SourcePath, &item.TargetPath, &item.Status, &item.Reason); err != nil {
			return nil, err
		}
		item.Filename = filenameFromPath(item.SourcePath)
		item.Retryable = domain.ImageMoveReasonIsRetryable(item.Reason)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *sqliteImageMoveHistoryRepository) scanBatch(row interface{ Scan(dest ...any) error }) (*domain.ImageMoveBatch, error) {
	return scanImageMoveBatch(row)
}

func scanImageMoveBatch(row interface{ Scan(dest ...any) error }) (*domain.ImageMoveBatch, error) {
	var (
		batch          domain.ImageMoveBatch
		sourceDirsJSON string
		finishedAt     sql.NullString
	)
	if err := row.Scan(
		&batch.ID,
		&batch.TagID,
		&sourceDirsJSON,
		&batch.TargetDir,
		&batch.ConflictStrategy,
		&batch.TotalMatched,
		&batch.Moved,
		&batch.Skipped,
		&batch.Failed,
		&batch.Status,
		&batch.CreatedAt,
		&finishedAt,
	); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(sourceDirsJSON), &batch.SourceDirs); err != nil {
		return nil, fmt.Errorf("decode source_dirs_json: %w", err)
	}
	if finishedAt.Valid {
		batch.FinishedAt = &finishedAt.String
	}
	return &batch, nil
}

func imageMoveProgressFromBatch(batch *domain.ImageMoveBatch) domain.ImageMoveProgress {
	processed := batch.Moved + batch.Skipped + batch.Failed
	return domain.ImageMoveProgress{
		Total:     batch.TotalMatched,
		Processed: processed,
		Moved:     batch.Moved,
		Skipped:   batch.Skipped,
		Failed:    batch.Failed,
	}
}

func filenameFromPath(path string) string {
	return filepath.Base(path)
}
