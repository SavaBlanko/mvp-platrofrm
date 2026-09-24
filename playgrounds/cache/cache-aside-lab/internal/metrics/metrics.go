package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var CacheRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "cache_requests_total",
		Help: "Total number of cache lookups.",
	},
	[]string{"cache", "result"},
)

var CacheOperationDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "cache_operation_duration_seconds",
		Help:    "Duration of cache operations.",
		Buckets: prometheus.ExponentialBuckets(0.0001, 2, 12),
	},
	[]string{"cache", "operation"},
)

var DBQueryDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "db_query_duration_seconds",
		Help:    "Duration of database queries.",
		Buckets: prometheus.ExponentialBuckets(0.0001, 2, 15),
	},
	[]string{"operation"},
)

var HTTPRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"method", "route", "cache"},
)

func Register() {
	prometheus.MustRegister(
		CacheRequests,
		CacheOperationDuration,
		DBQueryDuration,
		HTTPRequestDuration,
	)
}