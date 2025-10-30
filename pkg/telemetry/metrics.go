package telemetry

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ChallengeRecorder observes challenge-level metrics.
type ChallengeRecorder interface {
	ObserveChallenge(flow, outcome, errorCode string, duration time.Duration)
}

// MetricsProvider exposes Prometheus collectors and handlers.
type MetricsProvider interface {
	ChallengeRecorder
	Handler() http.Handler
}

// Metrics registers and publishes Prometheus collectors for the service.
type Metrics struct {
	registry         *prometheus.Registry
	challengeLatency *prometheus.HistogramVec
	challengeTotal   *prometheus.CounterVec
}

// NewMetrics constructs a Metrics instance backed by the provided Prometheus registry and registers
// challenge-related metrics on that registry. If reg is nil a new prometheus.Registry is created.
// It registers a histogram (labels: "flow", "outcome"; namespace: "oidc", subsystem: "challenge", name: "latency_seconds")
// and a counter (labels: "flow", "outcome", "error_code"; namespace: "oidc", subsystem: "challenge", name: "total").
func NewMetrics(reg *prometheus.Registry) *Metrics {
	if reg == nil {
		reg = prometheus.NewRegistry()
	}

	m := &Metrics{
		registry: reg,
		challengeLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "oidc",
			Subsystem: "challenge",
			Name:      "latency_seconds",
			Help:      "Latency of Hydra challenge handling.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"flow", "outcome"}),
		challengeTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "oidc",
			Subsystem: "challenge",
			Name:      "total",
			Help:      "Total number of Hydra challenges processed.",
		}, []string{"flow", "outcome", "error_code"}),
	}

	reg.MustRegister(m.challengeLatency, m.challengeTotal)

	return m
}

// Handler returns an HTTP handler exposing the registry's metrics.
func (m *Metrics) Handler() http.Handler {
	if m == nil {
		return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	}
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// ObserveChallenge records a challenge outcome and latency.
func (m *Metrics) ObserveChallenge(flow, outcome, errorCode string, duration time.Duration) {
	if m == nil {
		return
	}
	if flow == "" {
		flow = "unknown"
	}
	if outcome == "" {
		outcome = "unknown"
	}
	if errorCode == "" {
		errorCode = "none"
	}

	m.challengeLatency.WithLabelValues(flow, outcome).Observe(duration.Seconds())
	m.challengeTotal.WithLabelValues(flow, outcome, errorCode).Inc()
}