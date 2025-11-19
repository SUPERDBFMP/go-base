package prometheus

import "github.com/prometheus/client_golang/prometheus"

func init() {
	_ = prometheus.Register(HttpRequestsTotal)
	_ = prometheus.Register(HttpRequestDuration)
	_ = prometheus.Register(HttpRequestProcessing)
}
