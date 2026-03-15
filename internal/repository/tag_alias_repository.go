package repository

import (
	"context"
	"database/sql"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type TagAliasRepository interface {
	Save(ctx context.Context, alias *domain.TagAlias) error
	FindByID(ctx context.Context, id int64) (*domain.TagAlias, error)
	FindByTagID(ctx context.Context, tagID int64) ([]*domain.TagAlias, error)
	FindByNormalizedLabel(ctx context.Context, normalized string) (*domain.TagAlias, error)
	FindByLabelLike(ctx context.Context, query string) ([]*domain.TagAlias, error)
	Delete(ctx context.Context, id int64) error
}

type tagAliasRepository struct {
	db *sql.DB
}

func NewTagAliasRepository(db *sql.DB) TagAliasRepository {
	return &tagAliasRepository{db: db}
}

func NormalizeLabel(label string) string {
	return strings.ToLower(strings.TrimSpace(label))
}

func (r *tagAliasRepository) Save(ctx context.Context, alias *domain.TagAlias) error {
	alias.NormalizedLabel = NormalizeLabel(alias.Label)

	var (
		result sql.Result
		err    error
	)
	if alias.ID > 0 {
		result, err = r.db.ExecContext(ctx, `
			INSERT OR REPLACE INTO tag_aliases (id, tag_id, label, normalized_label, locale, alias_type, is_preferred)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, alias.ID, alias.TagID, alias.Label, alias.NormalizedLabel, alias.Locale, alias.AliasType, alias.IsPreferred)
	} else {
		result, err = r.db.ExecContext(ctx, `
			INSERT OR REPLACE INTO tag_aliases (tag_id, label, normalized_label, locale, alias_type, is_preferred)
			VALUES (?, ?, ?, ?, ?, ?)
		`, alias.TagID, alias.Label, alias.NormalizedLabel, alias.Locale, alias.AliasType, alias.IsPreferred)
	}
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	if id > 0 {
		alias.ID = id
	}

	return nil
}

func (r *tagAliasRepository) FindByID(ctx context.Context, id int64) (*domain.TagAlias, error) {
	return r.queryOne(ctx, `
		SELECT id, tag_id, label, normalized_label, locale, alias_type, is_preferred
		FROM tag_aliases WHERE id = ?
	`, id)
}

func (r *tagAliasRepository) FindByTagID(ctx context.Context, tagID int64) ([]*domain.TagAlias, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tag_id, label, normalized_label, locale, alias_type, is_preferred
		FROM tag_aliases WHERE tag_id = ? ORDER BY id ASC
	`, tagID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTagAliases(rows)
}

func (r *tagAliasRepository) FindByNormalizedLabel(ctx context.Context, normalized string) (*domain.TagAlias, error) {
	return r.queryOne(ctx, `
		SELECT id, tag_id, label, normalized_label, locale, alias_type, is_preferred
		FROM tag_aliases WHERE normalized_label = ?
	`, NormalizeLabel(normalized))
}

func (r *tagAliasRepository) FindByLabelLike(ctx context.Context, query string) ([]*domain.TagAlias, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tag_id, label, normalized_label, locale, alias_type, is_preferred
		FROM tag_aliases WHERE label LIKE ? ORDER BY id ASC
	`, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTagAliases(rows)
}

func (r *tagAliasRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM tag_aliases WHERE id = ?`, id)
	return err
}

func (r *tagAliasRepository) queryOne(ctx context.Context, query string, args ...any) (*domain.TagAlias, error) {
	alias := &domain.TagAlias{}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&alias.ID,
		&alias.TagID,
		&alias.Label,
		&alias.NormalizedLabel,
		&alias.Locale,
		&alias.AliasType,
		&alias.IsPreferred,
	)
	if err != nil {
		return nil, err
	}

	return alias, nil
}

func scanTagAliases(rows *sql.Rows) ([]*domain.TagAlias, error) {
	aliases := make([]*domain.TagAlias, 0)
	for rows.Next() {
		alias := &domain.TagAlias{}
		if err := rows.Scan(&alias.ID, &alias.TagID, &alias.Label, &alias.NormalizedLabel, &alias.Locale, &alias.AliasType, &alias.IsPreferred); err != nil {
			return nil, err
		}
		aliases = append(aliases, alias)
	}

	return aliases, rows.Err()
}
