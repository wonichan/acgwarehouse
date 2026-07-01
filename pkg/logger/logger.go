package logger

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gookit/rotatefile"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	appLogFilename    = "app.log"
	accessLogFilename = "access.log"
	defaultLogDir     = "data/log"
)

var (
	globalMu     sync.RWMutex
	global       = zap.NewNop()
	accessGlobal = zap.NewNop()
)

// FileConfig 保存文件日志配置。
type FileConfig struct {
	Level  string
	LogDir string
}

// Loggers 聚合应用日志与接口访问日志实例。
type Loggers struct {
	App    *zap.Logger
	Access *zap.Logger
}

// New 创建 zap 日志实例。
func New(level string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	parsedLevel, err := parseLevel(level)
	if err != nil {
		return nil, err
	}
	cfg.Level = zap.NewAtomicLevelAt(parsedLevel)

	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return logger, nil
}

// NewFiles 创建只写入文件的应用日志与接口日志实例。
func NewFiles(cfg FileConfig) (Loggers, error) {
	logDir := cfg.LogDir
	if logDir == "" {
		logDir = defaultLogDir
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return Loggers{}, errors.WithMessage(err, "create log dir")
	}

	parsedLevel, err := parseLevel(cfg.Level)
	if err != nil {
		return Loggers{}, err
	}
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	level := zap.NewAtomicLevelAt(parsedLevel)

	appWriter, err := newRotateWriter(filepath.Join(logDir, appLogFilename))
	if err != nil {
		return Loggers{}, errors.WithMessage(err, "create app log writer")
	}
	accessWriter, err := newRotateWriter(filepath.Join(logDir, accessLogFilename))
	if err != nil {
		closeRotateWriter(appWriter)
		return Loggers{}, errors.WithMessage(err, "create access log writer")
	}

	return Loggers{
		App: zap.New(zapcore.NewCore(encoder, zapcore.AddSync(appWriter), level)),
		Access: zap.New(zapcore.NewCore(
			encoder.Clone(),
			zapcore.AddSync(accessWriter),
			level,
		)),
	}, nil
}

// ReplaceGlobal 替换全局应用日志实例。
func ReplaceGlobal(next *zap.Logger) func() {
	globalMu.Lock()
	previous := global
	global = next
	globalMu.Unlock()

	return func() {
		globalMu.Lock()
		global = previous
		globalMu.Unlock()
	}
}

// ReplaceGlobals 替换全局应用日志与接口日志实例。
func ReplaceGlobals(next Loggers) func() {
	globalMu.Lock()
	previousApp := global
	previousAccess := accessGlobal
	global = next.App
	accessGlobal = next.Access
	globalMu.Unlock()

	return func() {
		globalMu.Lock()
		global = previousApp
		accessGlobal = previousAccess
		globalMu.Unlock()
	}
}

// Info 记录 info 级别结构化日志。
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	current().Info(msg, appendContextField(ctx, fields)...)
}

// Infof 记录 info 级别模板日志。
func Infof(ctx context.Context, format string, args ...interface{}) {
	current().Sugar().With(contextField(ctx)).Infof(format, args...)
}

// Warn 记录 warn 级别结构化日志。
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	current().Warn(msg, appendContextField(ctx, fields)...)
}

// Warnf 记录 warn 级别模板日志。
func Warnf(ctx context.Context, format string, args ...interface{}) {
	current().Sugar().With(contextField(ctx)).Warnf(format, args...)
}

// Error 记录 error 级别结构化日志。
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	current().Error(msg, appendContextField(ctx, fields)...)
}

// Errorf 记录 error 级别模板日志。
func Errorf(ctx context.Context, format string, args ...interface{}) {
	current().Sugar().With(contextField(ctx)).Errorf(format, args...)
}

// Access 记录接口访问日志。
func Access(ctx context.Context, fields ...zap.Field) {
	currentAccess().Info("api access", appendContextField(ctx, fields)...)
}

// Sync 刷新全局日志缓冲。
func Sync() error {
	if err := current().Sync(); shouldReturnSyncError(err) {
		return err
	}
	if err := currentAccess().Sync(); shouldReturnSyncError(err) {
		return err
	}
	return nil
}

// parseLevel 解析日志级别。
func parseLevel(level string) (zapcore.Level, error) {
	var parsed zapcore.Level
	if err := parsed.UnmarshalText([]byte(strings.ToLower(level))); err != nil {
		return zapcore.InfoLevel, err
	}
	return parsed, nil
}

// newRotateWriter 创建按日期和大小轮转的日志 writer。
func newRotateWriter(path string) (*rotatefile.Writer, error) {
	return rotatefile.NewConfig(path, func(c *rotatefile.Config) {
		c.RotateMode = rotatefile.ModeCreate
		c.RotateTime = rotatefile.EveryDay
		c.MaxSize = 100 * rotatefile.OneMByte
		c.Compress = true
	}).Create()
}

// closeRotateWriter 关闭已创建但不会继续使用的日志 writer。
func closeRotateWriter(writer *rotatefile.Writer) {
	if writer == nil {
		return
	}
	_ = writer.Close()
}

// current 返回当前全局应用日志实例。
func current() *zap.Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return global
}

// currentAccess 返回当前全局接口日志实例。
func currentAccess() *zap.Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return accessGlobal
}

// appendContextField 为结构化日志附加上下文占位字段。
func appendContextField(ctx context.Context, fields []zap.Field) []zap.Field {
	return append(fields, contextField(ctx))
}

// contextField 将上下文标记为日志字段。
func contextField(ctx context.Context) zap.Field {
	if ctx == nil {
		return zap.Bool("has_context", false)
	}
	return zap.Bool("has_context", true)
}

// shouldReturnSyncError 判断日志刷新错误是否需要上抛。
func shouldReturnSyncError(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(err.Error(), os.Stdout.Name()) {
		return false
	}
	if strings.Contains(err.Error(), os.Stderr.Name()) {
		return false
	}
	return true
}
