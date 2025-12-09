package middleware

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/SUPERDBFMP/go-base/config"
	"github.com/SUPERDBFMP/go-base/glog"
	"github.com/SUPERDBFMP/go-base/trace"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// 自定义 ResponseWriter 来捕获响应数据
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w bodyLogWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	n, err := w.ResponseWriter.WriteString(s)
	return n, err
}

// LoggerMiddleware 自定义日志中间件
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录请求开始时间
		start := time.Now()

		// 包装traceId
		newCtx := context.WithValue(c.Request.Context(), trace.TraceIdKey, trace.GenerateTraceId())
		c.Request = c.Request.WithContext(newCtx)
		method := c.Request.Method // 请求方法（GET/POST等）
		path := c.Request.URL.Path // 请求路径
		ip := c.ClientIP()         // 客户端 IP
		contentType := c.Request.Header.Get("Content-Type")
		var reqBodyStr string
		reqFields := logrus.Fields{
			"path":   path,
			"method": method,
			"ip":     ip,
		}
		if strings.Contains(contentType, "json") ||
			strings.Contains(contentType, "x-www-form-urlencoded") ||
			strings.Contains(contentType, "form-data") &&
				c.Request.ContentLength < 1*1024*1024 { // 1MB
			//copy := c.Copy()
			//// 读取复制的请求Body
			//body, err := io.ReadAll(copy.Request.Body)
			//if err != nil {
			//	reqBodyStr = "读取Body失败"
			//	glog.Errorf(newCtx, "读取Body失败: %v", err)
			//} else {
			//	reqBodyStr = string(body)
			//}
			//copy.Request.Body.Close()

			body, err := c.GetRawData()
			if err != nil {
				reqBodyStr = "读取Body失败"
				glog.Errorf(newCtx, "读取Body失败: %v", err)
			}
			// 重新设置请求体供后续使用
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
			reqBodyStr = string(body)
		} else {
			//尝试获取query
			reqBodyStr = c.Request.URL.Query().Encode()
			if reqBodyStr != "" {
				reqBodyStr = fmt.Sprintf("[query:%s body:二进制内容，大小: %d 字节]", reqBodyStr, c.Request.ContentLength)
			} else {
				reqBodyStr = fmt.Sprintf("[二进制内容，大小: %d 字节]", c.Request.ContentLength)
			}

		}

		glog.InfofWithFields(newCtx, reqFields, "req:%s", reqBodyStr)
		// 创建自定义 ResponseWriter 来捕获响应
		blw := &bodyLogWriter{
			body:           bytes.NewBufferString(""),
			ResponseWriter: c.Writer,
		}
		c.Writer = blw

		// 继续执行后续中间件或处理函数
		c.Next()

		// 请求处理完成后,记录日志（此时可获取状态码、耗时等）
		status := c.Writer.Status()   // 响应状态码
		duration := time.Since(start) // 耗时
		responseBody := blw.body.Bytes()
		fields := logrus.Fields{
			"path":   path,
			"method": method,
			"ip":     ip,
			"status": status,
			"cost":   duration.Milliseconds(),
		}

		var responseBodyStr string

		// 检测并处理压缩响应
		//contentEncoding := c.Writer.Header().Get("Content-Encoding")
		contentType = c.Writer.Header().Get("Content-Type")

		if strings.Contains(contentType, "json") ||
			strings.Contains(contentType, "x-www-form-urlencoded") ||
			strings.Contains(contentType, "form-data") {
			// 如果是文本或JSON内容，直接转换为字符串
			responseBodyStr = string(responseBody)
			//} else if contentEncoding == "gzip" && len(responseBody) > 0 {
			//	// 如果是GZIP压缩内容，尝试解压
			//	reader, errs := gzip.NewReader(bytes.NewReader(responseBody))
			//	if errs == nil {
			//		defer reader.Close()
			//		decompressed, errs := io.ReadAll(reader)
			//		if errs == nil {
			//			responseBodyStr = string(decompressed)
			//		} else {
			//			responseBodyStr = "[GZIP压缩内容，解压失败]"
			//		}
			//	} else {
			//		responseBodyStr = "[GZIP压缩内容，创建读取器失败]"
			//	}
		} else {
			// 二进制内容（如图片、文件等）
			responseBodyStr = fmt.Sprintf("[二进制内容，大小: %d 字节]", len(responseBody))
		}

		// 打印日志（生产环境建议使用结构化日志库如 zap/logrus）
		if status >= http.StatusNotFound {
			glog.ErrorWithFields(c.Request.Context(), fields, "请求失败")
		} else {
			if path != config.GlobalConf.WebServer.ContextPath+"/health" {
				if len(responseBodyStr) < 1*1024*1024 {
					glog.InfofWithFields(c.Request.Context(), fields, "resp:%s", responseBodyStr)
				} else {
					glog.InfofWithFields(c.Request.Context(), fields, "请求成功")
				}

			}
		}
	}
}
