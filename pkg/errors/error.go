package errors

import pkgerrors "github.com/pkg/errors"

const (
	// CodeSuccess 表示业务处理成功。
	CodeSuccess = 0
	// CodeInvalidParam 表示请求参数校验失败。
	CodeInvalidParam = 40001
	// CodeUnauthorized 表示认证失败或未登录。
	CodeUnauthorized = 40101
	// CodeForbidden 表示权限不足或拒绝访问。
	CodeForbidden = 40301
	// CodeNotFound 表示目标数据不存在。
	CodeNotFound = 40401
	// CodeInternal 表示服务器内部错误。
	CodeInternal = 50001
)

// New 创建带堆栈信息的新错误。
func New(message string) error {
	return pkgerrors.New(message)
}

// WithMessage 为下游错误附加上下文并保留堆栈链路。
func WithMessage(err error, message string) error {
	return pkgerrors.WithMessage(err, message)
}

// Cause 返回错误链中的根因。
func Cause(err error) error {
	return pkgerrors.Cause(err)
}
