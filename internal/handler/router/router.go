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

// Services 聚合路由层依赖的业务服务。
type Services struct {
	User       *service.UserService
	Image      *service.ImageService
	Tag        *service.TagService
	Rating     *service.RatingService
	Collection *service.CollectionService
	Ranking    *service.RankingService
}

// New 创建 Hertz 路由引擎并注册基础路由。
func New(cfg conf.Config, services Services) *server.Hertz {
	engine := server.Default(server.WithHostPorts(cfg.Server.Address()))
	engine.Use(middleware.CORS(cfg.CORS.AllowOrigin))
	Register(
		engine,
		services,
		jwtpkg.NewManager(cfg.Security.JWTSecret, cfg.Security.JWTDuration),
	)
	return engine
}

// Register 注册 API v1 路由骨架。
func Register(engine *server.Hertz, services Services, jwtManager *jwtpkg.Manager) {
	v1 := engine.Group("/api/v1")
	v1.GET("/ping", ping)
	registerUserRoutes(v1, services.User, jwtManager)
	registerImageRoutes(v1, services.Image, jwtManager)
	registerTagRoutes(v1, services.Tag, jwtManager)
	registerRatingRoutes(v1, services.Rating, jwtManager)
	registerCollectionRoutes(v1, services.Collection, jwtManager)
	registerRankingRoutes(v1, services.Ranking)
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

// registerRatingRoutes 注册评分路由。
func registerRatingRoutes(group *route.RouterGroup, ratingService *service.RatingService, jwtManager *jwtpkg.Manager) {
	ratingHandler := handler.NewRatingHandler(ratingService)
	group.PUT("/images/:id/rating", middleware.Auth(jwtManager), ratingHandler.Rate)
}

// registerCollectionRoutes 注册收藏夹路由。
func registerCollectionRoutes(group *route.RouterGroup, collectionService *service.CollectionService, jwtManager *jwtpkg.Manager) {
	collectionHandler := handler.NewCollectionHandler(collectionService)
	group.GET("/collections", middleware.Auth(jwtManager), collectionHandler.ListMine)
	group.POST("/collections", middleware.Auth(jwtManager), collectionHandler.Create)
	group.GET("/collections/:id", collectionHandler.Detail)
	group.PUT("/collections/:id", middleware.Auth(jwtManager), collectionHandler.Update)
	group.DELETE("/collections/:id", middleware.Auth(jwtManager), collectionHandler.Delete)
	group.POST("/collections/:id/items", middleware.Auth(jwtManager), collectionHandler.AddItem)
	group.DELETE("/collections/:id/items/:imageId", middleware.Auth(jwtManager), collectionHandler.RemoveItem)
}

// registerRankingRoutes 注册热榜查询路由。
func registerRankingRoutes(group *route.RouterGroup, rankingService *service.RankingService) {
	rankingHandler := handler.NewRankingHandler(rankingService)
	group.GET("/rankings", rankingHandler.List)
}

// ping 返回服务健康检查响应。
func ping(c context.Context, ctx *app.RequestContext) {
	handler.Success(ctx, map[string]string{"message": "pong"})
	ctx.SetStatusCode(consts.StatusOK)
}
