package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleConfig = `server:
  host: "127.0.0.1"
  port: 8080
  env: "development"
database:
  type: "sqlite"
  path: "./data/test.db"
  connection_string: ""
storage:
  scan_roots:
    - "./images"
ai:
  provider: "qwen"
  api_key: ""
  model: "qwen-vl-max"
  auto_ai_tag_on_import: false
  auto_scan_interval_minutes: 9
  auto_scan_batch_size: 42
cos:
  bucket_url: "https://example.cos.ap-shanghai.myqcloud.com"
  secret_id: ""
  secret_key: ""
worker_pool:
  worker_count: 2
  queue_size: 8
  refill_interval_seconds: 3
  refill_threshold: 0.25
  refill_batch_size: 7
`

const sampleConfigWithoutAutoAITag = `server:
  host: "127.0.0.1"
  port: 8080
  env: "development"
database:
  type: "sqlite"
  path: "./data/test.db"
  connection_string: ""
storage:
  scan_roots:
    - "./images"
ai:
  provider: "qwen"
  api_key: ""
  model: "qwen-vl-max"
cos:
  bucket_url: "https://example.cos.ap-shanghai.myqcloud.com"
  secret_id: ""
  secret_key: ""
`

const sampleConfigWithoutRefillBatchSize = `server:
  host: "127.0.0.1"
  port: 8080
  env: "development"
database:
  type: "sqlite"
  path: "./data/test.db"
  connection_string: ""
storage:
  scan_roots:
    - "./images"
ai:
  provider: "qwen"
  api_key: ""
  model: "qwen-vl-max"
cos:
  bucket_url: "https://example.cos.ap-shanghai.myqcloud.com"
  secret_id: ""
  secret_key: ""
worker_pool:
  worker_count: 2
  queue_size: 8
  refill_interval_seconds: 3
  refill_threshold: 0.25
`

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}

func TestLoadConfigUsesExplicitPath(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "custom.yaml")
	if err := os.WriteFile(configPath, []byte(sampleConfig), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Fatalf("expected host 127.0.0.1, got %q", cfg.Server.Host)
	}
	if cfg.Server.Env != "development" {
		t.Fatalf("expected env development, got %q", cfg.Server.Env)
	}
	if cfg.Database.Path != "./data/test.db" {
		t.Fatalf("expected database path ./data/test.db, got %q", cfg.Database.Path)
	}
	if cfg.WorkerPool.RefillBatchSize != 7 {
		t.Fatalf("expected refill batch size 7, got %d", cfg.WorkerPool.RefillBatchSize)
	}
}

func TestLoadConfigAppliesEnvOverrides(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(sampleConfig), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("SERVER_HOST", "0.0.0.0")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("SERVER_ENV", "production")
	t.Setenv("DATABASE_TYPE", "postgres")
	t.Setenv("DATABASE_PATH", "./data/override.db")
	t.Setenv("DATABASE_CONNECTION_STRING", "postgres://user:pass@localhost/acg")
	t.Setenv("ACG_D1_API_URL", "https://api.acgwarehouse.cloud")
	t.Setenv("ACG_D1_API_KEY", "d1-secret")
	t.Setenv("ACG_D1_READONLY", "true")
	t.Setenv("AI_PROVIDER", "doubao")
	t.Setenv("AI_API_KEY", "secret")
	t.Setenv("AI_MODEL", "doubao-vision")
	t.Setenv("AUTO_AI_TAG_ON_IMPORT", "true")
	t.Setenv("AUTO_AI_TAG_SCAN_INTERVAL_MINUTES", "7")
	t.Setenv("AUTO_AI_TAG_BATCH_SIZE", "33")
	t.Setenv("WORKER_REFILL_BATCH_SIZE", "21")
	t.Setenv("COS_SECRET_ID", "cos-id")
	t.Setenv("COS_SECRET_KEY", "cos-key")
	t.Setenv("COS_BUCKET_URL", "https://override.cos.test")

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.Server.Host != "0.0.0.0" {
		t.Fatalf("expected overridden host 0.0.0.0, got %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Fatalf("expected overridden port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Server.Env != "production" {
		t.Fatalf("expected overridden env production, got %q", cfg.Server.Env)
	}
	if cfg.Database.Type != "postgres" {
		t.Fatalf("expected overridden database type postgres, got %q", cfg.Database.Type)
	}
	if cfg.Database.Path != "./data/override.db" {
		t.Fatalf("expected overridden database path ./data/override.db, got %q", cfg.Database.Path)
	}
	if cfg.Database.ConnectionString != "postgres://user:pass@localhost/acg" {
		t.Fatalf("expected overridden connection string, got %q", cfg.Database.ConnectionString)
	}
	if cfg.Database.D1APIURL != "https://api.acgwarehouse.cloud" {
		t.Fatalf("expected overridden D1 API URL, got %q", cfg.Database.D1APIURL)
	}
	if cfg.Database.D1APIKey != "d1-secret" {
		t.Fatalf("expected overridden D1 API key, got %q", cfg.Database.D1APIKey)
	}
	if !cfg.Database.D1ReadOnly {
		t.Fatal("expected overridden D1 readonly true")
	}
	if cfg.AI.Provider != "doubao" {
		t.Fatalf("expected overridden AI provider doubao, got %q", cfg.AI.Provider)
	}
	if cfg.AI.APIKey != "secret" {
		t.Fatalf("expected overridden AI API key secret, got %q", cfg.AI.APIKey)
	}
	if cfg.AI.Model != "doubao-vision" {
		t.Fatalf("expected overridden AI model doubao-vision, got %q", cfg.AI.Model)
	}
	if cfg.COS.SecretID != "cos-id" {
		t.Fatalf("expected overridden COS secret id cos-id, got %q", cfg.COS.SecretID)
	}
	if cfg.COS.SecretKey != "cos-key" {
		t.Fatalf("expected overridden COS secret key cos-key, got %q", cfg.COS.SecretKey)
	}
	if cfg.COS.BucketURL != "https://override.cos.test" {
		t.Fatalf("expected overridden COS bucket url, got %q", cfg.COS.BucketURL)
	}
	if !cfg.AI.AutoAITagOnImport {
		t.Fatal("expected AUTO_AI_TAG_ON_IMPORT override to set true")
	}
	if cfg.AI.AutoScanIntervalMinutes != 7 {
		t.Fatalf("expected overridden auto scan interval 7, got %d", cfg.AI.AutoScanIntervalMinutes)
	}
	if cfg.AI.AutoScanBatchSize != 33 {
		t.Fatalf("expected overridden auto scan batch size 33, got %d", cfg.AI.AutoScanBatchSize)
	}
	if cfg.WorkerPool.RefillBatchSize != 21 {
		t.Fatalf("expected overridden refill batch size 21, got %d", cfg.WorkerPool.RefillBatchSize)
	}
}

func TestLoadConfigReadsAutoAITagFields(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(sampleConfig), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.AI.AutoAITagOnImport {
		t.Fatal("expected auto_ai_tag_on_import false from config file")
	}
	if cfg.AI.AutoScanIntervalMinutes != 9 {
		t.Fatalf("expected auto scan interval 9, got %d", cfg.AI.AutoScanIntervalMinutes)
	}
	if cfg.AI.AutoScanBatchSize != 42 {
		t.Fatalf("expected auto scan batch size 42, got %d", cfg.AI.AutoScanBatchSize)
	}
	if cfg.WorkerPool.RefillBatchSize != 7 {
		t.Fatalf("expected refill batch size 7, got %d", cfg.WorkerPool.RefillBatchSize)
	}
}

func TestLoadConfigAppliesAutoAITagDefaults(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(sampleConfigWithoutAutoAITag), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if !cfg.AI.AutoAITagOnImport {
		t.Fatal("expected auto_ai_tag_on_import default true")
	}
	if cfg.AI.AutoScanIntervalMinutes != 5 {
		t.Fatalf("expected default auto scan interval 5, got %d", cfg.AI.AutoScanIntervalMinutes)
	}
	if cfg.AI.AutoScanBatchSize != 100 {
		t.Fatalf("expected default auto scan batch size 100, got %d", cfg.AI.AutoScanBatchSize)
	}
}

func TestLoadConfigDefaultsD1RuntimeDatabasePath(t *testing.T) {
	tempDir := t.TempDir()
	runtimeDir := filepath.Join(tempDir, "runtime")
	t.Setenv("ACG_RUNTIME_ROOT", runtimeDir)

	configPath := filepath.Join(tempDir, "config.yaml")
	configYAML := `server:
  host: "127.0.0.1"
  port: 8080
database:
  type: "d1"
  d1_api_url: "https://api.acgwarehouse.cloud"
  d1_api_key: "test-key"
storage:
  scan_roots: []
ai: {}
`
	if err := os.WriteFile(configPath, []byte(configYAML), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	want := filepath.Join(tempDir, "storage", "acgwarehouse-runtime.db")
	if cfg.Database.Path != want {
		t.Fatalf("database path = %q, want %q", cfg.Database.Path, want)
	}
}

func TestLoadConfigAppliesRefillBatchSizeDefault(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(sampleConfigWithoutRefillBatchSize), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.WorkerPool.RefillBatchSize != cfg.WorkerPool.QueueSize {
		t.Fatalf("expected refill batch size default to queue size %d, got %d", cfg.WorkerPool.QueueSize, cfg.WorkerPool.RefillBatchSize)
	}
}

const sampleConfigWithFallbackModels = `server:
  host: "127.0.0.1"
  port: 8080
ai:
  provider: "doubao"
  api_key: "test-key-123"
  model: "doubao-seed-2-0-pro-260215"
  fallback_models:
    - "doubao-seed-2-0-lite-260215"
    - "doubao-seed-2-0-mini-260215"
database:
  type: "sqlite"
  path: "./data/test.db"
storage:
  scan_roots:
    - "./images"
`

func TestLoadConfigParsesFallbackModels(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(sampleConfigWithFallbackModels), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.AI.Provider != "doubao" {
		t.Fatalf("expected AI provider doubao, got %q", cfg.AI.Provider)
	}
	if cfg.AI.Model != "doubao-seed-2-0-pro-260215" {
		t.Fatalf("expected AI model doubao-seed-2-0-pro-260215, got %q", cfg.AI.Model)
	}
	if len(cfg.AI.FallbackModels) != 2 {
		t.Fatalf("expected 2 fallback models, got %d", len(cfg.AI.FallbackModels))
	}
	if cfg.AI.FallbackModels[0] != "doubao-seed-2-0-lite-260215" {
		t.Fatalf("expected first fallback model doubao-seed-2-0-lite-260215, got %q", cfg.AI.FallbackModels[0])
	}
	if cfg.AI.FallbackModels[1] != "doubao-seed-2-0-mini-260215" {
		t.Fatalf("expected second fallback model doubao-seed-2-0-mini-260215, got %q", cfg.AI.FallbackModels[1])
	}
}

func TestLoadConfigFallbackModelsEmptyWhenNotSpecified(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(sampleConfigWithoutAutoAITag), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if len(cfg.AI.FallbackModels) != 0 {
		t.Fatalf("expected empty fallback models when not specified, got %d items", len(cfg.AI.FallbackModels))
	}
}

const sampleConfigWithoutDoubaoBatchMode = `server:
  host: "127.0.0.1"
  port: 8080
  env: "development"
database:
  type: "sqlite"
  path: "./data/test.db"
  connection_string: ""
storage:
  scan_roots:
    - "./images"
ai:
  provider: "qwen"
  api_key: ""
  model: "qwen-vl-max"
cos:
  bucket_url: "https://example.cos.ap-shanghai.myqcloud.com"
  secret_id: ""
  secret_key: ""
`

func configWithDoubaoBatchMode(modeLine string) string {
	return strings.Replace(sampleConfigWithoutDoubaoBatchMode, "  model: \"qwen-vl-max\"\n", "  model: \"qwen-vl-max\"\n"+modeLine+"\n", 1)
}

func TestLoadConfigReadsDoubaoBatchMode(t *testing.T) {
	configYAML := strings.Replace(sampleConfigWithFallbackModels, "  fallback_models:\n    - \"doubao-seed-2-0-lite-260215\"\n    - \"doubao-seed-2-0-mini-260215\"\n", "  fallback_models:\n    - \"doubao-seed-2-0-lite-260215\"\n    - \"doubao-seed-2-0-mini-260215\"\n  doubao_batch_mode: \"single\"\n", 1)
	cfg, err := LoadConfig(writeConfigFile(t, configYAML))
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg.AI.DoubaoBatchMode != "single" {
		t.Fatalf("expected doubao batch mode single, got %q", cfg.AI.DoubaoBatchMode)
	}
}

func TestLoadConfigDefaultsDoubaoBatchModeToAuto(t *testing.T) {
	cfg, err := LoadConfig(writeConfigFile(t, sampleConfigWithoutDoubaoBatchMode))
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg.AI.DoubaoBatchMode != "auto" {
		t.Fatalf("expected default doubao batch mode auto, got %q", cfg.AI.DoubaoBatchMode)
	}
}

func TestLoadConfigAppliesDoubaoBatchModeEnvOverride(t *testing.T) {
	t.Setenv("AI_DOUBAO_BATCH_MODE", "multi")
	cfg, err := LoadConfig(writeConfigFile(t, sampleConfigWithoutDoubaoBatchMode))
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg.AI.DoubaoBatchMode != "multi" {
		t.Fatalf("expected env override doubao batch mode multi, got %q", cfg.AI.DoubaoBatchMode)
	}
}

func TestLoadConfigAppliesFallbackModelsEnvOverride(t *testing.T) {
	t.Setenv("AI_FALLBACK_MODELS", "fallback-a, fallback-b ,fallback-c")
	cfg, err := LoadConfig(writeConfigFile(t, sampleConfigWithoutDoubaoBatchMode))
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if strings.Join(cfg.AI.FallbackModels, ",") != "fallback-a,fallback-b,fallback-c" {
		t.Fatalf("unexpected fallback models from env: %v", cfg.AI.FallbackModels)
	}
}

func TestLoadConfigNormalizesYAMLDoubaoBatchModeValues(t *testing.T) {
	tests := []struct {
		name     string
		modeLine string
		want     string
	}{
		{name: "invalid", modeLine: "  doubao_batch_mode: \"invalid\"", want: "auto"},
		{name: "mixed-case", modeLine: "  doubao_batch_mode: \" SINGLE \"", want: "single"},
		{name: "whitespace-padded", modeLine: "  doubao_batch_mode: \"  multi  \"", want: "multi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadConfig(writeConfigFile(t, configWithDoubaoBatchMode(tt.modeLine)))
			if err != nil {
				t.Fatalf("LoadConfig returned error: %v", err)
			}
			if cfg.AI.DoubaoBatchMode != tt.want {
				t.Fatalf("expected doubao batch mode %q, got %q", tt.want, cfg.AI.DoubaoBatchMode)
			}
		})
	}
}
