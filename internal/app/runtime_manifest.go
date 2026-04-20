package app

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	RuntimeManifestSchemaVersion = 1
	runtimeManifestFileName      = "runtime-manifest.json"
	runtimeManifestEnvPath       = "ACG_RUNTIME_MANIFEST_PATH"
)

type RuntimeManifestPayload struct {
	Version     int                    `json:"version"`
	GeneratedAt string                 `json:"generated_at"`
	Go          runtimeManifestGoEntry `json:"go"`
}

type runtimeManifestGoEntry struct {
	BaseURL          string `json:"base_url"`
	ThumbnailBaseURL string `json:"thumbnail_base_url,omitempty"`
	Ready            bool   `json:"ready"`
	AdminBasicAuth   string `json:"admin_basic_auth,omitempty"`
}

var (
	runtimeManifestMkdirAll   = os.MkdirAll
	runtimeManifestCreateTmp  = os.CreateTemp
	runtimeManifestRenameFile = os.Rename
	runtimeManifestRemoveFile = os.Remove
)

func BuildRuntimeManifestPayload(baseURL, thumbnailBaseURL, adminUsername, adminPassword string, generatedAt time.Time) (RuntimeManifestPayload, error) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return RuntimeManifestPayload{}, fmt.Errorf("runtime manifest base URL is required")
	}
	if _, err := url.ParseRequestURI(trimmed); err != nil {
		return RuntimeManifestPayload{}, fmt.Errorf("runtime manifest base URL is invalid: %w", err)
	}

	thumbnailBase := strings.TrimRight(strings.TrimSpace(thumbnailBaseURL), "/")
	if thumbnailBase != "" {
		if _, err := url.ParseRequestURI(thumbnailBase); err != nil {
			return RuntimeManifestPayload{}, fmt.Errorf("runtime manifest thumbnail base URL is invalid: %w", err)
		}
	}

	stamp := generatedAt.UTC()
	if stamp.IsZero() {
		stamp = time.Now().UTC()
	}

	adminBasicAuth := ""
	if adminUsername != "" || adminPassword != "" {
		adminBasicAuth = "Basic " + base64.StdEncoding.EncodeToString([]byte(adminUsername+":"+adminPassword))
	}

	return RuntimeManifestPayload{
		Version:     RuntimeManifestSchemaVersion,
		GeneratedAt: stamp.Format(time.RFC3339),
		Go: runtimeManifestGoEntry{
			BaseURL:          trimmed,
			ThumbnailBaseURL: thumbnailBase,
			Ready:            true,
			AdminBasicAuth:   adminBasicAuth,
		},
	}, nil
}

func WriteRuntimeManifestAtomic(path string, payload RuntimeManifestPayload) error {
	if path == "" {
		return fmt.Errorf("runtime manifest path is required")
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal runtime manifest: %w", err)
	}

	dir := filepath.Dir(path)
	if err := runtimeManifestMkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create runtime manifest directory: %w", err)
	}

	tmpFile, err := runtimeManifestCreateTmp(dir, "runtime-manifest-*.tmp")
	if err != nil {
		return fmt.Errorf("create runtime manifest temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	cleanupTemp := func() {
		_ = runtimeManifestRemoveFile(tmpPath)
	}

	if _, err := tmpFile.Write(raw); err != nil {
		_ = tmpFile.Close()
		cleanupTemp()
		return fmt.Errorf("write runtime manifest temp file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		cleanupTemp()
		return fmt.Errorf("sync runtime manifest temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		cleanupTemp()
		return fmt.Errorf("close runtime manifest temp file: %w", err)
	}

	if err := runtimeManifestRenameFile(tmpPath, path); err != nil {
		cleanupTemp()
		return fmt.Errorf("rename runtime manifest temp file: %w", err)
	}

	return nil
}

func RemoveRuntimeManifest(path string) error {
	if path == "" {
		return nil
	}
	if err := runtimeManifestRemoveFile(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove runtime manifest: %w", err)
	}
	return nil
}

func ResolveRuntimeManifestPath() string {
	if configured := strings.TrimSpace(os.Getenv(runtimeManifestEnvPath)); configured != "" {
		return configured
	}
	if runtimeRoot := strings.TrimSpace(os.Getenv(portableRuntimeRootEnv)); runtimeRoot != "" {
		return ResolveRuntimeManifestPathForLayout(resolvePortableRuntimeLayoutRoot(runtimeRoot))
	}
	return filepath.Join(os.TempDir(), "acgwarehouse", runtimeManifestFileName)
}

func ResolveRuntimeManifestBaseURL(listenerAddr net.Addr, configuredHost string) (string, error) {
	tcpAddr, ok := listenerAddr.(*net.TCPAddr)
	if !ok {
		return "", fmt.Errorf("runtime manifest listener address must be TCP, got %T", listenerAddr)
	}

	host := strings.TrimSpace(configuredHost)
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		host = "127.0.0.1"
	}

	return fmt.Sprintf("http://%s:%d", host, tcpAddr.Port), nil
}
