package server

import (
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/challenge"
	middlewarepkg "github.com/alkem-io/oidc-service/internal/middleware"
)

// LogoutHandler processes Hydra logout challenges (the UI Hydra invokes via
// urls.logout). It auto-accepts the logout — the BFF or end_session entry has
// already authorised user intent — and redirects to Hydra's returned URL.
type LogoutHandler struct {
	logger   *zap.Logger
	service  challenge.Service
	paramKey string
}

// NewLogoutHandler constructs a logout handler with the provided dependencies.
func NewLogoutHandler(logger *zap.Logger, service challenge.Service) *LogoutHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	if service == nil {
		service = challenge.NewStubService()
	}
	return &LogoutHandler{
		logger:   logger,
		service:  service,
		paramKey: "logout_challenge",
	}
}

// Handle resolves the logout challenge and 302s to the redirect Hydra returned.
func (h *LogoutHandler) Handle(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	challengeID := strings.TrimSpace(r.URL.Query().Get(h.paramKey))
	if challengeID == "" {
		err := challenge.NewError(http.StatusBadRequest, "missing_challenge", "logout challenge is required", "", nil)
		middlewarepkg.WriteChallengeError(w, r, err)
		return
	}

	resolution, err := h.service.ResolveLogout(r.Context(), challengeID)
	if err != nil {
		middlewarepkg.WriteChallengeError(w, r, err)
		return
	}

	logger := middlewarepkg.Logger(r.Context())
	if logger == nil {
		logger = h.logger
	}

	logger.Info("logout challenge resolved",
		zap.String("challengeId", challengeID),
		zap.String("redirectTo", redactRedirectURL(resolution.RedirectURL)),
		zap.Duration("duration", time.Since(started)),
	)

	safeRedirect(w, r, resolution.RedirectURL)
}
