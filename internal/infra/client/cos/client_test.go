package cos

import (
	"errors"
	"testing"

	"github.com/yachiyo/acgwarehouse/internal/conf"
)

func Test_ValidateConfig_rejects_placeholder_credentials_when_sync_starts(t *testing.T) {
	// Given
	cfg := conf.COSConfig{
		SecretID:  "COS_SECRET_ID_PLACEHOLDER",
		SecretKey: "real-secret-key",
		Bucket:    "acgwarehouse-1301393037",
		Region:    "ap-shanghai",
		Domain:    "https://example.com",
		Prefix:    "/thumbnails",
	}

	// When
	err := ValidateConfig(cfg)

	// Then
	if !errors.Is(err, ErrInvalidCredential) {
		t.Fatalf("error = %v, want ErrInvalidCredential", err)
	}
}

func Test_NormalizePrefix_converts_config_prefix_to_cos_list_prefix(t *testing.T) {
	// Given
	raw := "/thumbnails"

	// When
	got := NormalizePrefix(raw)

	// Then
	if got != "thumbnails/" {
		t.Fatalf("prefix = %q, want thumbnails/", got)
	}
}

func Test_ObjectURL_joins_domain_and_key_without_duplicate_slashes(t *testing.T) {
	// Given
	client := Client{domain: "https://example.com/"}

	// When
	got := client.ObjectURL("/thumbnails/miku.png")

	// Then
	want := "https://example.com/thumbnails/miku.png"
	if got != want {
		t.Fatalf("url = %q, want %q", got, want)
	}
}
