package service

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/disintegration/imaging"
)

func TestThumbnailServiceGenerateThumbnailReturnsDataAndDimensions(t *testing.T) {
	t.Parallel()

	imgPath := writePNGFixture(t, 1200, 800)
	svc := NewThumbnailService()

	thumb, err := svc.GenerateThumbnail(imgPath, "small")
	if err != nil {
		t.Fatalf("GenerateThumbnail() error = %v", err)
	}
	if thumb.Size != "small" {
		t.Fatalf("Size = %q, want small", thumb.Size)
	}
	if thumb.Width != 200 || thumb.Height != 133 {
		t.Fatalf("thumbnail dimensions = %dx%d, want 200x133", thumb.Width, thumb.Height)
	}
	if len(thumb.Data) == 0 {
		t.Fatal("thumbnail data is empty")
	}
}

func TestThumbnailServiceGenerateThumbnailLargeDimensions(t *testing.T) {
	t.Parallel()

	imgPath := writePNGFixture(t, 1200, 800)
	svc := NewThumbnailService()

	thumb, err := svc.GenerateThumbnail(imgPath, "large")
	if err != nil {
		t.Fatalf("GenerateThumbnail() error = %v", err)
	}
	if thumb.Width != 600 || thumb.Height != 400 {
		t.Fatalf("thumbnail dimensions = %dx%d, want 600x400", thumb.Width, thumb.Height)
	}
}

func TestThumbnailServiceGenerateThumbnailAppliesJPEGQuality(t *testing.T) {
	t.Parallel()

	imgPath := writePNGFixture(t, 1200, 800)
	svc := NewThumbnailService()

	small, err := svc.GenerateThumbnail(imgPath, "small")
	if err != nil {
		t.Fatalf("GenerateThumbnail(small) error = %v", err)
	}
	large, err := svc.GenerateThumbnail(imgPath, "large")
	if err != nil {
		t.Fatalf("GenerateThumbnail(large) error = %v", err)
	}

	src, err := imaging.Open(imgPath)
	if err != nil {
		t.Fatalf("imaging.Open() error = %v", err)
	}

	expectedSmall := encodeExpectedJPEG(t, imaging.Resize(src, svc.SmallWidth, 0, imaging.Lanczos), svc.SmallQuality)
	expectedLarge := encodeExpectedJPEG(t, imaging.Resize(src, svc.LargeWidth, 0, imaging.Lanczos), svc.LargeQuality)

	if !bytes.Equal(small.Data, expectedSmall) {
		t.Fatal("small thumbnail bytes differ from expected JPEG encoding at quality 85")
	}
	if !bytes.Equal(large.Data, expectedLarge) {
		t.Fatal("large thumbnail bytes differ from expected JPEG encoding at quality 90")
	}

	if _, err := jpeg.Decode(bytes.NewReader(small.Data)); err != nil {
		t.Fatalf("small thumbnail is not valid jpeg: %v", err)
	}
	if _, err := jpeg.Decode(bytes.NewReader(large.Data)); err != nil {
		t.Fatalf("large thumbnail is not valid jpeg: %v", err)
	}
}

func TestThumbnailServiceGenerateThumbnailInvalidPath(t *testing.T) {
	t.Parallel()

	svc := NewThumbnailService()
	if _, err := svc.GenerateThumbnail(filepath.Join(t.TempDir(), "missing.png"), "small"); err == nil {
		t.Fatal("GenerateThumbnail() expected error for missing file")
	}
}

func TestThumbnailServiceGenerateThumbnailUnsupportedFormat(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "not-image.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write txt fixture: %v", err)
	}

	svc := NewThumbnailService()
	if _, err := svc.GenerateThumbnail(path, "small"); err == nil {
		t.Fatal("GenerateThumbnail() expected error for unsupported format")
	}
}

func TestThumbnailServiceGenerateBoth(t *testing.T) {
	t.Parallel()

	imgPath := writePNGFixture(t, 1200, 800)
	svc := NewThumbnailService()

	small, large, err := svc.GenerateBoth(imgPath)
	if err != nil {
		t.Fatalf("GenerateBoth() error = %v", err)
	}
	if small == nil || large == nil {
		t.Fatal("GenerateBoth() returned nil thumbnail")
	}
	if small.Size != "small" || large.Size != "large" {
		t.Fatalf("unexpected sizes: small=%q large=%q", small.Size, large.Size)
	}
}

func writePNGFixture(t *testing.T, width, height int) string {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8((x * 255) / width), G: uint8((y * 255) / height), B: 120, A: 255})
		}
	}

	path := filepath.Join(t.TempDir(), "fixture.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create fixture file: %v", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode fixture png: %v", err)
	}

	return path
}

func encodeExpectedJPEG(t *testing.T, img image.Image, quality int) []byte {
	t.Helper()

	var buf bytes.Buffer
	if err := imaging.Encode(&buf, img, imaging.JPEG, imaging.JPEGQuality(quality)); err != nil {
		t.Fatalf("encode expected jpeg: %v", err)
	}
	return buf.Bytes()
}
