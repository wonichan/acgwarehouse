package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/yachiyo/acgwarehouse/internal/handler"
	"github.com/yachiyo/acgwarehouse/pkg/errors"
	jwtpkg "github.com/yachiyo/acgwarehouse/pkg/jwt"
)

const (
	authorizationHeader = "Authorization"
	bearerPrefix        = "Bearer "
	userIDContextKey    = "user_id"
	roleContextKey      = "role"
	adminRole           = "admin"
)

// Auth 校验 Bearer JWT 并注入用户身份上下文。
func Auth(jwtManager *jwtpkg.Manager) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		token := bearerToken(ctx)
		if token == "" {
			handler.Fail(c, ctx, consts.StatusUnauthorized, errors.CodeUnauthorized, "请先登录", nil)
			ctx.Abort()
			return
		}
		claims, err := jwtManager.Parse(token, time.Now().UTC())
		if err != nil {
			handler.Fail(c, ctx, consts.StatusUnauthorized, errors.CodeUnauthorized, "请先登录", err)
			ctx.Abort()
			return
		}
		ctx.Set(userIDContextKey, claims.UserID)
		ctx.Set(roleContextKey, claims.Role)
		ctx.Next(c)
	}
}

// RequireAdmin 校验当前登录用户是否为管理员。
func RequireAdmin() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		role, ok := ctx.Get(roleContextKey)
		if !ok || role != adminRole {
			handler.Fail(c, ctx, consts.StatusForbidden, errors.CodeForbidden, "无权操作", nil)
			ctx.Abort()
			return
		}
		ctx.Next(c)
	}
}

// bearerToken 从 Authorization 头中提取 Bearer 令牌。
func bearerToken(ctx *app.RequestContext) string {
	header := string(ctx.GetHeader(authorizationHeader))
	if !strings.HasPrefix(header, bearerPrefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix))
}
