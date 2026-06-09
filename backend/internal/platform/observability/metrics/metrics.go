package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "leadcat_http_requests_total",
		Help: "Total HTTP requests",
	}, []string{"method", "path", "status"})
)

func IncHTTP(method, path string, status int) {
	HTTPRequests.WithLabelValues(method, path, statusLabel(status)).Inc()
}

func statusLabel(code int) string {
	switch {
	case code >= 500:
		return "5xx"
	case code >= 400:
		return "4xx"
	case code >= 300:
		return "3xx"
	case code >= 200:
		return "2xx"
	default:
		return "other"
	}
}
