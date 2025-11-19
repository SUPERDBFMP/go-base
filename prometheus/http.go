package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HttpRequestsTotal 定义一个计数器，用于记录HTTP请求总数，并包含方法、路径和状态码标签
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",            // 指标名称
			Help: "Total number of HTTP requests.", // 帮助信息
		},
		[]string{"method", "path", "status"}, // 标签维度
	)
	// HttpRequestDuration 定义一个直方图，用于记录HTTP请求的持续时间（单位：秒）
	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds.",
			Buckets: prometheus.DefBuckets, // 使用默认的桶定义，你也可以自定义 []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
		},
		[]string{"method", "path"}, // 标签维度
	)
	// HttpRequestProcessing 创建一个 gauge，用于记录当前处理中的HTTP请求数量
	HttpRequestProcessing = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_request_processing",
			Help: "Count of HTTP requests in process.",
		},
		[]string{"method", "path"},
	)
)
