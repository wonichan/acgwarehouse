package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/d1client"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type d1GalleryImageRepository struct {
	client *d1client.Client
	tagRepo TagRepository
}

func NewD1GalleryImageRepository(client *d1client.Client, tagRepo TagRepository) GalleryImageQuery {
	return &d1GalleryImageRepository{client: client, tagRepo: tagRepo}
}

func (r *d1GalleryImageRepository) FindByTagIDs(ctx context.Context, tagIDs []int64, limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
	if len(tagIDs) == 0 {
		return []domain.Image{}, nil
	}

	clauses, err := expandHierarchicalTagClausesD1(ctx, r.tagRepo, tagIDs)
	if err != nil {
		return nil, err
	}
	whereClause, args := buildImageTagClauseFilters(clauses, "i.id")

	sortColumn := validImageSortColumn(sortBy)
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}

	sql := fmt.Sprintf(`
		SELECT i.id, ci.collection_id, i.path, i.filename, i.source_root,
		       i.file_size, i.width, i.height, i.format,
		       COALESCE(i.phash, 0) as phash, COALESCE(i.phash_hex, '') as phash_hex,
		       COALESCE(i.sha256, '') as sha256, COALESCE(i.source_mtime_unix, 0) as source_mtime_unix,
		       i.thumbnail_small_url, i.thumbnail_large_url, i.created_at, i.updated_at
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		WHERE %s
		ORDER BY i.%s %s, i.id %s
		LIMIT ? OFFSET ?
	`, whereClause, sortColumn, sortDir, sortDir)

	args = append(args, int64(limit), int64(offset))
	rows, err := r.client.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return mapImagesFromD1(rows)
}

func (r *d1GalleryImageRepository) CountByTagIDs(ctx context.Context, tagIDs []int64) (int64, error) {
	if len(tagIDs) == 0 {
		return 0, nil
	}

	clauses, err := expandHierarchicalTagClausesD1(ctx, r.tagRepo, tagIDs)
	if err != nil {
		return 0, err
	}
	whereClause, args := buildImageTagClauseFilters(clauses, "i.id")

	sql := fmt.Sprintf(`
		SELECT COUNT(DISTINCT i.id) as cnt
		FROM images i
		WHERE %s
	`, whereClause)

	return r.client.QueryCount(ctx, sql, args...)
}

func (r *d1GalleryImageRepository) FindUntagged(ctx context.Context, limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
	sortColumn := validImageSortColumn(sortBy)
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}

	sql := fmt.Sprintf(`
		SELECT i.id, ci.collection_id, i.path, i.filename, i.source_root,
		       i.file_size, i.width, i.height, i.format,
		       COALESCE(i.phash, 0) as phash, COALESCE(i.phash_hex, '') as phash_hex,
		       COALESCE(i.sha256, '') as sha256, COALESCE(i.source_mtime_unix, 0) as source_mtime_unix,
		       i.thumbnail_small_url, i.thumbnail_large_url, i.created_at, i.updated_at
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		WHERE NOT EXISTS (
			SELECT 1 FROM image_tags it WHERE it.image_id = i.id AND it.review_state != ?
		)
		ORDER BY i.%s %s, i.id %s
		LIMIT ? OFFSET ?
	`, sortColumn, sortDir, sortDir)

	rows, err := r.client.Query(ctx, sql, domain.ReviewStateRejected, int64(limit), int64(offset))
	if err != nil {
		return nil, err
	}
	return mapImagesFromD1(rows)
}

func (r *d1GalleryImageRepository) CountUntagged(ctx context.Context) (int64, error) {
	return r.client.QueryCount(ctx, `
		SELECT COUNT(i.id) as cnt
		FROM images i
		WHERE NOT EXISTS (
			SELECT 1 FROM image_tags it WHERE it.image_id = i.id AND it.review_state != ?
		)
	`, domain.ReviewStateRejected)
}

func (r *d1GalleryImageRepository) FindPendingTags(ctx context.Context, limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
	sortColumn := validImageSortColumn(sortBy)
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}

	sql := fmt.Sprintf(`
		SELECT i.id, ci.collection_id, i.path, i.filename, i.source_root,
		       i.file_size, i.width, i.height, i.format,
		       COALESCE(i.phash, 0) as phash, COALESCE(i.phash_hex, '') as phash_hex,
		       COALESCE(i.sha256, '') as sha256, COALESCE(i.source_mtime_unix, 0) as source_mtime_unix,
		       i.thumbnail_small_url, i.thumbnail_large_url, i.created_at, i.updated_at
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		WHERE EXISTS (
			SELECT 1 FROM image_tags it WHERE it.image_id = i.id AND it.review_state = ?
		)
		ORDER BY i.%s %s, i.id %s
		LIMIT ? OFFSET ?
	`, sortColumn, sortDir, sortDir)

	rows, err := r.client.Query(ctx, sql, domain.ReviewStatePending, int64(limit), int64(offset))
	if err != nil {
		return nil, err
	}
	return mapImagesFromD1(rows)
}

func (r *d1GalleryImageRepository) CountPendingTags(ctx context.Context) (int64, error) {
	return r.client.QueryCount(ctx, `
		SELECT COUNT(i.id) as cnt
		FROM images i
		WHERE EXISTS (
			SELECT 1 FROM image_tags it WHERE it.image_id = i.id AND it.review_state = ?
		)
	`, domain.ReviewStatePending)
}

func (r *d1GalleryImageRepository) FindPendingTagsByTagIDs(ctx context.Context, tagIDs []int64, limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
	if len(tagIDs) == 0 {
		return []domain.Image{}, nil
	}

	clauses, err := expandHierarchicalTagClausesD1(ctx, r.tagRepo, tagIDs)
	if err != nil {
		return nil, err
	}
	tagWhereClause, args := buildImageTagClauseFilters(clauses, "i.id")

	sortColumn := validImageSortColumn(sortBy)
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}

	sql := fmt.Sprintf(`
		SELECT i.id, ci.collection_id, i.path, i.filename, i.source_root,
		       i.file_size, i.width, i.height, i.format,
		       COALESCE(i.phash, 0) as phash, COALESCE(i.phash_hex, '') as phash_hex,
		       COALESCE(i.sha256, '') as sha256, COALESCE(i.source_mtime_unix, 0) as source_mtime_unix,
		       i.thumbnail_small_url, i.thumbnail_large_url, i.created_at, i.updated_at
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		WHERE EXISTS (
			SELECT 1 FROM image_tags it WHERE it.image_id = i.id AND it.review_state = ?
		)
		AND %s
		ORDER BY i.%s %s, i.id %s
		LIMIT ? OFFSET ?
	`, tagWhereClause, sortColumn, sortDir, sortDir)

	args = append([]any{domain.ReviewStatePending}, args...)
	args = append(args, int64(limit), int64(offset))

	rows, err := r.client.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return mapImagesFromD1(rows)
}

func (r *d1GalleryImageRepository) CountPendingTagsByTagIDs(ctx context.Context, tagIDs []int64) (int64, error) {
	if len(tagIDs) == 0 {
		return 0, nil
	}

	clauses, err := expandHierarchicalTagClausesD1(ctx, r.tagRepo, tagIDs)
	if err != nil {
		return 0, err
	}
	tagWhereClause, args := buildImageTagClauseFilters(clauses, "i.id")

	sql := fmt.Sprintf(`
		SELECT COUNT(i.id) as cnt
		FROM images i
		WHERE EXISTS (
			SELECT 1 FROM image_tags it WHERE it.image_id = i.id AND it.review_state = ?
		)
		AND %s
	`, tagWhereClause)

	args = append([]any{domain.ReviewStatePending}, args...)
	return r.client.QueryCount(ctx, sql, args...)
}

func (r *d1GalleryImageRepository) FindByGalleryFilter(ctx context.Context, exactTagIDs, subtreeRootTagIDs []int64, limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
	if len(exactTagIDs) == 0 && len(subtreeRootTagIDs) == 0 {
		return []domain.Image{}, nil
	}

	whereClause, args, err := buildGalleryFilterClausesD1(ctx, r.tagRepo, exactTagIDs, subtreeRootTagIDs, "i.id")
	if err != nil {
		return nil, err
	}

	sortColumn := validImageSortColumn(sortBy)
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}

	sql := fmt.Sprintf(`
		SELECT i.id, ci.collection_id, i.path, i.filename, i.source_root,
		       i.file_size, i.width, i.height, i.format,
		       COALESCE(i.phash, 0) as phash, COALESCE(i.phash_hex, '') as phash_hex,
		       COALESCE(i.sha256, '') as sha256, COALESCE(i.source_mtime_unix, 0) as source_mtime_unix,
		       i.thumbnail_small_url, i.thumbnail_large_url, i.created_at, i.updated_at
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		WHERE %s
		ORDER BY %s %s, i.id %s
		LIMIT ? OFFSET ?
	`, whereClause, sortColumn, sortDir, sortDir)

	args = append(args, int64(limit), int64(offset))
	rows, err := r.client.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return mapImagesFromD1(rows)
}

func (r *d1GalleryImageRepository) CountByGalleryFilter(ctx context.Context, exactTagIDs, subtreeRootTagIDs []int64) (int64, error) {
	if len(exactTagIDs) == 0 && len(subtreeRootTagIDs) == 0 {
		return 0, nil
	}

	whereClause, args, err := buildGalleryFilterClausesD1(ctx, r.tagRepo, exactTagIDs, subtreeRootTagIDs, "i.id")
	if err != nil {
		return 0, err
	}

	sql := fmt.Sprintf(`
		SELECT COUNT(DISTINCT i.id) as cnt
		FROM images i
		WHERE %s
	`, whereClause)

	return r.client.QueryCount(ctx, sql, args...)
}

// expandHierarchicalTagClausesD1 is the D1 equivalent of expandHierarchicalTagClauses
// but uses the TagRepository interface instead of direct DB access.
func expandHierarchicalTagClausesD1(ctx context.Context, tagRepo TagRepository, tagIDs []int64) ([][]int64, error) {
	if len(tagIDs) == 0 {
		return nil, nil
	}

	resolved, err := tagRepo.ResolveDescendantIDs(ctx, tagIDs)
	if err != nil {
		return nil, err
	}

	clauses := make([][]int64, 0, len(tagIDs))
	for _, tagID := range tagIDs {
		ids := resolved[tagID]
		if len(ids) == 0 {
			ids = []int64{tagID}
		}
		clauses = append(clauses, ids)
	}
	return clauses, nil
}

// buildGalleryFilterClausesD1 is the D1 equivalent of buildGalleryFilterClauses
// but uses the TagRepository interface instead of direct DB access.
func buildGalleryFilterClausesD1(ctx context.Context, tagRepo TagRepository, exactTagIDs, subtreeRootTagIDs []int64, imageColumn string) (string, []any, error) {
	var parts []string
	var args []any

	for _, tagID := range exactTagIDs {
		parts = append(parts, fmt.Sprintf(`%s IN (
			SELECT it.image_id
			FROM image_tags it
			WHERE it.tag_id = ? AND it.review_state != 'rejected'
		)`, imageColumn))
		args = append(args, tagID)
	}

	if len(subtreeRootTagIDs) > 0 {
		clauses, err := expandHierarchicalTagClausesD1(ctx, tagRepo, subtreeRootTagIDs)
		if err != nil {
			return "", nil, err
		}
		subtreeWhere, subtreeArgs := buildImageTagClauseFilters(clauses, imageColumn)
		if subtreeWhere != "" {
			parts = append(parts, subtreeWhere)
			args = append(args, subtreeArgs...)
		}
	}

	if len(parts) == 0 {
		return "", nil, nil
	}
	return strings.Join(parts, " AND "), args, nil
}