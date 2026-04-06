package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

func TestLogStreamHandler_RejectsUnauthorized(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	handler := NewLogStreamHandler(nil, &config.Config{
		Admin: config.AdminConfig{Username: "admin", Password: "secret"},
	})
	server := httptest.NewServer(ginHandlerForLogStream(handler))
	defer server.Close()

	_, resp, err := websocket.DefaultDialer.Dial(wsTestURL(server.URL)+"?source=go", nil)
	if err == nil {
		t.Fatal("Dial() unexpectedly succeeded without credentials")
	}
	if resp == nil {
		t.Fatal("expected HTTP response for unauthorized WebSocket upgrade")
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	conn, resp, err := websocket.DefaultDialer.Dial(wsTestURL(server.URL)+"?source=go", wsAuthHeader("admin", "secret"))
	if err != nil {
		t.Fatalf("Dial() with valid credentials error = %v (status=%v)", err, resp)
	}
	_ = conn.Close()
}

func TestLogStreamHandler_RejectsInvalidSource(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(ginHandlerForLogStream(NewLogStreamHandler(nil, nil)))
	defer server.Close()

	resp, err := http.Get(server.URL + "?source=invalid")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestLogStreamHandler_RejectsMissingSource(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(ginHandlerForLogStream(NewLogStreamHandler(nil, nil)))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestLogStreamHandler_AcceptsValidSource(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	goLogPath := filepath.Join(tempDir, "app.log")
	if err := os.WriteFile(goLogPath, []byte("line one\nline two\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	svc := service.NewLogStreamService(goLogPath, filepath.Join(tempDir, "sidecar.log"))
	svc.Start(context.Background())
	defer svc.Stop()

	handler := NewLogStreamHandler(svc, nil)
	server := httptest.NewServer(ginHandlerForLogStream(handler))
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsTestURL(server.URL)+"?source=go", nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	var event service.LogEvent
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}
	if event.Type != "snapshot" {
		t.Fatalf("event.Type = %q, want snapshot", event.Type)
	}
	if event.Source != string(service.LogSourceGo) {
		t.Fatalf("event.Source = %q, want %q", event.Source, service.LogSourceGo)
	}
	var lines []string
	if err := json.Unmarshal([]byte(event.Payload), &lines); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(lines) != 2 || lines[0] != "line one" || lines[1] != "line two" {
		t.Fatalf("snapshot lines = %#v, want [line one line two]", lines)
	}
}

func TestLogStreamHandler_ServiceNil_SendsUnavailable(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(ginHandlerForLogStream(NewLogStreamHandler(nil, nil)))
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsTestURL(server.URL)+"?source=go", nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	var event service.LogEvent
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}
	if event.Type != "status" {
		t.Fatalf("event.Type = %q, want status", event.Type)
	}
	if event.Payload != "service unavailable" {
		t.Fatalf("event.Payload = %q, want %q", event.Payload, "service unavailable")
	}
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("ReadMessage() unexpectedly succeeded after unavailable status")
	}
}

func ginHandlerForLogStream(handler *LogStreamHandler) http.Handler {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", handler.HandleLogStream)
	return r
}

func wsAuthHeader(username, password string) http.Header {
	headers := http.Header{}
	headers.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
	return headers
}
