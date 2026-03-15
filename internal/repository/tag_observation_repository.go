package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type TagObservationRepository interface {
	Save(ctx context.Context, obs *domain.TagObservation) error
	FindByID(ctx context.Context, id int64) (*domain.TagObservation, error)
	FindByImageID(ctx context.Context, imageID int64) ([]*domain.TagObservation, error)
	FindByProvider(ctx context.Context, provider string, limit int) ([]*domain.TagObservation, error)
}

type sqliteTagObservationRepository struct {
	db *sql.DB
}

func NewTagObservationRepository(db *sql.DB) TagObservationRepository {
	return &sqliteTagObservationRepository{db: db}
}

func (r *sqliteTagObservationRepository) Save(ctx context.Context, obs *domain.TagObservation) error {
	if obs.CreatedAt.IsZero() {
		obs.CreatedAt = time.Now()
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO tag_observations (image_id, raw_text, confidence, evidence_type, provider, model_name, prompt_version, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, obs.ImageID, obs.RawText, obs.Confidence, obs.EvidenceType, obs.Provider, obs.ModelName, obs.PromptVersion, obs.CreatedAt)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	obs.ID = id

	return nil
}

func (r *sqliteTagObservationRepository) FindByID(ctx context.Context, id int64) (*domain.TagObservation, error) {
	obs := &domain.TagObservation{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, image_id, raw_text, confidence, evidence_type, provider, model_name, prompt_version, created_at
		FROM tag_observations WHERE id = ?
	`, id).Scan(&obs.ID, &obs.ImageID, &obs.RawText, &obs.Confidence, &obs.EvidenceType, &obs.Provider, &obs.ModelName, &obs.PromptVersion, &obs.CreatedAt)
	if err != nil {
		return nil, err
	}

	return obs, nil
}

func (r *sqliteTagObservationRepository) FindByImageID(ctx context.Context, imageID int64) ([]*domain.TagObservation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, image_id, raw_text, confidence, evidence_type, provider, model_name, prompt_version, created_at
		FROM tag_observations WHERE image_id = ? ORDER BY created_at DESC, id DESC
	`, imageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTagObservations(rows)
}

func (r *sqliteTagObservationRepository) FindByProvider(ctx context.Context, provider string, limit int) ([]*domain.TagObservation, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, image_id, raw_text, confidence, evidence_type, provider, model_name, prompt_version, created_at
		FROM tag_observations WHERE provider = ? ORDER BY created_at DESC, id DESC LIMIT ?
	`, provider, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTagObservations(rows)
}

func scanTagObservations(rows *sql.Rows) ([]*domain.TagObservation, error) {
	observations := make([]*domain.TagObservation, 0)
	for rows.Next() {
		obs := &domain.TagObservation{}
		if err := rows.Scan(&obs.ID, &obs.ImageID, &obs.RawText, &obs.Confidence, &obs.EvidenceType, &obs.Provider, &obs.ModelName, &obs.PromptVersion, &obs.CreatedAt); err != nil {
			return nil, err
		}
		observations = append(observations, obs)
	}

	return observations, rows.Err()
}
