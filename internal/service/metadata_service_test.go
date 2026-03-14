package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsImageRecognizesSupportedFormats(t *testing.T) {
	t.Parallel()

	svc := NewMetadataService()

	cases := []struct {
		name string
		path string
		want bool
	}{
		{name: "jpg", path: "a.jpg", want: true},
		{name: "jpeg", path: "a.jpeg", want: true},
		{name: "png", path: "a.png", want: true},
		{name: "webp", path: "a.webp", want: true},
		{name: "gif", path: "a.gif", want: true},
		{name: "txt", path: "a.txt", want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := svc.IsImage(tc.path); got != tc.want {
				t.Fatalf("IsImage(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestExtractMetadataUsesFileInfoWhenExifMissing(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	imagePath := filepath.Join(root, "tiny.png")

	// 1x1 transparent PNG.
	data := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	}

	if err := os.WriteFile(imagePath, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	svc := NewMetadataService()
	img, err := svc.ExtractMetadata(imagePath)
	if err != nil {
		t.Fatalf("ExtractMetadata() error = %v", err)
	}

	if img.Width != 1 || img.Height != 1 {
		t.Fatalf("unexpected dimensions: got %dx%d, want 1x1", img.Width, img.Height)
	}
	if img.Format != "png" {
		t.Fatalf("Format = %q, want png", img.Format)
	}
	if img.Path != imagePath {
		t.Fatalf("Path = %q, want %q", img.Path, imagePath)
	}
	if img.FileSize <= 0 {
		t.Fatalf("FileSize = %d, want > 0", img.FileSize)
	}
	if img.CreatedAt.IsZero() || img.UpdatedAt.IsZero() {
		t.Fatal("expected non-zero timestamps")
	}
}
