package ai

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"testing"
)

const maxAIImageSize = 10 * 1024 * 1024 // 10MB

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

	// Create a large image (4000x4000 should produce >10MB with quality 100)
	img := createTestImage(4000, 4000)
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

	// Create a large image
	img := createTestImage(4000, 4000)
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

// Helper function to encode PNG
func encodePNG(w *os.File, img image.Image) error {
	return png.Encode(w, img)
}
