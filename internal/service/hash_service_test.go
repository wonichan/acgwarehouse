package service

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func TestHashService_CalculateFileHash(t *testing.T) {
	service := NewHashService()

	// 创建临时测试目录
	tmpDir, err := os.MkdirTemp("", "hash_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试图片
	testPath := filepath.Join(tmpDir, "test.jpg")
	createTestImage(t, testPath, 100, 100, color.RGBA{R: 255, G: 0, B: 0, A: 255})

	// Test 1: CalculateFileHash 对已知文件返回正确的 SHA256 哈希
	hash1, err := service.CalculateFileHash(testPath)
	if err != nil {
		t.Fatalf("CalculateFileHash failed: %v", err)
	}
	if hash1 == "" {
		t.Error("Expected non-empty hash")
	}
	if len(hash1) != 64 { // SHA256 返回 64 个十六进制字符
		t.Errorf("Expected hash length 64, got %d", len(hash1))
	}

	// 再次计算同一文件，应该得到相同的哈希
	hash2, err := service.CalculateFileHash(testPath)
	if err != nil {
		t.Fatalf("Second CalculateFileHash failed: %v", err)
	}
	if hash1 != hash2 {
		t.Error("Expected same hash for same file")
	}

	// 创建不同内容的文件，应该得到不同的哈希
	testPath2 := filepath.Join(tmpDir, "test2.jpg")
	createTestImage(t, testPath2, 100, 100, color.RGBA{R: 0, G: 255, B: 0, A: 255})

	hash3, err := service.CalculateFileHash(testPath2)
	if err != nil {
		t.Fatalf("CalculateFileHash for test2 failed: %v", err)
	}
	if hash1 == hash3 {
		t.Error("Expected different hash for different file content")
	}
}

func TestHashService_CalculatePHash(t *testing.T) {
	service := NewHashService()

	// 创建临时测试目录
	tmpDir, err := os.MkdirTemp("", "phash_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test 2: CalculatePHash 对相同图片返回相同哈希
	testPath := filepath.Join(tmpDir, "test.jpg")
	createTestImage(t, testPath, 200, 200, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	phash1, err := service.CalculatePHash(testPath)
	if err != nil {
		t.Fatalf("CalculatePHash failed: %v", err)
	}
	if phash1 == 0 {
		t.Error("Expected non-zero pHash")
	}

	// 再次计算同一文件的 pHash
	phash2, err := service.CalculatePHash(testPath)
	if err != nil {
		t.Fatalf("Second CalculatePHash failed: %v", err)
	}
	if phash1 != phash2 {
		t.Errorf("Expected same pHash for same image, got %d and %d", phash1, phash2)
	}
}

func TestHashService_HammingDistance(t *testing.T) {
	service := NewHashService()

	// Test 3: HammingDistance 正确计算两个 int64 的汉明距离
	tests := []struct {
		hash1    int64
		hash2    int64
		expected int
	}{
		{0, 0, 0},                   // 相同值，距离为 0
		{0, 1, 1},                   // 只有一位不同
		{0b1111, 0b0000, 4},         // 4 位都不同
		{0b10101010, 0b01010101, 8}, // 8 位都不同
		{0x1234567890ABCDEF, 0x1234567890ABCDEF, 0}, // 相同
		{-1, 0, 64}, // -1 在 int64 中所有位都是 1，与 0 的距离是 64
	}

	for _, tt := range tests {
		result := service.HammingDistance(tt.hash1, tt.hash2)
		if result != tt.expected {
			t.Errorf("HammingDistance(%d, %d) = %d, expected %d", tt.hash1, tt.hash2, result, tt.expected)
		}
	}
}

func TestHashService_PHashSimilarImages(t *testing.T) {
	service := NewHashService()

	// 创建临时测试目录
	tmpDir, err := os.MkdirTemp("", "phash_similar_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test 4: 相同图片的 pHash 汉明距离为 0
	// 使用完全相同的图片文件
	testPath1 := filepath.Join(tmpDir, "original.jpg")
	createTestImage(t, testPath1, 200, 200, color.RGBA{R: 100, G: 150, B: 200, A: 255})

	phash1, err := service.CalculatePHash(testPath1)
	if err != nil {
		t.Fatalf("CalculatePHash failed: %v", err)
	}

	distance := service.HammingDistance(phash1, phash1)
	if distance != 0 {
		t.Errorf("Expected Hamming distance 0 for same pHash, got %d", distance)
	}

	// Test 5: 不同图片的 pHash 汉明距离 > 0
	// 创建具有不同结构的图片（渐变 vs 纯色）
	testPath2 := filepath.Join(tmpDir, "gradient.jpg")
	createGradientImage(t, testPath2, 200, 200)

	phash2, err := service.CalculatePHash(testPath2)
	if err != nil {
		t.Fatalf("CalculatePHash for different image failed: %v", err)
	}

	distance = service.HammingDistance(phash1, phash2)
	if distance == 0 {
		t.Error("Expected Hamming distance > 0 for different images")
	}
	t.Logf("Hamming distance between different images: %d (phash1=%d, phash2=%d)", distance, phash1, phash2)
}

func TestHashService_SupportedFormats(t *testing.T) {
	service := NewHashService()

	tmpDir, err := os.MkdirTemp("", "format_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	formats := []struct {
		ext   string
		valid bool
	}{
		{".jpg", true},
		{".jpeg", true},
		{".png", true},
		{".webp", true},
		{".gif", true},
		{".txt", false},
	}

	for _, tt := range formats {
		testPath := filepath.Join(tmpDir, "test"+tt.ext)
		if tt.valid {
			createTestImage(t, testPath, 100, 100, color.RGBA{R: 128, G: 128, B: 128, A: 255})
			_, err := service.CalculatePHash(testPath)
			if err != nil {
				t.Errorf("CalculatePHash failed for %s: %v", tt.ext, err)
			}
		}
	}
}

// 辅助函数：创建测试图片
func createTestImage(t *testing.T, path string, width, height int, c color.RGBA) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create test image file: %v", err)
	}
	defer file.Close()

	if err := jpeg.Encode(file, img, nil); err != nil {
		t.Fatalf("Failed to encode test image: %v", err)
	}
}

// 辅助函数：创建渐变测试图片（具有明显结构特征）
func createGradientImage(t *testing.T, path string, width, height int) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// 创建从左到右的渐变
			r := uint8((x * 255) / width)
			g := uint8((y * 255) / height)
			b := uint8(128)
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create test image file: %v", err)
	}
	defer file.Close()

	if err := jpeg.Encode(file, img, nil); err != nil {
		t.Fatalf("Failed to encode test image: %v", err)
	}
}
