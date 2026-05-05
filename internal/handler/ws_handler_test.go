package handler

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

func TestWSHandlerServeHTTPUpgradesAndStreamsEvents(t *testing.T) {
	t.Parallel()

	bus := service.NewMonitoringEventBus(nil)
	handler := NewWSHandler(bus)
	server := httptest.NewServer(handler)
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsTestURL(server.URL), nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	assertEventually(t, time.Second, func() bool {
		return wsSubscriberCount(bus) == 1
	})
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline() error = %v", err)
	}

	want := service.MonitoringEvent{
		Type:      "overview",
		Payload:   json.RawMessage(`{"queue":{"queue_size":5}}`),
		Timestamp: time.Now().UTC().Round(0),
	}
	bus.Broadcast(want)

	var got service.MonitoringEvent
	if err := conn.ReadJSON(&got); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}
	if got.Type != want.Type {
		t.Fatalf("event.Type = %q, want %q", got.Type, want.Type)
	}
	if string(got.Payload) != string(want.Payload) {
		t.Fatalf("event.Payload = %s, want %s", got.Payload, want.Payload)
	}
}

func TestWSHandlerServeHTTPUnsubscribesOnClientDisconnect(t *testing.T) {
	t.Parallel()

	bus := service.NewMonitoringEventBus(nil)
	handler := NewWSHandler(bus)
	server := httptest.NewServer(handler)
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsTestURL(server.URL), nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}

	assertEventually(t, time.Second, func() bool {
		return wsSubscriberCount(bus) == 1
	})

	if err := conn.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	assertEventually(t, 2*time.Second, func() bool {
		return wsSubscriberCount(bus) == 0
	})
}

func TestWSHandlerServeHTTPRejectsUnauthorizedRequests(t *testing.T) {
	t.Parallel()

	bus := service.NewMonitoringEventBus(nil)
	handler := NewWSHandler(bus)
	handler.cfg = &config.Config{
		Admin: config.AdminConfig{
			Username: "admin",
			Password: "secret",
		},
	}

	server := httptest.NewServer(handler)
	defer server.Close()

	_, resp, err := websocket.DefaultDialer.Dial(wsTestURL(server.URL), nil)
	if err == nil {
		t.Fatal("Dial() unexpectedly succeeded without credentials")
	}
	if resp == nil {
		t.Fatal("expected HTTP response for unauthorized WebSocket upgrade")
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	headers := http.Header{}
	headers.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	conn, resp, err := websocket.DefaultDialer.Dial(wsTestURL(server.URL), headers)
	if err != nil {
		t.Fatalf("Dial() with valid credentials error = %v (status=%v)", err, resp)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	assertEventually(t, time.Second, func() bool {
		return wsSubscriberCount(bus) == 0
	})
}

func wsTestURL(serverURL string) string {
	return "ws" + strings.TrimPrefix(serverURL, "http")
}

func wsSubscriberCount(bus *service.MonitoringEventBus) int {
	value := reflect.ValueOf(bus).Elem().FieldByName("subscribers")
	return value.Len()
}

func assertEventually(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}
