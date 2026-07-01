package logger_test

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/yachiyo/acgwarehouse/pkg/logger"
)

// TestFileLoggersWriteAppAndAccessToSeparateDateFiles 确认应用日志和接口日志写入不同的日期文件。
func TestFileLoggersWriteAppAndAccessToSeparateDateFiles(t *testing.T) {
	// Given
	logDir := t.TempDir()
	loggers, err := logger.NewFiles(logger.FileConfig{
		Level:  "info",
		LogDir: logDir,
	})
	if err != nil {
		t.Fatalf("NewFiles error = %v", err)
	}
	restore := logger.ReplaceGlobals(loggers)
	t.Cleanup(restore)
	ctx := context.Background()

	// When
	logger.Info(ctx, "app event", zap.String("component", "sqlite"))
	logger.Access(ctx,
		zap.String("route", "/api/v1/ping"),
		zap.Int("status_code", 200),
	)
	if err := logger.Sync(); err != nil {
		t.Fatalf("Sync error = %v", err)
	}

	// Then
	dateSuffix := time.Now().Format("20060102")
	appEntries := readJSONLogEntries(t, filepath.Join(logDir, "app."+dateSuffix+".log"))
	accessEntries := readJSONLogEntries(t, filepath.Join(logDir, "access."+dateSuffix+".log"))

	if len(appEntries) != 1 {
		t.Fatalf("app entries = %d, want 1", len(appEntries))
	}
	if appEntries[0]["msg"] != "app event" {
		t.Fatalf("app msg = %v, want app event", appEntries[0]["msg"])
	}
	if appEntries[0]["component"] != "sqlite" {
		t.Fatalf("app component = %v, want sqlite", appEntries[0]["component"])
	}

	if len(accessEntries) != 1 {
		t.Fatalf("access entries = %d, want 1", len(accessEntries))
	}
	if accessEntries[0]["msg"] != "api access" {
		t.Fatalf("access msg = %v, want api access", accessEntries[0]["msg"])
	}
	if accessEntries[0]["route"] != "/api/v1/ping" {
		t.Fatalf("access route = %v, want /api/v1/ping", accessEntries[0]["route"])
	}

	if _, ok := appEntries[0]["route"]; ok {
		t.Fatalf("app entry unexpectedly contains access route field: %#v", appEntries[0])
	}
	if _, ok := accessEntries[0]["component"]; ok {
		t.Fatalf("access entry unexpectedly contains app component field: %#v", accessEntries[0])
	}
}

// TestFileLoggersCreateLogDirectory 确认日志目录不存在时会自动创建。
func TestFileLoggersCreateLogDirectory(t *testing.T) {
	// Given
	logDir := filepath.Join(t.TempDir(), "data", "log")

	// When
	loggers, err := logger.NewFiles(logger.FileConfig{
		Level:  "info",
		LogDir: logDir,
	})
	if err != nil {
		t.Fatalf("NewFiles error = %v", err)
	}
	restore := logger.ReplaceGlobals(loggers)
	t.Cleanup(restore)
	logger.Info(context.Background(), "directory check")
	if err := logger.Sync(); err != nil {
		t.Fatalf("Sync error = %v", err)
	}

	// Then
	info, err := os.Stat(logDir)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", logDir, err)
	}
	if !info.IsDir() {
		t.Fatalf("%q is not a directory", logDir)
	}
}

func readJSONLogEntries(t *testing.T, path string) []map[string]any {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open(%q) error = %v", path, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatalf("Close(%q) error = %v", path, err)
		}
	}()

	entries := make([]map[string]any, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		entry := map[string]any{}
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf("Unmarshal log line %q error = %v", scanner.Text(), err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("Scan(%q) error = %v", path, err)
	}
	return entries
}
