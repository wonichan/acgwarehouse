package repository

import (
	"context"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

// HybridCollectionRepository delegates reads to D1 and writes to SQLite.
type HybridCollectionRepository struct {
	d1Reader CollectionRepository
	sqlite   CollectionRepository
}

func NewHybridCollectionRepository(d1Reader, sqlite CollectionRepository) *HybridCollectionRepository {
	return &HybridCollectionRepository{d1Reader: d1Reader, sqlite: sqlite}
}

func (h *HybridCollectionRepository) Save(ctx context.Context, collection *domain.Collection) error {
	return h.sqlite.Save(ctx, collection)
}

func (h *HybridCollectionRepository) FindByID(ctx context.Context, id int64) (*domain.Collection, error) {
	return h.d1Reader.FindByID(ctx, id)
}

func (h *HybridCollectionRepository) FindAll(ctx context.Context, limit, offset int) ([]domain.Collection, error) {
	return h.d1Reader.FindAll(ctx, limit, offset)
}

func (h *HybridCollectionRepository) FindByName(ctx context.Context, name string) (*domain.Collection, error) {
	return h.d1Reader.FindByName(ctx, name)
}

func (h *HybridCollectionRepository) Update(ctx context.Context, collection *domain.Collection) error {
	return h.sqlite.Update(ctx, collection)
}

func (h *HybridCollectionRepository) Delete(ctx context.Context, id int64) error {
	return h.sqlite.Delete(ctx, id)
}

func (h *HybridCollectionRepository) AddImage(ctx context.Context, collectionID, imageID int64) error {
	return h.sqlite.AddImage(ctx, collectionID, imageID)
}

func (h *HybridCollectionRepository) RemoveImage(ctx context.Context, collectionID, imageID int64) error {
	return h.sqlite.RemoveImage(ctx, collectionID, imageID)
}

func (h *HybridCollectionRepository) FindImagesByCollection(ctx context.Context, collectionID int64, limit, offset int) ([]domain.Image, error) {
	return h.d1Reader.FindImagesByCollection(ctx, collectionID, limit, offset)
}

func (h *HybridCollectionRepository) CountImages(ctx context.Context, collectionID int64) (int64, error) {
	return h.d1Reader.CountImages(ctx, collectionID)
}

func (h *HybridCollectionRepository) UpdateCover(ctx context.Context, collectionID, imageID int64) error {
	return h.sqlite.UpdateCover(ctx, collectionID, imageID)
}

func (h *HybridCollectionRepository) GetLatestImageID(ctx context.Context, collectionID int64) (*int64, error) {
	return h.d1Reader.GetLatestImageID(ctx, collectionID)
}

func (h *HybridCollectionRepository) Count(ctx context.Context) (int64, error) {
	return h.d1Reader.Count(ctx)
}

func (h *HybridCollectionRepository) FindCollectionIDsByImage(ctx context.Context, imageID int64) ([]int64, error) {
	return h.d1Reader.FindCollectionIDsByImage(ctx, imageID)
}

func (h *HybridCollectionRepository) ReconcileAfterImageDelete(ctx context.Context, collectionID int64) error {
	return h.sqlite.ReconcileAfterImageDelete(ctx, collectionID)
}