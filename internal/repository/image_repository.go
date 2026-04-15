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
	FindByIDRange(limit int, lastID int64) ([]domain.Image, error)
	FindBySourceRootsAfterID(limit int, lastID int64, sourceRoots []string) ([]domain.Image, error)
	FindByTagIDs(ctx context.Context, tagIDs []int64, limit, offset int, sortBy, sortDir string) ([]domain.Image, error)
	CountByTagIDs(ctx context.Context, tagIDs []int64) (int64, error)
	FindUntagged(ctx context.Context, limit, offset int, sortBy, sortDir string) ([]domain.Image, error)
	CountUntagged(ctx context.Context) (int64, error)
	FindPendingTags(ctx context.Context, limit, offset int, sortBy, sortDir string) ([]domain.Image, error)
	CountPendingTags(ctx context.Context) (int64, error)
	FindPendingTagsByTagIDs(ctx context.Context, tagIDs []int64, limit, offset int, sortBy, sortDir string) ([]domain.Image, error)
	CountPendingTagsByTagIDs(ctx context.Context, tagIDs []int64) (int64, error)
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
	UpdateImageDuplicateHashCache(imageID int64, sha256, phashHex string, sourceMTimeUnix int64) error
	UpdateThumbnails(id int64, smallURL, largeURL string) error
	Count() (int64, error)
	Delete(id int64) error
}

type sqliteImageRepository struct {
	db *sql.DB
}

const imageSelectColumns = `
	i.id,
	ci.collection_id,
	i.path,
	i.filename,
	i.source_root,
	i.file_size,
	i.width,
	i.height,
	i.format,
	COALESCE(i.phash, 0),
	COALESCE(i.phash_hex, ''),
	COALESCE(i.sha256, ''),
	COALESCE(i.source_mtime_unix, 0),
	i.thumbnail_small_url,
	i.thumbnail_large_url,
	i.created_at,
	i.updated_at
`

func NewImageRepository(db *sql.DB) ImageRepository {
	return &sqliteImageRepository{db: db}
}

func scanImage(scanner interface{ Scan(dest ...any) error }, image *domain.Image) error {
	var (
		collectionID                         sql.NullInt64
		thumbnailSmallURL, thumbnailLargeURL sql.NullString
	)

	err := scanner.Scan(
		&image.ID,
		&collectionID,
		&image.Path,
		&image.Filename,
		&image.SourceRoot,
		&image.FileSize,
		&image.Width,
		&image.Height,
		&image.Format,
		&image.PHash,
		&image.PHashHex,
		&image.SHA256,
		&image.SourceMTimeUnix,
		&thumbnailSmallURL,
		&thumbnailLargeURL,
		&image.CreatedAt,
		&image.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if collectionID.Valid {
		value := collectionID.Int64
		image.CollectionID = &value
	} else {
		image.CollectionID = nil
	}
	if thumbnailSmallURL.Valid {
		image.ThumbnailSmallUrl = thumbnailSmallURL.String
	}
	if thumbnailLargeURL.Valid {
		image.ThumbnailLargeUrl = thumbnailLargeURL.String
	}

	return nil
}

// SaveImage saves an image to the database using INSERT OR IGNORE.
// Returns (isNew, error) where isNew is true if a new record was inserted,
// false if the image already existed (path conflict, INSERT OR IGNORE took effect).
func (r *sqliteImageRepository) SaveImage(image *domain.Image) (bool, error) {
	result, err := r.db.Exec(`
		INSERT OR IGNORE INTO images
		(path, filename, source_root, file_size, width, height, format, phash, phash_hex, sha256, source_mtime_unix, thumbnail_small_url, thumbnail_large_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, image.Path, image.Filename, image.SourceRoot, image.FileSize, image.Width, image.Height, image.Format, image.PHash, image.PHashHex, image.SHA256, image.SourceMTimeUnix, image.ThumbnailSmallUrl, image.ThumbnailLargeUrl, image.CreatedAt, image.UpdatedAt)
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
		SELECT `+imageSelectColumns+`
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		WHERE i.path = ?
	`, path)
}

func (r *sqliteImageRepository) FindByID(id int64) (*domain.Image, error) {
	return r.queryOne(`
		SELECT `+imageSelectColumns+`
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		WHERE i.id = ?
	`, id)
}

func (r *sqliteImageRepository) queryOne(query string, args ...any) (*domain.Image, error) {
	image := &domain.Image{}
	err := scanImage(r.db.QueryRow(query, args...), image)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, err
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
		SELECT %s
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		ORDER BY i.%s %s, i.id %s LIMIT ? OFFSET ?
	`, imageSelectColumns, sortColumn, sortDir, sortDir)

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := make([]domain.Image, 0)
	for rows.Next() {
		var image domain.Image
		if err := scanImage(rows, &image); err != nil {
			return nil, err
		}
		images = append(images, image)
	}

	return images, rows.Err()
}

func (r *sqliteImageRepository) FindByIDRange(limit int, lastID int64) ([]domain.Image, error) {
	if limit <= 0 {
		limit = 1000
	}

	rows, err := r.db.Query(`
		SELECT `+imageSelectColumns+`
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		WHERE i.id > ?
		ORDER BY i.id ASC
		LIMIT ?
	`, lastID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := make([]domain.Image, 0, limit)
	for rows.Next() {
		var image domain.Image
		if err := scanImage(rows, &image); err != nil {
			return nil, err
		}
		images = append(images, image)
	}

	return images, rows.Err()
}

func (r *sqliteImageRepository) FindBySourceRootsAfterID(limit int, lastID int64, sourceRoots []string) ([]domain.Image, error) {
	if limit <= 0 {
		limit = 1000
	}

	roots := make([]string, 0, len(sourceRoots))
	for _, root := range sourceRoots {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			continue
		}
		roots = append(roots, trimmed)
	}
	if len(roots) == 0 {
		return []domain.Image{}, nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(roots)), ",")
	query := fmt.Sprintf(`
		SELECT %s
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		WHERE i.source_root IN (%s)
		  AND i.id > ?
		ORDER BY i.id ASC
		LIMIT ?
	`, imageSelectColumns, placeholders)

	args := make([]any, 0, len(roots)+2)
	for _, root := range roots {
		args = append(args, root)
	}
	args = append(args, lastID, limit)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := make([]domain.Image, 0)
	for rows.Next() {
		var image domain.Image
		if err := scanImage(rows, &image); err != nil {
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
func (r *sqliteImageRepository) FindByTagIDs(ctx context.Context, tagIDs []int64, limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
	if len(tagIDs) == 0 {
		return []domain.Image{}, nil
	}

	clauses, err := expandHierarchicalTagClauses(ctx, r.db, tagIDs)
	if err != nil {
		return nil, err
	}
	whereClause, args := buildImageTagClauseFilters(clauses, "i.id")

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
		SELECT %s
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		WHERE %s
		ORDER BY %s %s, i.id %s
		LIMIT ? OFFSET ?
	`, imageSelectColumns, whereClause, sortColumn, sortDir, sortDir)

	args = append(args, int64(limit), int64(offset))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := make([]domain.Image, 0)
	for rows.Next() {
		var image domain.Image
		if err := scanImage(rows, &image); err != nil {
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

	clauses, err := expandHierarchicalTagClauses(ctx, r.db, tagIDs)
	if err != nil {
		return 0, err
	}
	whereClause, args := buildImageTagClauseFilters(clauses, "i.id")

	query := fmt.Sprintf(`
		SELECT COUNT(DISTINCT i.id)
		FROM images i
		WHERE %s
	`, whereClause)

	var count int64
	err = r.db.QueryRowContext(ctx, query, args...).Scan(&count)
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
		SELECT %s
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		WHERE NOT EXISTS (
			SELECT 1 FROM image_tags it WHERE it.image_id = i.id AND it.review_state != 'rejected'
		)
		ORDER BY %s %s, i.id %s
		LIMIT ? OFFSET ?
	`, imageSelectColumns, sortColumn, sortDir, sortDir)

	rows, err := r.db.QueryContext(ctx, query, int64(limit), int64(offset))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := make([]domain.Image, 0)
	for rows.Next() {
		var image domain.Image
		if err := scanImage(rows, &image); err != nil {
			return nil, err
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
		WHERE NOT EXISTS (
			SELECT 1 FROM image_tags it WHERE it.image_id = i.id AND it.review_state != 'rejected'
		)
	`).Scan(&count)
	return count, err
}

// FindPendingTags returns images that have at least one pending tag association.
func (r *sqliteImageRepository) FindPendingTags(ctx context.Context, limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
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

	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}

	query := fmt.Sprintf(`
		SELECT %s
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		WHERE EXISTS (
			SELECT 1 FROM image_tags it WHERE it.image_id = i.id AND it.review_state = 'pending'
		)
		ORDER BY %s %s, i.id %s
		LIMIT ? OFFSET ?
	`, imageSelectColumns, sortColumn, sortDir, sortDir)

	rows, err := r.db.QueryContext(ctx, query, int64(limit), int64(offset))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := make([]domain.Image, 0)
	for rows.Next() {
		var image domain.Image
		if err := scanImage(rows, &image); err != nil {
			return nil, err
		}
		images = append(images, image)
	}

	return images, rows.Err()
}

// CountPendingTags returns the count of images that have at least one pending tag association.
func (r *sqliteImageRepository) CountPendingTags(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(i.id)
		FROM images i
		WHERE EXISTS (
			SELECT 1 FROM image_tags it WHERE it.image_id = i.id AND it.review_state = 'pending'
		)
	`).Scan(&count)
	return count, err
}

// FindPendingTagsByTagIDs returns images that have at least one pending tag association
// AND have ALL the specified tag IDs (AND semantics).
func (r *sqliteImageRepository) FindPendingTagsByTagIDs(ctx context.Context, tagIDs []int64, limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
	if len(tagIDs) == 0 {
		return []domain.Image{}, nil
	}

	clauses, err := expandHierarchicalTagClauses(ctx, r.db, tagIDs)
	if err != nil {
		return nil, err
	}
	tagWhereClause, args := buildImageTagClauseFilters(clauses, "i.id")

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

	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}

	query := fmt.Sprintf(`
		SELECT %s
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		WHERE EXISTS (
			SELECT 1 FROM image_tags it WHERE it.image_id = i.id AND it.review_state = 'pending'
		)
		AND %s
		ORDER BY %s %s, i.id %s
		LIMIT ? OFFSET ?
	`, imageSelectColumns, tagWhereClause, sortColumn, sortDir, sortDir)

	args = append(args, int64(limit), int64(offset))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := make([]domain.Image, 0)
	for rows.Next() {
		var image domain.Image
		if err := scanImage(rows, &image); err != nil {
			return nil, err
		}
		images = append(images, image)
	}

	return images, rows.Err()
}

// CountPendingTagsByTagIDs returns the count of images that have at least one pending tag association
// AND have ALL the specified tag IDs (AND semantics).
func (r *sqliteImageRepository) CountPendingTagsByTagIDs(ctx context.Context, tagIDs []int64) (int64, error) {
	if len(tagIDs) == 0 {
		return 0, nil
	}

	clauses, err := expandHierarchicalTagClauses(ctx, r.db, tagIDs)
	if err != nil {
		return 0, err
	}
	tagWhereClause, args := buildImageTagClauseFilters(clauses, "i.id")

	var count int64
	query := fmt.Sprintf(`
		SELECT COUNT(i.id)
		FROM images i
		WHERE EXISTS (
			SELECT 1 FROM image_tags it WHERE it.image_id = i.id AND it.review_state = 'pending'
		)
		AND %s
	`, tagWhereClause)
	err = r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

func (r *sqliteImageRepository) FindImagesWithoutAITags(ctx context.Context, limit int) ([]domain.Image, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT `+imageSelectColumns+`
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		WHERE i.thumbnail_small_url IS NOT NULL
		  AND i.thumbnail_small_url != ''
		  AND NOT EXISTS (
			  SELECT 1
			  FROM image_tags it
			  WHERE it.image_id = i.id
			    AND it.source = 'ai'
			    AND it.review_state != 'rejected'
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
		if err := scanImage(rows, &image); err != nil {
			return nil, err
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

func (r *sqliteImageRepository) UpdateImageDuplicateHashCache(imageID int64, sha256, phashHex string, sourceMTimeUnix int64) error {
	_, err := r.db.Exec(`
		UPDATE images
		SET sha256 = ?, phash_hex = ?, source_mtime_unix = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, sha256, phashHex, sourceMTimeUnix, imageID)
	return err
}

// buildBackfillBaseWhere constructs the WHERE clause fragment and args for the base filter.
// It handles TagIDs (AND semantics) and HasTags filtering.
func buildBackfillBaseWhere(ctx context.Context, db *sql.DB, filter BackfillCandidateFilter) (string, []any, error) {
	var conds []string
	var args []any

	if len(filter.TagIDs) > 0 {
		clauses, err := expandHierarchicalTagClauses(ctx, db, filter.TagIDs)
		if err != nil {
			return "", nil, err
		}
		cond, clauseArgs := buildImageTagClauseFilters(clauses, "i.id")
		conds = append(conds, cond)
		args = append(args, clauseArgs...)
	}

	if filter.HasTags != nil {
		if !*filter.HasTags {
			conds = append(conds, `NOT EXISTS (SELECT 1 FROM image_tags it2 WHERE it2.image_id = i.id AND it2.review_state != 'rejected')`)
		} else {
			conds = append(conds, `EXISTS (SELECT 1 FROM image_tags it2 WHERE it2.image_id = i.id AND it2.review_state != 'rejected')`)
		}
	}

	if len(conds) > 0 {
		return " AND " + strings.Join(conds, " AND "), args, nil
	}
	return "", args, nil
}

// FindBackfillCandidates returns images matching the filter that are eligible for AI backfill:
// no AI source tags and no active (pending/queued/running) AI tasks.
func (r *sqliteImageRepository) FindBackfillCandidates(ctx context.Context, filter BackfillCandidateFilter) ([]domain.Image, error) {
	baseWhere, baseArgs, err := buildBackfillBaseWhere(ctx, r.db, filter)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		SELECT %s
		FROM images i
		LEFT JOIN collection_images ci ON ci.image_id = i.id
		WHERE 1=1
		%s
		AND NOT EXISTS (
			SELECT 1 FROM image_tags it3 WHERE it3.image_id = i.id AND it3.source = 'ai' AND it3.review_state != 'rejected'
		)
		AND NOT EXISTS (
			SELECT 1 FROM platform_tasks pt WHERE pt.image_id = i.id
			AND pt.task_type = 'ai_tag_generation'
			AND pt.status IN ('pending', 'queued', 'running')
		)
		ORDER BY i.id ASC
	`, imageSelectColumns, baseWhere)

	rows, err := r.db.QueryContext(ctx, query, baseArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := make([]domain.Image, 0)
	for rows.Next() {
		var image domain.Image
		if err := scanImage(rows, &image); err != nil {
			return nil, err
		}
		images = append(images, image)
	}
	return images, rows.Err()
}

// CountBackfillCandidates returns the count of images matching the filter that are eligible for AI backfill.
func (r *sqliteImageRepository) CountBackfillCandidates(ctx context.Context, filter BackfillCandidateFilter) (int64, error) {
	baseWhere, baseArgs, err := buildBackfillBaseWhere(ctx, r.db, filter)
	if err != nil {
		return 0, err
	}

	query := fmt.Sprintf(`
		SELECT COUNT(i.id)
		FROM images i
		WHERE 1=1
		%s
		AND NOT EXISTS (
			SELECT 1 FROM image_tags it3 WHERE it3.image_id = i.id AND it3.source = 'ai' AND it3.review_state != 'rejected'
		)
		AND NOT EXISTS (
			SELECT 1 FROM platform_tasks pt WHERE pt.image_id = i.id
			AND pt.task_type = 'ai_tag_generation'
			AND pt.status IN ('pending', 'queued', 'running')
		)
	`, baseWhere)

	var count int64
	err = r.db.QueryRowContext(ctx, query, baseArgs...).Scan(&count)
	return count, err
}

// CountBackfillSkippedWithAITag returns the count of images matching the filter that already have AI source tags.
func (r *sqliteImageRepository) CountBackfillSkippedWithAITag(ctx context.Context, filter BackfillCandidateFilter) (int64, error) {
	baseWhere, baseArgs, err := buildBackfillBaseWhere(ctx, r.db, filter)
	if err != nil {
		return 0, err
	}

	query := fmt.Sprintf(`
		SELECT COUNT(i.id)
		FROM images i
		WHERE 1=1
		%s
		AND EXISTS (
			SELECT 1 FROM image_tags it3 WHERE it3.image_id = i.id AND it3.source = 'ai' AND it3.review_state != 'rejected'
		)
	`, baseWhere)

	var count int64
	err = r.db.QueryRowContext(ctx, query, baseArgs...).Scan(&count)
	return count, err
}

// CountBackfillSkippedWithActiveTask returns the count of images matching the filter that already have active AI tasks.
func (r *sqliteImageRepository) CountBackfillSkippedWithActiveTask(ctx context.Context, filter BackfillCandidateFilter) (int64, error) {
	baseWhere, baseArgs, err := buildBackfillBaseWhere(ctx, r.db, filter)
	if err != nil {
		return 0, err
	}

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
	err = r.db.QueryRowContext(ctx, query, baseArgs...).Scan(&count)
	return count, err
}

// CountBackfillHitCount returns the total count of images matching the filter (before any skip classification).
func (r *sqliteImageRepository) CountBackfillHitCount(ctx context.Context, filter BackfillCandidateFilter) (int64, error) {
	baseWhere, baseArgs, err := buildBackfillBaseWhere(ctx, r.db, filter)
	if err != nil {
		return 0, err
	}

	query := fmt.Sprintf(`
		SELECT COUNT(i.id)
		FROM images i
		WHERE 1=1
		%s
	`, baseWhere)

	var count int64
	err = r.db.QueryRowContext(ctx, query, baseArgs...).Scan(&count)
	return count, err
}
