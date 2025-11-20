package metric

import (
	"strconv"
	"time"

	"github.com/SUPERDBFMP/go-base/prometheus"

	"github.com/gin-gonic/gin"
)

// PrometheusMiddleware Gin中间件:收集HTTP指标
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 记录开始时间和请求信息
		start := time.Now()
		path := c.FullPath() // 获取路由路径（如"/api/fields/:id"）
		method := c.Request.Method

		if path == "" {
			//这个是实际的路由路径，如"/api/cust/123",但是不能加，因为会被攻击，搞摊机器
			//path = c.Request.URL.Path
			path = "/404"
		}

		// 2. 增加活跃请求数
		prometheus.HttpRequestProcessing.WithLabelValues(path, method).Inc()
		defer prometheus.HttpRequestProcessing.WithLabelValues(path, method).Dec() // 延迟减少

		// 3. 处理请求
		c.Next()

		// 4. 计算延迟并记录
		duration := time.Since(start).Seconds()
		prometheus.HttpRequestDuration.WithLabelValues(path, method).Observe(duration)

		// 5. 记录请求结果（状态码）
		status := strconv.Itoa(c.Writer.Status())
		prometheus.HttpRequestsTotal.WithLabelValues(path, method, status).Inc()
	}
}
