package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type ImageRepository interface {
	SaveImage(image *domain.Image) error
	FindByID(id int64) (*domain.Image, error)
	FindByPath(path string) (*domain.Image, error)
	FindAll(limit, offset int) ([]domain.Image, error)
	FindByTagIDs(ctx context.Context, tagIDs []int64, limit, offset int) ([]domain.Image, error)
	CountByTagIDs(ctx context.Context, tagIDs []int64) (int64, error)
	UpdateThumbnails(id int64, smallURL, largeURL string) error
	Count() (int64, error)
	Delete(id int64) error
}

type sqliteImageRepository struct {
	db *sql.DB
}

func NewImageRepository(db *sql.DB) ImageRepository {
	return &sqliteImageRepository{db: db}
}

func (r *sqliteImageRepository) SaveImage(image *domain.Image) error {
	result, err := r.db.Exec(`
		INSERT OR IGNORE INTO images
		(path, filename, source_root, file_size, width, height, format, phash, thumbnail_small_url, thumbnail_large_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, image.Path, image.Filename, image.SourceRoot, image.FileSize, image.Width, image.Height, image.Format, image.PHash, image.ThumbnailSmallUrl, image.ThumbnailLargeUrl, image.CreatedAt, image.UpdatedAt)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	if id > 0 {
		image.ID = id
		return nil
	}

	existing, err := r.FindByPath(image.Path)
	if err != nil {
		return err
	}
	image.ID = existing.ID
	return nil
}

func (r *sqliteImageRepository) FindByPath(path string) (*domain.Image, error) {
	return r.queryOne(`
		SELECT id, path, filename, source_root, file_size, width, height, format, COALESCE(phash, 0), COALESCE(thumbnail_small_url, ''), COALESCE(thumbnail_large_url, ''), created_at, updated_at
		FROM images WHERE path = ?
	`, path)
}

func (r *sqliteImageRepository) FindByID(id int64) (*domain.Image, error) {
	return r.queryOne(`
		SELECT id, path, filename, source_root, file_size, width, height, format, COALESCE(phash, 0), COALESCE(thumbnail_small_url, ''), COALESCE(thumbnail_large_url, ''), created_at, updated_at
		FROM images WHERE id = ?
	`, id)
}

func (r *sqliteImageRepository) queryOne(query string, args ...any) (*domain.Image, error) {
	image := &domain.Image{}
	err := r.db.QueryRow(query, args...).Scan(
		&image.ID,
		&image.Path,
		&image.Filename,
		&image.SourceRoot,
		&image.FileSize,
		&image.Width,
		&image.Height,
		&image.Format,
		&image.PHash,
		&image.ThumbnailSmallUrl,
		&image.ThumbnailLargeUrl,
		&image.CreatedAt,
		&image.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}

	return image, nil
}

func (r *sqliteImageRepository) FindAll(limit, offset int) ([]domain.Image, error) {
	rows, err := r.db.Query(`
		SELECT id, path, filename, source_root, file_size, width, height, format, COALESCE(phash, 0), COALESCE(thumbnail_small_url, ''), COALESCE(thumbnail_large_url, ''), created_at, updated_at
		FROM images ORDER BY id LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := make([]domain.Image, 0)
	for rows.Next() {
		var image domain.Image
		if err := rows.Scan(
			&image.ID,
			&image.Path,
			&image.Filename,
			&image.SourceRoot,
			&image.FileSize,
			&image.Width,
			&image.Height,
			&image.Format,
			&image.PHash,
			&image.ThumbnailSmallUrl,
			&image.ThumbnailLargeUrl,
			&image.CreatedAt,
			&image.UpdatedAt,
		); err != nil {
			return nil, err
		}
		images = append(images, image)
	}

	return images, rows.Err()
}

func (r *sqliteImageRepository) UpdateThumbnails(id int64, smallURL, largeURL string) error {
	_, err := r.db.Exec(`
		UPDATE images
		SET thumbnail_small_url = ?, thumbnail_large_url = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, smallURL, largeURL, id)
	return err
}

func (r *sqliteImageRepository) Count() (int64, error) {
	var count int64
	err := r.db.QueryRow(`SELECT COUNT(*) FROM images`).Scan(&count)
	return count, err
}

// FindByTagIDs returns images that have ALL the specified tag IDs (AND semantics).
// It joins images with image_tags and counts matched tags per image, returning only
// images where the matched count equals the number of requested tags.
func (r *sqliteImageRepository) FindByTagIDs(ctx context.Context, tagIDs []int64, limit, offset int) ([]domain.Image, error) {
	if len(tagIDs) == 0 {
		return []domain.Image{}, nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(tagIDs))
	args := make([]any, 0, len(tagIDs)+2)
	for i, id := range tagIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	// Query images where matched tag count equals the number of requested tags (AND semantics)
	query := fmt.Sprintf(`
		SELECT i.id, i.path, i.filename, i.source_root, i.file_size, i.width, i.height, i.format, COALESCE(i.phash, 0), COALESCE(i.thumbnail_small_url, ''), COALESCE(i.thumbnail_large_url, ''), i.created_at, i.updated_at
		FROM images i
		INNER JOIN image_tags it ON it.image_id = i.id
		WHERE it.tag_id IN (%s)
		GROUP BY i.id
		HAVING COUNT(DISTINCT it.tag_id) = ?
		ORDER BY i.id
		LIMIT ? OFFSET ?
	`, strings.Join(placeholders, ", "))

	args = append(args, int64(len(tagIDs)), int64(limit), int64(offset))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := make([]domain.Image, 0)
	for rows.Next() {
		var image domain.Image
		if err := rows.Scan(
			&image.ID,
			&image.Path,
			&image.Filename,
			&image.SourceRoot,
			&image.FileSize,
			&image.Width,
			&image.Height,
			&image.Format,
			&image.PHash,
			&image.ThumbnailSmallUrl,
			&image.ThumbnailLargeUrl,
			&image.CreatedAt,
			&image.UpdatedAt,
		); err != nil {
			return nil, err
		}
		images = append(images, image)
	}

	return images, rows.Err()
}

// CountByTagIDs returns the count of images that have ALL the specified tag IDs (AND semantics).
func (r *sqliteImageRepository) CountByTagIDs(ctx context.Context, tagIDs []int64) (int64, error) {
	if len(tagIDs) == 0 {
		return 0, nil
	}

	placeholders := make([]string, len(tagIDs))
	args := make([]any, 0, len(tagIDs)+1)
	for i, id := range tagIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf(`
		SELECT COUNT(DISTINCT sub.image_id)
		FROM (
			SELECT it.image_id
			FROM image_tags it
			WHERE it.tag_id IN (%s)
			GROUP BY it.image_id
			HAVING COUNT(DISTINCT it.tag_id) = ?
		) sub
	`, strings.Join(placeholders, ", "))

	args = append(args, int64(len(tagIDs)))

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

// Delete removes an image by ID. Due to ON DELETE CASCADE in the schema,
// this will also remove related records in image_tags and collection_images.
func (r *sqliteImageRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM images WHERE id = ?`, id)
	return err
}
