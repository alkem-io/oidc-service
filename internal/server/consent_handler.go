package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/alkem-io/oidc-service/internal/challenge"
	middlewarepkg "github.com/alkem-io/oidc-service/internal/middleware"
	"github.com/alkem-io/oidc-service/pkg/telemetry"
	"go.uber.org/zap"
)

// ConsentHandler processes consent challenges and delegates to the challenge service.
type ConsentHandler struct {
	logger   *zap.Logger
	service  challenge.Service
	metrics  telemetry.ChallengeRecorder
	paramKey string
}

// NewConsentHandler creates a ConsentHandler configured with the provided dependencies.
// If logger is nil a no-op logger is used, if service is nil a stub service is used; the returned handler uses "consent_challenge" as the query parameter key and accepts a nil metrics recorder.
func NewConsentHandler(logger *zap.Logger, service challenge.Service, metrics telemetry.ChallengeRecorder) *ConsentHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	if service == nil {
		service = challenge.NewStubService()
	}

	return &ConsentHandler{
		logger:   logger,
		service:  service,
		metrics:  metrics,
		paramKey: "consent_challenge",
	}
}

// Handle resolves the consent challenge and redirects according to Hydra response.
func (h *ConsentHandler) Handle(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	challengeID := strings.TrimSpace(r.URL.Query().Get(h.paramKey))
	if challengeID == "" {
		err := challenge.NewError(http.StatusBadRequest, "missing_challenge", "consent challenge is required", "", nil)
		h.observe(started, err)
		middlewarepkg.WriteChallengeError(w, r, err)
		return
	}

	resolution, err := h.service.ResolveConsent(r.Context(), challengeID)
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

	logger.Info("consent challenge resolved",
		zap.String("challengeId", challengeID),
		zap.String("redirectTo", redactRedirectURL(redirectTo)),
		zap.Duration("duration", time.Since(started)),
	)

	http.Redirect(w, r, redirectTo, http.StatusFound)
}

func (h *ConsentHandler) observe(started time.Time, err error) {
	if h.metrics == nil {
		return
	}

	outcome := "success"
	errorCode := "none"
	if err != nil {
		outcome, errorCode = classifyOutcome(err)
	}

	h.metrics.ObserveChallenge("consent", outcome, errorCode, time.Since(started))
}