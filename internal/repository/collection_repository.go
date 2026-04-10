package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

// CollectionRepository defines the interface for collection data access
type CollectionRepository interface {
	// Collection CRUD operations
	Save(ctx context.Context, collection *domain.Collection) error
	FindByID(ctx context.Context, id int64) (*domain.Collection, error)
	FindAll(ctx context.Context, limit, offset int) ([]domain.Collection, error)
	FindByName(ctx context.Context, name string) (*domain.Collection, error)
	Update(ctx context.Context, collection *domain.Collection) error
	Delete(ctx context.Context, id int64) error

	// Collection Image operations
	AddImage(ctx context.Context, collectionID, imageID int64) error
	RemoveImage(ctx context.Context, collectionID, imageID int64) error
	FindImagesByCollection(ctx context.Context, collectionID int64, limit, offset int) ([]domain.Image, error)
	CountImages(ctx context.Context, collectionID int64) (int64, error)

	// Cover operations
	UpdateCover(ctx context.Context, collectionID, imageID int64) error
	GetLatestImageID(ctx context.Context, collectionID int64) (*int64, error)

	// Statistics
	Count(ctx context.Context) (int64, error)

	FindCollectionIDsByImage(ctx context.Context, imageID int64) ([]int64, error)
	ReconcileAfterImageDelete(ctx context.Context, collectionID int64) error
}

type sqliteCollectionRepository struct {
	db *sql.DB
}

// NewCollectionRepository creates a new CollectionRepository instance
func NewCollectionRepository(db *sql.DB) CollectionRepository {
	return &sqliteCollectionRepository{db: db}
}

// Save creates a new collection and updates the ID field
func (r *sqliteCollectionRepository) Save(ctx context.Context, collection *domain.Collection) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO collections (name, description, cover_image_id, image_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, collection.Name, collection.Description, collection.CoverImageID, collection.ImageCount, now, now)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	collection.ID = id
	collection.CreatedAt = now
	collection.UpdatedAt = now
	return nil
}

// FindByID retrieves a collection by its ID
func (r *sqliteCollectionRepository) FindByID(ctx context.Context, id int64) (*domain.Collection, error) {
	collection := &domain.Collection{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, description, cover_image_id, image_count, created_at, updated_at
		FROM collections WHERE id = ?
	`, id).Scan(
		&collection.ID,
		&collection.Name,
		&collection.Description,
		&collection.CoverImageID,
		&collection.ImageCount,
		&collection.CreatedAt,
		&collection.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}
	return collection, nil
}

// FindAll retrieves all collections with pagination
func (r *sqliteCollectionRepository) FindAll(ctx context.Context, limit, offset int) ([]domain.Collection, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, description, cover_image_id, image_count, created_at, updated_at
		FROM collections ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	collections := make([]domain.Collection, 0)
	for rows.Next() {
		var c domain.Collection
		if err := rows.Scan(
			&c.ID,
			&c.Name,
			&c.Description,
			&c.CoverImageID,
			&c.ImageCount,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		collections = append(collections, c)
	}
	return collections, rows.Err()
}

// FindByName retrieves a collection by its name
func (r *sqliteCollectionRepository) FindByName(ctx context.Context, name string) (*domain.Collection, error) {
	collection := &domain.Collection{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, description, cover_image_id, image_count, created_at, updated_at
		FROM collections WHERE name = ?
	`, name).Scan(
		&collection.ID,
		&collection.Name,
		&collection.Description,
		&collection.CoverImageID,
		&collection.ImageCount,
		&collection.CreatedAt,
		&collection.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}
	return collection, nil
}

// Update modifies an existing collection
func (r *sqliteCollectionRepository) Update(ctx context.Context, collection *domain.Collection) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE collections SET name = ?, description = ?, cover_image_id = ?, image_count = ?, updated_at = ?
		WHERE id = ?
	`, collection.Name, collection.Description, collection.CoverImageID, collection.ImageCount, now, collection.ID)
	if err != nil {
		return err
	}
	collection.UpdatedAt = now
	return nil
}

// Delete removes a collection by its ID
func (r *sqliteCollectionRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM collections WHERE id = ?`, id)
	return err
}

// AddImage adds an image to a collection
func (r *sqliteCollectionRepository) AddImage(ctx context.Context, collectionID, imageID int64) error {
	now := time.Now()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var existingCollectionID sql.NullInt64
	err = tx.QueryRowContext(ctx, `
		SELECT collection_id
		FROM collection_images
		WHERE image_id = ?
		LIMIT 1
	`, imageID).Scan(&existingCollectionID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if existingCollectionID.Valid && existingCollectionID.Int64 == collectionID {
		return tx.Commit()
	}

	if _, err = tx.ExecContext(ctx, `DELETE FROM collection_images WHERE image_id = ?`, imageID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO collection_images (collection_id, image_id, added_at)
		VALUES (?, ?, ?)
	`, collectionID, imageID, now); err != nil {
		return err
	}

	if existingCollectionID.Valid {
		if _, err = tx.ExecContext(ctx, `
			UPDATE collections SET image_count = (
				SELECT COUNT(*) FROM collection_images WHERE collection_id = ?
			), updated_at = ? WHERE id = ?
		`, existingCollectionID.Int64, now, existingCollectionID.Int64); err != nil {
			return err
		}
	}

	if _, err = tx.ExecContext(ctx, `
		UPDATE collections SET image_count = (
			SELECT COUNT(*) FROM collection_images WHERE collection_id = ?
		), updated_at = ? WHERE id = ?
	`, collectionID, now, collectionID); err != nil {
		return err
	}

	return tx.Commit()
}

// RemoveImage removes an image from a collection
func (r *sqliteCollectionRepository) RemoveImage(ctx context.Context, collectionID, imageID int64) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM collection_images WHERE collection_id = ? AND image_id = ?
	`, collectionID, imageID)
	if err != nil {
		return err
	}

	// Update image_count in collection
	_, err = r.db.ExecContext(ctx, `
		UPDATE collections SET image_count = (
			SELECT COUNT(*) FROM collection_images WHERE collection_id = ?
		), updated_at = ? WHERE id = ?
	`, collectionID, now, collectionID)
	return err
}

// FindImagesByCollection retrieves all images in a collection with pagination
func (r *sqliteCollectionRepository) FindImagesByCollection(ctx context.Context, collectionID int64, limit, offset int) ([]domain.Image, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT i.id, ci.collection_id, i.path, i.filename, i.source_root, i.file_size, i.width, i.height, i.format,
		       COALESCE(i.phash, 0), COALESCE(i.phash_hex, ''), COALESCE(i.sha256, ''), COALESCE(i.source_mtime_unix, 0),
		       i.thumbnail_small_url, i.thumbnail_large_url, i.created_at, i.updated_at
		FROM images i
		INNER JOIN collection_images ci ON ci.image_id = i.id
		WHERE ci.collection_id = ?
		ORDER BY ci.added_at DESC
		LIMIT ? OFFSET ?
	`, collectionID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := make([]domain.Image, 0)
	for rows.Next() {
		var img domain.Image
		var collectionID sql.NullInt64
		var thumbnailSmallUrl, thumbnailLargeUrl sql.NullString
		if err := rows.Scan(
			&img.ID,
			&collectionID,
			&img.Path,
			&img.Filename,
			&img.SourceRoot,
			&img.FileSize,
			&img.Width,
			&img.Height,
			&img.Format,
			&img.PHash,
			&img.PHashHex,
			&img.SHA256,
			&img.SourceMTimeUnix,
			&thumbnailSmallUrl,
			&thumbnailLargeUrl,
			&img.CreatedAt,
			&img.UpdatedAt,
		); err != nil {
			return nil, err
		}
		// Convert NullString to string (empty if NULL)
		if thumbnailSmallUrl.Valid {
			img.ThumbnailSmallUrl = thumbnailSmallUrl.String
		}
		if thumbnailLargeUrl.Valid {
			img.ThumbnailLargeUrl = thumbnailLargeUrl.String
		}
		if collectionID.Valid {
			value := collectionID.Int64
			img.CollectionID = &value
		}
		images = append(images, img)
	}
	return images, rows.Err()
}

// CountImages returns the number of images in a collection
func (r *sqliteCollectionRepository) CountImages(ctx context.Context, collectionID int64) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM collection_images WHERE collection_id = ?
	`, collectionID).Scan(&count)
	return count, err
}

// UpdateCover sets the cover image for a collection
func (r *sqliteCollectionRepository) UpdateCover(ctx context.Context, collectionID, imageID int64) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE collections SET cover_image_id = ?, updated_at = ? WHERE id = ?
	`, imageID, now, collectionID)
	return err
}

// GetLatestImageID returns the most recently added image ID in a collection
func (r *sqliteCollectionRepository) GetLatestImageID(ctx context.Context, collectionID int64) (*int64, error) {
	var imageID int64
	err := r.db.QueryRowContext(ctx, `
		SELECT image_id FROM collection_images
		WHERE collection_id = ?
		ORDER BY added_at DESC
		LIMIT 1
	`, collectionID).Scan(&imageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &imageID, nil
}

// Count returns the total number of collections
func (r *sqliteCollectionRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM collections`).Scan(&count)
	return count, err
}

func (r *sqliteCollectionRepository) FindCollectionIDsByImage(ctx context.Context, imageID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT collection_id
		FROM collection_images
		WHERE image_id = ?
	`, imageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

func (r *sqliteCollectionRepository) ReconcileAfterImageDelete(ctx context.Context, collectionID int64) error {
	var imageCount int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM collection_images WHERE collection_id = ?
	`, collectionID).Scan(&imageCount); err != nil {
		return err
	}

	var latestImageID sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
		SELECT image_id FROM collection_images
		WHERE collection_id = ?
		ORDER BY added_at DESC
		LIMIT 1
	`, collectionID).Scan(&latestImageID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	now := time.Now()
	if latestImageID.Valid {
		_, err = r.db.ExecContext(ctx, `
			UPDATE collections
			SET image_count = ?, cover_image_id = ?, updated_at = ?
			WHERE id = ?
		`, imageCount, latestImageID.Int64, now, collectionID)
		return err
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE collections
		SET image_count = ?, cover_image_id = NULL, updated_at = ?
		WHERE id = ?
	`, imageCount, now, collectionID)
	return err
}
