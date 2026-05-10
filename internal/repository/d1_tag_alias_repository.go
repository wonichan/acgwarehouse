package repository

import (
	"context"
	"database/sql"

	"github.com/wonichan/acgwarehouse-backend/internal/d1client"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type d1TagAliasRepository struct {
	client *d1client.Client
}

func NewD1TagAliasRepository(client *d1client.Client) TagAliasRepository {
	return &d1TagAliasRepository{client: client}
}

func (r *d1TagAliasRepository) FindByID(ctx context.Context, id int64) (*domain.TagAlias, error) {
	row, err := r.client.QueryOne(ctx, `
		SELECT id, tag_id, label, normalized_label, locale, alias_type, is_preferred
		FROM tag_aliases WHERE id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, sql.ErrNoRows
	}
	return mapTagAliasFromD1(row)
}

func (r *d1TagAliasRepository) FindByTagID(ctx context.Context, tagID int64) ([]*domain.TagAlias, error) {
	rows, err := r.client.Query(ctx, `
		SELECT id, tag_id, label, normalized_label, locale, alias_type, is_preferred
		FROM tag_aliases WHERE tag_id = ? ORDER BY id ASC
	`, tagID)
	if err != nil {
		return nil, err
	}
	return mapTagAliasesFromD1(rows)
}

func (r *d1TagAliasRepository) FindByNormalizedLabel(ctx context.Context, normalized string) (*domain.TagAlias, error) {
	row, err := r.client.QueryOne(ctx, `
		SELECT id, tag_id, label, normalized_label, locale, alias_type, is_preferred
		FROM tag_aliases WHERE normalized_label = ?
	`, NormalizeLabel(normalized))
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, sql.ErrNoRows
	}
	return mapTagAliasFromD1(row)
}

func (r *d1TagAliasRepository) FindByLabelLike(ctx context.Context, query string) ([]*domain.TagAlias, error) {
	rows, err := r.client.Query(ctx, `
		SELECT id, tag_id, label, normalized_label, locale, alias_type, is_preferred
		FROM tag_aliases WHERE label LIKE ? ORDER BY id ASC
	`, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	return mapTagAliasesFromD1(rows)
}

func (r *d1TagAliasRepository) Save(ctx context.Context, alias *domain.TagAlias) error {
	alias.NormalizedLabel = NormalizeLabel(alias.Label)

	if alias.ID > 0 {
		return r.client.Exec(ctx, `
			INSERT OR REPLACE INTO tag_aliases (id, tag_id, label, normalized_label, locale, alias_type, is_preferred)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, alias.ID, alias.TagID, alias.Label, alias.NormalizedLabel, alias.Locale, alias.AliasType, alias.IsPreferred)
	}

	id, err := r.client.ExecReturningID(ctx, `
		INSERT OR REPLACE INTO tag_aliases (tag_id, label, normalized_label, locale, alias_type, is_preferred)
		VALUES (?, ?, ?, ?, ?, ?)
	`, alias.TagID, alias.Label, alias.NormalizedLabel, alias.Locale, alias.AliasType, alias.IsPreferred)
	if err != nil {
		return err
	}
	if id > 0 {
		alias.ID = id
	}
	return nil
}

func (r *d1TagAliasRepository) Delete(ctx context.Context, id int64) error {
	return r.client.Exec(ctx, `DELETE FROM tag_aliases WHERE id = ?`, id)
}

func mapTagAliasesFromD1(rows []map[string]any) ([]*domain.TagAlias, error) {
	aliases := make([]*domain.TagAlias, 0, len(rows))
	for _, row := range rows {
		alias, err := mapTagAliasFromD1(row)
		if err != nil {
			return nil, err
		}
		aliases = append(aliases, alias)
	}
	return aliases, nil
}
