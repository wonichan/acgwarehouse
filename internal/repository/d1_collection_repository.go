package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/d1client"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type d1CollectionRepository struct {
	client *d1client.Client
}

func NewD1CollectionRepository(client *d1client.Client) CollectionRepository {
	return &d1CollectionRepository{client: client}
}

func (r *d1CollectionRepository) FindByID(ctx context.Context, id int64) (*domain.Collection, error) {
	row, err := r.client.QueryOne(ctx, `
		SELECT id, name, description, cover_image_id, image_count, created_at, updated_at
		FROM collections WHERE id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, sql.ErrNoRows
	}
	return mapCollectionFromD1(row)
}

func (r *d1CollectionRepository) FindAll(ctx context.Context, limit, offset int) ([]domain.Collection, error) {
	rows, err := r.client.Query(ctx, `
		SELECT id, name, description, cover_image_id, image_count, created_at, updated_at
		FROM collections ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, int64(limit), int64(offset))
	if err != nil {
		return nil, err
	}
	return mapCollectionsFromD1(rows)
}

func (r *d1CollectionRepository) FindByName(ctx context.Context, name string) (*domain.Collection, error) {
	row, err := r.client.QueryOne(ctx, `
		SELECT id, name, description, cover_image_id, image_count, created_at, updated_at
		FROM collections WHERE name = ?
	`, name)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, sql.ErrNoRows
	}
	return mapCollectionFromD1(row)
}

func (r *d1CollectionRepository) Count(ctx context.Context) (int64, error) {
	return r.client.QueryCount(ctx, `SELECT COUNT(*) as cnt FROM collections`)
}

func (r *d1CollectionRepository) FindImagesByCollection(ctx context.Context, collectionID int64, limit, offset int) ([]domain.Image, error) {
	rows, err := r.client.Query(ctx, `
		SELECT i.id, ci2.collection_id, i.path, i.filename, i.source_root,
		       i.file_size, i.width, i.height, i.format,
		       COALESCE(i.phash, 0) as phash, COALESCE(i.phash_hex, '') as phash_hex,
		       COALESCE(i.sha256, '') as sha256, COALESCE(i.source_mtime_unix, 0) as source_mtime_unix,
		       i.thumbnail_small_url, i.thumbnail_large_url, i.created_at, i.updated_at
		FROM collection_images ci
		JOIN images i ON i.id = ci.image_id
		LEFT JOIN collection_images ci2 ON ci2.image_id = i.id AND ci2.collection_id = ?
		WHERE ci.collection_id = ?
		ORDER BY ci.image_id DESC
		LIMIT ? OFFSET ?
	`, collectionID, collectionID, int64(limit), int64(offset))
	if err != nil {
		return nil, err
	}
	return mapImagesFromD1(rows)
}

func (r *d1CollectionRepository) CountImages(ctx context.Context, collectionID int64) (int64, error) {
	return r.client.QueryCount(ctx, `
		SELECT COUNT(*) as cnt FROM collection_images WHERE collection_id = ?
	`, collectionID)
}

func (r *d1CollectionRepository) GetLatestImageID(ctx context.Context, collectionID int64) (*int64, error) {
	row, err := r.client.QueryOne(ctx, `
		SELECT image_id FROM collection_images WHERE collection_id = ? ORDER BY image_id DESC LIMIT 1
	`, collectionID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	id, err := toInt64(row["image_id"])
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (r *d1CollectionRepository) FindCollectionIDsByImage(ctx context.Context, imageID int64) ([]int64, error) {
	rows, err := r.client.Query(ctx, `
		SELECT collection_id FROM collection_images WHERE image_id = ?
	`, imageID)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		id, err := toInt64(row["collection_id"])
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *d1CollectionRepository) Save(ctx context.Context, collection *domain.Collection) error {
	now := time.Now()
	id, err := r.client.ExecReturningID(ctx, `
		INSERT INTO collections (name, description, cover_image_id, image_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, collection.Name, collection.Description, collection.CoverImageID, collection.ImageCount, now, now)
	if err != nil {
		return err
	}
	collection.ID = id
	collection.CreatedAt = now
	collection.UpdatedAt = now
	return nil
}

func (r *d1CollectionRepository) Update(ctx context.Context, collection *domain.Collection) error {
	now := time.Now()
	err := r.client.Exec(ctx, `
		UPDATE collections SET name = ?, description = ?, cover_image_id = ?, image_count = ?, updated_at = ?
		WHERE id = ?
	`, collection.Name, collection.Description, collection.CoverImageID, collection.ImageCount, now, collection.ID)
	if err != nil {
		return err
	}
	collection.UpdatedAt = now
	return nil
}

func (r *d1CollectionRepository) Delete(ctx context.Context, id int64) error {
	return r.client.Exec(ctx, `DELETE FROM collections WHERE id = ?`, id)
}

func (r *d1CollectionRepository) AddImage(ctx context.Context, collectionID, imageID int64) error {
	now := time.Now()
	row, _ := r.client.QueryOne(ctx, `SELECT collection_id FROM collection_images WHERE image_id = ? LIMIT 1`, imageID)
	if row != nil {
		if existingCID, err := toInt64(row["collection_id"]); err == nil && existingCID == collectionID {
			return nil
		}
		if err := r.client.Exec(ctx, `DELETE FROM collection_images WHERE image_id = ?`, imageID); err != nil {
			return err
		}
		if row != nil {
			if existingCID, err := toInt64(row["collection_id"]); err == nil && existingCID != 0 {
				_ = r.client.Exec(ctx, `UPDATE collections SET image_count = (SELECT COUNT(*) FROM collection_images WHERE collection_id = ?), updated_at = ? WHERE id = ?`, existingCID, now, existingCID)
			}
		}
	}
	if err := r.client.Exec(ctx, `INSERT INTO collection_images (collection_id, image_id, added_at) VALUES (?, ?, ?)`, collectionID, imageID, now); err != nil {
		return err
	}
	return r.client.Exec(ctx, `UPDATE collections SET image_count = (SELECT COUNT(*) FROM collection_images WHERE collection_id = ?), updated_at = ? WHERE id = ?`, collectionID, now, collectionID)
}

func (r *d1CollectionRepository) RemoveImage(ctx context.Context, collectionID, imageID int64) error {
	now := time.Now()
	if err := r.client.Exec(ctx, `DELETE FROM collection_images WHERE collection_id = ? AND image_id = ?`, collectionID, imageID); err != nil {
		return err
	}
	return r.client.Exec(ctx, `UPDATE collections SET image_count = (SELECT COUNT(*) FROM collection_images WHERE collection_id = ?), updated_at = ? WHERE id = ?`, collectionID, now, collectionID)
}

func (r *d1CollectionRepository) UpdateCover(ctx context.Context, collectionID, imageID int64) error {
	now := time.Now()
	return r.client.Exec(ctx, `UPDATE collections SET cover_image_id = ?, updated_at = ? WHERE id = ?`, imageID, now, collectionID)
}

func (r *d1CollectionRepository) ReconcileAfterImageDelete(ctx context.Context, collectionID int64) error {
	now := time.Now()
	cnt, err := r.client.QueryCount(ctx, `SELECT COUNT(*) as cnt FROM collection_images WHERE collection_id = ?`, collectionID)
	if err != nil {
		return err
	}
	row, err := r.client.QueryOne(ctx, `SELECT image_id FROM collection_images WHERE collection_id = ? ORDER BY added_at DESC LIMIT 1`, collectionID)
	if err != nil {
		return err
	}
	if row != nil {
		latestID, _ := toInt64(row["image_id"])
		return r.client.Exec(ctx, `UPDATE collections SET image_count = ?, cover_image_id = ?, updated_at = ? WHERE id = ?`, cnt, latestID, now, collectionID)
	}
	return r.client.Exec(ctx, `UPDATE collections SET image_count = ?, cover_image_id = NULL, updated_at = ? WHERE id = ?`, cnt, now, collectionID)
}

func mapCollectionsFromD1(rows []map[string]any) ([]domain.Collection, error) {
	collections := make([]domain.Collection, 0, len(rows))
	for _, row := range rows {
		c, err := mapCollectionFromD1(row)
		if err != nil {
			return nil, err
		}
		collections = append(collections, *c)
	}
	return collections, nil
}
