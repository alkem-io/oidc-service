package server

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/alkem-io/oidc-service/internal/challenge"
	middlewarepkg "github.com/alkem-io/oidc-service/internal/middleware"
	"github.com/alkem-io/oidc-service/pkg/telemetry"
	"go.uber.org/zap"
)

// SessionIdentityResolver resolves a Kratos session cookie into an identity identifier.
type SessionIdentityResolver interface {
	IdentityID(ctx context.Context, sessionCookie string) (string, error)
}

// LoginHandler processes login challenges and delegates to the challenge service.
type LoginHandler struct {
	logger          *zap.Logger
	service         challenge.Service
	metrics         telemetry.ChallengeRecorder
	sessionResolver SessionIdentityResolver
	paramKey        string
	sessionCookie   string
}

// LoginHandlerConfig captures dependencies used by LoginHandler.
type LoginHandlerConfig struct {
	Logger          *zap.Logger
	Challenge       challenge.Service
	Metrics         telemetry.ChallengeRecorder
	SessionResolver SessionIdentityResolver
	SessionCookie   string
}

// NewLoginHandler constructs a LoginHandler from the supplied configuration.
func NewLoginHandler(cfg LoginHandlerConfig) *LoginHandler {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	svc := cfg.Challenge
	if svc == nil {
		svc = challenge.NewStubService()
	}

	cookie := strings.TrimSpace(cfg.SessionCookie)
	if cookie == "" {
		cookie = "ory_kratos_session"
	}

	return &LoginHandler{
		logger:          logger,
		service:         svc,
		metrics:         cfg.Metrics,
		sessionResolver: cfg.SessionResolver,
		paramKey:        "login_challenge",
		sessionCookie:   cookie,
	}
}

// Handle parses the login challenge, injects session hints, and redirects on success.
func (h *LoginHandler) Handle(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	challengeID := strings.TrimSpace(r.URL.Query().Get(h.paramKey))
	if challengeID == "" {
		err := challenge.NewError(http.StatusBadRequest, "missing_challenge", "login challenge is required", "", nil)
		h.observe(started, err)
		middlewarepkg.WriteChallengeError(w, r, err)
		return
	}

	ctx := h.attachIdentityHint(r.Context(), r)

	resolution, err := h.service.ResolveLogin(ctx, challengeID)
	if err != nil {
		h.observe(started, err)
		middlewarepkg.WriteChallengeError(w, r, err)
		return
	}

	h.observe(started, nil)

	redirectTo := resolution.RedirectURL
	logger := middlewarepkg.Logger(r.Context())
	if logger == nil {
		logger = h.logger
	}

	logger.Info("login challenge resolved",
		zap.String("challengeId", challengeID),
		zap.String("redirectTo", redactRedirectURL(redirectTo)),
		zap.Duration("duration", time.Since(started)),
	)

	http.Redirect(w, r, redirectTo, http.StatusFound)
}

func (h *LoginHandler) attachIdentityHint(ctx context.Context, r *http.Request) context.Context {
	if h.sessionResolver == nil {
		return ctx
	}

	provider := h.buildHintProvider(r)
	if provider == nil {
		return ctx
	}

	return challenge.WithIdentityHintProvider(ctx, provider)
}

func (h *LoginHandler) buildHintProvider(r *http.Request) challenge.IdentityHintProvider {
	if h.sessionResolver == nil {
		return nil
	}

	cookie, err := r.Cookie(h.sessionCookie)
	if err != nil {
		return challenge.IdentityHintFunc(func(context.Context) (string, error) {
			return "", challenge.ErrIdentitySessionRequired
		})
	}

	sessionToken := strings.TrimSpace(cookie.Value)
	if sessionToken == "" {
		return challenge.IdentityHintFunc(func(context.Context) (string, error) {
			return "", challenge.ErrIdentitySessionInvalid
		})
	}

	return challenge.IdentityHintFunc(func(ctx context.Context) (string, error) {
		return h.sessionResolver.IdentityID(ctx, sessionToken)
	})
}

func (h *LoginHandler) observe(started time.Time, err error) {
	if h.metrics == nil {
		return
	}

	outcome := "success"
	errorCode := "none"
	if err != nil {
		outcome, errorCode = classifyOutcome(err)
	}

	h.metrics.ObserveChallenge("login", outcome, errorCode, time.Since(started))
}

func classifyOutcome(err error) (string, string) {
	var chErr challenge.Error
	if errors.As(err, &chErr) {
		code := chErr.Code()
		switch status := chErr.StatusCode(); {
		case status == http.StatusServiceUnavailable:
			return "maintenance", code
		case status >= 400 && status < 500:
			return "client_error", code
		default:
			return "server_error", code
		}
	}
	return "server_error", "unexpected_error"
}
