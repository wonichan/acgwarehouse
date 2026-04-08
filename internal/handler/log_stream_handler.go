package handler

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

const defaultLogTail = 200

type LogStreamHandler struct {
	svc *service.LogStreamService
	cfg *config.Config
}

func NewLogStreamHandler(svc *service.LogStreamService, cfg *config.Config) *LogStreamHandler {
	return &LogStreamHandler{svc: svc, cfg: cfg}
}

func (h *LogStreamHandler) HandleLogStream(c *gin.Context) {
	if !authorizeAdminWS(c, h.cfg) {
		return
	}

	sourceValue := strings.TrimSpace(c.Query("source"))
	if sourceValue == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source query parameter is required"})
		return
	}

	source := service.LogSource(sourceValue)
	if source != service.LogSourceGo {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source"})
		return
	}

	tail := defaultLogTail
	if tailValue := strings.TrimSpace(c.Query("tail")); tailValue != "" {
		parsedTail, err := strconv.Atoi(tailValue)
		if err != nil || parsedTail <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tail"})
			return
		}
		tail = parsedTail
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	if h.svc == nil {
		_ = conn.WriteJSON(service.LogEvent{
			Type:      "status",
			Source:    string(source),
			Payload:   "service unavailable",
			Timestamp: time.Now().UTC(),
		})
		return
	}

	events, unsubscribe := h.svc.Subscribe(source, tail)
	defer unsubscribe()

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
		case <-c.Request.Context().Done():
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
