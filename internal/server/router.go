package server

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/challenge"
	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/maintenance"
	middlewarepkg "github.com/alkem-io/oidc-service/internal/middleware"
	"github.com/alkem-io/oidc-service/pkg/telemetry"
)

// Options bundles router dependencies.
type Options struct {
	Logger          *zap.Logger
	Maintenance     *maintenance.State
	Challenge       challenge.Service
	Metrics         telemetry.MetricsProvider
	SessionResolver SessionIdentityResolver
	SessionCookie   string
}

// NewRouter constructs and returns an HTTP router configured with common middleware,
// maintenance handling, health endpoints, OIDC login/consent routes, and a metrics endpoint.
// 
// The router applies RealIP, recover, request-context, and maintenance middleware. If any
// of the Options dependencies (Logger, Maintenance, Challenge, Metrics) are nil, sensible
// defaults are used. It exposes:
//   - GET /health/live: returns a simple alive status.
//   - GET /health/ready: returns readiness that combines maintenance state and the
//     Challenge service readiness, and sets HTTP status and Retry-After when appropriate.
//   - GET /v1/oidc/login: delegates to the configured login handler.
//   - GET /v1/oidc/consent: delegates to the configured consent handler.
//   - GET /metrics: serves metrics from the configured metrics provider.
func NewRouter(opts Options) http.Handler {
	if opts.Logger == nil {
		opts.Logger = zap.NewNop()
	}
	if opts.Maintenance == nil {
		opts.Maintenance = maintenance.NewState(config.MaintenanceState{})
	}
	if opts.Challenge == nil {
		opts.Challenge = challenge.NewStubService()
	}
	if opts.Metrics == nil {
		opts.Metrics = telemetry.NewMetrics(telemetry.NewRegistry())
	}

	r := chi.NewRouter()
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Recoverer)
	r.Use(middlewarepkg.RequestContext(opts.Logger))
	r.Use(
		middlewarepkg.Maintenance(
			middlewarepkg.MaintenanceOptions{
				State:   opts.Maintenance,
				Metrics: opts.Metrics,
				Skip: func(r *http.Request) bool {
					return r.URL.Path == "/health/live" || r.URL.Path == "/health/ready"
				},
			},
		),
	)

	r.Get(
		"/health/live", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]any{"status": "alive"})
		},
	)

	r.Get(
		"/health/ready", func(w http.ResponseWriter, r *http.Request) {
			state := opts.Maintenance.Snapshot()
			readiness := opts.Challenge.Readiness(r.Context())

			payload := map[string]any{
				"status":      readiness.Status,
				"hydra":       readiness.Hydra,
				"kratos":      readiness.Kratos,
				"maintenance": state.Enabled,
			}
			if readiness.Version != "" {
				payload["version"] = readiness.Version
			}

			statusCode := http.StatusOK
			if readiness.Status != "ready" {
				statusCode = http.StatusServiceUnavailable
				payload["status"] = readiness.Status
			}
			if state.Enabled {
				statusCode = http.StatusServiceUnavailable
				payload["status"] = "maintenance"
				if value := retryAfterHeader(state); value != "" {
					w.Header().Set("Retry-After", value)
				}
			}

			writeJSON(w, statusCode, payload)
		},
	)

	loginHandler := NewLoginHandler(
		LoginHandlerConfig{
			Logger:          opts.Logger,
			Challenge:       opts.Challenge,
			Metrics:         opts.Metrics,
			SessionResolver: opts.SessionResolver,
			SessionCookie:   opts.SessionCookie,
		},
	)
	r.Get(
		"/v1/oidc/login", func(w http.ResponseWriter, r *http.Request) {
			loginHandler.Handle(w, r)
		},
	)

	consentHandler := NewConsentHandler(opts.Logger, opts.Challenge, opts.Metrics)
	r.Get(
		"/v1/oidc/consent", func(w http.ResponseWriter, r *http.Request) {
			consentHandler.Handle(w, r)
		},
	)

	r.Get(
		"/metrics", func(w http.ResponseWriter, r *http.Request) {
			opts.Metrics.Handler().ServeHTTP(w, r)
		},
	)

	return r
}

// writeJSON writes the given payload as JSON to w and sets the provided HTTP status code.
// It sets Content-Type to "application/json" and encodes the payload; any encoding error is ignored.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// retryAfterHeader returns the string value to use for an HTTP Retry-After header based on
// state.RetryAfter. It returns an empty string if RetryAfter is nil, "0" if the rounded-up
// seconds are less than or equal to zero, or the number of seconds (rounded up) as a decimal string.
func retryAfterHeader(state config.MaintenanceState) string {
	if state.RetryAfter == nil {
		return ""
	}

	seconds := int(math.Ceil(state.RetryAfter.Seconds()))
	if seconds <= 0 {
		return "0"
	}

	return strconv.Itoa(seconds)
}