package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/wonichan/acgwarehouse-backend/internal/logger"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server                   ServerConfig     `yaml:"server"`
	Database                 DatabaseConfig   `yaml:"database"`
	Storage                  StorageConfig    `yaml:"storage"`
	AI                       AIConfig         `yaml:"ai"`
	ThumbnailStorageProvider string           `yaml:"thumbnail_storage_provider"`
	COS                      COSConfig        `yaml:"cos"`
	Minio                    MinioConfig      `yaml:"minio"`
	Admin                    AdminConfig      `yaml:"admin"`
	WorkerPool               WorkerPoolConfig `yaml:"worker_pool"`
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
	Provider                string   `yaml:"provider"`
	APIKey                  string   `yaml:"api_key"`
	Model                   string   `yaml:"model"`
	FallbackModels          []string `yaml:"fallback_models"`
	DoubaoBatchMode         string   `yaml:"doubao_batch_mode"`
	RequestsPerMinute       int      `yaml:"requests_per_minute"` // 限流：每分钟请求数，默认 60
	MaxConcurrency          int      `yaml:"max_concurrency"`     // 并发限制：同时执行的最大AI请求数，默认 3
	AutoAITagOnImport       bool     `yaml:"auto_ai_tag_on_import"`
	AutoScanIntervalMinutes int      `yaml:"auto_scan_interval_minutes"`
	AutoScanBatchSize       int      `yaml:"auto_scan_batch_size"`
}

type COSConfig struct {
	BucketURL string `yaml:"bucket_url"`
	SecretID  string `yaml:"secret_id"`
	SecretKey string `yaml:"secret_key"`
}

type MinioConfig struct {
	Endpoint  string `yaml:"endpoint"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Bucket    string `yaml:"bucket"`
	UseSSL    bool   `yaml:"use_ssl"`
}

// AdminConfig holds configuration for the admin dashboard access.
// It supports simple local/internal protection suitable for personal use.
type AdminConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// WorkerPoolConfig holds configuration for the job worker pool.
type WorkerPoolConfig struct {
	// WorkerCount is the number of concurrent workers processing jobs
	WorkerCount int `yaml:"worker_count"`
	// QueueSize is the buffer size of the job queue (cannot be changed at runtime)
	QueueSize int `yaml:"queue_size"`
	// RefillIntervalSeconds is how often to check for pending jobs to refill the queue
	RefillIntervalSeconds int `yaml:"refill_interval_seconds"`
	// RefillThreshold is the queue size below which refill is triggered (fraction of QueueSize, e.g., 0.5)
	RefillThreshold float64 `yaml:"refill_threshold"`
	// RefillBatchSize is the max number of ready jobs loaded into the in-memory queue per refill tick.
	// When unset or <= 0, it defaults to QueueSize to preserve the previous behavior.
	RefillBatchSize int `yaml:"refill_batch_size"`
}

// ConfigChangeCallback is called when configuration changes.
type ConfigChangeCallback func(old, new *Config)

// Reloader manages config hot-reloading.
type Reloader struct {
	path      string
	config    *Config
	mu        sync.RWMutex
	callbacks []ConfigChangeCallback
	watcher   *fsnotify.Watcher
	stopCh    chan struct{}
}

// NewReloader creates a config reloader that watches for file changes.
func NewReloader(path string) (*Reloader, error) {
	cfg, err := loadConfig(path)
	if err != nil {
		return nil, err
	}

	return &Reloader{
		path:      path,
		config:    cfg,
		callbacks: make([]ConfigChangeCallback, 0),
		stopCh:    make(chan struct{}),
	}, nil
}

// Get returns the current config (thread-safe).
func (r *Reloader) Get() *Config {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}

// OnChange registers a callback to be called when config changes.
func (r *Reloader) OnChange(callback ConfigChangeCallback) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.callbacks = append(r.callbacks, callback)
}

// Start begins watching the config file for changes.
func (r *Reloader) Start() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	r.watcher = watcher

	if err := watcher.Add(r.path); err != nil {
		watcher.Close()
		return fmt.Errorf("watch config file: %w", err)
	}

	go r.watchLoop()
	logger.Infof("配置热重载已启用，监听文件: %s", r.path)
	return nil
}

// Stop stops watching for config changes.
func (r *Reloader) Stop() {
	close(r.stopCh)
	if r.watcher != nil {
		r.watcher.Close()
	}
}

func (r *Reloader) watchLoop() {
	for {
		select {
		case <-r.stopCh:
			return
		case event, ok := <-r.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				r.reload()
			}
		case err, ok := <-r.watcher.Errors:
			if !ok {
				return
			}
			logger.Errorf("配置文件监听错误: %v", err)
		}
	}
}

func (r *Reloader) reload() {
	newCfg, err := loadConfig(r.path)
	if err != nil {
		logger.Errorf("重载配置失败: %v", err)
		return
	}

	r.mu.Lock()
	oldCfg := r.config
	r.config = newCfg
	callbacks := make([]ConfigChangeCallback, len(r.callbacks))
	copy(callbacks, r.callbacks)
	r.mu.Unlock()

	logger.Infof("配置已重新加载")

	// Call all registered callbacks
	for _, cb := range callbacks {
		cb(oldCfg, newCfg)
	}
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	cfg := Config{
		AI: AIConfig{
			AutoAITagOnImport:       true,
			AutoScanIntervalMinutes: 5,
			AutoScanBatchSize:       100,
		},
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	applyDefaults(&cfg)
	applyEnvOverrides(&cfg)
	return &cfg, nil
}

// LoadConfig is a convenience function for one-time config loading.
func LoadConfig(paths ...string) (*Config, error) {
	path := "config.yaml"
	if len(paths) > 0 && paths[0] != "" {
		path = paths[0]
	}
	return loadConfig(path)
}

// applyDefaults sets default values for optional configuration fields.
func applyDefaults(cfg *Config) {
	// WorkerPool defaults
	if cfg.WorkerPool.WorkerCount <= 0 {
		cfg.WorkerPool.WorkerCount = 4
	}
	if cfg.WorkerPool.QueueSize <= 0 {
		cfg.WorkerPool.QueueSize = 512
	}
	if cfg.WorkerPool.RefillIntervalSeconds <= 0 {
		cfg.WorkerPool.RefillIntervalSeconds = 5
	}
	if cfg.WorkerPool.RefillThreshold <= 0 || cfg.WorkerPool.RefillThreshold > 1 {
		cfg.WorkerPool.RefillThreshold = 0.5
	}

	// AI defaults
	if cfg.AI.RequestsPerMinute <= 0 {
		cfg.AI.RequestsPerMinute = 60
	}
	if cfg.AI.MaxConcurrency <= 0 {
		cfg.AI.MaxConcurrency = 3
	}
	if cfg.AI.AutoScanIntervalMinutes <= 0 {
		cfg.AI.AutoScanIntervalMinutes = 5
	}
	if cfg.AI.AutoScanBatchSize <= 0 {
		cfg.AI.AutoScanBatchSize = 100
	}
	cfg.AI.FallbackModels = normalizeFallbackModels(cfg.AI.FallbackModels)
	applyDoubaoBatchMode(cfg, cfg.AI.DoubaoBatchMode)
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

	if v := os.Getenv("AI_FALLBACK_MODELS"); v != "" {
		cfg.AI.FallbackModels = normalizeFallbackModels(strings.Split(v, ","))
	}

	if v := os.Getenv("AI_DOUBAO_BATCH_MODE"); v != "" {
		applyDoubaoBatchMode(cfg, v)
	}

	if v := os.Getenv("THUMBNAIL_STORAGE_PROVIDER"); v != "" {
		cfg.ThumbnailStorageProvider = v
	}

	if v := os.Getenv("COS_BUCKET_URL"); v != "" {
		cfg.COS.BucketURL = v
	}

	if v := os.Getenv("COS_SECRET_ID"); v != "" {
		cfg.COS.SecretID = v
	}

	if v := os.Getenv("COS_SECRET_KEY"); v != "" {
		cfg.COS.SecretKey = v
	}

	if v := os.Getenv("MINIO_ENDPOINT"); v != "" {
		cfg.Minio.Endpoint = v
	}

	if v := os.Getenv("MINIO_ACCESS_KEY"); v != "" {
		cfg.Minio.AccessKey = v
	}

	if v := os.Getenv("MINIO_SECRET_KEY"); v != "" {
		cfg.Minio.SecretKey = v
	}

	if v := os.Getenv("MINIO_BUCKET"); v != "" {
		cfg.Minio.Bucket = v
	}

	if v := os.Getenv("MINIO_USE_SSL"); v != "" {
		cfg.Minio.UseSSL = strings.EqualFold(v, "true") || v == "1"
	}

	if v := os.Getenv("ADMIN_USERNAME"); v != "" {
		cfg.Admin.Username = v
	}

	if v := os.Getenv("ADMIN_PASSWORD"); v != "" {
		cfg.Admin.Password = v
	}

	if v := os.Getenv("WORKER_COUNT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.WorkerPool.WorkerCount = n
		}
	}

	if v := os.Getenv("QUEUE_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.WorkerPool.QueueSize = n
		}
	}

	if v := os.Getenv("REFILL_INTERVAL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.WorkerPool.RefillIntervalSeconds = n
		}
	}

	if v := os.Getenv("REFILL_THRESHOLD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.WorkerPool.RefillThreshold = f
		}
	}

	if v := os.Getenv("REFILL_BATCH_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.WorkerPool.RefillBatchSize = n
		}
	}
}
