package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/alkem-io/oidc-service/internal/challenge"
	middlewarepkg "github.com/alkem-io/oidc-service/internal/middleware"
	"github.com/alkem-io/oidc-service/pkg/telemetry"
	"go.uber.org/zap"
)

// LoginHandler processes incoming login challenges and delegates to the challenge service.
type LoginHandler struct {
	logger   *zap.Logger
	service  challenge.Service
	metrics  telemetry.ChallengeRecorder
	paramKey string
}

// NewLoginHandler constructs a login handler with the provided dependencies.
func NewLoginHandler(logger *zap.Logger, service challenge.Service, metrics telemetry.ChallengeRecorder) *LoginHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	if service == nil {
		service = challenge.NewStubService()
	}

	return &LoginHandler{
		logger:   logger,
		service:  service,
		metrics:  metrics,
		paramKey: "login_challenge",
	}
}

// Handle parses the login challenge, invokes the orchestrator, and redirects on success.
func (h *LoginHandler) Handle(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	challengeID := strings.TrimSpace(r.URL.Query().Get(h.paramKey))
	if challengeID == "" {
		err := challenge.NewError(http.StatusBadRequest, "missing_challenge", "login challenge is required", "", nil)
		h.observe(started, err)
		middlewarepkg.WriteChallengeError(w, r, err)
		return
	}

	resolution, err := h.service.ResolveLogin(r.Context(), challengeID)
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
