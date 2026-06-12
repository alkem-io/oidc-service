package server

import (
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/challenge"
	middlewarepkg "github.com/alkem-io/oidc-service/internal/middleware"
)

// ConsentHandler processes consent challenges and delegates to the challenge service.
type ConsentHandler struct {
	logger   *zap.Logger
	service  challenge.Service
	paramKey string
}

// NewConsentHandler constructs a consent handler with the provided dependencies.
func NewConsentHandler(logger *zap.Logger, service challenge.Service) *ConsentHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	if service == nil {
		service = challenge.NewStubService()
	}

	return &ConsentHandler{
		logger:   logger,
		service:  service,
		paramKey: "consent_challenge",
	}
}

// Handle resolves the consent challenge and redirects according to Hydra response.
func (h *ConsentHandler) Handle(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	challengeID := strings.TrimSpace(r.URL.Query().Get(h.paramKey))
	if challengeID == "" {
		err := challenge.NewError(http.StatusBadRequest, "missing_challenge", "consent challenge is required", "", nil)
		middlewarepkg.WriteChallengeError(w, r, err)
		return
	}

	resolution, err := h.service.ResolveConsent(r.Context(), challengeID)
	if err != nil {
		middlewarepkg.WriteChallengeError(w, r, err)
		return
	}

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

	safeRedirect(w, r, redirectTo)
}
