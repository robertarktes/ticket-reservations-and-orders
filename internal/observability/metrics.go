package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tro_requests_total",
			Help: "Total number of requests",
		},
		[]string{"route", "code", "method"},
	)

	DBTxDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "tro_db_tx_seconds",
			Help:    "Duration of DB transactions",
			Buckets: prometheus.DefBuckets,
		},
	)

	OutboxLag = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tro_outbox_lag_seconds",
			Help: "Lag of outbox publishing",
		},
	)

	RabbitPublishRetries = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tro_rabbit_publish_retries_total",
			Help: "Total rabbit publish retries",
		},
	)

	RateLimitExceeded = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tro_rate_limit_exceeded_total",
			Help: "Total rate limit exceeded",
		},
	)
)

func InitMetrics() {
	prometheus.MustRegister(RequestsTotal, DBTxDuration, OutboxLag, RabbitPublishRetries, RateLimitExceeded)
}
