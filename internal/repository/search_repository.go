package repository

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
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

	// SearchImages performs a query-backed search with optional tag filtering.
	SearchImages(ctx context.Context, opts SearchQueryOptions) ([]domain.Image, error)

	// CountSearchImages returns the total count for a query-backed search with optional tag filtering.
	CountSearchImages(ctx context.Context, opts SearchQueryOptions) (int64, error)
}

type SearchQueryOptions struct {
	Query     string
	TagIDs    []int64
	SortBy    string
	SortOrder string
	Limit     int
	Offset    int
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
		SELECT image_id FROM images_fts WHERE images_fts MATCH ? ORDER BY rank, image_id ASC LIMIT ? OFFSET ?
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

	for i, word := range words {
		words[i] = buildFTSToken(word)
	}

	return strings.Join(words, " AND ")
}

var simpleFTSPrefixToken = regexp.MustCompile(`^[\pL\pN_]+$`)

func buildFTSToken(word string) string {
	if strings.HasSuffix(word, "*") {
		prefix := strings.TrimSuffix(word, "*")
		if prefix != "" && simpleFTSPrefixToken.MatchString(prefix) {
			return prefix + "*"
		}
	}
	return `"` + strings.ReplaceAll(word, `"`, `""`) + `"`
}

func (r *sqliteSearchRepository) SearchImages(ctx context.Context, opts SearchQueryOptions) ([]domain.Image, error) {
	if strings.TrimSpace(opts.Query) == "" {
		return []domain.Image{}, nil
	}

	sortColumn, sortOrder, orderBy := normalizeSearchSort(opts.SortBy, opts.SortOrder)
	baseWhere, args := r.buildSearchWhere(opts.Query, opts.TagIDs)
	query := fmt.Sprintf(`
		SELECT i.id, i.path, i.filename, i.source_root, i.file_size, i.width, i.height, i.format, COALESCE(i.phash, 0), COALESCE(i.phash_hex, ''), i.thumbnail_small_url, i.thumbnail_large_url, i.created_at, i.updated_at
		FROM images i
		JOIN images_fts ON images_fts.image_id = i.id
		WHERE %s
		ORDER BY %s
		LIMIT ? OFFSET ?
	`, baseWhere, orderBy)
	_ = sortColumn
	_ = sortOrder
	args = append(args, opts.Limit, opts.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := make([]domain.Image, 0)
	for rows.Next() {
		var image domain.Image
		var thumbnailSmallURL, thumbnailLargeURL sql.NullString
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
			&image.PHashHex,
			&thumbnailSmallURL,
			&thumbnailLargeURL,
			&image.CreatedAt,
			&image.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if thumbnailSmallURL.Valid {
			image.ThumbnailSmallUrl = thumbnailSmallURL.String
		}
		if thumbnailLargeURL.Valid {
			image.ThumbnailLargeUrl = thumbnailLargeURL.String
		}
		images = append(images, image)
	}

	return images, rows.Err()
}

func (r *sqliteSearchRepository) CountSearchImages(ctx context.Context, opts SearchQueryOptions) (int64, error) {
	if strings.TrimSpace(opts.Query) == "" {
		return 0, nil
	}
	baseWhere, args := r.buildSearchWhere(opts.Query, opts.TagIDs)
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM images i
		JOIN images_fts ON images_fts.image_id = i.id
		WHERE %s
	`, baseWhere)
	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

func normalizeSearchSort(sortBy, sortOrder string) (string, string, string) {
	validSortFields := map[string]string{
		"relevance":  "rank",
		"created_at": "i.created_at",
		"filename":   "i.filename",
		"file_size":  "i.file_size",
		"id":         "i.id",
	}
	sortColumn := validSortFields[sortBy]
	if sortColumn == "" {
		sortColumn = "rank"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}
	if sortColumn == "rank" {
		return sortColumn, sortOrder, "rank, i.id ASC"
	}
	return sortColumn, sortOrder, fmt.Sprintf("%s %s, i.id %s", sortColumn, sortOrder, sortOrder)
}

func (r *sqliteSearchRepository) buildSearchWhere(query string, tagIDs []int64) (string, []any) {
	clauses := []string{"images_fts MATCH ?"}
	args := []any{r.buildFTSQuery(query)}
	if len(tagIDs) > 0 {
		placeholders := make([]string, len(tagIDs))
		for i, id := range tagIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		clauses = append(clauses, fmt.Sprintf(`i.id IN (
			SELECT it.image_id
			FROM image_tags it
			WHERE it.tag_id IN (%s) AND it.review_state != 'rejected'
			GROUP BY it.image_id
			HAVING COUNT(DISTINCT it.tag_id) = ?
		)`, strings.Join(placeholders, ", ")))
		args = append(args, int64(len(tagIDs)))
	}
	return strings.Join(clauses, " AND "), args
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
