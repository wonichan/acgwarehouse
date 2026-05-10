package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/d1client"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type d1TagObservationRepository struct {
	client *d1client.Client
}

func NewD1TagObservationRepository(client *d1client.Client) TagObservationRepository {
	return &d1TagObservationRepository{client: client}
}

func (r *d1TagObservationRepository) Save(ctx context.Context, obs *domain.TagObservation) error {
	if obs.CreatedAt.IsZero() {
		obs.CreatedAt = time.Now()
	}

	id, err := r.client.ExecReturningID(ctx, `
		INSERT INTO tag_observations (image_id, raw_text, confidence, evidence_type, provider, model_name, prompt_version, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, obs.ImageID, obs.RawText, obs.Confidence, obs.EvidenceType, obs.Provider, obs.ModelName, obs.PromptVersion, obs.CreatedAt)
	if err != nil {
		return err
	}
	obs.ID = id
	return nil
}

func (r *d1TagObservationRepository) FindByID(ctx context.Context, id int64) (*domain.TagObservation, error) {
	row, err := r.client.QueryOne(ctx, `
		SELECT id, image_id, raw_text, confidence, evidence_type, provider, model_name, prompt_version, created_at
		FROM tag_observations
		WHERE id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, sql.ErrNoRows
	}
	return mapTagObservationFromD1(row)
}

func (r *d1TagObservationRepository) FindByImageID(ctx context.Context, imageID int64) ([]*domain.TagObservation, error) {
	rows, err := r.client.Query(ctx, `
		SELECT id, image_id, raw_text, confidence, evidence_type, provider, model_name, prompt_version, created_at
		FROM tag_observations
		WHERE image_id = ?
		ORDER BY created_at DESC, id DESC
	`, imageID)
	if err != nil {
		return nil, err
	}
	return mapTagObservationsFromD1(rows)
}

func (r *d1TagObservationRepository) FindByProvider(ctx context.Context, provider string, limit int) ([]*domain.TagObservation, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.client.Query(ctx, `
		SELECT id, image_id, raw_text, confidence, evidence_type, provider, model_name, prompt_version, created_at
		FROM tag_observations
		WHERE provider = ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?
	`, provider, limit)
	if err != nil {
		return nil, err
	}
	return mapTagObservationsFromD1(rows)
}

func mapTagObservationsFromD1(rows []map[string]any) ([]*domain.TagObservation, error) {
	observations := make([]*domain.TagObservation, 0, len(rows))
	for _, row := range rows {
		obs, err := mapTagObservationFromD1(row)
		if err != nil {
			return nil, err
		}
		observations = append(observations, obs)
	}
	return observations, nil
}

func mapTagObservationFromD1(row map[string]any) (*domain.TagObservation, error) {
	id, err := toInt64(row["id"])
	if err != nil {
		return nil, err
	}
	imageID, err := toInt64(row["image_id"])
	if err != nil {
		return nil, err
	}
	confidence, _ := toFloat64(row["confidence"])
	createdAt, _ := toTime(row["created_at"])
	return &domain.TagObservation{
		ID:            id,
		ImageID:       imageID,
		RawText:       toStringDefault(row["raw_text"], ""),
		Confidence:    confidence,
		EvidenceType:  toStringDefault(row["evidence_type"], ""),
		Provider:      toStringDefault(row["provider"], ""),
		ModelName:     toStringDefault(row["model_name"], ""),
		PromptVersion: toStringDefault(row["prompt_version"], ""),
		CreatedAt:     createdAt,
	}, nil
}
