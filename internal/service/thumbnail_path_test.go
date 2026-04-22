package service

import (
	"testing"

	"github.com/wonichan/acgwarehouse-backend/internal/config"
)

func TestResolveThumbnailBaseURLUsesMinIOEndpoint(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ThumbnailStorageProvider: "minio",
		Minio: config.MinioConfig{
			Endpoint: "minio.internal:9000",
			UseSSL:   true,
		},
	}

	got := ResolveThumbnailBaseURL(cfg)
	if got != "https://minio.internal:9000" {
		t.Fatalf("ResolveThumbnailBaseURL() = %q, want %q", got, "https://minio.internal:9000")
	}
}

func TestResolveThumbnailBaseURLUsesCOSBucketURL(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ThumbnailStorageProvider: "cos",
		COS: config.COSConfig{
			BucketURL: "https://acg-1250000000.cos.ap-shanghai.myqcloud.com/",
		},
	}

	got := ResolveThumbnailBaseURL(cfg)
	if got != "https://acg-1250000000.cos.ap-shanghai.myqcloud.com" {
		t.Fatalf("ResolveThumbnailBaseURL() = %q, want %q", got, "https://acg-1250000000.cos.ap-shanghai.myqcloud.com")
	}
}

func TestResolveThumbnailBaseURLNormalizesProviderCase(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ThumbnailStorageProvider: "  CoS  ",
		COS: config.COSConfig{
			BucketURL: "https://acg-1250000000.cos.ap-shanghai.myqcloud.com",
		},
	}

	got := ResolveThumbnailBaseURL(cfg)
	if got != "https://acg-1250000000.cos.ap-shanghai.myqcloud.com" {
		t.Fatalf("ResolveThumbnailBaseURL() = %q, want normalized COS base URL", got)
	}
}

func TestResolveThumbnailBaseURLReturnsEmptyForEmptyProvider(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ThumbnailStorageProvider: "   ",
		Minio: config.MinioConfig{
			Endpoint: "minio.internal:9000",
			UseSSL:   true,
		},
	}

	got := ResolveThumbnailBaseURL(cfg)
	if got != "" {
		t.Fatalf("ResolveThumbnailBaseURL() = %q, want empty for empty provider", got)
	}
}

func TestResolveThumbnailBaseURLReturnsEmptyForNilConfig(t *testing.T) {
	t.Parallel()

	if got := ResolveThumbnailBaseURL(nil); got != "" {
		t.Fatalf("ResolveThumbnailBaseURL(nil) = %q, want empty", got)
	}
}

func TestResolveThumbnailBaseURLReturnsEmptyForInvalidCOSBucketURL(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ThumbnailStorageProvider: "cos",
		COS: config.COSConfig{
			BucketURL: "not-a-url",
		},
	}

	got := ResolveThumbnailBaseURL(cfg)
	if got != "" {
		t.Fatalf("ResolveThumbnailBaseURL() = %q, want empty for invalid COS bucket URL", got)
	}
}

func TestResolveThumbnailBaseURLReturnsEmptyForInvalidMinIOEndpoint(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ThumbnailStorageProvider: "minio",
		Minio: config.MinioConfig{
			Endpoint: "",
			UseSSL:   false,
		},
	}

	got := ResolveThumbnailBaseURL(cfg)
	if got != "" {
		t.Fatalf("ResolveThumbnailBaseURL() = %q, want empty for invalid MinIO endpoint", got)
	}
}

func TestResolveThumbnailBaseURLReturnsEmptyForUnknownProvider(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ThumbnailStorageProvider: "s3",
		Minio: config.MinioConfig{
			Endpoint: "minio.internal:9000",
			UseSSL:   false,
		},
	}

	got := ResolveThumbnailBaseURL(cfg)
	if got != "" {
		t.Fatalf("ResolveThumbnailBaseURL() = %q, want empty for unknown provider", got)
	}
}

func TestResolveThumbnailURLKeepsAbsoluteURLWithoutRewriting(t *testing.T) {
	t.Parallel()

	got := ResolveThumbnailURL("http://minio.internal:9000", "https://cdn.example.com/acg/thumbnails/example.jpg")
	if got != "https://cdn.example.com/acg/thumbnails/example.jpg" {
		t.Fatalf("ResolveThumbnailURL() = %q, want original absolute URL", got)
	}
}

func TestResolveThumbnailURLReturnsRelativePathWhenBaseURLIsEmpty(t *testing.T) {
	t.Parallel()

	got := ResolveThumbnailURL("", "acg/thumbnails/example.jpg")
	if got != "acg/thumbnails/example.jpg" {
		t.Fatalf("ResolveThumbnailURL() = %q, want original relative path", got)
	}
}

func TestResolveThumbnailURLReturnsRelativePathWhenBaseURLIsInvalid(t *testing.T) {
	t.Parallel()

	got := ResolveThumbnailURL("not-a-url", "acg/thumbnails/example.jpg")
	if got != "acg/thumbnails/example.jpg" {
		t.Fatalf("ResolveThumbnailURL() = %q, want original relative path when base invalid", got)
	}
}

func TestResolveThumbnailURLReturnsEmptyWhenRawURLIsEmpty(t *testing.T) {
	t.Parallel()

	if got := ResolveThumbnailURL("http://minio.internal:9000", ""); got != "" {
		t.Fatalf("ResolveThumbnailURL() = %q, want empty when raw is empty", got)
	}
}
