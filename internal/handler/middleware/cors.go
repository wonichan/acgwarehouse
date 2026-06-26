package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

const (
	allowMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	allowHeaders = "Content-Type, Authorization, X-Requested-With"
)

// CORS 返回跨域处理中间件。
func CORS(allowOrigin string) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		ctx.Header("Access-Control-Allow-Origin", allowOrigin)
		ctx.Header("Access-Control-Allow-Methods", allowMethods)
		ctx.Header("Access-Control-Allow-Headers", allowHeaders)
		if allowOrigin != "*" {
			ctx.Header("Access-Control-Allow-Credentials", "true")
		}

		if string(ctx.Method()) == consts.MethodOptions {
			ctx.AbortWithStatus(consts.StatusNoContent)
			return
		}
		ctx.Next(c)
	}
}
