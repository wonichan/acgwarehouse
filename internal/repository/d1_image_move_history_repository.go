package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/wonichan/acgwarehouse-backend/internal/d1client"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type d1ImageMoveHistoryRepository struct {
	client *d1client.Client
}

func NewD1ImageMoveHistoryRepository(client *d1client.Client) ImageMoveHistoryRepository {
	return &d1ImageMoveHistoryRepository{client: client}
}

func (r *d1ImageMoveHistoryRepository) CreateImageMoveBatch(ctx context.Context, batch *domain.ImageMoveBatch) error {
	sourceDirsJSON, err := json.Marshal(batch.SourceDirs)
	if err != nil {
		return err
	}
	id, err := r.client.ExecReturningID(ctx, `
		INSERT INTO image_move_batches
		(tag_id, source_dirs_json, target_dir, conflict_strategy, total_matched, moved, skipped, failed, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, batch.TagID, string(sourceDirsJSON), batch.TargetDir, batch.ConflictStrategy, batch.TotalMatched, batch.Moved, batch.Skipped, batch.Failed, batch.Status)
	if err != nil {
		return err
	}
	batch.ID = id
	return nil
}

func (r *d1ImageMoveHistoryRepository) UpdateImageMoveBatch(ctx context.Context, batch *domain.ImageMoveBatch) error {
	return r.client.Exec(ctx, `
		UPDATE image_move_batches
		SET total_matched = ?, moved = ?, skipped = ?, failed = ?, status = ?,
		    finished_at = CASE WHEN ? IN ('completed', 'failed', 'cancelled') THEN COALESCE(finished_at, CURRENT_TIMESTAMP) ELSE finished_at END
		WHERE id = ?
	`, batch.TotalMatched, batch.Moved, batch.Skipped, batch.Failed, batch.Status, batch.Status, batch.ID)
}

func (r *d1ImageMoveHistoryRepository) AddImageMoveItem(ctx context.Context, batchID int64, item domain.ImageMoveItem) error {
	return r.client.Exec(ctx, `
		INSERT INTO image_move_items
		(batch_id, image_id, source_path, target_path, status, reason)
		VALUES (?, ?, ?, ?, ?, ?)
	`, batchID, item.ImageID, item.SourcePath, item.TargetPath, item.Status, item.Reason)
}

func (r *d1ImageMoveHistoryRepository) FindImageMoveBatch(ctx context.Context, id int64) (*domain.ImageMoveBatch, error) {
	row, err := r.client.QueryOne(ctx, `
		SELECT id, tag_id, source_dirs_json, target_dir, conflict_strategy,
		       total_matched, moved, skipped, failed, status, created_at, finished_at
		FROM image_move_batches
		WHERE id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, fmt.Errorf("image move batch not found: %d", id)
	}
	batch, err := mapImageMoveBatchFromD1(row)
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

func (r *d1ImageMoveHistoryRepository) ListImageMoveBatches(ctx context.Context, limit int) ([]domain.ImageMoveBatch, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.client.Query(ctx, `
		SELECT id, tag_id, source_dirs_json, target_dir, conflict_strategy,
		       total_matched, moved, skipped, failed, status, created_at, finished_at
		FROM image_move_batches
		ORDER BY id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	batches := make([]domain.ImageMoveBatch, 0, len(rows))
	for _, row := range rows {
		batch, err := mapImageMoveBatchFromD1(row)
		if err != nil {
			return nil, err
		}
		batch.Progress = imageMoveProgressFromBatch(batch)
		batches = append(batches, *batch)
	}
	return batches, nil
}

func (r *d1ImageMoveHistoryRepository) listBatchItems(ctx context.Context, batchID int64) ([]domain.ImageMoveItem, error) {
	rows, err := r.client.Query(ctx, `
		SELECT image_id, source_path, target_path, status, reason
		FROM image_move_items
		WHERE batch_id = ?
		ORDER BY id ASC
	`, batchID)
	if err != nil {
		return nil, err
	}
	items := make([]domain.ImageMoveItem, 0, len(rows))
	for _, row := range rows {
		imageID, err := toInt64(row["image_id"])
		if err != nil {
			return nil, err
		}
		item := domain.ImageMoveItem{
			ImageID:    imageID,
			SourcePath: toStringDefault(row["source_path"], ""),
			TargetPath: toStringDefault(row["target_path"], ""),
			Status:     toStringDefault(row["status"], ""),
			Reason:     toStringDefault(row["reason"], ""),
		}
		item.Filename = filepath.Base(item.SourcePath)
		item.Retryable = domain.ImageMoveReasonIsRetryable(item.Reason)
		items = append(items, item)
	}
	return items, nil
}

func mapImageMoveBatchFromD1(row map[string]any) (*domain.ImageMoveBatch, error) {
	id, err := toInt64(row["id"])
	if err != nil {
		return nil, err
	}
	tagID, err := toInt64(row["tag_id"])
	if err != nil {
		return nil, err
	}
	sourceDirsJSON := toStringDefault(row["source_dirs_json"], "[]")
	var sourceDirs []string
	if err := json.Unmarshal([]byte(sourceDirsJSON), &sourceDirs); err != nil {
		return nil, fmt.Errorf("decode source_dirs_json: %w", err)
	}
	totalMatched, _ := toInt64(row["total_matched"])
	moved, _ := toInt64(row["moved"])
	skipped, _ := toInt64(row["skipped"])
	failed, _ := toInt64(row["failed"])
	var finishedAt *string
	if value := toStringDefault(row["finished_at"], ""); value != "" {
		finishedAt = &value
	}
	return &domain.ImageMoveBatch{
		ID:               id,
		TagID:            tagID,
		SourceDirs:       sourceDirs,
		TargetDir:        toStringDefault(row["target_dir"], ""),
		ConflictStrategy: toStringDefault(row["conflict_strategy"], ""),
		TotalMatched:     totalMatched,
		Moved:            moved,
		Skipped:          skipped,
		Failed:           failed,
		Status:           toStringDefault(row["status"], ""),
		CreatedAt:        toStringDefault(row["created_at"], ""),
		FinishedAt:       finishedAt,
	}, nil
}
