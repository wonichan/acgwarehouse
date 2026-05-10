package repository

import (
	"context"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

// HybridTagAliasRepository delegates reads to D1 and writes to SQLite.
type HybridTagAliasRepository struct {
	d1Reader TagAliasRepository
	sqlite   TagAliasRepository
}

func NewHybridTagAliasRepository(d1Reader, sqlite TagAliasRepository) *HybridTagAliasRepository {
	return &HybridTagAliasRepository{d1Reader: d1Reader, sqlite: sqlite}
}

func (h *HybridTagAliasRepository) Save(ctx context.Context, alias *domain.TagAlias) error {
	return h.sqlite.Save(ctx, alias)
}

func (h *HybridTagAliasRepository) FindByID(ctx context.Context, id int64) (*domain.TagAlias, error) {
	return h.d1Reader.FindByID(ctx, id)
}

func (h *HybridTagAliasRepository) FindByTagID(ctx context.Context, tagID int64) ([]*domain.TagAlias, error) {
	return h.d1Reader.FindByTagID(ctx, tagID)
}

func (h *HybridTagAliasRepository) FindByNormalizedLabel(ctx context.Context, normalized string) (*domain.TagAlias, error) {
	return h.d1Reader.FindByNormalizedLabel(ctx, normalized)
}

func (h *HybridTagAliasRepository) FindByLabelLike(ctx context.Context, query string) ([]*domain.TagAlias, error) {
	return h.d1Reader.FindByLabelLike(ctx, query)
}

func (h *HybridTagAliasRepository) Delete(ctx context.Context, id int64) error {
	return h.sqlite.Delete(ctx, id)
}