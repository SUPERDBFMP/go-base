package errs

import "fmt"

// WebApi错误码
var (
	Success         = NewBizError("Success", "操作成功")
	ErrSystem       = NewBizError("Err.System", "系统异常，请稍后再试")
	ErrInvalidParam = NewBizError("Err.InvalidParam", "参数错误")
)

// BizError 业务异常
type BizError struct {
	Code    string
	Message string
}

// Error 实现 error 接口，使 BizError 可作为 error 类型使用
func (e *BizError) Error() string {
	return fmt.Sprintf("code: %s, msg: %s", e.Code, e.Message)
}

// NewBizError 创建一个新的业务错误
func NewBizError(code, msg string) *BizError {
	return &BizError{
		Code:    code,
		Message: msg,
	}
}
