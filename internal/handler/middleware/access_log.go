package middleware

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"go.uber.org/zap"

	"github.com/yachiyo/acgwarehouse/pkg/logger"
)

// AccessLog 记录接口访问日志。
func AccessLog() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		start := time.Now()
		requestBodyBytes := ctx.Request.Header.ContentLength()
		if requestBodyBytes < 0 {
			requestBodyBytes = 0
		}

		ctx.Next(c)

		path := string(ctx.Path())
		route := ctx.FullPath()
		if route == "" {
			route = path
		}

		logger.Access(c,
			zap.String("route", route),
			zap.String("method", string(ctx.Method())),
			zap.String("path", path),
			zap.String("client_ip", ctx.ClientIP()),
			zap.String("user_agent", string(ctx.UserAgent())),
			zap.Int("request_body_bytes", requestBodyBytes),
			zap.Int("response_body_bytes", len(ctx.Response.BodyBytes())),
			zap.Int64("duration_ms", time.Since(start).Milliseconds()),
			zap.Int("status_code", ctx.Response.StatusCode()),
		)
	}
}
