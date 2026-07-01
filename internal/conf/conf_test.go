package conf_test

import (
	"testing"

	"github.com/yachiyo/acgwarehouse/internal/conf"
)

// TestLoadRejectsDefaultJWTSecret 确认默认占位 JWT 密钥不能启动服务。
func TestLoadRejectsDefaultJWTSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")

	_, err := conf.Load()

	if err == nil {
		t.Fatal("Load() error = nil, want weak jwt secret error")
	}
}

// TestLoadDatabaseUsesSQLiteEnvWithoutJWTSecret 确认 CLI 可只加载数据库配置而不要求 Web 密钥。
func TestLoadDatabaseUsesSQLiteEnvWithoutJWTSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("SQLITE_PATH", "data/custom.db")

	cfg := conf.LoadDatabase()

	if cfg.Path != "data/custom.db" {
		t.Fatalf("Path = %q, want data/custom.db", cfg.Path)
	}
	if cfg.WriteMaxOpenConns != 1 {
		t.Fatalf("WriteMaxOpenConns = %d, want 1", cfg.WriteMaxOpenConns)
	}
}

// TestLoadAcceptsStrongJWTSecret 确认强密钥与安全默认配置可正常加载。
func TestLoadAcceptsStrongJWTSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("CORS_ALLOW_ORIGIN", "https://app.example.com, https://admin.example.com")

	cfg, err := conf.Load()

	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if cfg.Security.MaxRequestBodyBytes != 1048576 {
		t.Fatalf("MaxRequestBodyBytes = %d, want 1048576", cfg.Security.MaxRequestBodyBytes)
	}
	if cfg.Security.RateLimitRPS != 2 || cfg.Security.RateLimitBurst != 5 {
		t.Fatalf("rate limit = %f/%d, want 2/5", cfg.Security.RateLimitRPS, cfg.Security.RateLimitBurst)
	}
	if len(cfg.CORS.AllowOrigins) != 2 || cfg.CORS.AllowOrigins[0] != "https://app.example.com" {
		t.Fatalf("AllowOrigins = %#v, want parsed origins", cfg.CORS.AllowOrigins)
	}
}

// TestLoadRejectsInvalidSecurityNumbers 确认非法安全数值会被拒绝。
func TestLoadRejectsInvalidSecurityNumbers(t *testing.T) {
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("MAX_REQUEST_BODY_BYTES", "0")

	_, err := conf.Load()

	if err == nil {
		t.Fatal("Load() error = nil, want invalid max body error")
	}
}
