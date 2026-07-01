package router_test

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/yachiyo/acgwarehouse/internal/handler/router"
	"github.com/yachiyo/acgwarehouse/pkg/jwt"
	"github.com/yachiyo/acgwarehouse/pkg/logger"
)

// TestRegisterWithOptionsWritesAccessLogForAPIV1Route_whenPingCompletes 确认 API v1 路由写入接口日志。
func TestRegisterWithOptionsWritesAccessLogForAPIV1Route_whenPingCompletes(t *testing.T) {
	// Given
	logDir := setupRouterAccessLogTest(t)
	engine := server.Default()
	router.RegisterWithOptions(engine, router.Services{}, jwt.NewManager("test-secret", time.Hour), router.Options{})

	// When
	response := ut.PerformRequest(engine.Engine, consts.MethodGet, "/api/v1/ping", nil)
	if response.Code != consts.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if err := logger.Sync(); err != nil {
		t.Fatalf("Sync error = %v", err)
	}

	// Then
	entries := readRouterAccessEntries(t, logDir)
	if len(entries) != 1 {
		t.Fatalf("access entries = %d, want 1: %#v", len(entries), entries)
	}
	if entries[0]["route"] != "/api/v1/ping" {
		t.Fatalf("route = %v, want /api/v1/ping", entries[0]["route"])
	}
}

// TestRegisterWithOptionsSkipsAccessLogForNonAPIRoute_whenOutsideGroupCompletes 确认非 API v1 路由不写接口日志。
func TestRegisterWithOptionsSkipsAccessLogForNonAPIRoute_whenOutsideGroupCompletes(t *testing.T) {
	// Given
	logDir := setupRouterAccessLogTest(t)
	engine := server.Default()
	router.RegisterWithOptions(engine, router.Services{}, jwt.NewManager("test-secret", time.Hour), router.Options{})
	engine.GET("/health", func(_ context.Context, ctx *app.RequestContext) {
		ctx.String(consts.StatusOK, "ok")
	})

	// When
	response := ut.PerformRequest(engine.Engine, consts.MethodGet, "/health", nil)
	if response.Code != consts.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if err := logger.Sync(); err != nil {
		t.Fatalf("Sync error = %v", err)
	}

	// Then
	entries := readRouterAccessEntries(t, logDir)
	if len(entries) != 0 {
		t.Fatalf("access entries = %d, want 0: %#v", len(entries), entries)
	}
}

func setupRouterAccessLogTest(t *testing.T) string {
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

func readRouterAccessEntries(t *testing.T, logDir string) []map[string]any {
	t.Helper()

	path := filepath.Join(logDir, "access."+time.Now().Format("20060102")+".log")
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
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
