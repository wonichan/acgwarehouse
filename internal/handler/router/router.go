package router

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/yachiyo/acgwarehouse/internal/conf"
	"github.com/yachiyo/acgwarehouse/internal/handler"
	"github.com/yachiyo/acgwarehouse/internal/handler/middleware"
)

// New 创建 Hertz 路由引擎并注册基础路由。
func New(cfg conf.Config) *server.Hertz {
	engine := server.Default(server.WithHostPorts(cfg.Server.Address()))
	engine.Use(middleware.CORS(cfg.CORS.AllowOrigin))
	Register(engine)
	return engine
}

// Register 注册 API v1 路由骨架。
func Register(engine *server.Hertz) {
	v1 := engine.Group("/api/v1")
	v1.GET("/ping", ping)
}

// ping 返回服务健康检查响应。
func ping(c context.Context, ctx *app.RequestContext) {
	handler.Success(ctx, map[string]string{"message": "pong"})
	ctx.SetStatusCode(consts.StatusOK)
}
