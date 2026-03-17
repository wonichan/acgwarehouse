package repository

import (
	"context"
	"database/sql"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

// SearchRepository provides search functionality using FTS5.
type SearchRepository interface {
	// FTSFullTextSearch performs a full-text search and returns matching image IDs.
	FTSFullTextSearch(ctx context.Context, query string, limit, offset int) ([]int64, error)

	// CountFTSFullTextSearch returns the count of matching images for a full-text search.
	CountFTSFullTextSearch(ctx context.Context, query string) (int64, error)

	// SearchByFilenames performs a LIKE search on filenames.
	SearchByFilenames(ctx context.Context, pattern string, limit, offset int) ([]domain.Image, error)

	// CountByFilenames returns the count of images matching the filename pattern.
	CountByFilenames(ctx context.Context, pattern string) (int64, error)
}

type sqliteSearchRepository struct {
	db *sql.DB
}

// NewSearchRepository creates a new search repository.
func NewSearchRepository(db *sql.DB) SearchRepository {
	return &sqliteSearchRepository{db: db}
}

// FTSFullTextSearch performs a full-text search using FTS5.
// The query is converted to FTS5 format with wildcard support.
func (r *sqliteSearchRepository) FTSFullTextSearch(ctx context.Context, query string, limit, offset int) ([]int64, error) {
	if query == "" {
		return []int64{}, nil
	}

	// Convert query to FTS5 format
	ftsQuery := r.buildFTSQuery(query)

	// Query FTS5 virtual table
	rows, err := r.db.QueryContext(ctx, `
		SELECT image_id FROM images_fts WHERE images_fts MATCH ? ORDER BY rank LIMIT ? OFFSET ?
	`, ftsQuery, limit, offset)
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

// CountFTSFullTextSearch returns the count of matching images.
func (r *sqliteSearchRepository) CountFTSFullTextSearch(ctx context.Context, query string) (int64, error) {
	if query == "" {
		return 0, nil
	}

	ftsQuery := r.buildFTSQuery(query)

	var count int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM images_fts WHERE images_fts MATCH ?
	`, ftsQuery).Scan(&count)
	return count, err
}

// buildFTSQuery converts a user query to FTS5 format.
// Supports:
// - Multiple words: "cat dog" -> "cat AND dog"
// - Wildcards: "cat*" -> "cat*"
func (r *sqliteSearchRepository) buildFTSQuery(query string) string {
	words := strings.Fields(query)
	if len(words) == 0 {
		return ""
	}

	// Join words with AND operator
	for i, word := range words {
		// If word doesn't end with *, add quotes for exact match
		if !strings.HasSuffix(word, "*") {
			words[i] = `"` + word + `"`
		}
	}

	return strings.Join(words, " AND ")
}

// SearchByFilenames performs a LIKE search on filenames.
func (r *sqliteSearchRepository) SearchByFilenames(ctx context.Context, pattern string, limit, offset int) ([]domain.Image, error) {
	if pattern == "" {
		return []domain.Image{}, nil
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, path, filename, source_root, file_size, width, height, format, COALESCE(phash, 0), created_at, updated_at
		FROM images
		WHERE filename LIKE ?
		ORDER BY id
		LIMIT ? OFFSET ?
	`, "%"+pattern+"%", limit, offset)
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

// CountByFilenames returns the count of images matching the filename pattern.
func (r *sqliteSearchRepository) CountByFilenames(ctx context.Context, pattern string) (int64, error) {
	if pattern == "" {
		return 0, nil
	}

	var count int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM images WHERE filename LIKE ?
	`, "%"+pattern+"%").Scan(&count)
	return count, err
}
