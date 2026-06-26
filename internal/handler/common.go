package handler

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"go.uber.org/zap"

	apperrors "github.com/yachiyo/acgwarehouse/pkg/errors"
	"github.com/yachiyo/acgwarehouse/pkg/logger"
)

const (
	DefaultPage      = 1
	DefaultPageSize  = 20
	MaxPageSize      = 100
	DefaultSortField = "created_at"
	DefaultSortOrder = "desc"
)

// Response 定义统一 HTTP 响应结构。
type Response struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

// PageQuery 定义统一分页查询参数。
type PageQuery struct {
	Page  int
	Size  int
	Sort  string
	Order string
}

// ListResponse 定义统一分页列表响应数据。
type ListResponse struct {
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
	List  interface{} `json:"list"`
}

// Success 输出成功响应。
func Success(ctx *app.RequestContext, data interface{}) {
	ctx.JSON(consts.StatusOK, Response{Code: apperrors.CodeSuccess, Data: data, Msg: ""})
}

// Fail 输出失败响应并在 handler 层统一记录错误。
func Fail(c context.Context, ctx *app.RequestContext, status int, code int, msg string, err error) {
	if err != nil {
		logger.Error(c, "http request failed", zap.Error(err), zap.Int("code", code), zap.String("path", path(ctx)))
	}
	ctx.JSON(status, Response{Code: code, Data: nil, Msg: msg})
}

// NewListResponse 创建统一分页列表响应。
func NewListResponse(total int64, page PageQuery, list interface{}) ListResponse {
	return ListResponse{Total: total, Page: page.Page, Size: page.Size, List: list}
}

// ParsePageQuery 从请求中解析分页与排序参数。
func ParsePageQuery(ctx *app.RequestContext) PageQuery {
	return PageQuery{
		Page:  parsePositiveInt(query(ctx, "page"), DefaultPage),
		Size:  parsePageSize(query(ctx, "size")),
		Sort:  parseSort(query(ctx, "sort")),
		Order: parseOrder(query(ctx, "order")),
	}
}

// FormatTime 将时间转换为 UTC RFC3339 字符串。
func FormatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

// query 读取查询参数字符串。
func query(ctx *app.RequestContext, key string) string {
	return string(ctx.Query(key))
}

// path 读取请求路径字符串。
func path(ctx *app.RequestContext) string {
	return string(ctx.Path())
}

// parsePositiveInt 解析正整数。
func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

// parsePageSize 解析分页大小并按上限截断。
func parsePageSize(raw string) int {
	size := parsePositiveInt(raw, DefaultPageSize)
	if size > MaxPageSize {
		return MaxPageSize
	}
	return size
}

// parseSort 解析排序字段。
func parseSort(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return DefaultSortField
	}
	return value
}

// parseOrder 解析排序方向。
func parseOrder(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "asc" {
		return "asc"
	}
	return DefaultSortOrder
}
