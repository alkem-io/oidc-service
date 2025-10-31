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

// TokenClaimsRecorder observes token claim generation metrics.
type TokenClaimsRecorder interface {
	ObserveTokenClaims(tokenType string, claimsCount int, hasEnhancedClaims bool)
}

// MetricsProvider exposes Prometheus collectors and handlers.
type MetricsProvider interface {
	ChallengeRecorder
	TokenClaimsRecorder
	Handler() http.Handler
}

// Metrics registers and publishes Prometheus collectors for the service.
type Metrics struct {
	registry             *prometheus.Registry
	challengeLatency     *prometheus.HistogramVec
	challengeTotal       *prometheus.CounterVec
	tokenClaimsTotal     *prometheus.CounterVec
	tokenClaimsGenerated *prometheus.HistogramVec
}

// NewMetrics constructs a metrics provider backed by the supplied registry.
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
		tokenClaimsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "oidc",
			Subsystem: "token",
			Name:      "claims_total",
			Help:      "Total number of tokens generated with enhanced claims.",
		}, []string{"token_type", "has_enhanced_claims"}),
		tokenClaimsGenerated: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "oidc",
			Subsystem: "token",
			Name:      "claims_count",
			Help:      "Number of enhanced claims added to tokens.",
			Buckets:   []float64{0, 1, 2, 3, 4, 5, 10},
		}, []string{"token_type"}),
	}

	reg.MustRegister(m.challengeLatency, m.challengeTotal, m.tokenClaimsTotal, m.tokenClaimsGenerated)

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

// ObserveTokenClaims records token claim generation metrics.
func (m *Metrics) ObserveTokenClaims(tokenType string, claimsCount int, hasEnhancedClaims bool) {
	if m == nil {
		return
	}

	if tokenType == "" {
		tokenType = "unknown"
	}

	enhancedClaimsStr := "false"
	if hasEnhancedClaims {
		enhancedClaimsStr = "true"
	}

	m.tokenClaimsTotal.WithLabelValues(tokenType, enhancedClaimsStr).Inc()
	m.tokenClaimsGenerated.WithLabelValues(tokenType).Observe(float64(claimsCount))
}
