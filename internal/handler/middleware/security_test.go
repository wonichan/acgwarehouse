package middleware_test

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/yachiyo/acgwarehouse/internal/handler/middleware"
)

// TestSecurityHeadersWritesBrowserProtectionHeaders 确认基础安全响应头存在。
func TestSecurityHeadersWritesBrowserProtectionHeaders(t *testing.T) {
	engine := server.Default()
	engine.Use(middleware.SecurityHeaders())
	engine.GET("/ping", okHandler)

	recorder := ut.PerformRequest(engine.Engine, consts.MethodGet, "/ping", nil)
	headers := recorder.Result().Header

	if headers.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", headers.Get("X-Content-Type-Options"))
	}
	if headers.Get("X-Frame-Options") != "DENY" {
		t.Fatalf("X-Frame-Options = %q, want DENY", headers.Get("X-Frame-Options"))
	}
	if headers.Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("Referrer-Policy = %q, want no-referrer", headers.Get("Referrer-Policy"))
	}
	if headers.Get("Content-Security-Policy") == "" {
		t.Fatal("Content-Security-Policy is empty")
	}
}

// TestRequestBodyLimitRejectsLargeBody 确认请求体超过上限会被拒绝。
func TestRequestBodyLimitRejectsLargeBody(t *testing.T) {
	engine := server.Default()
	engine.Use(middleware.RequestBodyLimit(4))
	engine.POST("/echo", okHandler)
	body := &ut.Body{Body: strings.NewReader("12345"), Len: 5}

	recorder := ut.PerformRequest(engine.Engine, consts.MethodPost, "/echo", body)

	if recorder.Code != consts.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d body=%s, want 413", recorder.Code, recorder.Body.String())
	}
}

// TestRateLimitRejectsWhenBurstExceeded 确认内存限流会拒绝高频请求。
func TestRateLimitRejectsWhenBurstExceeded(t *testing.T) {
	engine := server.Default()
	engine.Use(middleware.RateLimit(middleware.NewRateLimiter(1, 1)))
	engine.POST("/login", okHandler)

	first := ut.PerformRequest(engine.Engine, consts.MethodPost, "/login", nil)
	second := ut.PerformRequest(engine.Engine, consts.MethodPost, "/login", nil)

	if first.Code != consts.StatusOK {
		t.Fatalf("first status = %d, want 200", first.Code)
	}
	if second.Code != consts.StatusTooManyRequests {
		t.Fatalf("second status = %d, want 429", second.Code)
	}
}

// okHandler 返回测试成功响应。
func okHandler(_ context.Context, ctx *app.RequestContext) {
	ctx.String(consts.StatusOK, "ok")
}
