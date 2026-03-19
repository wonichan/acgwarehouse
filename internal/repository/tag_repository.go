package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type TagRepository interface {
	Save(ctx context.Context, tag *domain.Tag) error
	FindByID(ctx context.Context, id int64) (*domain.Tag, error)
	FindByLabel(ctx context.Context, label string) (*domain.Tag, error)
	FindByLabelLike(ctx context.Context, query string, limit int) ([]*domain.Tag, error)
	FindAll(ctx context.Context, limit, offset int) ([]*domain.Tag, error)
	UpdateReviewState(ctx context.Context, id int64, state string) error
	IncrementUsageCount(ctx context.Context, id int64) error
	DecrementUsageCount(ctx context.Context, id int64) error
	Delete(ctx context.Context, id int64) error
	Count(ctx context.Context) (int, error)
}

type tagRepository struct {
	db *sql.DB
}

func NewTagRepository(db *sql.DB) TagRepository {
	return &tagRepository{db: db}
}

func (r *tagRepository) Save(ctx context.Context, tag *domain.Tag) error {
	if tag.CreatedAt.IsZero() {
		tag.CreatedAt = time.Now()
	}

	var (
		result sql.Result
		err    error
	)
	if tag.ID > 0 {
		result, err = r.db.ExecContext(ctx, `
			INSERT OR REPLACE INTO tags (id, preferred_label, slug, primary_category, review_state, trust_score, usage_count, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, tag.ID, tag.PreferredLabel, tag.Slug, tag.PrimaryCategory, tag.ReviewState, tag.TrustScore, tag.UsageCount, tag.CreatedAt)
	} else {
		result, err = r.db.ExecContext(ctx, `
			INSERT OR REPLACE INTO tags (preferred_label, slug, primary_category, review_state, trust_score, usage_count, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, tag.PreferredLabel, tag.Slug, tag.PrimaryCategory, tag.ReviewState, tag.TrustScore, tag.UsageCount, tag.CreatedAt)
	}
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

func (r *tagRepository) FindByID(ctx context.Context, id int64) (*domain.Tag, error) {
	return r.queryOne(ctx, `
		SELECT id, preferred_label, slug, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags WHERE id = ?
	`, id)
}

func (r *tagRepository) FindByLabel(ctx context.Context, label string) (*domain.Tag, error) {
	return r.queryOne(ctx, `
		SELECT id, preferred_label, slug, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags WHERE preferred_label = ?
	`, label)
}

func (r *tagRepository) FindByLabelLike(ctx context.Context, query string, limit int) ([]*domain.Tag, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, preferred_label, slug, primary_category, review_state, trust_score, usage_count, created_at
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
		SELECT id, preferred_label, slug, primary_category, review_state, trust_score, usage_count, created_at
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

func (r *tagRepository) queryOne(ctx context.Context, query string, args ...any) (*domain.Tag, error) {
	tag := &domain.Tag{}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&tag.ID,
		&tag.PreferredLabel,
		&tag.Slug,
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
