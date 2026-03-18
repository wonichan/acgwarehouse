package config

import (
	"os"
	"path/filepath"
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
cos:
  bucket_url: "https://example.cos.ap-shanghai.myqcloud.com"
  secret_id: ""
  secret_key: ""
`

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
	t.Setenv("AI_PROVIDER", "doubao")
	t.Setenv("AI_API_KEY", "secret")
	t.Setenv("AI_MODEL", "doubao-vision")
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
}
