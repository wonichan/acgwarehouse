package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type TagRepository interface {
	Save(ctx context.Context, tag *domain.Tag) error
	Update(ctx context.Context, tag *domain.Tag) error
	FindByID(ctx context.Context, id int64) (*domain.Tag, error)
	FindByLabel(ctx context.Context, label string) (*domain.Tag, error)
	FindByLabelLike(ctx context.Context, query string, limit int) ([]*domain.Tag, error)
	FindAll(ctx context.Context, limit, offset int) ([]*domain.Tag, error)
	FindRoots(ctx context.Context) ([]*domain.Tag, error)
	FindChildrenByParent(ctx context.Context, parentID int64) ([]*domain.Tag, error)
	FindValidParentCandidates(ctx context.Context, targetLevel string) ([]*domain.Tag, error)
	ResolveDescendantIDs(ctx context.Context, tagIDs []int64) (map[int64][]int64, error)
	ResolveAllDescendantIDs(ctx context.Context, tagIDs []int64) ([]int64, error)
	UpdateReviewState(ctx context.Context, id int64, state string) error
	IncrementUsageCount(ctx context.Context, id int64) error
	DecrementUsageCount(ctx context.Context, id int64) error
	Delete(ctx context.Context, id int64) error
	Count(ctx context.Context) (int, error)
	FindTreeRoots(ctx context.Context) ([]*TagBrowseNode, error)
	FindTreeChildren(ctx context.Context, parentID int64) ([]*TagBrowseNode, error)
	ListOrphanTags(ctx context.Context, search string, limit, offset int) ([]*TagBrowseNode, error)
	CountOrphanTags(ctx context.Context, search string) (int, error)
}

type tagRepository struct {
	db *sql.DB
}

func NewTagRepository(db *sql.DB) TagRepository {
	return &tagRepository{db: db}
}

// Save creates new tags with INSERT and delegates existing tags to Update to avoid replace-triggered cascade deletes.
func (r *tagRepository) Save(ctx context.Context, tag *domain.Tag) error {
	if tag.CreatedAt.IsZero() {
		tag.CreatedAt = time.Now()
	}
	if tag.Level == "" {
		tag.Level = domain.TagLevelChild
	}

	if tag.ID > 0 {
		return r.Update(ctx, tag)
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO tags (preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tag.PreferredLabel, tag.Slug, tag.Level, tag.ParentID, tag.PrimaryCategory, tag.ReviewState, tag.TrustScore, tag.UsageCount, tag.CreatedAt)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	if id > 0 {
		tag.ID = id
	}

	return nil
}

// Update updates an existing tag using UPDATE statement to avoid cascade delete
func (r *tagRepository) Update(ctx context.Context, tag *domain.Tag) error {
	if tag.Level == "" {
		tag.Level = domain.TagLevelChild
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE tags 
		SET preferred_label = ?, slug = ?, level = ?, parent_id = ?, primary_category = ?, review_state = ?, trust_score = ?, usage_count = ?
		WHERE id = ?
	`, tag.PreferredLabel, tag.Slug, tag.Level, tag.ParentID, tag.PrimaryCategory, tag.ReviewState, tag.TrustScore, tag.UsageCount, tag.ID)
	return err
}

func (r *tagRepository) FindByID(ctx context.Context, id int64) (*domain.Tag, error) {
	return r.queryOne(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags WHERE id = ?
	`, id)
}

func (r *tagRepository) FindByLabel(ctx context.Context, label string) (*domain.Tag, error) {
	return r.queryOne(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags WHERE preferred_label = ?
	`, label)
}

func (r *tagRepository) FindByLabelLike(ctx context.Context, query string, limit int) ([]*domain.Tag, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags
		WHERE preferred_label LIKE ?
		ORDER BY usage_count DESC, id ASC
		LIMIT ?
	`, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTags(rows)
}

func (r *tagRepository) FindAll(ctx context.Context, limit, offset int) ([]*domain.Tag, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags
		ORDER BY usage_count DESC, id ASC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTags(rows)
}

func (r *tagRepository) FindRoots(ctx context.Context) ([]*domain.Tag, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags
		WHERE level = ?
		ORDER BY usage_count DESC, id ASC
	`, domain.TagLevelRoot)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTags(rows)
}

func (r *tagRepository) FindChildrenByParent(ctx context.Context, parentID int64) ([]*domain.Tag, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags
		WHERE parent_id = ?
		ORDER BY usage_count DESC, id ASC
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTags(rows)
}

func (r *tagRepository) FindValidParentCandidates(ctx context.Context, targetLevel string) ([]*domain.Tag, error) {
	var parentLevel string
	switch targetLevel {
	case domain.TagLevelParent:
		parentLevel = domain.TagLevelRoot
	case domain.TagLevelChild:
		parentLevel = domain.TagLevelParent
	default:
		return []*domain.Tag{}, nil
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags
		WHERE level = ?
		ORDER BY usage_count DESC, id ASC
	`, parentLevel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTags(rows)
}

// ResolveDescendantIDs returns a map from each requested tagID to a slice containing the tag itself plus all its descendants. The tag's own ID is always included in the result.
func (r *tagRepository) ResolveDescendantIDs(ctx context.Context, tagIDs []int64) (map[int64][]int64, error) {
	resolved := make(map[int64][]int64, len(tagIDs))
	for _, tagID := range tagIDs {
		ids, err := r.resolveDescendantIDsForSingle(ctx, tagID)
		if err != nil {
			return nil, err
		}
		resolved[tagID] = ids
	}
	return resolved, nil
}

// ResolveAllDescendantIDs returns a deduplicated slice containing each requested tag ID plus all of their descendants. Each requested tag's own ID is always included in the result.
func (r *tagRepository) ResolveAllDescendantIDs(ctx context.Context, tagIDs []int64) ([]int64, error) {
	resolved, err := r.ResolveDescendantIDs(ctx, tagIDs)
	if err != nil {
		return nil, err
	}

	seen := make(map[int64]struct{})
	merged := make([]int64, 0)
	for _, tagID := range tagIDs {
		for _, id := range resolved[tagID] {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			merged = append(merged, id)
		}
	}

	return merged, nil
}

func (r *tagRepository) UpdateReviewState(ctx context.Context, id int64, state string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tags SET review_state = ? WHERE id = ?`, state, id)
	return err
}

func (r *tagRepository) IncrementUsageCount(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tags SET usage_count = usage_count + 1 WHERE id = ?`, id)
	return err
}

func (r *tagRepository) DecrementUsageCount(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tags SET usage_count = MAX(usage_count - 1, 0) WHERE id = ?`, id)
	return err
}

func (r *tagRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, id)
	return err
}

func (r *tagRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tags`).Scan(&count)
	return count, err
}

// resolveDescendantIDsForSingle returns a slice containing the requested tagID plus all of its descendants. The tag's own ID is always included in the result.
func (r *tagRepository) resolveDescendantIDsForSingle(ctx context.Context, tagID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		WITH RECURSIVE descendants(id) AS (
			SELECT id FROM tags WHERE id = ?
			UNION ALL
			SELECT t.id
			FROM tags t
			INNER JOIN descendants d ON t.parent_id = d.id
		)
		SELECT DISTINCT id FROM descendants ORDER BY id ASC
	`, tagID)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ids, nil
}

func (r *tagRepository) queryOne(ctx context.Context, query string, args ...any) (*domain.Tag, error) {
	tag := &domain.Tag{}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&tag.ID,
		&tag.PreferredLabel,
		&tag.Slug,
		&tag.Level,
		&tag.ParentID,
		&tag.PrimaryCategory,
		&tag.ReviewState,
		&tag.TrustScore,
		&tag.UsageCount,
		&tag.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return tag, nil
}

func scanTags(rows *sql.Rows) ([]*domain.Tag, error) {
	tags := make([]*domain.Tag, 0)
	for rows.Next() {
		tag := &domain.Tag{}
		if err := rows.Scan(
			&tag.ID,
			&tag.PreferredLabel,
			&tag.Slug,
			&tag.Level,
			&tag.ParentID,
			&tag.PrimaryCategory,
			&tag.ReviewState,
			&tag.TrustScore,
			&tag.UsageCount,
			&tag.CreatedAt,
		); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	return tags, rows.Err()
}

type TagBrowseNode struct {
	ID              int64
	PreferredLabel  string
	Slug            string
	Level           string
	ParentID        *int64
	PrimaryCategory string
	ReviewState     string
	TrustScore      float64
	UsageCount      int
	CreatedAt       time.Time
	HasChildren     bool
}

func (r *tagRepository) FindTreeRoots(ctx context.Context) ([]*TagBrowseNode, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at,
			EXISTS (SELECT 1 FROM tags child WHERE child.parent_id = tags.id) AS has_children
		FROM tags
		WHERE level = ?
		ORDER BY usage_count DESC, id ASC
	`, domain.TagLevelRoot)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanBrowseNodes(rows)
}

func (r *tagRepository) FindTreeChildren(ctx context.Context, parentID int64) ([]*TagBrowseNode, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at,
			EXISTS (SELECT 1 FROM tags child WHERE child.parent_id = tags.id) AS has_children
		FROM tags
		WHERE parent_id = ?
		ORDER BY usage_count DESC, id ASC
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanBrowseNodes(rows)
}

func (r *tagRepository) ListOrphanTags(ctx context.Context, search string, limit, offset int) ([]*TagBrowseNode, error) {
	query := `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at,
			EXISTS (SELECT 1 FROM tags child WHERE child.parent_id = tags.id) AS has_children
		FROM tags
		WHERE parent_id IS NULL AND level != ?
	`
	args := []any{domain.TagLevelRoot}

	if search != "" {
		query += ` AND preferred_label LIKE ?`
		args = append(args, "%"+search+"%")
	}

	query += ` ORDER BY usage_count DESC, id ASC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanBrowseNodes(rows)
}

func (r *tagRepository) CountOrphanTags(ctx context.Context, search string) (int, error) {
	query := `SELECT COUNT(*) FROM tags WHERE parent_id IS NULL AND level != ?`
	args := []any{domain.TagLevelRoot}

	if search != "" {
		query += ` AND preferred_label LIKE ?`
		args = append(args, "%"+search+"%")
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

func scanBrowseNodes(rows *sql.Rows) ([]*TagBrowseNode, error) {
	nodes := make([]*TagBrowseNode, 0)
	for rows.Next() {
		node := &TagBrowseNode{}
		if err := rows.Scan(
			&node.ID,
			&node.PreferredLabel,
			&node.Slug,
			&node.Level,
			&node.ParentID,
			&node.PrimaryCategory,
			&node.ReviewState,
			&node.TrustScore,
			&node.UsageCount,
			&node.CreatedAt,
			&node.HasChildren,
		); err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}

	return nodes, rows.Err()
}
