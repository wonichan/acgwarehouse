package repository

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// 创建临时数据库文件
	tmpFile, err := os.CreateTemp("", "duplicate_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpPath) })

	// 打开数据库连接
	db, err := sql.Open("sqlite3", tmpPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// 初始化 schema
	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	return db
}

func insertTestImages(t *testing.T, db *sql.DB) []int64 {
	t.Helper()

	now := time.Now()
	ids := make([]int64, 3)

	for i := 0; i < 3; i++ {
		result, err := db.Exec(`
			INSERT INTO images (path, filename, source_root, file_size, width, height, format, phash, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			"/path/image"+string(rune('A'+i))+".jpg",
			"image"+string(rune('A'+i))+".jpg",
			"/path",
			1024*int64(i+1),
			100*(i+1),
			100*(i+1),
			"jpg",
			12345+int64(i),
			now,
			now,
		)
		if err != nil {
			t.Fatalf("Failed to insert test image: %v", err)
		}
		id, _ := result.LastInsertId()
		ids[i] = id
	}

	return ids
}

func TestDuplicateRepository_SaveDuplicateGroup(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDuplicateRepository(db)
	imageIDs := insertTestImages(t, db)

	// Test 1: SaveDuplicateGroup 保存重复组并返回 ID
	group := &domain.DuplicateGroup{
		RecommendedImageID:  imageIDs[0],
		SimilarityThreshold: 10,
		CreatedAt:           time.Now(),
	}

	relations := []domain.DuplicateRelation{
		{
			ImageID:       imageIDs[0],
			IsRecommended: true,
			FileHash:      "hash1",
			PHashDistance: 0,
		},
		{
			ImageID:       imageIDs[1],
			IsRecommended: false,
			FileHash:      "hash2",
			PHashDistance: 5,
		},
		{
			ImageID:       imageIDs[2],
			IsRecommended: false,
			FileHash:      "hash3",
			PHashDistance: 8,
		},
	}

	err := repo.SaveDuplicateGroup(group, relations)
	if err != nil {
		t.Fatalf("SaveDuplicateGroup failed: %v", err)
	}

	if group.ID == 0 {
		t.Error("Expected group ID to be set after save")
	}
}

func TestDuplicateRepository_FindDuplicateGroups(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDuplicateRepository(db)
	imageIDs := insertTestImages(t, db)

	// 创建测试数据
	group := &domain.DuplicateGroup{
		RecommendedImageID:  imageIDs[0],
		SimilarityThreshold: 10,
		CreatedAt:           time.Now(),
	}
	relations := []domain.DuplicateRelation{
		{ImageID: imageIDs[0], IsRecommended: true, FileHash: "hash1", PHashDistance: 0},
		{ImageID: imageIDs[1], IsRecommended: false, FileHash: "hash2", PHashDistance: 5},
	}
	repo.SaveDuplicateGroup(group, relations)

	// Test 2: FindDuplicateGroups 返回所有重复组
	groups, err := repo.FindDuplicateGroups(10, 0)
	if err != nil {
		t.Fatalf("FindDuplicateGroups failed: %v", err)
	}

	if len(groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(groups))
	}

	if groups[0].RecommendedImageID != imageIDs[0] {
		t.Errorf("Expected recommended image ID %d, got %d", imageIDs[0], groups[0].RecommendedImageID)
	}
}

func TestDuplicateRepository_FindDuplicateGroupByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDuplicateRepository(db)
	imageIDs := insertTestImages(t, db)

	// 创建测试数据
	group := &domain.DuplicateGroup{
		RecommendedImageID:  imageIDs[0],
		SimilarityThreshold: 15,
		CreatedAt:           time.Now(),
	}
	relations := []domain.DuplicateRelation{
		{ImageID: imageIDs[0], IsRecommended: true, FileHash: "hash1", PHashDistance: 0},
		{ImageID: imageIDs[1], IsRecommended: false, FileHash: "hash2", PHashDistance: 7},
	}
	repo.SaveDuplicateGroup(group, relations)

	// Test 3: FindDuplicateGroupByID 返回重复组详情
	foundGroup, foundRelations, err := repo.FindDuplicateGroupByID(group.ID)
	if err != nil {
		t.Fatalf("FindDuplicateGroupByID failed: %v", err)
	}

	if foundGroup.ID != group.ID {
		t.Errorf("Expected group ID %d, got %d", group.ID, foundGroup.ID)
	}

	if len(foundRelations) != 2 {
		t.Errorf("Expected 2 relations, got %d", len(foundRelations))
	}
}

func TestDuplicateRepository_FindDuplicateGroupByImageID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDuplicateRepository(db)
	imageIDs := insertTestImages(t, db)

	// 创建测试数据
	group := &domain.DuplicateGroup{
		RecommendedImageID:  imageIDs[0],
		SimilarityThreshold: 10,
		CreatedAt:           time.Now(),
	}
	relations := []domain.DuplicateRelation{
		{ImageID: imageIDs[0], IsRecommended: true, FileHash: "hash1", PHashDistance: 0},
		{ImageID: imageIDs[1], IsRecommended: false, FileHash: "hash2", PHashDistance: 3},
	}
	repo.SaveDuplicateGroup(group, relations)

	// Test 4: FindDuplicateGroupByImageID 返回图片所属的重复组
	foundGroup, foundRelations, err := repo.FindDuplicateGroupByImageID(imageIDs[1])
	if err != nil {
		t.Fatalf("FindDuplicateGroupByImageID failed: %v", err)
	}

	if foundGroup.ID != group.ID {
		t.Errorf("Expected group ID %d, got %d", group.ID, foundGroup.ID)
	}

	if len(foundRelations) != 2 {
		t.Errorf("Expected 2 relations, got %d", len(foundRelations))
	}
}

func TestDuplicateRepository_DeleteDuplicateGroup(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDuplicateRepository(db)
	imageIDs := insertTestImages(t, db)

	// 创建测试数据
	group := &domain.DuplicateGroup{
		RecommendedImageID:  imageIDs[0],
		SimilarityThreshold: 10,
		CreatedAt:           time.Now(),
	}
	relations := []domain.DuplicateRelation{
		{ImageID: imageIDs[0], IsRecommended: true, FileHash: "hash1", PHashDistance: 0},
		{ImageID: imageIDs[1], IsRecommended: false, FileHash: "hash2", PHashDistance: 5},
	}
	repo.SaveDuplicateGroup(group, relations)

	// Test 5: DeleteDuplicateGroup 删除重复组
	err := repo.DeleteDuplicateGroup(group.ID)
	if err != nil {
		t.Fatalf("DeleteDuplicateGroup failed: %v", err)
	}

	// 验证已删除
	_, _, err = repo.FindDuplicateGroupByID(group.ID)
	if err == nil {
		t.Error("Expected error when finding deleted group")
	}
}

func TestDuplicateRepository_CountDuplicateGroups(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDuplicateRepository(db)
	imageIDs := insertTestImages(t, db)

	// 初始计数
	count, err := repo.CountDuplicateGroups()
	if err != nil {
		t.Fatalf("CountDuplicateGroups failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 groups initially, got %d", count)
	}

	// 创建测试数据
	group := &domain.DuplicateGroup{
		RecommendedImageID:  imageIDs[0],
		SimilarityThreshold: 10,
		CreatedAt:           time.Now(),
	}
	relations := []domain.DuplicateRelation{
		{ImageID: imageIDs[0], IsRecommended: true, FileHash: "hash1", PHashDistance: 0},
		{ImageID: imageIDs[1], IsRecommended: false, FileHash: "hash2", PHashDistance: 5},
	}
	repo.SaveDuplicateGroup(group, relations)

	// 再次计数
	count, err = repo.CountDuplicateGroups()
	if err != nil {
		t.Fatalf("CountDuplicateGroups failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 group, got %d", count)
	}
}

func TestDuplicateRelation_RecommendationColumns(t *testing.T) {
	db := setupTestDB(t)
	imageIDs := insertTestImages(t, db)

	if _, err := db.Exec(`
		INSERT INTO duplicate_groups (recommended_image_id, similarity_threshold, created_at)
		VALUES (?, ?, ?)
	`, imageIDs[0], 10, time.Now()); err != nil {
		t.Fatalf("insert duplicate group: %v", err)
	}

	var groupID int64
	if err := db.QueryRow(`SELECT id FROM duplicate_groups ORDER BY id DESC LIMIT 1`).Scan(&groupID); err != nil {
		t.Fatalf("query duplicate group id: %v", err)
	}

	rationale := `{"reasons":[{"factor":"resolution","weight":0.5}],"score":85.5}`
	if _, err := db.Exec(`
		INSERT INTO duplicate_relations (
			group_id, image_id, is_recommended, file_hash, phash_distance, recommendation_score, recommendation_rationale
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, groupID, imageIDs[0], 1, "hash-roundtrip", 0, 85.5, rationale); err != nil {
		t.Fatalf("insert relation with recommendation columns: %v", err)
	}

	var score float64
	var rawRationale string
	if err := db.QueryRow(`
		SELECT recommendation_score, recommendation_rationale
		FROM duplicate_relations
		WHERE group_id = ? AND image_id = ?
	`, groupID, imageIDs[0]).Scan(&score, &rawRationale); err != nil {
		t.Fatalf("select relation with recommendation columns: %v", err)
	}

	if score != 85.5 {
		t.Fatalf("recommendation_score = %v, want 85.5", score)
	}
	if rawRationale != rationale {
		t.Fatalf("recommendation_rationale = %q, want %q", rawRationale, rationale)
	}
}
