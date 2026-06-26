package router

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"

	"github.com/yachiyo/acgwarehouse/internal/conf"
	"github.com/yachiyo/acgwarehouse/internal/handler"
	"github.com/yachiyo/acgwarehouse/internal/handler/middleware"
	"github.com/yachiyo/acgwarehouse/internal/service"
	jwtpkg "github.com/yachiyo/acgwarehouse/pkg/jwt"
)

// New 创建 Hertz 路由引擎并注册基础路由。
func New(
	cfg conf.Config,
	userService *service.UserService,
	imageService *service.ImageService,
	tagService *service.TagService,
) *server.Hertz {
	engine := server.Default(server.WithHostPorts(cfg.Server.Address()))
	engine.Use(middleware.CORS(cfg.CORS.AllowOrigin))
	Register(engine, userService, imageService, tagService, jwtpkg.NewManager(cfg.Security.JWTSecret, cfg.Security.JWTDuration))
	return engine
}

// Register 注册 API v1 路由骨架。
func Register(
	engine *server.Hertz,
	userService *service.UserService,
	imageService *service.ImageService,
	tagService *service.TagService,
	jwtManager *jwtpkg.Manager,
) {
	v1 := engine.Group("/api/v1")
	v1.GET("/ping", ping)
	registerUserRoutes(v1, userService, jwtManager)
	registerImageRoutes(v1, imageService, jwtManager)
	registerTagRoutes(v1, tagService, jwtManager)
}

// registerUserRoutes 注册用户认证路由。
func registerUserRoutes(group *route.RouterGroup, userService *service.UserService, jwtManager *jwtpkg.Manager) {
	userHandler := handler.NewUserHandler(userService)
	users := group.Group("/users")
	users.POST("/register", userHandler.Register)
	users.POST("/login", userHandler.Login)
	users.GET("/me", middleware.Auth(jwtManager), userHandler.Me)
}

// registerImageRoutes 注册图片查询与生命周期路由。
func registerImageRoutes(group *route.RouterGroup, imageService *service.ImageService, jwtManager *jwtpkg.Manager) {
	imageHandler := handler.NewImageHandler(imageService)
	group.GET("/images", imageHandler.List)
	group.GET("/images/:id", imageHandler.Detail)
	group.GET("/search", imageHandler.Search)
	group.DELETE("/images/:id", middleware.Auth(jwtManager), middleware.RequireAdmin(), imageHandler.SoftDelete)
	group.POST("/images/:id/restore", middleware.Auth(jwtManager), middleware.RequireAdmin(), imageHandler.Restore)
}

// registerTagRoutes 注册标签管理路由。
func registerTagRoutes(group *route.RouterGroup, tagService *service.TagService, jwtManager *jwtpkg.Manager) {
	tagHandler := handler.NewTagHandler(tagService)
	group.GET("/tags", tagHandler.List)
	group.GET("/tags/suggest", tagHandler.Suggest)
	group.POST("/tags", middleware.Auth(jwtManager), tagHandler.Create)
	group.PUT("/tags/:id", middleware.Auth(jwtManager), middleware.RequireAdmin(), tagHandler.Update)
	group.DELETE("/tags/:id", middleware.Auth(jwtManager), middleware.RequireAdmin(), tagHandler.Delete)
	group.POST("/images/tags", middleware.Auth(jwtManager), tagHandler.Assign)
	group.DELETE("/images/tags", middleware.Auth(jwtManager), tagHandler.Unassign)
}

// ping 返回服务健康检查响应。
func ping(c context.Context, ctx *app.RequestContext) {
	handler.Success(ctx, map[string]string{"message": "pong"})
	ctx.SetStatusCode(consts.StatusOK)
}
