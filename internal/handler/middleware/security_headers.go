package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

const (
	headerContentTypeOptions = "X-Content-Type-Options"
	headerFrameOptions       = "X-Frame-Options"
	headerReferrerPolicy     = "Referrer-Policy"
	headerCSP                = "Content-Security-Policy"
)

// SecurityHeaders 为所有响应增加基础浏览器安全头。
func SecurityHeaders() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		ctx.Header(headerContentTypeOptions, "nosniff")
		ctx.Header(headerFrameOptions, "DENY")
		ctx.Header(headerReferrerPolicy, "no-referrer")
		ctx.Header(headerCSP, "default-src 'self'; frame-ancestors 'none'")
		ctx.Next(c)
	}
}
