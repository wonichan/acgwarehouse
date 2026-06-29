package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/yachiyo/acgwarehouse/internal/handler"
	"github.com/yachiyo/acgwarehouse/pkg/errors"
)

// RequestBodyLimit 拒绝超过上限的请求体。
func RequestBodyLimit(maxBytes int) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		if maxBytes <= 0 {
			ctx.Next(c)
			return
		}
		contentLength := ctx.Request.Header.ContentLength()
		if contentLength > maxBytes || len(ctx.Request.Body()) > maxBytes {
			handler.Fail(c, ctx, consts.StatusRequestEntityTooLarge, errors.CodeInvalidParam, "请求体过大", nil)
			ctx.Abort()
			return
		}
		ctx.Next(c)
	}
}
