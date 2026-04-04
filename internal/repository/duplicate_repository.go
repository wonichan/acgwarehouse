package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

// DuplicateRepository 定义重复检测数据访问接口
type DuplicateRepository interface {
	// SaveDuplicateGroup 保存重复组及其关联关系
	SaveDuplicateGroup(group *domain.DuplicateGroup, relations []domain.DuplicateRelation) error

	// FindDuplicateGroups 获取重复组列表（分页）
	FindDuplicateGroups(limit, offset int) ([]domain.DuplicateGroup, error)

	// FindDuplicateGroupByID 根据 ID 获取重复组及其关联关系
	FindDuplicateGroupByID(id int64) (*domain.DuplicateGroup, []domain.DuplicateRelation, error)

	// FindDuplicateGroupByImageID 根据图片 ID 查找所属重复组
	FindDuplicateGroupByImageID(imageID int64) (*domain.DuplicateGroup, []domain.DuplicateRelation, error)

	// DeleteDuplicateGroup 删除重复组（级联删除关联关系）
	DeleteDuplicateGroup(id int64) error

	// CountDuplicateGroups 统计重复组总数
	CountDuplicateGroups() (int64, error)

	// DeleteAllDuplicateGroups 删除所有重复组数据
	DeleteAllDuplicateGroups() error
}

type sqliteDuplicateRepository struct {
	db *sql.DB
}

// NewDuplicateRepository 创建重复检测仓储实例
func NewDuplicateRepository(db *sql.DB) DuplicateRepository {
	return &sqliteDuplicateRepository{db: db}
}

func (r *sqliteDuplicateRepository) SaveDuplicateGroup(group *domain.DuplicateGroup, relations []domain.DuplicateRelation) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 插入重复组
	result, err := tx.Exec(`
		INSERT INTO duplicate_groups (recommended_image_id, similarity_threshold, created_at)
		VALUES (?, ?, ?)
	`, group.RecommendedImageID, group.SimilarityThreshold, group.CreatedAt)
	if err != nil {
		return err
	}

	groupID, err := result.LastInsertId()
	if err != nil {
		return err
	}
	group.ID = groupID

	// 插入关联关系
	stmt, err := tx.Prepare(`
		INSERT INTO duplicate_relations (
			group_id, image_id, is_recommended, file_hash, phash_distance, recommendation_score, recommendation_rationale
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, rel := range relations {
		isRecommended := 0
		if rel.IsRecommended {
			isRecommended = 1
		}
		_, err := stmt.Exec(
			groupID,
			rel.ImageID,
			isRecommended,
			rel.FileHash,
			rel.PHashDistance,
			rel.RecommendationScore,
			string(rel.RecommendationRationale),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *sqliteDuplicateRepository) FindDuplicateGroups(limit, offset int) ([]domain.DuplicateGroup, error) {
	rows, err := r.db.Query(`
		SELECT id, recommended_image_id, similarity_threshold, created_at
		FROM duplicate_groups
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := make([]domain.DuplicateGroup, 0)
	for rows.Next() {
		var group domain.DuplicateGroup
		err := rows.Scan(
			&group.ID,
			&group.RecommendedImageID,
			&group.SimilarityThreshold,
			&group.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}

	return groups, rows.Err()
}

func (r *sqliteDuplicateRepository) FindDuplicateGroupByID(id int64) (*domain.DuplicateGroup, []domain.DuplicateRelation, error) {
	var group domain.DuplicateGroup
	err := r.db.QueryRow(`
		SELECT id, recommended_image_id, similarity_threshold, created_at
		FROM duplicate_groups
		WHERE id = ?
	`, id).Scan(
		&group.ID,
		&group.RecommendedImageID,
		&group.SimilarityThreshold,
		&group.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, err
		}
		return nil, nil, err
	}

	relations, err := r.findRelationsByGroupID(group.ID)
	if err != nil {
		return nil, nil, err
	}

	return &group, relations, nil
}

func (r *sqliteDuplicateRepository) FindDuplicateGroupByImageID(imageID int64) (*domain.DuplicateGroup, []domain.DuplicateRelation, error) {
	var group domain.DuplicateGroup
	err := r.db.QueryRow(`
		SELECT dg.id, dg.recommended_image_id, dg.similarity_threshold, dg.created_at
		FROM duplicate_groups dg
		INNER JOIN duplicate_relations dr ON dr.group_id = dg.id
		WHERE dr.image_id = ?
		LIMIT 1
	`, imageID).Scan(
		&group.ID,
		&group.RecommendedImageID,
		&group.SimilarityThreshold,
		&group.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, err
		}
		return nil, nil, err
	}

	relations, err := r.findRelationsByGroupID(group.ID)
	if err != nil {
		return nil, nil, err
	}

	return &group, relations, nil
}

func (r *sqliteDuplicateRepository) findRelationsByGroupID(groupID int64) ([]domain.DuplicateRelation, error) {
	rows, err := r.db.Query(`
		SELECT group_id, image_id, is_recommended, file_hash, phash_distance,
		       COALESCE(recommendation_score, 0), COALESCE(recommendation_rationale, '')
		FROM duplicate_relations
		WHERE group_id = ?
		ORDER BY is_recommended DESC, phash_distance ASC
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	relations := make([]domain.DuplicateRelation, 0)
	for rows.Next() {
		var rel domain.DuplicateRelation
		var isRecommended int
		var recommendationRationale string
		err := rows.Scan(
			&rel.GroupID,
			&rel.ImageID,
			&isRecommended,
			&rel.FileHash,
			&rel.PHashDistance,
			&rel.RecommendationScore,
			&recommendationRationale,
		)
		if err != nil {
			return nil, err
		}
		rel.IsRecommended = isRecommended == 1
		rel.GroupID = groupID
		if recommendationRationale != "" {
			rel.RecommendationRationale = json.RawMessage(recommendationRationale)
		}
		relations = append(relations, rel)
	}

	return relations, rows.Err()
}

func (r *sqliteDuplicateRepository) DeleteDuplicateGroup(id int64) error {
	_, err := r.db.Exec(`DELETE FROM duplicate_groups WHERE id = ?`, id)
	return err
}

func (r *sqliteDuplicateRepository) CountDuplicateGroups() (int64, error) {
	var count int64
	err := r.db.QueryRow(`SELECT COUNT(*) FROM duplicate_groups`).Scan(&count)
	return count, err
}

func (r *sqliteDuplicateRepository) DeleteAllDuplicateGroups() error {
	_, err := r.db.Exec(`DELETE FROM duplicate_groups`)
	return err
}

// FindImagesWithPHash 查找所有有 pHash 的图片
func (r *sqliteDuplicateRepository) FindImagesWithPHash() ([]domain.Image, error) {
	rows, err := r.db.Query(`
		SELECT id, path, filename, source_root, file_size, width, height, format, COALESCE(phash, 0), created_at, updated_at
		FROM images
		WHERE phash IS NOT NULL AND phash != 0
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := make([]domain.Image, 0)
	for rows.Next() {
		var img domain.Image
		err := rows.Scan(
			&img.ID,
			&img.Path,
			&img.Filename,
			&img.SourceRoot,
			&img.FileSize,
			&img.Width,
			&img.Height,
			&img.Format,
			&img.PHash,
			&img.CreatedAt,
			&img.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		images = append(images, img)
	}

	return images, rows.Err()
}

// UpdateImagePHash 更新图片的 pHash 值
func (r *sqliteDuplicateRepository) UpdateImagePHash(imageID int64, pHash int64) error {
	_, err := r.db.Exec(`
		UPDATE images SET phash = ?, updated_at = ? WHERE id = ?
	`, pHash, time.Now(), imageID)
	return err
}
