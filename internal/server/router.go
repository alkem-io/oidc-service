package server

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/audit"
	"github.com/alkem-io/oidc-service/internal/challenge"
	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/maintenance"
	middlewarepkg "github.com/alkem-io/oidc-service/internal/middleware"
)

// WebhookHandler defines the interface for Kratos webhook handling.
type WebhookHandler interface {
	// PostLogin handles POST /webhooks/kratos/post-login requests.
	PostLogin(w http.ResponseWriter, r *http.Request)
}

// Options bundles router dependencies.
type Options struct {
	Logger              *zap.Logger
	Maintenance         *maintenance.State
	Challenge           challenge.Service
	SessionResolver     SessionIdentityResolver
	SessionCookie       string
	KratosBrowserURL    string
	LoginReturnBaseURL  string
	WebhookHandler      WebhookHandler
	Audit               *audit.Emitter
	EndSessionConfig    *EndSessionConfig
	RefreshResolver     RefreshActorIDResolver
	PostLogoutAllowList []string
}

// NewRouter wires core middleware and health endpoints.
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
	r := chi.NewRouter()
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Recoverer)
	r.Use(middlewarepkg.RequestContext(opts.Logger))
	r.Use(
		middlewarepkg.Maintenance(
			middlewarepkg.MaintenanceOptions{
				State: opts.Maintenance,
				Skip: func(r *http.Request) bool {
					return r.URL.Path == "/health/live" || r.URL.Path == "/health/ready"
				},
			},
		),
	)

	r.Get(
		"/health/live", func(w http.ResponseWriter, _ *http.Request) {
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
			Logger:           opts.Logger,
			Challenge:        opts.Challenge,
			SessionResolver:  opts.SessionResolver,
			SessionCookie:    opts.SessionCookie,
			KratosBrowserURL: opts.KratosBrowserURL,
			ReturnBaseURL:    opts.LoginReturnBaseURL,
		},
	)
	for _, path := range []string{"/v1/oidc/login", "/oidc/login"} {
		route := path
		r.Get(
			route, func(w http.ResponseWriter, r *http.Request) {
				loginHandler.Handle(w, r)
			},
		)
	}

	consentHandler := NewConsentHandler(opts.Logger, opts.Challenge)
	for _, path := range []string{"/v1/oidc/consent", "/oidc/consent"} {
		route := path
		r.Get(
			route, func(w http.ResponseWriter, r *http.Request) {
				consentHandler.Handle(w, r)
			},
		)
	}

	logoutHandler := NewLogoutHandler(opts.Logger, opts.Challenge)
	for _, path := range []string{"/v1/oidc/logout", "/oidc/logout"} {
		route := path
		r.Get(
			route, func(w http.ResponseWriter, r *http.Request) {
				logoutHandler.Handle(w, r)
			},
		)
	}

	if opts.WebhookHandler != nil {
		r.Post("/webhooks/kratos/post-login", opts.WebhookHandler.PostLogin)
	}

	endSessionCfg := EndSessionConfig{Logger: opts.Logger, Audit: opts.Audit}
	if opts.EndSessionConfig != nil {
		endSessionCfg = *opts.EndSessionConfig
		if endSessionCfg.Logger == nil {
			endSessionCfg.Logger = opts.Logger
		}
		if endSessionCfg.Audit == nil {
			endSessionCfg.Audit = opts.Audit
		}
	}
	if len(endSessionCfg.PostLogoutAllowList) == 0 && len(opts.PostLogoutAllowList) > 0 {
		endSessionCfg.PostLogoutAllowList = append([]string(nil), opts.PostLogoutAllowList...)
	}
	endSessionHandler := NewEndSessionHandlerWithConfig(endSessionCfg)
	for _, path := range []string{"/v1/oidc/end_session", "/oidc/end_session"} {
		route := path
		r.Get(route, endSessionHandler.ServeHTTP)
	}

	tokenHookHandler := NewTokenHookHandler(opts.Logger, opts.Audit, opts.RefreshResolver)
	for _, path := range []string{"/v1/oidc/token-hook", "/oidc/token-hook"} {
		route := path
		r.Post(route, tokenHookHandler.ServeHTTP)
	}

	return r
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(payload); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

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
