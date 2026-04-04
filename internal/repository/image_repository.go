package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

// BackfillCandidateFilter defines filter criteria for backfill candidate queries.
type BackfillCandidateFilter struct {
	TagIDs  []int64
	HasTags *bool // false = untagged only; nil = no has_tags filter
	SortBy  string
	SortDir string
}

// BackfillCandidateStats holds skip-reason counts for a backfill preview.
type BackfillCandidateStats struct {
	HitCount              int64
	EnqueueableCount      int64
	SkippedWithAITag      int64
	SkippedWithActiveTask int64
}

type ImageRepository interface {
	// SaveImage saves an image to the database.
	// Returns (isNew, error) where isNew is true if a new record was inserted,
	// false if the image already existed (INSERT OR IGNORE took effect).
	SaveImage(image *domain.Image) (isNew bool, err error)
	FindByID(id int64) (*domain.Image, error)
	FindByPath(path string) (*domain.Image, error)
	FindAll(limit, offset int, sortBy, sortDir string) ([]domain.Image, error)
	FindByTagIDs(ctx context.Context, tagIDs []int64, limit, offset int, sortBy, sortDir string) ([]domain.Image, error)
	CountByTagIDs(ctx context.Context, tagIDs []int64) (int64, error)
	FindUntagged(ctx context.Context, limit, offset int, sortBy, sortDir string) ([]domain.Image, error)
	CountUntagged(ctx context.Context) (int64, error)
	FindImagesWithoutAITags(ctx context.Context, limit int) ([]domain.Image, error)
	// FindBackfillCandidates returns images matching the filter that are eligible for AI backfill:
	// no AI source tags and no active (pending/queued/running) AI tasks.
	FindBackfillCandidates(ctx context.Context, filter BackfillCandidateFilter) ([]domain.Image, error)
	// CountBackfillCandidates returns the count of images matching the filter that are eligible for AI backfill.
	CountBackfillCandidates(ctx context.Context, filter BackfillCandidateFilter) (int64, error)
	// CountBackfillSkippedWithAITag returns the count of images matching the filter that already have AI source tags.
	CountBackfillSkippedWithAITag(ctx context.Context, filter BackfillCandidateFilter) (int64, error)
	// CountBackfillSkippedWithActiveTask returns the count of images matching the filter that already have active AI tasks.
	CountBackfillSkippedWithActiveTask(ctx context.Context, filter BackfillCandidateFilter) (int64, error)
	// CountBackfillHitCount returns the total count of images matching the filter (before any skip classification).
	CountBackfillHitCount(ctx context.Context, filter BackfillCandidateFilter) (int64, error)
	UpdateImagePHashHex(imageID int64, phashHex string) error
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

// SaveImage saves an image to the database using INSERT OR IGNORE.
// Returns (isNew, error) where isNew is true if a new record was inserted,
// false if the image already existed (path conflict, INSERT OR IGNORE took effect).
func (r *sqliteImageRepository) SaveImage(image *domain.Image) (bool, error) {
	result, err := r.db.Exec(`
		INSERT OR IGNORE INTO images
		(path, filename, source_root, file_size, width, height, format, phash, phash_hex, thumbnail_small_url, thumbnail_large_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, image.Path, image.Filename, image.SourceRoot, image.FileSize, image.Width, image.Height, image.Format, image.PHash, image.PHashHex, image.ThumbnailSmallUrl, image.ThumbnailLargeUrl, image.CreatedAt, image.UpdatedAt)
	if err != nil {
		return false, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return false, err
	}
	if id > 0 {
		image.ID = id
		return true, nil // 新插入的记录
	}

	// INSERT OR IGNORE took effect - record already exists
	existing, err := r.FindByPath(image.Path)
	if err != nil {
		return false, err
	}
	image.ID = existing.ID
	return false, nil // 已存在的记录
}

func (r *sqliteImageRepository) FindByPath(path string) (*domain.Image, error) {
	return r.queryOne(`
		SELECT id, path, filename, source_root, file_size, width, height, format, COALESCE(phash, 0), COALESCE(phash_hex, ''), thumbnail_small_url, thumbnail_large_url, created_at, updated_at
		FROM images WHERE path = ?
	`, path)
}

func (r *sqliteImageRepository) FindByID(id int64) (*domain.Image, error) {
	return r.queryOne(`
		SELECT id, path, filename, source_root, file_size, width, height, format, COALESCE(phash, 0), COALESCE(phash_hex, ''), thumbnail_small_url, thumbnail_large_url, created_at, updated_at
		FROM images WHERE id = ?
	`, id)
}

func (r *sqliteImageRepository) queryOne(query string, args ...any) (*domain.Image, error) {
	image := &domain.Image{}
	var thumbnailSmallUrl, thumbnailLargeUrl sql.NullString
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
		&image.PHashHex,
		&thumbnailSmallUrl,
		&thumbnailLargeUrl,
		&image.CreatedAt,
		&image.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}

	// Convert NullString to string (empty if NULL)
	if thumbnailSmallUrl.Valid {
		image.ThumbnailSmallUrl = thumbnailSmallUrl.String
	}
	if thumbnailLargeUrl.Valid {
		image.ThumbnailLargeUrl = thumbnailLargeUrl.String
	}

	return image, nil
}

func (r *sqliteImageRepository) FindAll(limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
	// 验证并设置默认排序字段
	validSortFields := map[string]string{
		"created_at": "created_at",
		"filename":   "filename",
		"file_size":  "file_size",
		"id":         "id",
	}

	sortColumn := validSortFields[sortBy]
	if sortColumn == "" {
		sortColumn = "id"
	}

	// 验证排序方向
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}

	query := fmt.Sprintf(`
		SELECT id, path, filename, source_root, file_size, width, height, format, COALESCE(phash, 0), COALESCE(phash_hex, ''), thumbnail_small_url, thumbnail_large_url, created_at, updated_at
		FROM images ORDER BY %s %s LIMIT ? OFFSET ?
	`, sortColumn, sortDir)

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := make([]domain.Image, 0)
	for rows.Next() {
		var image domain.Image
		var thumbnailSmallUrl, thumbnailLargeUrl sql.NullString
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
			&thumbnailSmallUrl,
			&thumbnailLargeUrl,
			&image.CreatedAt,
			&image.UpdatedAt,
		); err != nil {
			return nil, err
		}
		// Convert NullString to string (empty if NULL)
		if thumbnailSmallUrl.Valid {
			image.ThumbnailSmallUrl = thumbnailSmallUrl.String
		}
		if thumbnailLargeUrl.Valid {
			image.ThumbnailLargeUrl = thumbnailLargeUrl.String
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
func (r *sqliteImageRepository) FindByTagIDs(ctx context.Context, tagIDs []int64, limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
	if len(tagIDs) == 0 {
		return []domain.Image{}, nil
	}

	// 验证并设置默认排序字段
	validSortFields := map[string]string{
		"created_at": "i.created_at",
		"filename":   "i.filename",
		"file_size":  "i.file_size",
		"id":         "i.id",
	}

	sortColumn := validSortFields[sortBy]
	if sortColumn == "" {
		sortColumn = "i.id"
	}

	// 验证排序方向
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
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
		SELECT i.id, i.path, i.filename, i.source_root, i.file_size, i.width, i.height, i.format, COALESCE(i.phash, 0), COALESCE(i.phash_hex, ''), i.thumbnail_small_url, i.thumbnail_large_url, i.created_at, i.updated_at
		FROM images i
		INNER JOIN image_tags it ON it.image_id = i.id
		WHERE it.tag_id IN (%s) AND it.review_state != 'rejected'
		GROUP BY i.id
		HAVING COUNT(DISTINCT it.tag_id) = ?
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, strings.Join(placeholders, ", "), sortColumn, sortDir)

	args = append(args, int64(len(tagIDs)), int64(limit), int64(offset))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := make([]domain.Image, 0)
	for rows.Next() {
		var image domain.Image
		var thumbnailSmallUrl, thumbnailLargeUrl sql.NullString
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
			&thumbnailSmallUrl,
			&thumbnailLargeUrl,
			&image.CreatedAt,
			&image.UpdatedAt,
		); err != nil {
			return nil, err
		}
		// Convert NullString to string (empty if NULL)
		if thumbnailSmallUrl.Valid {
			image.ThumbnailSmallUrl = thumbnailSmallUrl.String
		}
		if thumbnailLargeUrl.Valid {
			image.ThumbnailLargeUrl = thumbnailLargeUrl.String
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
			WHERE it.tag_id IN (%s) AND it.review_state != 'rejected'
			GROUP BY it.image_id
			HAVING COUNT(DISTINCT it.tag_id) = ?
		) sub
	`, strings.Join(placeholders, ", "))

	args = append(args, int64(len(tagIDs)))

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

// FindUntagged returns images that have no tags associated with them.
// Uses LEFT JOIN to find images without any corresponding image_tags entries.
func (r *sqliteImageRepository) FindUntagged(ctx context.Context, limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
	// 验证并设置默认排序字段
	validSortFields := map[string]string{
		"created_at": "i.created_at",
		"filename":   "i.filename",
		"file_size":  "i.file_size",
		"id":         "i.id",
	}

	sortColumn := validSortFields[sortBy]
	if sortColumn == "" {
		sortColumn = "i.id"
	}

	// 验证排序方向
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}

	query := fmt.Sprintf(`
		SELECT i.id, i.path, i.filename, i.source_root, i.file_size, i.width, i.height, i.format, COALESCE(i.phash, 0), COALESCE(i.phash_hex, ''), i.thumbnail_small_url, i.thumbnail_large_url, i.created_at, i.updated_at
		FROM images i
		LEFT JOIN image_tags it ON it.image_id = i.id
		WHERE it.image_id IS NULL
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, sortColumn, sortDir)

	rows, err := r.db.QueryContext(ctx, query, int64(limit), int64(offset))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := make([]domain.Image, 0)
	for rows.Next() {
		var image domain.Image
		var thumbnailSmallUrl, thumbnailLargeUrl sql.NullString
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
			&thumbnailSmallUrl,
			&thumbnailLargeUrl,
			&image.CreatedAt,
			&image.UpdatedAt,
		); err != nil {
			return nil, err
		}
		// Convert NullString to string (empty if NULL)
		if thumbnailSmallUrl.Valid {
			image.ThumbnailSmallUrl = thumbnailSmallUrl.String
		}
		if thumbnailLargeUrl.Valid {
			image.ThumbnailLargeUrl = thumbnailLargeUrl.String
		}
		images = append(images, image)
	}

	return images, rows.Err()
}

// CountUntagged returns the count of images that have no tags associated with them.
func (r *sqliteImageRepository) CountUntagged(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(i.id)
		FROM images i
		LEFT JOIN image_tags it ON it.image_id = i.id
		WHERE it.image_id IS NULL
	`).Scan(&count)
	return count, err
}

func (r *sqliteImageRepository) FindImagesWithoutAITags(ctx context.Context, limit int) ([]domain.Image, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT i.id, i.path, i.filename, i.source_root, i.file_size, i.width, i.height, i.format, COALESCE(i.phash, 0), COALESCE(i.phash_hex, ''), i.thumbnail_small_url, i.thumbnail_large_url, i.created_at, i.updated_at
		FROM images i
		WHERE i.thumbnail_small_url IS NOT NULL
		  AND i.thumbnail_small_url != ''
		  AND NOT EXISTS (
			  SELECT 1
			  FROM image_tags it
			  WHERE it.image_id = i.id
			    AND it.source = 'ai'
		  )
		  AND NOT EXISTS (
			  SELECT 1
			  FROM platform_tasks pt
			  WHERE pt.image_id = i.id
			    AND pt.task_type = 'ai_tag_generation'
			    AND pt.status IN ('pending', 'queued', 'running')
		  )
		ORDER BY i.id ASC
		LIMIT ?
	`, int64(limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := make([]domain.Image, 0)
	for rows.Next() {
		var image domain.Image
		var thumbnailSmallURL sql.NullString
		var thumbnailLargeURL sql.NullString
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

// Delete removes an image by ID. Due to ON DELETE CASCADE in the schema,
// this will also remove related records in image_tags and collection_images.
func (r *sqliteImageRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM images WHERE id = ?`, id)
	return err
}

func (r *sqliteImageRepository) UpdateImagePHashHex(imageID int64, phashHex string) error {
	_, err := r.db.Exec(`UPDATE images SET phash_hex = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, phashHex, imageID)
	return err
}

// buildBackfillBaseWhere constructs the WHERE clause fragment and args for the base filter.
// It handles TagIDs (AND semantics) and HasTags filtering.
func buildBackfillBaseWhere(filter BackfillCandidateFilter) (string, []any) {
	var conds []string
	var args []any

	if len(filter.TagIDs) > 0 {
		placeholders := make([]string, len(filter.TagIDs))
		for i, id := range filter.TagIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		cond := fmt.Sprintf(`i.id IN (
			SELECT it.image_id FROM image_tags it
			WHERE it.tag_id IN (%s) AND it.review_state != 'rejected'
			GROUP BY it.image_id
			HAVING COUNT(DISTINCT it.tag_id) = ?
		)`, strings.Join(placeholders, ", "))
		args = append(args, int64(len(filter.TagIDs)))
		conds = append(conds, cond)
	}

	if filter.HasTags != nil {
		if !*filter.HasTags {
			conds = append(conds, `NOT EXISTS (SELECT 1 FROM image_tags it2 WHERE it2.image_id = i.id)`)
		} else {
			conds = append(conds, `EXISTS (SELECT 1 FROM image_tags it2 WHERE it2.image_id = i.id)`)
		}
	}

	if len(conds) > 0 {
		return " AND " + strings.Join(conds, " AND "), args
	}
	return "", args
}

// FindBackfillCandidates returns images matching the filter that are eligible for AI backfill:
// no AI source tags and no active (pending/queued/running) AI tasks.
func (r *sqliteImageRepository) FindBackfillCandidates(ctx context.Context, filter BackfillCandidateFilter) ([]domain.Image, error) {
	baseWhere, baseArgs := buildBackfillBaseWhere(filter)

	query := fmt.Sprintf(`
		SELECT i.id, i.path, i.filename, i.source_root, i.file_size, i.width, i.height, i.format,
		       COALESCE(i.phash, 0), COALESCE(i.phash_hex, ''), i.thumbnail_small_url, i.thumbnail_large_url, i.created_at, i.updated_at
		FROM images i
		WHERE 1=1
		%s
		AND NOT EXISTS (
			SELECT 1 FROM image_tags it3 WHERE it3.image_id = i.id AND it3.source = 'ai'
		)
		AND NOT EXISTS (
			SELECT 1 FROM platform_tasks pt WHERE pt.image_id = i.id
			AND pt.task_type = 'ai_tag_generation'
			AND pt.status IN ('pending', 'queued', 'running')
		)
		ORDER BY i.id ASC
	`, baseWhere)

	rows, err := r.db.QueryContext(ctx, query, baseArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := make([]domain.Image, 0)
	for rows.Next() {
		var image domain.Image
		var thumbnailSmallURL, thumbnailLargeURL sql.NullString
		if err := rows.Scan(
			&image.ID, &image.Path, &image.Filename, &image.SourceRoot,
			&image.FileSize, &image.Width, &image.Height, &image.Format,
			&image.PHash, &image.PHashHex, &thumbnailSmallURL, &thumbnailLargeURL,
			&image.CreatedAt, &image.UpdatedAt,
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

// CountBackfillCandidates returns the count of images matching the filter that are eligible for AI backfill.
func (r *sqliteImageRepository) CountBackfillCandidates(ctx context.Context, filter BackfillCandidateFilter) (int64, error) {
	baseWhere, baseArgs := buildBackfillBaseWhere(filter)

	query := fmt.Sprintf(`
		SELECT COUNT(i.id)
		FROM images i
		WHERE 1=1
		%s
		AND NOT EXISTS (
			SELECT 1 FROM image_tags it3 WHERE it3.image_id = i.id AND it3.source = 'ai'
		)
		AND NOT EXISTS (
			SELECT 1 FROM platform_tasks pt WHERE pt.image_id = i.id
			AND pt.task_type = 'ai_tag_generation'
			AND pt.status IN ('pending', 'queued', 'running')
		)
	`, baseWhere)

	var count int64
	err := r.db.QueryRowContext(ctx, query, baseArgs...).Scan(&count)
	return count, err
}

// CountBackfillSkippedWithAITag returns the count of images matching the filter that already have AI source tags.
func (r *sqliteImageRepository) CountBackfillSkippedWithAITag(ctx context.Context, filter BackfillCandidateFilter) (int64, error) {
	baseWhere, baseArgs := buildBackfillBaseWhere(filter)

	query := fmt.Sprintf(`
		SELECT COUNT(i.id)
		FROM images i
		WHERE 1=1
		%s
		AND EXISTS (
			SELECT 1 FROM image_tags it3 WHERE it3.image_id = i.id AND it3.source = 'ai'
		)
	`, baseWhere)

	var count int64
	err := r.db.QueryRowContext(ctx, query, baseArgs...).Scan(&count)
	return count, err
}

// CountBackfillSkippedWithActiveTask returns the count of images matching the filter that already have active AI tasks.
func (r *sqliteImageRepository) CountBackfillSkippedWithActiveTask(ctx context.Context, filter BackfillCandidateFilter) (int64, error) {
	baseWhere, baseArgs := buildBackfillBaseWhere(filter)

	query := fmt.Sprintf(`
		SELECT COUNT(i.id)
		FROM images i
		WHERE 1=1
		%s
		AND EXISTS (
			SELECT 1 FROM platform_tasks pt WHERE pt.image_id = i.id
			AND pt.task_type = 'ai_tag_generation'
			AND pt.status IN ('pending', 'queued', 'running')
		)
	`, baseWhere)

	var count int64
	err := r.db.QueryRowContext(ctx, query, baseArgs...).Scan(&count)
	return count, err
}

// CountBackfillHitCount returns the total count of images matching the filter (before any skip classification).
func (r *sqliteImageRepository) CountBackfillHitCount(ctx context.Context, filter BackfillCandidateFilter) (int64, error) {
	baseWhere, baseArgs := buildBackfillBaseWhere(filter)

	query := fmt.Sprintf(`
		SELECT COUNT(i.id)
		FROM images i
		WHERE 1=1
		%s
	`, baseWhere)

	var count int64
	err := r.db.QueryRowContext(ctx, query, baseArgs...).Scan(&count)
	return count, err
}
