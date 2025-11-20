package web

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/SUPERDBFMP/go-base/errs"
	"github.com/SUPERDBFMP/go-base/glog"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type BaseResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WrapBizError 包装现有错误为业务错误
func WrapBizError(err *errs.BizError) *BaseResponse {
	return &BaseResponse{
		Code:    err.Code,
		Message: err.Message,
	}
}

func WrapSuccessResponse(resp *BaseResponse) {
	resp.Code = errs.Success.Code
	resp.Message = errs.Success.Message
}

var (
	// SuccessResponse 成功响应对象
	SuccessResponse = BaseResponse{Code: errs.Success.Code, Message: errs.Success.Message}

	// SystemErrResponse 系统异常
	SystemErrResponse = BaseResponse{Code: errs.ErrSystem.Code, Message: errs.ErrSystem.Message}
)

// BindAndValidate 绑定请求参数并校验，失败时打印原始参数和错误
// 参数：c Gin上下文，req 待绑定的结构体指针
// 返回：true=校验通过，false=校验失败（已自动返回错误响应）
func BindAndValidate(c *gin.Context, req interface{}) bool {
	// 1. 绑定请求参数到结构体（支持JSON/Form等）
	if err := c.ShouldBind(req); err != nil {

		// 2. 解析校验错误详情（区分普通错误和validator错误）
		var errMsg string
		var validationErrs validator.ValidationErrors
		if errors.As(err, &validationErrs) {
			// 自定义错误信息：包含字段名、实际值、校验规则
			errMsg = "参数错误："
			for _, e := range validationErrs {
				// 获取字段的实际值（通过反射）
				fieldValue := reflect.ValueOf(req).Elem().FieldByName(e.Field()).Interface()
				errMsg += fmt.Sprintf(
					"字段 %s（值：%v）不符合规则",
					e.Field(),  // 字段名
					fieldValue, // 字段实际值（原始参数）
				)
			}
		}

		// 3. 打印原始参数（结构体中已解析的值）
		glog.Errorf(c.Request.Context(), "参数校验失败: %s,原始参数: %+v", errMsg, req)

		// 4. 返回错误响应
		c.JSON(http.StatusOK, BaseResponse{Code: errs.ErrInvalidParam.Code, Message: errMsg})
		return false
	}
	return true
}
