package middleware_test

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/yachiyo/acgwarehouse/internal/handler/middleware"
	"github.com/yachiyo/acgwarehouse/pkg/logger"
)

// TestAccessLogWritesRequiredFields_whenAPIRequestCompletes 确认接口完成日志包含必需字段。
func TestAccessLogWritesRequiredFields_whenAPIRequestCompletes(t *testing.T) {
	// Given
	logDir := setupAccessLogTest(t)
	engine := server.Default()
	engine.Use(middleware.AccessLog())
	engine.POST("/api/v1/images/:id/rating", func(_ context.Context, ctx *app.RequestContext) {
		ctx.String(consts.StatusCreated, "created")
	})
	body := &ut.Body{Body: strings.NewReader(`{"score":80}`), Len: 12}

	// When
	request := ut.PerformRequest(engine.Engine, consts.MethodPost, "/api/v1/images/42/rating?token=secret", body,
		ut.Header{Key: "User-Agent", Value: "acg-client/1.0"},
		ut.Header{Key: "Authorization", Value: "Bearer secret-token"},
		ut.Header{Key: "Cookie", Value: "session=secret-cookie"},
	)
	if request.Code != consts.StatusCreated {
		t.Fatalf("status = %d, want 201", request.Code)
	}
	if err := logger.Sync(); err != nil {
		t.Fatalf("Sync error = %v", err)
	}

	// Then
	entry := singleAccessEntry(t, logDir)
	assertStringField(t, entry, "msg", "api access")
	assertStringField(t, entry, "route", "/api/v1/images/:id/rating")
	assertStringField(t, entry, "method", consts.MethodPost)
	assertStringField(t, entry, "path", "/api/v1/images/42/rating")
	assertStringField(t, entry, "user_agent", "acg-client/1.0")
	assertNumberField(t, entry, "request_body_bytes", 12)
	assertNumberField(t, entry, "response_body_bytes", len("created"))
	assertNumberField(t, entry, "status_code", consts.StatusCreated)
	if _, ok := entry["client_ip"]; !ok {
		t.Fatalf("client_ip field missing: %#v", entry)
	}
	if duration, ok := entry["duration_ms"].(float64); !ok || duration < 0 {
		t.Fatalf("duration_ms = %#v, want non-negative number", entry["duration_ms"])
	}
	assertLogOmits(t, entry, "token=secret", "secret-token", "secret-cookie", `{"score":80}`, "created")
}

// TestAccessLogNormalizesUnknownRequestBodySize_whenContentLengthMissing 确认未知请求体大小记录为 0。
func TestAccessLogNormalizesUnknownRequestBodySize_whenContentLengthMissing(t *testing.T) {
	// Given
	logDir := setupAccessLogTest(t)
	engine := server.Default()
	engine.Use(middleware.AccessLog())
	engine.GET("/api/v1/ping", func(_ context.Context, ctx *app.RequestContext) {
		ctx.String(consts.StatusOK, "ok")
	})

	// When
	request := ut.PerformRequest(engine.Engine, consts.MethodGet, "/api/v1/ping", nil)
	if request.Code != consts.StatusOK {
		t.Fatalf("status = %d, want 200", request.Code)
	}
	if err := logger.Sync(); err != nil {
		t.Fatalf("Sync error = %v", err)
	}

	// Then
	entry := singleAccessEntry(t, logDir)
	assertNumberField(t, entry, "request_body_bytes", 0)
}

func setupAccessLogTest(t *testing.T) string {
	t.Helper()

	logDir := t.TempDir()
	loggers, err := logger.NewFiles(logger.FileConfig{Level: "info", LogDir: logDir})
	if err != nil {
		t.Fatalf("NewFiles error = %v", err)
	}
	restore := logger.ReplaceGlobals(loggers)
	t.Cleanup(restore)
	return logDir
}

func singleAccessEntry(t *testing.T, logDir string) map[string]any {
	t.Helper()

	entries := readAccessEntries(t, logDir)
	if len(entries) != 1 {
		t.Fatalf("access entries = %d, want 1: %#v", len(entries), entries)
	}
	return entries[0]
}

func readAccessEntries(t *testing.T, logDir string) []map[string]any {
	t.Helper()

	path := filepath.Join(logDir, "access."+time.Now().Format("20060102")+".log")
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

func assertStringField(t *testing.T, entry map[string]any, key string, want string) {
	t.Helper()

	got, ok := entry[key].(string)
	if !ok {
		t.Fatalf("%s = %#v, want string %q", key, entry[key], want)
	}
	if got != want {
		t.Fatalf("%s = %q, want %q", key, got, want)
	}
}

func assertNumberField(t *testing.T, entry map[string]any, key string, want int) {
	t.Helper()

	got, ok := entry[key].(float64)
	if !ok {
		t.Fatalf("%s = %#v, want number %d", key, entry[key], want)
	}
	if int(got) != want {
		t.Fatalf("%s = %v, want %d", key, got, want)
	}
}

func assertLogOmits(t *testing.T, entry map[string]any, values ...string) {
	t.Helper()

	encoded, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Marshal entry error = %v", err)
	}
	line := string(encoded)
	for _, value := range values {
		if strings.Contains(line, value) {
			t.Fatalf("log entry %s unexpectedly contains %q", line, value)
		}
	}
}
