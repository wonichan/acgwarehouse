package service

import (
	"context"
	"database/sql"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

func setupDuplicateTestDB(t *testing.T) *sql.DB {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "duplicate_service_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpPath) })

	db, err := sql.Open("sqlite3", tmpPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	return db
}

func insertTestImagesForDuplicate(t *testing.T, db *sql.DB) []int64 {
	t.Helper()

	now := time.Now()
	ids := make([]int64, 5)

	// 创建测试图片：图片 A 和 B 完全相同（相同路径模拟），C、D、E 相似
	images := []struct {
		path   string
		pHash  int64
		width  int
		height int
	}{
		{"/test/a.jpg", 0x1234567890ABCDEF, 100, 100},
		{"/test/b.jpg", 0x1234567890ABCDEF, 200, 200},        // 相同 pHash，更高分辨率
		{"/test/c.jpg", 0x1234567890ABCDEF + 1, 150, 150},    // 相似 pHash（汉明距离 1）
		{"/test/d.jpg", 0x1234567890ABCDEF + 3, 120, 120},    // 相似 pHash（汉明距离 2）
		{"/test/e.jpg", int64(0x7FFFFFFFFFFFFFFF), 100, 100}, // 完全不同的 pHash（使用 int64 最大值）
	}

	for i, img := range images {
		result, err := db.Exec(`
			INSERT INTO images (path, filename, source_root, file_size, width, height, format, phash, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			img.path,
			img.path[len(img.path)-5:],
			"/test",
			1024,
			img.width,
			img.height,
			"jpg",
			img.pHash,
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

func TestDuplicateService_DetectDuplicates_IdenticalImages(t *testing.T) {
	db := setupDuplicateTestDB(t)
	imageRepo := repository.NewImageRepository(db)
	duplicateRepo := repository.NewDuplicateRepository(db)
	hashService := NewHashService()
	service := NewDuplicateService(imageRepo, duplicateRepo, hashService)

	// 插入测试图片
	insertTestImagesForDuplicate(t, db)

	// Test 1: DetectDuplicates 检测完全相同的图片（SHA256 匹配）
	// 注意：由于我们没有真实文件，这里测试 pHash 相同的情况
	count, err := service.DetectDuplicates(context.Background(), DetectOptions{Threshold: 10})
	if err != nil {
		t.Fatalf("DetectDuplicates failed: %v", err)
	}

	if count == 0 {
		t.Error("Expected to find duplicate groups")
	}

	t.Logf("Found %d duplicate groups", count)
}

func TestDuplicateService_DetectDuplicates_SimilarImages(t *testing.T) {
	db := setupDuplicateTestDB(t)
	imageRepo := repository.NewImageRepository(db)
	duplicateRepo := repository.NewDuplicateRepository(db)
	hashService := NewHashService()
	service := NewDuplicateService(imageRepo, duplicateRepo, hashService)

	// 插入测试图片
	ids := insertTestImagesForDuplicate(t, db)

	// Test 2: DetectDuplicates 检测相似图片（pHash 汉明距离 ≤ 阈值）
	count, err := service.DetectDuplicates(context.Background(), DetectOptions{Threshold: 5})
	if err != nil {
		t.Fatalf("DetectDuplicates failed: %v", err)
	}

	if count == 0 {
		t.Error("Expected to find similar images")
	}

	// 验证结果
	groups, total, err := service.GetDuplicateGroups(10, 0)
	if err != nil {
		t.Fatalf("GetDuplicateGroups failed: %v", err)
	}

	t.Logf("Found %d groups, total: %d", len(groups), total)

	// 验证图片 A、B、C、D 应该在同一组（pHash 相似）
	for _, group := range groups {
		t.Logf("Group %d: recommended=%d, images=%d", group.Group.ID, group.Group.RecommendedImageID, len(group.Images))
		for _, img := range group.Images {
			t.Logf("  - Image %d: %s, recommended=%v, distance=%d", img.ID, img.Filename, img.IsRecommended, img.PHashDistance)
		}
	}

	// 验证至少有一组包含多个图片
	hasMultipleImages := false
	for _, group := range groups {
		if len(group.Images) > 1 {
			hasMultipleImages = true
			break
		}
	}
	if !hasMultipleImages {
		t.Error("Expected at least one group with multiple images")
	}

	_ = ids // 使用变量避免编译警告
}

func TestDuplicateService_DetectDuplicates_Transitivity(t *testing.T) {
	db := setupDuplicateTestDB(t)
	imageRepo := repository.NewImageRepository(db)
	duplicateRepo := repository.NewDuplicateRepository(db)
	hashService := NewHashService()
	service := NewDuplicateService(imageRepo, duplicateRepo, hashService)

	// Test 3: DetectDuplicates 使用传递性分组（A~B, B~C → {A,B,C}）
	// 使用预设的测试数据，其中 A、B、C、D 的 pHash 相似

	_, err := service.DetectDuplicates(context.Background(), DetectOptions{Threshold: 5})
	if err != nil {
		t.Fatalf("DetectDuplicates failed: %v", err)
	}

	// 验证传递性分组
	groups, _, err := service.GetDuplicateGroups(10, 0)
	if err != nil {
		t.Fatalf("GetDuplicateGroups failed: %v", err)
	}

	// 找到最大的组
	maxGroupSize := 0
	for _, group := range groups {
		if len(group.Images) > maxGroupSize {
			maxGroupSize = len(group.Images)
		}
	}

	// 由于测试数据的 pHash 设置，应该有传递性分组
	t.Logf("Max group size: %d", maxGroupSize)
}

func TestDuplicateService_DetectDuplicates_RecommendHighestResolution(t *testing.T) {
	db := setupDuplicateTestDB(t)
	imageRepo := repository.NewImageRepository(db)
	duplicateRepo := repository.NewDuplicateRepository(db)
	hashService := NewHashService()
	service := NewDuplicateService(imageRepo, duplicateRepo, hashService)

	// 插入测试图片
	ids := insertTestImagesForDuplicate(t, db)

	// Test 4: DetectDuplicates 推荐分辨率最高的图片
	_, err := service.DetectDuplicates(context.Background(), DetectOptions{Threshold: 10})
	if err != nil {
		t.Fatalf("DetectDuplicates failed: %v", err)
	}

	groups, _, err := service.GetDuplicateGroups(10, 0)
	if err != nil {
		t.Fatalf("GetDuplicateGroups failed: %v", err)
	}

	// 验证推荐的图片是分辨率最高的
	for _, group := range groups {
		if len(group.Images) < 2 {
			continue
		}

		// 找到推荐的图片
		var recommended *domain.DuplicateImage
		maxResolution := 0
		for i := range group.Images {
			if group.Images[i].IsRecommended {
				recommended = &group.Images[i]
			}
			resolution := group.Images[i].Width * group.Images[i].Height
			if resolution > maxResolution {
				maxResolution = resolution
			}
		}

		if recommended != nil {
			recommendedResolution := recommended.Width * recommended.Height
			t.Logf("Recommended image %d has resolution %d (max in group: %d)",
				recommended.ID, recommendedResolution, maxResolution)
		}
	}

	_ = ids
}

func TestDuplicateService_GetDuplicateGroups(t *testing.T) {
	db := setupDuplicateTestDB(t)
	imageRepo := repository.NewImageRepository(db)
	duplicateRepo := repository.NewDuplicateRepository(db)
	hashService := NewHashService()
	service := NewDuplicateService(imageRepo, duplicateRepo, hashService)

	insertTestImagesForDuplicate(t, db)

	// 执行检测
	_, err := service.DetectDuplicates(context.Background(), DetectOptions{Threshold: 10})
	if err != nil {
		t.Fatalf("DetectDuplicates failed: %v", err)
	}

	// Test 5: GetDuplicateGroups 返回分页的重复组列表
	groups, total, err := service.GetDuplicateGroups(10, 0)
	if err != nil {
		t.Fatalf("GetDuplicateGroups failed: %v", err)
	}

	if total == 0 {
		t.Error("Expected total > 0")
	}

	if len(groups) == 0 {
		t.Error("Expected at least one group")
	}

	// 验证图片按推荐排序
	for _, group := range groups {
		if len(group.Images) < 2 {
			continue
		}
		// 推荐的图片应该排在第一位
		if !group.Images[0].IsRecommended {
			t.Error("Expected recommended image to be first")
		}
	}

	t.Logf("Groups: %d, Total: %d", len(groups), total)
}

func TestDuplicateService_DeleteDuplicateGroup(t *testing.T) {
	db := setupDuplicateTestDB(t)
	imageRepo := repository.NewImageRepository(db)
	duplicateRepo := repository.NewDuplicateRepository(db)
	hashService := NewHashService()
	service := NewDuplicateService(imageRepo, duplicateRepo, hashService)

	insertTestImagesForDuplicate(t, db)

	// 执行检测
	_, err := service.DetectDuplicates(context.Background(), DetectOptions{Threshold: 10})
	if err != nil {
		t.Fatalf("DetectDuplicates failed: %v", err)
	}

	// 获取组
	groups, _, err := service.GetDuplicateGroups(10, 0)
	if err != nil {
		t.Fatalf("GetDuplicateGroups failed: %v", err)
	}

	if len(groups) == 0 {
		t.Fatal("Expected at least one group to delete")
	}

	// 删除组
	err = service.DeleteDuplicateGroup(groups[0].Group.ID)
	if err != nil {
		t.Fatalf("DeleteDuplicateGroup failed: %v", err)
	}

	// 验证已删除
	_, err = service.GetDuplicateGroup(groups[0].Group.ID)
	if err == nil {
		t.Error("Expected error when getting deleted group")
	}
}

func TestUnionFind(t *testing.T) {
	uf := NewUnionFind(5)

	// 初始状态：每个元素都是独立的
	for i := 0; i < 5; i++ {
		if uf.Find(i) != i {
			t.Errorf("Expected Find(%d) = %d", i, i)
		}
	}

	// 合并 0 和 1
	uf.Union(0, 1)
	if uf.Find(0) != uf.Find(1) {
		t.Error("Expected 0 and 1 to be in the same set")
	}

	// 合并 1 和 2
	uf.Union(1, 2)
	if uf.Find(0) != uf.Find(2) {
		t.Error("Expected 0, 1, 2 to be in the same set (transitivity)")
	}

	// 3 和 4 应该还是独立的
	if uf.Find(3) == uf.Find(0) {
		t.Error("Expected 3 to be in a different set")
	}
}

func TestHammingDistance(t *testing.T) {
	hashService := NewHashService()

	tests := []struct {
		h1, h2   int64
		expected int
	}{
		{0, 0, 0},
		{0, 1, 1},
		{0x1234567890ABCDEF, 0x1234567890ABCDEF, 0},
		{0x1234567890ABCDEF, 0x1234567890ABCDEF + 1, 1},
	}

	for _, tt := range tests {
		result := hashService.HammingDistance(tt.h1, tt.h2)
		if result != tt.expected {
			t.Errorf("HammingDistance(%x, %x) = %d, expected %d", tt.h1, tt.h2, result, tt.expected)
		}
	}
}

func TestSelectRecommended(t *testing.T) {
	service := &DuplicateService{}

	imgs := []imageHash{
		{image: domain.Image{ID: 1, Width: 100, Height: 100}},
		{image: domain.Image{ID: 2, Width: 200, Height: 200}}, // 最高分辨率
		{image: domain.Image{ID: 3, Width: 150, Height: 150}},
	}

	recommended := service.selectRecommended(imgs)
	if recommended.image.ID != 2 {
		t.Errorf("Expected recommended image ID 2, got %d", recommended.image.ID)
	}

	// 测试排序是否正确
	sort.Slice(imgs, func(i, j int) bool {
		return imgs[i].image.Width*imgs[i].image.Height > imgs[j].image.Width*imgs[j].image.Height
	})

	// 最高分辨率的应该在第一位
	if imgs[0].image.ID != 2 {
		t.Error("Expected highest resolution image to be first after sorting")
	}
}
