package logger

import (
	"context"
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	globalMu sync.RWMutex
	global   = zap.NewNop()
)

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

// ReplaceGlobal 替换全局日志实例。
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

// Sync 刷新全局日志缓冲。
func Sync() error {
	err := current().Sync()
	if err != nil && strings.Contains(err.Error(), os.Stdout.Name()) {
		return nil
	}
	if err != nil && strings.Contains(err.Error(), os.Stderr.Name()) {
		return nil
	}
	return err
}

// parseLevel 解析日志级别。
func parseLevel(level string) (zapcore.Level, error) {
	var parsed zapcore.Level
	if err := parsed.UnmarshalText([]byte(strings.ToLower(level))); err != nil {
		return zapcore.InfoLevel, err
	}
	return parsed, nil
}

// current 返回当前全局日志实例。
func current() *zap.Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return global
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
