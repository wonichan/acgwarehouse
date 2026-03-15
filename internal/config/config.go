package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Storage  StorageConfig  `yaml:"storage"`
	AI       AIConfig       `yaml:"ai"`
	COS      COSConfig      `yaml:"cos"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Env  string `yaml:"env"`
}

type DatabaseConfig struct {
	Type             string `yaml:"type"`
	Path             string `yaml:"path"`
	ConnectionString string `yaml:"connection_string"`
}

type StorageConfig struct {
	ScanRoots []string `yaml:"scan_roots"`
}

type AIConfig struct {
	Provider          string `yaml:"provider"`
	APIKey            string `yaml:"api_key"`
	Model             string `yaml:"model"`
	RequestsPerMinute int    `yaml:"requests_per_minute"` // 限流：每分钟请求数，默认 60
}

type COSConfig struct {
	BucketURL string `yaml:"bucket_url"`
	SecretID  string `yaml:"secret_id"`
	SecretKey string `yaml:"secret_key"`
}

func LoadConfig(paths ...string) (*Config, error) {
	path := "config.yaml"
	if len(paths) > 0 && paths[0] != "" {
		path = paths[0]
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	applyEnvOverrides(&cfg)
	return &cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}

	if v := os.Getenv("SERVER_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = p
		}
	}

	if v := os.Getenv("SERVER_ENV"); v != "" {
		cfg.Server.Env = v
	}

	if v := os.Getenv("DATABASE_TYPE"); v != "" {
		cfg.Database.Type = v
	}

	if v := os.Getenv("DATABASE_PATH"); v != "" {
		cfg.Database.Path = v
	}

	if v := os.Getenv("DATABASE_CONNECTION_STRING"); v != "" {
		cfg.Database.ConnectionString = v
	}

	if v := os.Getenv("AI_PROVIDER"); v != "" {
		cfg.AI.Provider = v
	}

	if v := os.Getenv("AI_API_KEY"); v != "" {
		cfg.AI.APIKey = v
	}

	if v := os.Getenv("AI_MODEL"); v != "" {
		cfg.AI.Model = v
	}

	if v := os.Getenv("AI_REQUESTS_PER_MINUTE"); v != "" {
		if rpm, err := strconv.Atoi(v); err == nil {
			cfg.AI.RequestsPerMinute = rpm
		}
	}

	// COS 环境变量覆盖
	if v := os.Getenv("COS_SECRET_ID"); v != "" {
		cfg.COS.SecretID = v
	}
	if v := os.Getenv("COS_SECRET_KEY"); v != "" {
		cfg.COS.SecretKey = v
	}
}
