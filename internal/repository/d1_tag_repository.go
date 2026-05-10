package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/d1client"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type d1TagRepository struct {
	client *d1client.Client
}

func NewD1TagRepository(client *d1client.Client) TagRepository {
	return &d1TagRepository{client: client}
}

func (r *d1TagRepository) FindByID(ctx context.Context, id int64) (*domain.Tag, error) {
	row, err := r.client.QueryOne(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags WHERE id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, sql.ErrNoRows
	}
	return mapTagFromD1(row)
}

func (r *d1TagRepository) FindByLabel(ctx context.Context, label string) (*domain.Tag, error) {
	row, err := r.client.QueryOne(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags WHERE preferred_label = ?
	`, label)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, sql.ErrNoRows
	}
	return mapTagFromD1(row)
}

func (r *d1TagRepository) FindByLabelLike(ctx context.Context, query string, limit int) ([]*domain.Tag, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.client.Query(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags
		WHERE preferred_label LIKE ?
		ORDER BY usage_count DESC, id ASC
		LIMIT ?
	`, "%"+query+"%", int64(limit))
	if err != nil {
		return nil, err
	}
	return mapTagsFromD1(rows)
}

func (r *d1TagRepository) FindAll(ctx context.Context, limit, offset int) ([]*domain.Tag, error) {
	rows, err := r.client.Query(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags
		ORDER BY usage_count DESC, id ASC
		LIMIT ? OFFSET ?
	`, int64(limit), int64(offset))
	if err != nil {
		return nil, err
	}
	return mapTagsFromD1(rows)
}

func (r *d1TagRepository) FindRoots(ctx context.Context) ([]*domain.Tag, error) {
	rows, err := r.client.Query(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags
		WHERE level = ?
		ORDER BY usage_count DESC, id ASC
	`, domain.TagLevelRoot)
	if err != nil {
		return nil, err
	}
	return mapTagsFromD1(rows)
}

func (r *d1TagRepository) FindChildrenByParent(ctx context.Context, parentID int64) ([]*domain.Tag, error) {
	rows, err := r.client.Query(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags
		WHERE parent_id = ?
		ORDER BY usage_count DESC, id ASC
	`, parentID)
	if err != nil {
		return nil, err
	}
	return mapTagsFromD1(rows)
}

func (r *d1TagRepository) FindValidParentCandidates(ctx context.Context, targetLevel string) ([]*domain.Tag, error) {
	var parentLevel string
	switch targetLevel {
	case domain.TagLevelParent:
		parentLevel = domain.TagLevelRoot
	case domain.TagLevelChild:
		parentLevel = domain.TagLevelParent
	default:
		return []*domain.Tag{}, nil
	}
	rows, err := r.client.Query(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags
		WHERE level = ?
		ORDER BY usage_count DESC, id ASC
	`, parentLevel)
	if err != nil {
		return nil, err
	}
	return mapTagsFromD1(rows)
}

func (r *d1TagRepository) ResolveDescendantIDs(ctx context.Context, tagIDs []int64) (map[int64][]int64, error) {
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

func (r *d1TagRepository) resolveDescendantIDsForSingle(ctx context.Context, tagID int64) ([]int64, error) {
	rows, err := r.client.Query(ctx, `
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

	ids := make([]int64, 0)
	for _, row := range rows {
		id, err := toInt64(row["id"])
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *d1TagRepository) ResolveAllDescendantIDs(ctx context.Context, tagIDs []int64) ([]int64, error) {
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

func (r *d1TagRepository) Count(ctx context.Context) (int, error) {
	cnt, err := r.client.QueryCount(ctx, `SELECT COUNT(*) as cnt FROM tags`)
	if err != nil {
		return 0, err
	}
	return int(cnt), nil
}

func (r *d1TagRepository) FindTreeRoots(ctx context.Context) ([]*TagBrowseNode, error) {
	rows, err := r.client.Query(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at,
			EXISTS (SELECT 1 FROM tags child WHERE child.parent_id = tags.id) as has_children
		FROM tags
		WHERE level = ?
		ORDER BY usage_count DESC, id ASC
	`, domain.TagLevelRoot)
	if err != nil {
		return nil, err
	}
	return mapBrowseNodesFromD1(rows)
}

func (r *d1TagRepository) FindTreeChildren(ctx context.Context, parentID int64) ([]*TagBrowseNode, error) {
	rows, err := r.client.Query(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at,
			EXISTS (SELECT 1 FROM tags child WHERE child.parent_id = tags.id) as has_children
		FROM tags
		WHERE parent_id = ?
		ORDER BY usage_count DESC, id ASC
	`, parentID)
	if err != nil {
		return nil, err
	}
	return mapBrowseNodesFromD1(rows)
}

func (r *d1TagRepository) ListOrphanTags(ctx context.Context, search string, limit, offset int) ([]*TagBrowseNode, error) {
	query := `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at,
			EXISTS (SELECT 1 FROM tags child WHERE child.parent_id = tags.id) as has_children
		FROM tags
		WHERE parent_id IS NULL AND level != ?
	`
	args := []any{domain.TagLevelRoot}

	if search != "" {
		query += ` AND preferred_label LIKE ?`
		args = append(args, "%"+search+"%")
	}

	query += ` ORDER BY usage_count DESC, id ASC LIMIT ? OFFSET ?`
	args = append(args, int64(limit), int64(offset))

	rows, err := r.client.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return mapBrowseNodesFromD1(rows)
}

func (r *d1TagRepository) CountOrphanTags(ctx context.Context, search string) (int, error) {
	query := `SELECT COUNT(*) as cnt FROM tags WHERE parent_id IS NULL AND level != ?`
	args := []any{domain.TagLevelRoot}

	if search != "" {
		query += ` AND preferred_label LIKE ?`
		args = append(args, "%"+search+"%")
	}

	cnt, err := r.client.QueryCount(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return int(cnt), nil
}

func (r *d1TagRepository) Save(ctx context.Context, tag *domain.Tag) error {
	if tag.CreatedAt.IsZero() {
		tag.CreatedAt = time.Now()
	}
	if tag.Level == "" {
		tag.Level = domain.TagLevelChild
	}
	if tag.ID > 0 {
		return r.Update(ctx, tag)
	}
	id, err := r.client.ExecReturningID(ctx, `
		INSERT INTO tags (preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tag.PreferredLabel, tag.Slug, tag.Level, tag.ParentID, tag.PrimaryCategory, tag.ReviewState, tag.TrustScore, tag.UsageCount, tag.CreatedAt)
	if err != nil {
		return err
	}
	if id > 0 {
		tag.ID = id
	}
	return nil
}

func (r *d1TagRepository) Update(ctx context.Context, tag *domain.Tag) error {
	if tag.Level == "" {
		tag.Level = domain.TagLevelChild
	}
	return r.client.Exec(ctx, `
		UPDATE tags 
		SET preferred_label = ?, slug = ?, level = ?, parent_id = ?, primary_category = ?, review_state = ?, trust_score = ?, usage_count = ?
		WHERE id = ?
	`, tag.PreferredLabel, tag.Slug, tag.Level, tag.ParentID, tag.PrimaryCategory, tag.ReviewState, tag.TrustScore, tag.UsageCount, tag.ID)
}

func (r *d1TagRepository) UpdateReviewState(ctx context.Context, id int64, state string) error {
	return r.client.Exec(ctx, `UPDATE tags SET review_state = ? WHERE id = ?`, state, id)
}

func (r *d1TagRepository) IncrementUsageCount(ctx context.Context, id int64) error {
	return r.client.Exec(ctx, `UPDATE tags SET usage_count = usage_count + 1 WHERE id = ?`, id)
}

func (r *d1TagRepository) DecrementUsageCount(ctx context.Context, id int64) error {
	return r.client.Exec(ctx, `UPDATE tags SET usage_count = MAX(usage_count - 1, 0) WHERE id = ?`, id)
}

func (r *d1TagRepository) Delete(ctx context.Context, id int64) error {
	return r.client.Exec(ctx, `DELETE FROM tags WHERE id = ?`, id)
}

func mapTagsFromD1(rows []map[string]any) ([]*domain.Tag, error) {
	tags := make([]*domain.Tag, 0, len(rows))
	for _, row := range rows {
		tag, err := mapTagFromD1(row)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func mapBrowseNodesFromD1(rows []map[string]any) ([]*TagBrowseNode, error) {
	nodes := make([]*TagBrowseNode, 0, len(rows))
	for _, row := range rows {
		id, err := toInt64(row["id"])
		if err != nil {
			return nil, err
		}
		var parentID *int64
		if pid, err := toInt64(row["parent_id"]); err == nil && pid != 0 {
			parentID = &pid
		}
		var primaryCategory *string
		if pc, ok := row["primary_category"].(string); ok && pc != "" {
			primaryCategory = &pc
		}
		var reviewState *string
		if rs, ok := row["review_state"].(string); ok && rs != "" {
			reviewState = &rs
		}
		usageCount, _ := toInt(row["usage_count"])
		trustScore, _ := toFloat64(row["trust_score"])
		hasChildren, _ := toBool(row["has_children"])
		createdAt, _ := toTime(row["created_at"])

		nodes = append(nodes, &TagBrowseNode{
			ID:              id,
			PreferredLabel:  toStringDefault(row["preferred_label"], ""),
			Slug:            toStringDefault(row["slug"], ""),
			Level:           toStringDefault(row["level"], ""),
			ParentID:        parentID,
			PrimaryCategory: primaryCategory,
			ReviewState:     reviewState,
			TrustScore:      trustScore,
			UsageCount:      usageCount,
			CreatedAt:       createdAt,
			HasChildren:     hasChildren,
		})
	}
	return nodes, nil
}
