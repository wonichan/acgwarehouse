package router_test

import (
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/yachiyo/acgwarehouse/internal/handler/router"
	"github.com/yachiyo/acgwarehouse/internal/service"
	jwtpkg "github.com/yachiyo/acgwarehouse/pkg/jwt"
)

// TestRegisterWithOptionsRateLimitsLoginRoute 确认登录路由可按安全选项限流。
func TestRegisterWithOptionsRateLimitsLoginRoute(t *testing.T) {
	repo := newMemoryRouterUserRepository()
	svc := service.NewUserService(repo, jwtpkg.NewManager("test-secret", 0))
	engine := server.Default()
	router.RegisterWithOptions(engine, router.Services{User: svc}, jwtpkg.NewManager("test-secret", 0), router.Options{
		RateLimitRPS:   1,
		RateLimitBurst: 1,
	})
	bodyText := `{"username":"alice","password":"secret1"}`

	first := ut.PerformRequest(
		engine.Engine,
		consts.MethodPost,
		"/api/v1/users/login",
		&ut.Body{Body: strings.NewReader(bodyText), Len: len(bodyText)},
		jsonHeader(),
	)
	second := ut.PerformRequest(
		engine.Engine,
		consts.MethodPost,
		"/api/v1/users/login",
		&ut.Body{Body: strings.NewReader(bodyText), Len: len(bodyText)},
		jsonHeader(),
	)

	if first.Code == consts.StatusTooManyRequests {
		t.Fatalf("first status = %d, want not 429", first.Code)
	}
	if second.Code != consts.StatusTooManyRequests {
		t.Fatalf("second status = %d, want 429", second.Code)
	}
}
