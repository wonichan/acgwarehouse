package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/d1client"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type d1ImageRepository struct {
	client  *d1client.Client
	tagRepo TagRepository
}

func NewD1ImageRepository(client *d1client.Client) ImageRepository {
	return &d1ImageRepository{client: client}
}

func NewD1ImageRepositoryWithTags(client *d1client.Client, tagRepo TagRepository) ImageRepository {
	return &d1ImageRepository{client: client, tagRepo: tagRepo}
}

func (r *d1ImageRepository) FindByID(id int64) (*domain.Image, error) {
	row, err := r.client.QueryOne(context.Background(), `
		SELECT i.id, ci.collection_id, i.path, i.filename, i.source_root,
		       i.file_size, i.width, i.height, i.format,
		       COALESCE(i.phash, 0) as phash, COALESCE(i.phash_hex, '') as phash_hex,
		       COALESCE(i.sha256, '') as sha256, COALESCE(i.source_mtime_unix, 0) as source_mtime_unix,
		       i.thumbnail_small_url, i.thumbnail_large_url, i.created_at, i.updated_at
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		WHERE i.id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, sql.ErrNoRows
	}
	return mapImageFromD1(row)
}

func (r *d1ImageRepository) FindByPath(path string) (*domain.Image, error) {
	row, err := r.client.QueryOne(context.Background(), `
		SELECT i.id, ci.collection_id, i.path, i.filename, i.source_root,
		       i.file_size, i.width, i.height, i.format,
		       COALESCE(i.phash, 0) as phash, COALESCE(i.phash_hex, '') as phash_hex,
		       COALESCE(i.sha256, '') as sha256, COALESCE(i.source_mtime_unix, 0) as source_mtime_unix,
		       i.thumbnail_small_url, i.thumbnail_large_url, i.created_at, i.updated_at
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		WHERE i.path = ?
	`, path)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, sql.ErrNoRows
	}
	return mapImageFromD1(row)
}

func (r *d1ImageRepository) FindAll(limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
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
		ORDER BY i.%s %s, i.id %s
		LIMIT ? OFFSET ?
	`, sortColumn, sortDir, sortDir)

	rows, err := r.client.Query(context.Background(), sql, int64(limit), int64(offset))
	if err != nil {
		return nil, err
	}
	return mapImagesFromD1(rows)
}

func (r *d1ImageRepository) FindByIDRange(limit int, lastID int64) ([]domain.Image, error) {
	if limit <= 0 {
		limit = 1000
	}
	rows, err := r.client.Query(context.Background(), `
		SELECT i.id, ci.collection_id, i.path, i.filename, i.source_root,
		       i.file_size, i.width, i.height, i.format,
		       COALESCE(i.phash, 0) as phash, COALESCE(i.phash_hex, '') as phash_hex,
		       COALESCE(i.sha256, '') as sha256, COALESCE(i.source_mtime_unix, 0) as source_mtime_unix,
		       i.thumbnail_small_url, i.thumbnail_large_url, i.created_at, i.updated_at
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		WHERE i.id > ?
		ORDER BY i.id ASC
		LIMIT ?
	`, lastID, int64(limit))
	if err != nil {
		return nil, err
	}
	return mapImagesFromD1(rows)
}

func (r *d1ImageRepository) FindBySourceRootsAfterID(limit int, lastID int64, sourceRoots []string) ([]domain.Image, error) {
	if limit <= 0 {
		limit = 1000
	}
	roots := make([]string, 0, len(sourceRoots))
	for _, root := range sourceRoots {
		if trimmed := strings.TrimSpace(root); trimmed != "" {
			roots = append(roots, trimmed)
		}
	}
	if len(roots) == 0 {
		return []domain.Image{}, nil
	}

	placeholders := strings.Repeat("?,", len(roots))
	placeholders = strings.TrimRight(placeholders, ",")
	sql := fmt.Sprintf(`
		SELECT i.id, ci.collection_id, i.path, i.filename, i.source_root,
		       i.file_size, i.width, i.height, i.format,
		       COALESCE(i.phash, 0) as phash, COALESCE(i.phash_hex, '') as phash_hex,
		       COALESCE(i.sha256, '') as sha256, COALESCE(i.source_mtime_unix, 0) as source_mtime_unix,
		       i.thumbnail_small_url, i.thumbnail_large_url, i.created_at, i.updated_at
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		WHERE i.source_root IN (%s)
		  AND i.id > ?
		ORDER BY i.id ASC
		LIMIT ?
	`, placeholders)

	args := make([]any, 0, len(roots)+2)
	for _, root := range roots {
		args = append(args, root)
	}
	args = append(args, lastID, int64(limit))

	rows, err := r.client.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, err
	}
	return mapImagesFromD1(rows)
}

func (r *d1ImageRepository) Count() (int64, error) {
	return r.client.QueryCount(context.Background(), `SELECT COUNT(*) as cnt FROM images`)
}

func validImageSortColumn(sortBy string) string {
	valid := map[string]string{
		"created_at": "created_at",
		"filename":   "filename",
		"file_size":  "file_size",
		"id":         "id",
	}
	if col, ok := valid[sortBy]; ok {
		return col
	}
	return "id"
}

func (r *d1ImageRepository) SaveImage(image *domain.Image) (bool, error) {
	id, err := r.client.ExecReturningID(context.Background(), `
		INSERT OR IGNORE INTO images
		(path, filename, source_root, file_size, width, height, format, phash, phash_hex, sha256, source_mtime_unix, thumbnail_small_url, thumbnail_large_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, image.Path, image.Filename, image.SourceRoot, image.FileSize, image.Width, image.Height, image.Format, image.PHash, image.PHashHex, image.SHA256, image.SourceMTimeUnix, image.ThumbnailSmallUrl, image.ThumbnailLargeUrl, image.CreatedAt, image.UpdatedAt)
	if err != nil {
		return false, err
	}
	if id > 0 {
		image.ID = id
		return true, nil
	}
	existing, err := r.FindByPath(image.Path)
	if err != nil {
		return false, err
	}
	image.ID = existing.ID
	return false, nil
}

func (r *d1ImageRepository) UpdateImagePHashHex(imageID int64, phashHex string) error {
	return r.client.Exec(context.Background(), `
		UPDATE images SET phash_hex = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, phashHex, imageID)
}

func (r *d1ImageRepository) UpdateImageDuplicateHashCache(imageID int64, sha256, phashHex string, sourceMTimeUnix int64) error {
	return r.client.Exec(context.Background(), `
		UPDATE images SET sha256 = ?, phash_hex = ?, source_mtime_unix = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, sha256, phashHex, sourceMTimeUnix, imageID)
}

func (r *d1ImageRepository) UpdateThumbnails(id int64, smallURL, largeURL string) error {
	return r.client.Exec(context.Background(), `
		UPDATE images SET thumbnail_small_url = ?, thumbnail_large_url = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, smallURL, largeURL, id)
}

func (r *d1ImageRepository) Delete(id int64) error {
	return r.client.Exec(context.Background(), `DELETE FROM images WHERE id = ?`, id)
}

func (r *d1ImageRepository) FindByTagIDs(ctx context.Context, tagIDs []int64, limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
	if len(tagIDs) == 0 {
		return []domain.Image{}, nil
	}
	if r.tagRepo == nil {
		return nil, fmt.Errorf("d1ImageRepository: tagRepo not configured for gallery queries")
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

func (r *d1ImageRepository) CountByTagIDs(ctx context.Context, tagIDs []int64) (int64, error) {
	if len(tagIDs) == 0 {
		return 0, nil
	}
	if r.tagRepo == nil {
		return 0, fmt.Errorf("d1ImageRepository: tagRepo not configured for gallery queries")
	}
	clauses, err := expandHierarchicalTagClausesD1(ctx, r.tagRepo, tagIDs)
	if err != nil {
		return 0, err
	}
	whereClause, args := buildImageTagClauseFilters(clauses, "i.id")
	sql := fmt.Sprintf(`SELECT COUNT(DISTINCT i.id) as cnt FROM images i WHERE %s`, whereClause)
	return r.client.QueryCount(ctx, sql, args...)
}

func (r *d1ImageRepository) FindUntagged(ctx context.Context, limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
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

func (r *d1ImageRepository) CountUntagged(ctx context.Context) (int64, error) {
	return r.client.QueryCount(ctx, `
		SELECT COUNT(i.id) as cnt
		FROM images i
		WHERE NOT EXISTS (
			SELECT 1 FROM image_tags it WHERE it.image_id = i.id AND it.review_state != ?
		)
	`, domain.ReviewStateRejected)
}

func (r *d1ImageRepository) FindPendingTags(ctx context.Context, limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
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

func (r *d1ImageRepository) CountPendingTags(ctx context.Context) (int64, error) {
	return r.client.QueryCount(ctx, `
		SELECT COUNT(i.id) as cnt
		FROM images i
		WHERE EXISTS (
			SELECT 1 FROM image_tags it WHERE it.image_id = i.id AND it.review_state = ?
		)
	`, domain.ReviewStatePending)
}

func (r *d1ImageRepository) FindPendingTagsByTagIDs(ctx context.Context, tagIDs []int64, limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
	if len(tagIDs) == 0 {
		return []domain.Image{}, nil
	}
	if r.tagRepo == nil {
		return nil, fmt.Errorf("d1ImageRepository: tagRepo not configured for gallery queries")
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
		WHERE EXISTS (
			SELECT 1 FROM image_tags it WHERE it.image_id = i.id AND it.review_state = ?
		) AND %s
		ORDER BY i.%s %s, i.id %s
		LIMIT ? OFFSET ?
	`, whereClause, sortColumn, sortDir, sortDir)
	args = append([]any{domain.ReviewStatePending}, args...)
	args = append(args, int64(limit), int64(offset))
	rows, err := r.client.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return mapImagesFromD1(rows)
}

func (r *d1ImageRepository) CountPendingTagsByTagIDs(ctx context.Context, tagIDs []int64) (int64, error) {
	if len(tagIDs) == 0 {
		return 0, nil
	}
	if r.tagRepo == nil {
		return 0, fmt.Errorf("d1ImageRepository: tagRepo not configured for gallery queries")
	}
	clauses, err := expandHierarchicalTagClausesD1(ctx, r.tagRepo, tagIDs)
	if err != nil {
		return 0, err
	}
	whereClause, args := buildImageTagClauseFilters(clauses, "i.id")
	sql := fmt.Sprintf(`
		SELECT COUNT(DISTINCT i.id) as cnt
		FROM images i
		WHERE EXISTS (
			SELECT 1 FROM image_tags it WHERE it.image_id = i.id AND it.review_state = ?
		) AND %s
	`, whereClause)
	args = append([]any{domain.ReviewStatePending}, args...)
	return r.client.QueryCount(ctx, sql, args...)
}

func (r *d1ImageRepository) FindByGalleryFilter(ctx context.Context, exactTagIDs, subtreeRootTagIDs []int64, limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
	if len(exactTagIDs) == 0 && len(subtreeRootTagIDs) == 0 {
		return []domain.Image{}, nil
	}
	if r.tagRepo == nil {
		return nil, fmt.Errorf("d1ImageRepository: tagRepo not configured for gallery queries")
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

func (r *d1ImageRepository) CountByGalleryFilter(ctx context.Context, exactTagIDs, subtreeRootTagIDs []int64) (int64, error) {
	if len(exactTagIDs) == 0 && len(subtreeRootTagIDs) == 0 {
		return 0, nil
	}
	if r.tagRepo == nil {
		return 0, fmt.Errorf("d1ImageRepository: tagRepo not configured for gallery queries")
	}

	whereClause, args, err := buildGalleryFilterClausesD1(ctx, r.tagRepo, exactTagIDs, subtreeRootTagIDs, "i.id")
	if err != nil {
		return 0, err
	}
	sql := fmt.Sprintf(`SELECT COUNT(DISTINCT i.id) as cnt FROM images i WHERE %s`, whereClause)
	return r.client.QueryCount(ctx, sql, args...)
}

func (r *d1ImageRepository) FindImagesWithoutAITags(ctx context.Context, limit int) ([]domain.Image, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.client.Query(ctx, `
		SELECT i.id, ci.collection_id, i.path, i.filename, i.source_root,
		       i.file_size, i.width, i.height, i.format,
		       COALESCE(i.phash, 0) as phash, COALESCE(i.phash_hex, '') as phash_hex,
		       COALESCE(i.sha256, '') as sha256, COALESCE(i.source_mtime_unix, 0) as source_mtime_unix,
		       i.thumbnail_small_url, i.thumbnail_large_url, i.created_at, i.updated_at
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		WHERE i.thumbnail_small_url IS NOT NULL
		  AND i.thumbnail_small_url != ''
		  AND NOT EXISTS (
			  SELECT 1 FROM image_tags it WHERE it.image_id = i.id AND it.review_state != ?
		  )
		  AND NOT EXISTS (
			  SELECT 1 FROM image_tags it WHERE it.image_id = i.id AND it.source = ? AND it.review_state != ?
		  )
		  AND NOT EXISTS (
			  SELECT 1
			  FROM platform_tasks pt
			  WHERE pt.image_id = i.id
			    AND pt.task_type = ?
			    AND pt.status IN (?, ?, ?)
		  )
		ORDER BY i.id ASC
		LIMIT ?
	`, domain.ReviewStateRejected, domain.ImageTagSourceAI, domain.ReviewStateRejected, domain.PlatformTaskTypeAITagGeneration, domain.PlatformTaskStatusPending, domain.PlatformTaskStatusQueued, domain.PlatformTaskStatusRunning, int64(limit))
	if err != nil {
		return nil, err
	}
	return mapImagesFromD1(rows)
}

func (r *d1ImageRepository) FindBackfillCandidates(ctx context.Context, filter BackfillCandidateFilter) ([]domain.Image, error) {
	where, args, err := r.buildBackfillWhere(ctx, filter, d1BackfillEligible)
	if err != nil {
		return nil, err
	}
	orderBy := "i.id ASC"
	if strings.EqualFold(filter.SortDir, "desc") {
		orderBy = "i.id DESC"
	}
	if filter.SortBy != "" && filter.SortBy != "id" {
		switch filter.SortBy {
		case "created_at":
			orderBy = "i.created_at " + d1SortDir(filter.SortDir) + ", i.id " + d1SortDir(filter.SortDir)
		case "filename":
			orderBy = "i.filename " + d1SortDir(filter.SortDir) + ", i.id " + d1SortDir(filter.SortDir)
		case "file_size":
			orderBy = "i.file_size " + d1SortDir(filter.SortDir) + ", i.id " + d1SortDir(filter.SortDir)
		}
	}
	query := fmt.Sprintf(`
		SELECT i.id, ci.collection_id, i.path, i.filename, i.source_root,
		       i.file_size, i.width, i.height, i.format,
		       COALESCE(i.phash, 0) as phash, COALESCE(i.phash_hex, '') as phash_hex,
		       COALESCE(i.sha256, '') as sha256, COALESCE(i.source_mtime_unix, 0) as source_mtime_unix,
		       i.thumbnail_small_url, i.thumbnail_large_url, i.created_at, i.updated_at
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		WHERE 1=1
		%s
		ORDER BY %s
	`, where, orderBy)
	rows, err := r.client.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return mapImagesFromD1(rows)
}

func (r *d1ImageRepository) CountBackfillCandidates(ctx context.Context, filter BackfillCandidateFilter) (int64, error) {
	return r.countBackfill(ctx, filter, d1BackfillEligible)
}

func (r *d1ImageRepository) CountBackfillSkippedWithAITag(ctx context.Context, filter BackfillCandidateFilter) (int64, error) {
	return r.countBackfill(ctx, filter, d1BackfillSkippedWithAITag)
}

func (r *d1ImageRepository) CountBackfillSkippedWithActiveTask(ctx context.Context, filter BackfillCandidateFilter) (int64, error) {
	return r.countBackfill(ctx, filter, d1BackfillSkippedWithActiveTask)
}

func (r *d1ImageRepository) CountBackfillHitCount(ctx context.Context, filter BackfillCandidateFilter) (int64, error) {
	return r.countBackfill(ctx, filter, d1BackfillHit)
}

func (r *d1ImageRepository) FindBySourceDirsAndTag(ctx context.Context, sourceDirs []string, tagID int64, limit, offset int) ([]domain.Image, error) {
	if tagID <= 0 {
		return []domain.Image{}, nil
	}
	if limit <= 0 {
		limit = 1000
	}
	clauses := make([]string, 0, len(sourceDirs))
	args := make([]any, 0, len(sourceDirs)*3+4)
	for _, dir := range sourceDirs {
		trimmed := strings.TrimSpace(dir)
		if trimmed == "" {
			continue
		}
		trimmed = strings.TrimRight(trimmed, `/\`)
		if trimmed == "" {
			continue
		}
		clauses = append(clauses, `(i.path = ? OR i.path LIKE ? OR i.path LIKE ?)`)
		args = append(args, trimmed, trimmed+`\%`, trimmed+`/%`)
	}
	if len(clauses) == 0 {
		return []domain.Image{}, nil
	}
	whereClause := strings.Join(clauses, " OR ")
	sql := fmt.Sprintf(`
		SELECT i.id, ci.collection_id, i.path, i.filename, i.source_root,
		       i.file_size, i.width, i.height, i.format,
		       COALESCE(i.phash, 0) as phash, COALESCE(i.phash_hex, '') as phash_hex,
		       COALESCE(i.sha256, '') as sha256, COALESCE(i.source_mtime_unix, 0) as source_mtime_unix,
		       i.thumbnail_small_url, i.thumbnail_large_url, i.created_at, i.updated_at
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		JOIN image_tags it ON it.image_id = i.id
		WHERE it.tag_id = ?
		  AND it.review_state != ?
		  AND (%s)
		ORDER BY i.id ASC
		LIMIT ? OFFSET ?
	`, whereClause)
	queryArgs := make([]any, 0, len(args)+4)
	queryArgs = append(queryArgs, tagID, domain.ReviewStateRejected)
	queryArgs = append(queryArgs, args...)
	queryArgs = append(queryArgs, int64(limit), int64(offset))
	rows, err := r.client.Query(ctx, sql, queryArgs...)
	if err != nil {
		return nil, err
	}
	return mapImagesFromD1(rows)
}

func (r *d1ImageRepository) CountBySourceDirsAndTag(ctx context.Context, sourceDirs []string, tagID int64) (int64, error) {
	if tagID <= 0 {
		return 0, nil
	}
	clauses := make([]string, 0, len(sourceDirs))
	args := make([]any, 0, len(sourceDirs)*3)
	for _, dir := range sourceDirs {
		trimmed := strings.TrimSpace(dir)
		if trimmed == "" {
			continue
		}
		trimmed = strings.TrimRight(trimmed, `/\`)
		if trimmed == "" {
			continue
		}
		clauses = append(clauses, `(i.path = ? OR i.path LIKE ? OR i.path LIKE ?)`)
		args = append(args, trimmed, trimmed+`\%`, trimmed+`/%`)
	}
	if len(clauses) == 0 {
		return 0, nil
	}
	whereClause := strings.Join(clauses, " OR ")
	sql := fmt.Sprintf(`
		SELECT COUNT(DISTINCT i.id) as cnt
		FROM images i
		JOIN image_tags it ON it.image_id = i.id
		WHERE it.tag_id = ?
		  AND it.review_state != ?
		  AND (%s)
	`, whereClause)
	queryArgs := make([]any, 0, len(args)+2)
	queryArgs = append(queryArgs, tagID, domain.ReviewStateRejected)
	queryArgs = append(queryArgs, args...)
	return r.client.QueryCount(ctx, sql, queryArgs...)
}

func (r *d1ImageRepository) UpdateImageLocation(ctx context.Context, imageID int64, path, filename, sourceRoot string) error {
	return r.client.Exec(ctx, `
		UPDATE images SET path = ?, filename = ?, source_root = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, path, filename, sourceRoot, imageID)
}

func mapImagesFromD1(rows []map[string]any) ([]domain.Image, error) {
	images := make([]domain.Image, 0, len(rows))
	for _, row := range rows {
		img, err := mapImageFromD1(row)
		if err != nil {
			return nil, err
		}
		images = append(images, *img)
	}
	return images, nil
}

type d1BackfillMode int

const (
	d1BackfillHit d1BackfillMode = iota
	d1BackfillEligible
	d1BackfillSkippedWithAITag
	d1BackfillSkippedWithActiveTask
)

func (r *d1ImageRepository) countBackfill(ctx context.Context, filter BackfillCandidateFilter, mode d1BackfillMode) (int64, error) {
	where, args, err := r.buildBackfillWhere(ctx, filter, mode)
	if err != nil {
		return 0, err
	}
	query := fmt.Sprintf(`
		SELECT COUNT(i.id) as cnt
		FROM images i
		WHERE 1=1
		%s
	`, where)
	return r.client.QueryCount(ctx, query, args...)
}

func (r *d1ImageRepository) buildBackfillWhere(ctx context.Context, filter BackfillCandidateFilter, mode d1BackfillMode) (string, []any, error) {
	var conds []string
	args := make([]any, 0)

	if len(filter.TagIDs) > 0 {
		if r.tagRepo == nil {
			return "", nil, fmt.Errorf("d1ImageRepository: tagRepo not configured for backfill queries")
		}
		clauses, err := expandHierarchicalTagClausesD1(ctx, r.tagRepo, filter.TagIDs)
		if err != nil {
			return "", nil, err
		}
		cond, clauseArgs := buildImageTagClauseFilters(clauses, "i.id")
		conds = append(conds, cond)
		args = append(args, clauseArgs...)
	}

	if filter.HasTags != nil {
		if *filter.HasTags {
			conds = append(conds, `EXISTS (SELECT 1 FROM image_tags it2 WHERE it2.image_id = i.id AND it2.review_state != ?)`)
		} else {
			conds = append(conds, `NOT EXISTS (SELECT 1 FROM image_tags it2 WHERE it2.image_id = i.id AND it2.review_state != ?)`)
		}
		args = append(args, domain.ReviewStateRejected)
	}

	aiExists := `EXISTS (
		SELECT 1 FROM image_tags it3
		WHERE it3.image_id = i.id
		  AND it3.source = ?
		  AND it3.review_state != ?
	)`
	activeTaskExists := `EXISTS (
		SELECT 1 FROM platform_tasks pt
		WHERE pt.image_id = i.id
		  AND pt.task_type = ?
		  AND pt.status IN (?, ?, ?)
	)`
	switch mode {
	case d1BackfillEligible:
		conds = append(conds, "NOT "+aiExists, "NOT "+activeTaskExists)
		args = append(args, domain.ImageTagSourceAI, domain.ReviewStateRejected)
		args = append(args, domain.PlatformTaskTypeAITagGeneration, domain.PlatformTaskStatusPending, domain.PlatformTaskStatusQueued, domain.PlatformTaskStatusRunning)
	case d1BackfillSkippedWithAITag:
		conds = append(conds, aiExists)
		args = append(args, domain.ImageTagSourceAI, domain.ReviewStateRejected)
	case d1BackfillSkippedWithActiveTask:
		conds = append(conds, activeTaskExists)
		args = append(args, domain.PlatformTaskTypeAITagGeneration, domain.PlatformTaskStatusPending, domain.PlatformTaskStatusQueued, domain.PlatformTaskStatusRunning)
	}

	if len(conds) == 0 {
		return "", args, nil
	}
	return " AND " + strings.Join(conds, " AND "), args, nil
}

func d1SortDir(raw string) string {
	if strings.EqualFold(raw, "asc") {
		return "asc"
	}
	return "desc"
}
