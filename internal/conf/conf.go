package conf

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	pkgerrors "github.com/pkg/errors"
)

const (
	defaultPort              = "8080"
	defaultDBPath            = "data/acgwarehouse.db"
	defaultBlevePath         = "data/bleve"
	defaultCOSBucket         = "acgwarehouse-1301393037"
	defaultCOSRegion         = "ap-shanghai"
	defaultCOSDomain         = "https://acgwarehouse-1301393037.cos.ap-shanghai.myqcloud.com"
	defaultCOSPrefix         = "/thumbnails"
	defaultCOSSecretID       = "COS_SECRET_ID_PLACEHOLDER"
	defaultCOSSecretKey      = "COS_SECRET_KEY_PLACEHOLDER"
	defaultJWTSecret         = "JWT_SECRET_PLACEHOLDER"
	defaultJWTDuration       = "168h"
	defaultCORSAllowOrigin   = ""
	defaultMaxRequestBody    = "1048576"
	defaultRateLimitRPS      = "2"
	defaultRateLimitBurst    = "5"
	defaultRankingWeight     = "1"
	defaultBayesianC         = "10"
	defaultRankingInterval   = "10m"
	defaultViewFlushInterval = "1s"
	defaultSQLiteTimeout     = "5000"
	defaultLogLevel          = "info"
	minJWTSecretLength       = 32
)

// Config 保存服务启动所需的全部环境配置。
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Search   SearchConfig
	COS      COSConfig
	Security SecurityConfig
	Ranking  RankingConfig
	View     ViewConfig
	Admin    AdminConfig
	CORS     CORSConfig
	Log      LogConfig
}

// ServerConfig 保存 HTTP 服务配置。
type ServerConfig struct {
	Port            string
	ShutdownTimeout time.Duration
}

// Address 返回 Hertz 监听地址。
func (c ServerConfig) Address() string {
	return ":" + c.Port
}

// DatabaseConfig 保存 SQLite 双连接池配置。
type DatabaseConfig struct {
	Path              string
	BusyTimeoutMS     int
	ReadMaxOpenConns  int
	WriteMaxOpenConns int
}

// SearchConfig 保存搜索索引配置。
type SearchConfig struct {
	BlevePath string
}

// COSConfig 保存腾讯云 COS 访问配置。
type COSConfig struct {
	SecretID  string
	SecretKey string
	Bucket    string
	Region    string
	Domain    string
	Prefix    string
}

// SecurityConfig 保存认证与令牌配置。
type SecurityConfig struct {
	JWTSecret           string
	JWTDuration         time.Duration
	MaxRequestBodyBytes int
	RateLimitRPS        float64
	RateLimitBurst      int
}

// RankingConfig 保存热榜计算配置。
type RankingConfig struct {
	WeightRating      float64
	WeightFavorite    float64
	WeightView        float64
	BayesianC         float64
	RecomputeInterval time.Duration
}

// ViewConfig 保存浏览事件缓冲配置。
type ViewConfig struct {
	FlushInterval time.Duration
}

// AdminConfig 保存首个管理员引导配置。
type AdminConfig struct {
	Username string
	Password string
}

// CORSConfig 保存跨域访问配置。
type CORSConfig struct {
	AllowOrigins []string
}

// LogConfig 保存日志配置。
type LogConfig struct {
	Level string
}

// Load 从环境变量读取配置，并为非凭证项填充默认值。
func Load() (Config, error) {
	jwtDuration, err := envDuration("JWT_DURATION", defaultJWTDuration)
	if err != nil {
		return Config{}, pkgerrors.WithMessage(err, "load jwt duration")
	}

	rankingInterval, err := envDuration("RANKING_RECOMPUTE_INTERVAL", defaultRankingInterval)
	if err != nil {
		return Config{}, pkgerrors.WithMessage(err, "load ranking recompute interval")
	}

	viewFlushInterval, err := envDuration("VIEW_FLUSH_INTERVAL", defaultViewFlushInterval)
	if err != nil {
		return Config{}, pkgerrors.WithMessage(err, "load view flush interval")
	}

	cfg := Config{
		Server:   loadServerConfig(),
		Database: loadDatabaseConfig(),
		Search:   SearchConfig{BlevePath: envString("BLEVE_PATH", defaultBlevePath)},
		COS:      loadCOSConfig(),
		Security: loadSecurityConfig(jwtDuration),
		Ranking:  loadRankingConfig(rankingInterval),
		View:     ViewConfig{FlushInterval: viewFlushInterval},
		Admin:    loadAdminConfig(),
		CORS:     CORSConfig{AllowOrigins: envList("CORS_ALLOW_ORIGIN", defaultCORSAllowOrigin)},
		Log:      LogConfig{Level: envString("LOG_LEVEL", defaultLogLevel)},
	}
	if err := validateSecurityConfig(cfg.Security); err != nil {
		return Config{}, pkgerrors.WithMessage(err, "validate security config")
	}
	return cfg, nil
}

// loadServerConfig 读取 HTTP 服务配置。
func loadServerConfig() ServerConfig {
	return ServerConfig{
		Port:            envString("PORT", defaultPort),
		ShutdownTimeout: 10 * time.Second,
	}
}

// loadDatabaseConfig 读取 SQLite 配置。
func loadDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Path:              envString("SQLITE_PATH", defaultDBPath),
		BusyTimeoutMS:     envInt("SQLITE_BUSY_TIMEOUT_MS", defaultSQLiteTimeout),
		ReadMaxOpenConns:  envIntWithFallback("SQLITE_READ_MAX_OPEN_CONNS", runtime.NumCPU()*4), // 增加读连接池，提升并发能力
		WriteMaxOpenConns: 1,
	}
}

// loadCOSConfig 读取腾讯云 COS 配置。
func loadCOSConfig() COSConfig {
	return COSConfig{
		SecretID:  envString("COS_SECRET_ID", defaultCOSSecretID),
		SecretKey: envString("COS_SECRET_KEY", defaultCOSSecretKey),
		Bucket:    envString("COS_BUCKET", defaultCOSBucket),
		Region:    envString("COS_REGION", defaultCOSRegion),
		Domain:    envString("COS_DOMAIN", defaultCOSDomain),
		Prefix:    envString("COS_PREFIX", defaultCOSPrefix),
	}
}

// loadSecurityConfig 读取认证、防刷与请求体限制配置。
func loadSecurityConfig(jwtDuration time.Duration) SecurityConfig {
	return SecurityConfig{
		JWTSecret:           envString("JWT_SECRET", defaultJWTSecret),
		JWTDuration:         jwtDuration,
		MaxRequestBodyBytes: envIntWithFallback("MAX_REQUEST_BODY_BYTES", mustAtoi(defaultMaxRequestBody)),
		RateLimitRPS:        envFloatWithFallback("RATE_LIMIT_RPS", mustAtof(defaultRateLimitRPS)),
		RateLimitBurst:      envIntWithFallback("RATE_LIMIT_BURST", mustAtoi(defaultRateLimitBurst)),
	}
}

// validateSecurityConfig 拒绝会让公网服务裸奔的弱安全配置。
func validateSecurityConfig(cfg SecurityConfig) error {
	secret := strings.TrimSpace(cfg.JWTSecret)
	if secret == "" || secret == defaultJWTSecret || len(secret) < minJWTSecretLength {
		return pkgerrors.New("jwt secret must be configured with at least 32 characters")
	}
	if cfg.MaxRequestBodyBytes <= 0 {
		return pkgerrors.New("max request body bytes must be positive")
	}
	if cfg.RateLimitRPS <= 0 {
		return pkgerrors.New("rate limit rps must be positive")
	}
	if cfg.RateLimitBurst <= 0 {
		return pkgerrors.New("rate limit burst must be positive")
	}
	return nil
}

// loadRankingConfig 读取热榜计算配置。
func loadRankingConfig(recomputeInterval time.Duration) RankingConfig {
	return RankingConfig{
		WeightRating:      envFloat("RANKING_WEIGHT_RATING", defaultRankingWeight),
		WeightFavorite:    envFloat("RANKING_WEIGHT_FAVORITE", defaultRankingWeight),
		WeightView:        envFloat("RANKING_WEIGHT_VIEW", defaultRankingWeight),
		BayesianC:         envFloat("RANKING_BAYESIAN_C", defaultBayesianC),
		RecomputeInterval: recomputeInterval,
	}
}

// loadAdminConfig 读取首个管理员引导配置。
func loadAdminConfig() AdminConfig {
	return AdminConfig{
		Username: envString("ADMIN_USERNAME", ""),
		Password: envString("ADMIN_PASSWORD", ""),
	}
}

// envString 读取字符串环境变量。
func envString(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

// envList 读取逗号分隔的字符串列表，并去除空白项。
func envList(key string, fallback string) []string {
	raw := envString(key, fallback)
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

// envInt 读取整数环境变量。
func envInt(key string, fallback string) int {
	parsed, err := strconv.Atoi(envString(key, fallback))
	if err != nil {
		return 0
	}
	return parsed
}

// envIntWithFallback 读取整数环境变量，解析失败时返回数值默认值。
func envIntWithFallback(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

// envFloat 读取浮点数环境变量。
func envFloat(key string, fallback string) float64 {
	parsed, err := strconv.ParseFloat(envString(key, fallback), 64)
	if err != nil {
		return 0
	}
	return parsed
}

// envFloatWithFallback 读取浮点环境变量，解析失败时返回数值默认值。
func envFloatWithFallback(key string, fallback float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

// envDuration 读取时间间隔环境变量。
func envDuration(key string, fallback string) (time.Duration, error) {
	duration, err := time.ParseDuration(envString(key, fallback))
	if err != nil {
		return 0, pkgerrors.WithMessage(err, "parse duration")
	}
	return duration, nil
}

// mustAtoi 将常量默认值转换为整数。
func mustAtoi(value string) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		panic(err)
	}
	return parsed
}

// mustAtof 将常量默认值转换为浮点数。
func mustAtof(value string) float64 {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		panic(err)
	}
	return parsed
}
