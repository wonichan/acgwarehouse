package handler

import (
	"encoding/base64"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

type WSHandler struct {
	bus      *service.MonitoringEventBus
	upgrader websocket.Upgrader
	cfg      *config.Config
}

func NewWSHandler(bus *service.MonitoringEventBus) *WSHandler {
	return &WSHandler{
		bus: bus,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.authorized(w, r) {
		return
	}
	if h.bus == nil {
		http.Error(w, "monitoring bus not configured", http.StatusServiceUnavailable)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	events, unsubscribe := h.bus.Subscribe()
	defer unsubscribe()
	defer conn.Close()

	done := make(chan struct{})
	var doneOnce sync.Once
	closeDone := func() {
		doneOnce.Do(func() {
			close(done)
		})
	}

	go func() {
		defer closeDone()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-done:
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if err := conn.WriteJSON(event); err != nil {
				closeDone()
				return
			}
		}
	}
}

func (h *WSHandler) authorized(w http.ResponseWriter, r *http.Request) bool {
	if h.cfg == nil || (h.cfg.Admin.Username == "" && h.cfg.Admin.Password == "") {
		return true
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.Header().Set("WWW-Authenticate", `Basic realm="admin"`)
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	if !strings.HasPrefix(authHeader, "Basic ") {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}

	if parts[0] != h.cfg.Admin.Username || parts[1] != h.cfg.Admin.Password {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}

	return true
}
