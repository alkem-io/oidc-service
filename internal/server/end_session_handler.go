package server

import (
	"net/http"

	"github.com/alkem-io/oidc-service/internal/audit"
)

// EndSessionHandler implements the RP-initiated logout endpoint (FR-010 / T031).
// It validates the id_token_hint against Hydra's JWKS + iss + exp, derives
// client_id from the token's aud/azp, validates post_logout_redirect_uri
// against the RP's registered list, calls Hydra admin
// DELETE /admin/oauth2/auth/sessions/consent?client=<id>&subject=<sub>,
// emits a `session.end_session` audit, and 302s to the post-logout URI.
type EndSessionHandler struct {
	Audit *audit.Emitter
}

// NewEndSessionHandler constructs the handler.
func NewEndSessionHandler(emitter *audit.Emitter) *EndSessionHandler {
	return &EndSessionHandler{Audit: emitter}
}

// ServeHTTP is the handler entry point. T031 — implementation pending.
func (h *EndSessionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "end_session handler not implemented", http.StatusNotImplemented)
}
