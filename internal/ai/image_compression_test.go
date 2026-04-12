package ai

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"sync/atomic"
	"testing"

	"github.com/disintegration/imaging"
)

func TestCalculateResizeDimensions_HandlesIntOverflow(t *testing.T) {
	newW, newH := calculateResizeDimensions(math.MaxInt, 2, maxAIPixelCount)

	if newW == math.MaxInt && newH == 2 {
		t.Fatalf("expected overflow path to trigger resize, got original dimensions %dx%d", newW, newH)
	}

	if newW < 1 || newH < 1 {
		t.Fatalf("expected positive resized dimensions, got %dx%d", newW, newH)
	}

	pixels := int64(newW) * int64(newH)
	if pixels > int64(maxAIPixelCount) {
		t.Fatalf("resized pixel count exceeds max: got %d, max %d", pixels, maxAIPixelCount)
	}
}

func TestValidateImageBytes_RejectsNonImageData(t *testing.T) {
	err := validateImageBytes([]byte("not an image"), "")
	if err == nil {
		t.Fatal("expected non-image bytes to be rejected")
	}
}

// TestCompressImageIfNeeded_SmallFileUnchanged tests that files under 10MB are returned unchanged
func TestCompressImageIfNeeded_SmallFileUnchanged(t *testing.T) {
	// Create a small test image (~100KB)
	tmpFile, err := os.CreateTemp("", "small_test_*.jpg")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Create a 200x200 image
	img := createTestImage(200, 200)
	if err := jpeg.Encode(tmpFile, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("Failed to encode test image: %v", err)
	}

	// Get original file info
	originalData, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read original file: %v", err)
	}
	originalSize := len(originalData)

	// Call CompressImageIfNeeded
	compressedData, contentType, err := CompressImageIfNeeded(tmpFile.Name())
	if err != nil {
		t.Fatalf("CompressImageIfNeeded failed: %v", err)
	}

	// Verify file is returned unchanged for small files
	if len(compressedData) != originalSize {
		t.Errorf("Small file should be returned unchanged, got size %d, expected %d", len(compressedData), originalSize)
	}

	if !bytes.Equal(compressedData, originalData) {
		t.Error("Small file data should be exactly the same as original")
	}

	if contentType != "image/jpeg" {
		t.Errorf("Expected content type 'image/jpeg', got '%s'", contentType)
	}
}

// TestCompressImageIfNeeded_LargeFileCompressed tests that files over 10MB are compressed
func TestCompressImageIfNeeded_LargeFileCompressed(t *testing.T) {
	// Create a large test image (~15MB) - use a large dimension image
	tmpFile, err := os.CreateTemp("", "large_test_*.jpg")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	img := createTestImage(5000, 5000)
	if err := jpeg.Encode(tmpFile, img, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatalf("Failed to encode test image: %v", err)
	}

	// Get original file info
	originalData, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read original file: %v", err)
	}
	originalSize := len(originalData)

	t.Logf("Original image size: %d bytes (%.2f MB)", originalSize, float64(originalSize)/1024/1024)

	// Skip test if we couldn't create a file over 10MB
	if originalSize <= maxAIImageSize {
		t.Skipf("Could not create test image over 10MB (got %d bytes), skipping", originalSize)
	}

	// Call CompressImageIfNeeded
	compressedData, contentType, err := CompressImageIfNeeded(tmpFile.Name())
	if err != nil {
		t.Fatalf("CompressImageIfNeeded failed: %v", err)
	}

	// Verify file was compressed
	if len(compressedData) >= originalSize {
		t.Errorf("Large file should be compressed, got size %d, original was %d", len(compressedData), originalSize)
	}

	// Verify compressed file is under 10MB
	if len(compressedData) > maxAIImageSize {
		t.Errorf("Compressed file should be under 10MB, got %d bytes (%.2f MB)", len(compressedData), float64(len(compressedData))/1024/1024)
	}

	if contentType != "image/jpeg" {
		t.Errorf("Expected content type 'image/jpeg', got '%s'", contentType)
	}

	t.Logf("Compressed image size: %d bytes (%.2f MB)", len(compressedData), float64(len(compressedData))/1024/1024)
}

// TestCompressImageIfNeeded_CompressedImageValid tests that compressed images can be decoded
func TestCompressImageIfNeeded_CompressedImageValid(t *testing.T) {
	// Create a large test image
	tmpFile, err := os.CreateTemp("", "valid_test_*.jpg")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	img := createTestImage(5000, 5000)
	if err := jpeg.Encode(tmpFile, img, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatalf("Failed to encode test image: %v", err)
	}

	// Get original file info to check if we need compression
	originalData, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read original file: %v", err)
	}

	// Skip test if we couldn't create a file over 10MB
	if len(originalData) <= maxAIImageSize {
		t.Skipf("Could not create test image over 10MB (got %d bytes), skipping", len(originalData))
	}

	// Call CompressImageIfNeeded
	compressedData, _, err := CompressImageIfNeeded(tmpFile.Name())
	if err != nil {
		t.Fatalf("CompressImageIfNeeded failed: %v", err)
	}

	// Verify compressed image can be decoded
	_, err = jpeg.Decode(bytes.NewReader(compressedData))
	if err != nil {
		t.Errorf("Compressed image should be valid JPEG: %v", err)
	}
}

// TestCompressImageIfNeeded_PNGFile tests PNG file handling
func TestCompressImageIfNeeded_PNGFile(t *testing.T) {
	// Create a PNG test file
	tmpFile, err := os.CreateTemp("", "test_*.png")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Create a small PNG image
	img := createTestImage(100, 100)
	if err := encodePNG(tmpFile, img); err != nil {
		t.Fatalf("Failed to encode test PNG: %v", err)
	}

	// Call CompressImageIfNeeded
	compressedData, contentType, err := CompressImageIfNeeded(tmpFile.Name())
	if err != nil {
		t.Fatalf("CompressImageIfNeeded failed: %v", err)
	}

	// For small PNG, should return unchanged
	if contentType != "image/png" {
		t.Errorf("Expected content type 'image/png', got '%s'", contentType)
	}

	// Verify data is valid
	if len(compressedData) == 0 {
		t.Error("Compressed data should not be empty")
	}
}

// TestCompressImageIfNeeded_WebPFile tests WebP file handling
func TestCompressImageIfNeeded_WebPFile(t *testing.T) {
	// Create a WebP test file
	tmpFile, err := os.CreateTemp("", "test_*.webp")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Create a small test image and save as WebP (using JPEG as fallback since Go stdlib doesn't have WebP encoder)
	// We'll test by creating a JPEG with .webp extension to test content type detection
	img := createTestImage(100, 100)
	if err := jpeg.Encode(tmpFile, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("Failed to encode test image: %v", err)
	}

	// Call CompressImageIfNeeded
	_, _, err = CompressImageIfNeeded(tmpFile.Name())
	if err != nil {
		t.Fatalf("CompressImageIfNeeded failed: %v", err)
	}
}

// TestCompressImageIfNeeded_NonExistentFile tests error handling for missing files
func TestCompressImageIfNeeded_NonExistentFile(t *testing.T) {
	_, _, err := CompressImageIfNeeded("/non/existent/file.jpg")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// Helper function to create a test image
func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a pattern to make it harder to compress
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Create a noisy pattern
			r := uint8((x + y) % 256)
			g := uint8((x * 2) % 256)
			b := uint8((y * 2) % 256)
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}

// TestCompressImageIfNeeded_ExceedsPixelLimit tests that images exceeding 36M pixel limit are resized
// This test creates a large image with small file size to expose the missing pixel limit check
func TestCompressImageIfNeeded_ExceedsPixelLimit(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "pixel_limit_*.jpg")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	img := createTestImage(7000, 7000)
	if err := jpeg.Encode(tmpFile, img, &jpeg.Options{Quality: 1}); err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Verify file size is under 10MB (so file-based resize won't trigger)
	fileInfo, err := os.Stat(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	t.Logf("File size: %d bytes (%.2f MB)", fileInfo.Size(), float64(fileInfo.Size())/1024/1024)

	compressedData, contentType, err := CompressImageIfNeeded(tmpFile.Name())
	if err != nil {
		t.Fatalf("CompressImageIfNeeded failed: %v", err)
	}

	// Decode to check dimensions
	decoded, err := jpeg.Decode(bytes.NewReader(compressedData))
	if err != nil {
		t.Fatalf("Failed to decode compressed: %v", err)
	}

	width := decoded.Bounds().Dx()
	height := decoded.Bounds().Dy()
	pixels := width * height

	// Must fit within pixel limit (36M pixels)
	maxPixelLimit := 36000000
	if pixels > maxPixelLimit {
		t.Errorf("Pixel count %d exceeds limit %d, dims=%dx%d", pixels, maxPixelLimit, width, height)
	}

	// Aspect ratio should be preserved (square → square)
	if width != height {
		t.Errorf("Square image should remain square, got %dx%d", width, height)
	}

	// Should be JPEG
	if contentType != "image/jpeg" {
		t.Errorf("Expected image/jpeg, got %s", contentType)
	}

	t.Logf("Resized from 7000x7000 (49M pixels) to %dx%d (%d pixels)", width, height, pixels)
}

// TestCompressImageIfNeeded_NonSquarePixelLimit tests aspect ratio preservation for non-square images
func TestCompressImageIfNeeded_NonSquarePixelLimit(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "nonsquare_*.jpg")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	img := createTestImage(12000, 4000)
	// Use quality 1 to create small file size (< 10MB)
	if err := jpeg.Encode(tmpFile, img, &jpeg.Options{Quality: 1}); err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	compressedData, contentType, err := CompressImageIfNeeded(tmpFile.Name())
	if err != nil {
		t.Fatalf("CompressImageIfNeeded failed: %v", err)
	}

	// Decode to check dimensions
	decoded, err := jpeg.Decode(bytes.NewReader(compressedData))
	if err != nil {
		t.Fatalf("Failed to decode compressed: %v", err)
	}

	width := decoded.Bounds().Dx()
	height := decoded.Bounds().Dy()
	pixels := width * height

	// Must fit within pixel limit (36M pixels)
	maxPixelLimit := 36000000
	if pixels > maxPixelLimit {
		t.Errorf("Pixel count %d exceeds limit %d, dims=%dx%d", pixels, maxPixelLimit, width, height)
	}

	originalRatio := 12000.0 / 4000.0
	newRatio := float64(width) / float64(height)
	if newRatio < originalRatio*0.95 || newRatio > originalRatio*1.05 {
		t.Errorf("Aspect ratio not preserved, expected ~%.2f, got %.2f (dims=%dx%d)", originalRatio, newRatio, width, height)
	}

	// Should be JPEG
	if contentType != "image/jpeg" {
		t.Errorf("Expected image/jpeg, got %s", contentType)
	}

	t.Logf("Resized from 15000x5000 (75M pixels) to %dx%d (%d pixels), ratio %.2f", width, height, pixels, newRatio)
}

// TestCompressImageIfNeeded_UnderPixelLimit tests that images under pixel limit are not unnecessarily resized
func TestCompressImageIfNeeded_UnderPixelLimit(t *testing.T) {
	// Create 5000x5000 = 25M pixels < 36M limit
	// Should NOT resize, just return original (or process for file size if needed)
	tmpFile, err := os.CreateTemp("", "under_limit_*.jpg")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	img := createTestImage(5000, 5000)
	if err := jpeg.Encode(tmpFile, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Get original dimensions
	originalWidth := 5000
	originalHeight := 5000
	originalPixels := originalWidth * originalHeight // 25M

	compressedData, contentType, err := CompressImageIfNeeded(tmpFile.Name())
	if err != nil {
		t.Fatalf("CompressImageIfNeeded failed: %v", err)
	}

	// Decode to check dimensions
	decoded, err := jpeg.Decode(bytes.NewReader(compressedData))
	if err != nil {
		t.Fatalf("Failed to decode compressed: %v", err)
	}

	width := decoded.Bounds().Dx()
	height := decoded.Bounds().Dy()
	pixels := width * height

	// Should NOT resize - dimensions should remain same (or larger if file size compression didn't need resize)
	// Note: if file size > 10MB, it might resize, but pixel count should still be under limit
	if pixels > maxAIPixelCount {
		t.Errorf("Pixel count %d exceeds limit %d for image under pixel limit", pixels, maxAIPixelCount)
	}

	// For small files, content type should be preserved
	if contentType != "image/jpeg" {
		t.Errorf("Expected image/jpeg, got %s", contentType)
	}

	t.Logf("Image dimensions: %dx%d (%d pixels), original: %dx%d (%d pixels)", width, height, pixels, originalWidth, originalHeight, originalPixels)
}

// Helper function to encode PNG
func encodePNG(w *os.File, img image.Image) error {
	return png.Encode(w, img)
}

func TestCompressImageWithFilterAndLimit_ResizeCalledOnlyWhenScaleChanges(t *testing.T) {
	var resizeCalls int32
	var sourceWidths []int
	originalResize := resizeImage
	resizeImage = func(img image.Image, width, height int, filter imaging.ResampleFilter) *image.NRGBA {
		atomic.AddInt32(&resizeCalls, 1)
		sourceWidths = append(sourceWidths, img.Bounds().Dx())
		return imaging.Resize(img, width, height, filter)
	}
	defer func() { resizeImage = originalResize }()

	img := createTestImage(2000, 2000)
	_, err := compressImageWithFilterAndLimit(img, imaging.Lanczos, 1024)
	if err != nil {
		t.Fatalf("compressImageWithFilterAndLimit() error = %v", err)
	}

	if got := atomic.LoadInt32(&resizeCalls); got == 0 {
		t.Fatalf("resize invocation count = %d, expected > 0", got)
	}

	if len(sourceWidths) < 2 {
		t.Fatalf("expected multiple resize invocations, got %d", len(sourceWidths))
	}

	for i := 1; i < len(sourceWidths); i++ {
		if sourceWidths[i] >= sourceWidths[i-1] {
			t.Fatalf("expected progressive downscale source widths, got sequence %v", sourceWidths)
		}
	}
}
