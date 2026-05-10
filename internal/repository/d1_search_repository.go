package repository

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/d1client"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type d1SearchRepository struct {
	client *d1client.Client
}

func NewD1SearchRepository(client *d1client.Client) SearchRepository {
	return &d1SearchRepository{client: client}
}

func (r *d1SearchRepository) FTSFullTextSearch(ctx context.Context, query string, limit, offset int) ([]int64, error) {
	if query == "" {
		return []int64{}, nil
	}
	ftsQuery := buildD1FTSQuery(query)
	rows, err := r.client.Query(ctx, `
		SELECT image_id FROM images_fts WHERE images_fts MATCH ? ORDER BY rank, image_id ASC LIMIT ? OFFSET ?
	`, ftsQuery, int64(limit), int64(offset))
	if err != nil {
		return nil, err
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		id, err := toInt64(row["image_id"])
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *d1SearchRepository) CountFTSFullTextSearch(ctx context.Context, query string) (int64, error) {
	if query == "" {
		return 0, nil
	}
	ftsQuery := buildD1FTSQuery(query)
	return r.client.QueryCount(ctx, `
		SELECT COUNT(*) as cnt FROM images_fts WHERE images_fts MATCH ?
	`, ftsQuery)
}

func (r *d1SearchRepository) SearchByFilenames(ctx context.Context, pattern string, limit, offset int) ([]domain.Image, error) {
	if pattern == "" {
		return []domain.Image{}, nil
	}
	rows, err := r.client.Query(ctx, `
		SELECT id, path, filename, source_root, file_size, width, height, format,
		       COALESCE(phash, 0) as phash, COALESCE(phash_hex, '') as phash_hex,
		       COALESCE(sha256, '') as sha256, COALESCE(source_mtime_unix, 0) as source_mtime_unix,
		       thumbnail_small_url, thumbnail_large_url, created_at, updated_at
		FROM images
		WHERE filename LIKE ?
		ORDER BY id
		LIMIT ? OFFSET ?
	`, "%"+pattern+"%", int64(limit), int64(offset))
	if err != nil {
		return nil, err
	}
	return mapImagesFromD1(rows)
}

func (r *d1SearchRepository) CountByFilenames(ctx context.Context, pattern string) (int64, error) {
	if pattern == "" {
		return 0, nil
	}
	return r.client.QueryCount(ctx, `
		SELECT COUNT(*) as cnt FROM images WHERE filename LIKE ?
	`, "%"+pattern+"%")
}

func (r *d1SearchRepository) SearchImages(ctx context.Context, opts SearchQueryOptions) ([]domain.Image, error) {
	if strings.TrimSpace(opts.Query) == "" {
		return []domain.Image{}, nil
	}

	_, _, orderBy := normalizeD1SearchSort(opts.SortBy, opts.SortOrder)
	baseWhere, args, err := r.buildSearchWhere(ctx, opts.Query, opts.TagIDs)
	if err != nil {
		return nil, err
	}

	sql := fmt.Sprintf(`
		SELECT i.id, i.path, i.filename, i.source_root, i.file_size, i.width, i.height, i.format,
		       COALESCE(i.phash, 0) as phash, COALESCE(i.phash_hex, '') as phash_hex,
		       COALESCE(i.sha256, '') as sha256, COALESCE(i.source_mtime_unix, 0) as source_mtime_unix,
		       i.thumbnail_small_url, i.thumbnail_large_url, i.created_at, i.updated_at
		FROM images i
		JOIN images_fts ON images_fts.image_id = i.id
		WHERE %s
		ORDER BY %s
		LIMIT ? OFFSET ?
	`, baseWhere, orderBy)

	args = append(args, int64(opts.Limit), int64(opts.Offset))
	rows, err := r.client.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return mapImagesFromD1(rows)
}

func (r *d1SearchRepository) CountSearchImages(ctx context.Context, opts SearchQueryOptions) (int64, error) {
	if strings.TrimSpace(opts.Query) == "" {
		return 0, nil
	}
	baseWhere, args, err := r.buildSearchWhere(ctx, opts.Query, opts.TagIDs)
	if err != nil {
		return 0, err
	}

	sql := fmt.Sprintf(`
		SELECT COUNT(*) as cnt
		FROM images i
		JOIN images_fts ON images_fts.image_id = i.id
		WHERE %s
	`, baseWhere)

	return r.client.QueryCount(ctx, sql, args...)
}

func (r *d1SearchRepository) buildSearchWhere(_ context.Context, query string, tagIDs []int64) (string, []any, error) {
	clauses := []string{"images_fts MATCH ?"}
	args := []any{buildD1FTSQuery(query)}
	if len(tagIDs) > 0 {
		placeholders := make([]string, len(tagIDs))
		for i, id := range tagIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		tagClause := fmt.Sprintf(`i.id IN (SELECT it.image_id FROM image_tags it WHERE it.tag_id IN (%s) AND it.review_state != 'rejected')`, strings.Join(placeholders, ", "))
		clauses = append(clauses, tagClause)
	}
	return strings.Join(clauses, " AND "), args, nil
}

func normalizeD1SearchSort(sortBy, sortOrder string) (string, string, string) {
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

var d1FTSPrefixToken = regexp.MustCompile(`^[\pL\pN_]+$`)

func buildD1FTSQuery(query string) string {
	words := strings.Fields(query)
	if len(words) == 0 {
		return ""
	}
	for i, word := range words {
		words[i] = buildD1FTSToken(word)
	}
	return strings.Join(words, " AND ")
}

func buildD1FTSToken(word string) string {
	if prefix, ok := strings.CutSuffix(word, "*"); ok {
		if prefix != "" && d1FTSPrefixToken.MatchString(prefix) {
			return prefix + "*"
		}
	}
	return `"` + strings.ReplaceAll(word, `"`, `""`) + `"`
}