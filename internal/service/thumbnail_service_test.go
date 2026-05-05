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

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
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
	if thumb.Width != 800 || thumb.Height != 533 {
		t.Fatalf("thumbnail dimensions = %dx%d, want 800x533", thumb.Width, thumb.Height)
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

	if len(large.Data) <= len(small.Data) {
		t.Fatalf("large thumbnail should generally be larger than small, got large=%d small=%d", len(large.Data), len(small.Data))
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

func TestThumbnailPolicyParamsBySize(t *testing.T) {
	t.Parallel()

	svc := &ThumbnailService{
		SmallWidth:   240,
		LargeWidth:   960,
		SmallQuality: 80,
		LargeQuality: 92,
	}

	width, quality, err := svc.paramsBySize("small")
	if err != nil {
		t.Fatalf("paramsBySize(small) error = %v", err)
	}
	if width != 240 || quality != 80 {
		t.Fatalf("paramsBySize(small) = %d/%d, want 240/80", width, quality)
	}

	width, quality, err = svc.paramsBySize("large")
	if err != nil {
		t.Fatalf("paramsBySize(large) error = %v", err)
	}
	if width != 960 || quality != 92 {
		t.Fatalf("paramsBySize(large) = %d/%d, want 960/92", width, quality)
	}

	if _, _, err := svc.paramsBySize("medium"); err == nil {
		t.Fatal("paramsBySize(medium) expected error")
	}
}

func TestThumbnailPolicySelectResizeProfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fileSize int64
		want     thumbnailResizeProfile
	}{
		{name: "small file", fileSize: smallFileThreshold - 1, want: thumbnailResizeHighQuality},
		{name: "medium file", fileSize: smallFileThreshold, want: thumbnailResizeBalanced},
		{name: "large file", fileSize: mediumFileThreshold, want: thumbnailResizeFast},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := selectThumbnailResizeProfile(tt.fileSize); got != tt.want {
				t.Fatalf("selectThumbnailResizeProfile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestThumbnailPolicyLargeMinimumSizeTracksSmallThumbnail(t *testing.T) {
	t.Parallel()

	if got := minLargeThumbnailSize(minLargeSize - 1); got != minLargeSize {
		t.Fatalf("minLargeThumbnailSize(below minimum) = %d, want %d", got, minLargeSize)
	}

	smallSize := minLargeSize + 8
	want := smallSize + 100*1024
	if got := minLargeThumbnailSize(smallSize); got != want {
		t.Fatalf("minLargeThumbnailSize(large small) = %d, want %d", got, want)
	}
}

func TestThumbnailPolicyAdjustsSmallWidthBeforeQuality(t *testing.T) {
	t.Parallel()

	var attempts []struct {
		width   int
		quality int
	}

	thumb, err := runSmallThumbnailPolicy(450, 200, 85, func(width, quality int) (*domain.Thumbnail, error) {
		attempts = append(attempts, struct {
			width   int
			quality int
		}{width: width, quality: quality})
		return &domain.Thumbnail{Data: make([]byte, 10)}, nil
	})
	if err != nil {
		t.Fatalf("runSmallThumbnailPolicy() error = %v", err)
	}
	if len(thumb.Data) != 10 {
		t.Fatalf("thumb data length = %d, want 10", len(thumb.Data))
	}

	want := []struct {
		width   int
		quality int
	}{
		{width: 200, quality: 85},
		{width: 300, quality: 85},
		{width: 400, quality: 85},
		{width: 450, quality: 85},
		{width: 450, quality: 90},
	}
	if len(attempts) < len(want) {
		t.Fatalf("attempt count = %d, want at least %d", len(attempts), len(want))
	}
	for i := range want {
		if attempts[i] != want[i] {
			t.Fatalf("attempt[%d] = %+v, want %+v", i, attempts[i], want[i])
		}
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
