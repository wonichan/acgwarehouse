package repository

import (
	"database/sql"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

// TagObservationRepository 标签观测存储接口
type TagObservationRepository interface {
	Save(obs *domain.TagObservation) error
	FindByImageID(imageID int64) ([]domain.TagObservation, error)
	FindByID(id int64) (*domain.TagObservation, error)
}

type sqliteTagObservationRepository struct {
	db *sql.DB
}

// NewTagObservationRepository 创建标签观测存储实例
func NewTagObservationRepository(db *sql.DB) TagObservationRepository {
	return &sqliteTagObservationRepository{db: db}
}

func (r *sqliteTagObservationRepository) Save(obs *domain.TagObservation) error {
	if obs.CreatedAt.IsZero() {
		obs.CreatedAt = time.Now()
	}
	result, err := r.db.Exec(`
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

func (r *sqliteTagObservationRepository) FindByImageID(imageID int64) ([]domain.TagObservation, error) {
	rows, err := r.db.Query(`
		SELECT id, image_id, raw_text, confidence, evidence_type, provider, model_name, prompt_version, created_at
		FROM tag_observations WHERE image_id = ? ORDER BY created_at DESC
	`, imageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	observations := make([]domain.TagObservation, 0)
	for rows.Next() {
		var obs domain.TagObservation
		if err := rows.Scan(&obs.ID, &obs.ImageID, &obs.RawText, &obs.Confidence, &obs.EvidenceType, &obs.Provider, &obs.ModelName, &obs.PromptVersion, &obs.CreatedAt); err != nil {
			return nil, err
		}
		observations = append(observations, obs)
	}
	return observations, rows.Err()
}

func (r *sqliteTagObservationRepository) FindByID(id int64) (*domain.TagObservation, error) {
	obs := &domain.TagObservation{}
	err := r.db.QueryRow(`
		SELECT id, image_id, raw_text, confidence, evidence_type, provider, model_name, prompt_version, created_at
		FROM tag_observations WHERE id = ?
	`, id).Scan(&obs.ID, &obs.ImageID, &obs.RawText, &obs.Confidence, &obs.EvidenceType, &obs.Provider, &obs.ModelName, &obs.PromptVersion, &obs.CreatedAt)
	if err != nil {
		return nil, err
	}
	return obs, nil
}
