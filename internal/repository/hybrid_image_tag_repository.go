package repository

import (
	"context"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

// HybridImageTagRepository delegates reads to D1 and writes to SQLite.
type HybridImageTagRepository struct {
	d1Reader ImageTagRepository
	sqlite   ImageTagRepository
}

func NewHybridImageTagRepository(d1Reader, sqlite ImageTagRepository) *HybridImageTagRepository {
	return &HybridImageTagRepository{d1Reader: d1Reader, sqlite: sqlite}
}

func (h *HybridImageTagRepository) Save(ctx context.Context, imageTag *domain.ImageTag) error {
	return h.sqlite.Save(ctx, imageTag)
}

func (h *HybridImageTagRepository) FindByImageID(ctx context.Context, imageID int64) ([]*domain.ImageTag, error) {
	return h.d1Reader.FindByImageID(ctx, imageID)
}

func (h *HybridImageTagRepository) FindByTagID(ctx context.Context, tagID int64, limit, offset int) ([]*domain.ImageTag, error) {
	return h.d1Reader.FindByTagID(ctx, tagID, limit, offset)
}

func (h *HybridImageTagRepository) HasAITags(ctx context.Context, imageID int64) (bool, error) {
	return h.d1Reader.HasAITags(ctx, imageID)
}

func (h *HybridImageTagRepository) UpdateReviewState(ctx context.Context, imageID, tagID int64, state string) error {
	return h.sqlite.UpdateReviewState(ctx, imageID, tagID, state)
}

func (h *HybridImageTagRepository) Delete(ctx context.Context, imageID, tagID int64) (int64, error) {
	return h.sqlite.Delete(ctx, imageID, tagID)
}

func (h *HybridImageTagRepository) BatchUpdateReviewState(ctx context.Context, imageID int64, tagIDs []int64, state string) error {
	return h.sqlite.BatchUpdateReviewState(ctx, imageID, tagIDs, state)
}

func (h *HybridImageTagRepository) MergeImageTag(ctx context.Context, imageID, sourceTagID, targetTagID int64) error {
	return h.sqlite.MergeImageTag(ctx, imageID, sourceTagID, targetTagID)
}

func (h *HybridImageTagRepository) GetTagStats(ctx context.Context, tagID int64) (*TagStats, error) {
	return h.d1Reader.GetTagStats(ctx, tagID)
}

func (h *HybridImageTagRepository) SyncFTSForTag(ctx context.Context, tagID int64) error {
	return h.sqlite.SyncFTSForTag(ctx, tagID)
}

func (h *HybridImageTagRepository) Exists(ctx context.Context, imageID, tagID int64) (bool, error) {
	return h.d1Reader.Exists(ctx, imageID, tagID)
}