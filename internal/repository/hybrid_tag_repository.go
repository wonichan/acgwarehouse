package repository

import (
	"context"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

// HybridTagRepository delegates reads to D1 and writes to SQLite.
type HybridTagRepository struct {
	d1Reader TagRepository
	sqlite   TagRepository
}

func NewHybridTagRepository(d1Reader, sqlite TagRepository) *HybridTagRepository {
	return &HybridTagRepository{d1Reader: d1Reader, sqlite: sqlite}
}

func (h *HybridTagRepository) Save(ctx context.Context, tag *domain.Tag) error {
	return h.sqlite.Save(ctx, tag)
}

func (h *HybridTagRepository) Update(ctx context.Context, tag *domain.Tag) error {
	return h.sqlite.Update(ctx, tag)
}

func (h *HybridTagRepository) FindByID(ctx context.Context, id int64) (*domain.Tag, error) {
	return h.d1Reader.FindByID(ctx, id)
}

func (h *HybridTagRepository) FindByLabel(ctx context.Context, label string) (*domain.Tag, error) {
	return h.d1Reader.FindByLabel(ctx, label)
}

func (h *HybridTagRepository) FindByLabelLike(ctx context.Context, query string, limit int) ([]*domain.Tag, error) {
	return h.d1Reader.FindByLabelLike(ctx, query, limit)
}

func (h *HybridTagRepository) FindAll(ctx context.Context, limit, offset int) ([]*domain.Tag, error) {
	return h.d1Reader.FindAll(ctx, limit, offset)
}

func (h *HybridTagRepository) FindRoots(ctx context.Context) ([]*domain.Tag, error) {
	return h.d1Reader.FindRoots(ctx)
}

func (h *HybridTagRepository) FindChildrenByParent(ctx context.Context, parentID int64) ([]*domain.Tag, error) {
	return h.d1Reader.FindChildrenByParent(ctx, parentID)
}

func (h *HybridTagRepository) FindValidParentCandidates(ctx context.Context, targetLevel string) ([]*domain.Tag, error) {
	return h.d1Reader.FindValidParentCandidates(ctx, targetLevel)
}

func (h *HybridTagRepository) ResolveDescendantIDs(ctx context.Context, tagIDs []int64) (map[int64][]int64, error) {
	return h.d1Reader.ResolveDescendantIDs(ctx, tagIDs)
}

func (h *HybridTagRepository) ResolveAllDescendantIDs(ctx context.Context, tagIDs []int64) ([]int64, error) {
	return h.d1Reader.ResolveAllDescendantIDs(ctx, tagIDs)
}

func (h *HybridTagRepository) UpdateReviewState(ctx context.Context, id int64, state string) error {
	return h.sqlite.UpdateReviewState(ctx, id, state)
}

func (h *HybridTagRepository) IncrementUsageCount(ctx context.Context, id int64) error {
	return h.sqlite.IncrementUsageCount(ctx, id)
}

func (h *HybridTagRepository) DecrementUsageCount(ctx context.Context, id int64) error {
	return h.sqlite.DecrementUsageCount(ctx, id)
}

func (h *HybridTagRepository) Delete(ctx context.Context, id int64) error {
	return h.sqlite.Delete(ctx, id)
}

func (h *HybridTagRepository) Count(ctx context.Context) (int, error) {
	return h.d1Reader.Count(ctx)
}

func (h *HybridTagRepository) FindTreeRoots(ctx context.Context) ([]*TagBrowseNode, error) {
	return h.d1Reader.FindTreeRoots(ctx)
}

func (h *HybridTagRepository) FindTreeChildren(ctx context.Context, parentID int64) ([]*TagBrowseNode, error) {
	return h.d1Reader.FindTreeChildren(ctx, parentID)
}

func (h *HybridTagRepository) ListOrphanTags(ctx context.Context, search string, limit, offset int) ([]*TagBrowseNode, error) {
	return h.d1Reader.ListOrphanTags(ctx, search, limit, offset)
}

func (h *HybridTagRepository) CountOrphanTags(ctx context.Context, search string) (int, error) {
	return h.d1Reader.CountOrphanTags(ctx, search)
}