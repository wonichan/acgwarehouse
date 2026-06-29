package middleware

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

const (
	allowMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	allowHeaders = "Content-Type, Authorization, X-Requested-With"
)

// CORS 返回按白名单放行的跨域处理中间件。
func CORS(allowOrigins ...string) app.HandlerFunc {
	allowed := normalizeAllowedOrigins(allowOrigins)
	return func(c context.Context, ctx *app.RequestContext) {
		origin := strings.TrimSpace(string(ctx.GetHeader("Origin")))
		if allowedOrigin, ok := matchAllowedOrigin(origin, allowed); ok {
			ctx.Header("Access-Control-Allow-Origin", allowedOrigin)
			ctx.Header("Vary", "Origin")
			ctx.Header("Access-Control-Allow-Methods", allowMethods)
			ctx.Header("Access-Control-Allow-Headers", allowHeaders)
			if allowedOrigin != "*" {
				ctx.Header("Access-Control-Allow-Credentials", "true")
			}
		}

		if string(ctx.Method()) == consts.MethodOptions {
			ctx.AbortWithStatus(consts.StatusNoContent)
			return
		}
		ctx.Next(c)
	}
}

// normalizeAllowedOrigins 标准化跨域白名单配置。
func normalizeAllowedOrigins(origins []string) map[string]struct{} {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			allowed[trimmed] = struct{}{}
		}
	}
	return allowed
}

// matchAllowedOrigin 判断请求 Origin 是否命中白名单。
func matchAllowedOrigin(origin string, allowed map[string]struct{}) (string, bool) {
	if origin == "" {
		return "", false
	}
	if _, ok := allowed["*"]; ok {
		return "*", true
	}
	if _, ok := allowed[origin]; ok {
		return origin, true
	}
	return "", false
}
