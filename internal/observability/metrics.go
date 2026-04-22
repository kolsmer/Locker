package observability

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	registerMetricsOnce sync.Once

	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "locker",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total count of HTTP requests.",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "locker",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)
)

func RegisterMetrics(registry prometheus.Registerer) {
	registerMetricsOnce.Do(func() {
		registry.MustRegister(httpRequestsTotal)
		registry.MustRegister(httpRequestDurationSeconds)
	})
}
