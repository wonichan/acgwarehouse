package repository

import (
	"database/sql"
	"errors"

	"github.com/yourusername/acgwarehouse-backend/internal/domain"
)

type ImageRepository interface {
	SaveImage(image *domain.Image) error
	FindByPath(path string) (*domain.Image, error)
	FindAll(limit, offset int) ([]domain.Image, error)
	Count() (int64, error)
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
		(path, filename, source_root, file_size, width, height, format, phash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, image.Path, image.Filename, image.SourceRoot, image.FileSize, image.Width, image.Height, image.Format, image.PHash, image.CreatedAt, image.UpdatedAt)
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
	image := &domain.Image{}
	err := r.db.QueryRow(`
		SELECT id, path, filename, source_root, file_size, width, height, format, phash, created_at, updated_at
		FROM images WHERE path = ?
	`, path).Scan(
		&image.ID,
		&image.Path,
		&image.Filename,
		&image.SourceRoot,
		&image.FileSize,
		&image.Width,
		&image.Height,
		&image.Format,
		&image.PHash,
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
		SELECT id, path, filename, source_root, file_size, width, height, format, phash, created_at, updated_at
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
			&image.CreatedAt,
			&image.UpdatedAt,
		); err != nil {
			return nil, err
		}
		images = append(images, image)
	}

	return images, rows.Err()
}

func (r *sqliteImageRepository) Count() (int64, error) {
	var count int64
	err := r.db.QueryRow(`SELECT COUNT(*) FROM images`).Scan(&count)
	return count, err
}
