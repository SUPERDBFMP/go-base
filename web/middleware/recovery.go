package middleware

import (
	"go-base/glog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RecoveryMiddleware 自定义 Panic 恢复中间件
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 使用 defer 捕获 panic
		defer func() {
			if err := recover(); err != nil {
				// 记录 Panic 信息（包含堆栈）
				stack := string(debug.Stack()) // 获取堆栈跟踪
				fields := logrus.Fields{
					"path":   c.Request.URL.Path,
					"method": c.Request.Method,
					"ip":     c.ClientIP(),
					"stack":  stack,
				}

				glog.ErrorWithFields(c.Request.Context(), fields, "Recovery from panic")
				// 返回友好错误响应（避免暴露敏感信息）
				c.AbortWithStatusJSON(
					http.StatusInternalServerError, gin.H{
						"code":    500,
						"message": "服务器内部错误,请稍后再试",
					},
				)
			}
		}()

		// 继续执行后续中间件或处理函数
		c.Next()
	}
}
