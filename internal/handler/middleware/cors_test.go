package middleware_test

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/yachiyo/acgwarehouse/internal/handler/middleware"
)

// TestCORSDoesNotAllowOriginWhenWhitelistEmpty 确认空白名单不会默认放行任意 Origin。
func TestCORSDoesNotAllowOriginWhenWhitelistEmpty(t *testing.T) {
	engine := corsTestEngine(middleware.CORS())

	recorder := ut.PerformRequest(
		engine.Engine,
		consts.MethodGet,
		"/ping",
		nil,
		ut.Header{Key: "Origin", Value: "https://evil.example.com"},
	)

	if got := recorder.Result().Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
}

// TestCORSAllowsOnlyConfiguredOrigin 确认只有白名单 Origin 会被放行。
func TestCORSAllowsOnlyConfiguredOrigin(t *testing.T) {
	engine := corsTestEngine(middleware.CORS("https://app.example.com"))

	recorder := ut.PerformRequest(
		engine.Engine,
		consts.MethodGet,
		"/ping",
		nil,
		ut.Header{Key: "Origin", Value: "https://app.example.com"},
	)

	if got := recorder.Result().Header.Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want configured origin", got)
	}
	if got := recorder.Result().Header.Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want true", got)
	}
}

// TestCORSPreflightReturnsNoContent 确认预检请求仍能正常结束。
func TestCORSPreflightReturnsNoContent(t *testing.T) {
	engine := corsTestEngine(middleware.CORS("https://app.example.com"))

	recorder := ut.PerformRequest(
		engine.Engine,
		consts.MethodOptions,
		"/ping",
		nil,
		ut.Header{Key: "Origin", Value: "https://app.example.com"},
	)

	if recorder.Code != consts.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
}

// corsTestEngine 创建 CORS 中间件测试引擎。
func corsTestEngine(cors app.HandlerFunc) *server.Hertz {
	engine := server.Default()
	engine.Use(cors)
	engine.GET("/ping", func(c context.Context, ctx *app.RequestContext) {
		ctx.String(consts.StatusOK, "pong")
	})
	return engine
}
