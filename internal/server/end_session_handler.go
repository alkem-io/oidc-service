package server

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/audit"
)

// IDTokenValidator validates an id_token_hint and extracts the subject and
// client_id required for RP-initiated logout per FR-010. Production wires a
// JWKS-backed validator that performs full Hydra signature/iss/exp checks;
// the contract test surface relies on the shape-only default.
type IDTokenValidator interface {
	Validate(ctx context.Context, idToken string) (subject string, clientID string, err error)
}

// HydraConsentRevoker revokes the calling RP's consent session in Hydra
// (DELETE /admin/oauth2/auth/sessions/consent?client=<id>&subject=<sub>).
// Tests pass nil to skip the call entirely; the response path is unchanged.
type HydraConsentRevoker interface {
	RevokeConsent(ctx context.Context, clientID, subject string) error
}

// EndSessionConfig wires production dependencies for the RP-initiated logout
// endpoint. Zero-value config plus a non-nil emitter is sufficient for the
// contract tests; the default validator + allow-list make the local-dev URL
// the only acceptable post-logout target.
type EndSessionConfig struct {
	Logger              *zap.Logger
	Audit               *audit.Emitter
	IDTokenValidator    IDTokenValidator
	PostLogoutAllowList []string
	Revoker             HydraConsentRevoker
}

// EndSessionHandler implements the RP-initiated logout endpoint (FR-010 / T031).
// It validates the id_token_hint, validates post_logout_redirect_uri against
// the per-RP allow-list, calls Hydra admin to revoke ONLY the calling RP's
// consent session (Kratos SSO is preserved), emits a `session.end_session`
// audit, and 302s to the post-logout URI.
type EndSessionHandler struct {
	logger    *zap.Logger
	audit     *audit.Emitter
	validator IDTokenValidator
	allowList []string
	revoker   HydraConsentRevoker
}

const defaultPostLogoutLocalDevURL = "http://localhost:3000/logout"

// NewEndSessionHandler constructs the handler with audit emitter and the
// shape-only id_token_hint validator suited to the contract test surface.
func NewEndSessionHandler(emitter *audit.Emitter) *EndSessionHandler {
	return NewEndSessionHandlerWithConfig(EndSessionConfig{Audit: emitter})
}

// NewEndSessionHandlerWithConfig constructs the handler from cfg. Production
// wires a JWKS-aware IDTokenValidator, the per-environment allow-list, and a
// Hydra HydraConsentRevoker; tests typically pass only Audit.
func NewEndSessionHandlerWithConfig(cfg EndSessionConfig) *EndSessionHandler {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	validator := cfg.IDTokenValidator
	if validator == nil {
		validator = shapeOnlyIDTokenValidator{}
	}
	allow := cfg.PostLogoutAllowList
	if len(allow) == 0 {
		allow = []string{defaultPostLogoutLocalDevURL}
	}
	return &EndSessionHandler{
		logger:    logger,
		audit:     cfg.Audit,
		validator: validator,
		allowList: append([]string(nil), allow...),
		revoker:   cfg.Revoker,
	}
}

// ServeHTTP implements http.Handler.
func (h *EndSessionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	hint := strings.TrimSpace(query.Get("id_token_hint"))
	postLogout := strings.TrimSpace(query.Get("post_logout_redirect_uri"))

	if hint == "" {
		h.fail(w, r, "", "", http.StatusBadRequest, "missing_id_token_hint")
		return
	}

	subject, clientID, err := h.validator.Validate(r.Context(), hint)
	if err != nil {
		h.fail(w, r, "", "", http.StatusBadRequest, "invalid_id_token_hint")
		return
	}

	if !h.allowedRedirect(postLogout) {
		h.fail(w, r, subject, clientID, http.StatusBadRequest, "unregistered_post_logout_redirect")
		return
	}

	if h.revoker != nil && clientID != "" && subject != "" {
		if err := h.revoker.RevokeConsent(r.Context(), clientID, subject); err != nil {
			h.logger.Warn(
				"hydra revoke consent failed; continuing logout",
				zap.String("client_id", clientID),
				zap.Error(err),
			)
		}
	}

	h.emit(
		r.Context(), audit.Event{
			EventType: "session.end_session",
			Outcome:   audit.OutcomeSuccess,
			Sub:       subject,
			ClientID:  clientID,
		},
	)

	http.Redirect(w, r, postLogout, http.StatusFound)
}

func (h *EndSessionHandler) allowedRedirect(uri string) bool {
	if uri == "" {
		return false
	}
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	return slices.Contains(h.allowList, uri)
}

func (h *EndSessionHandler) fail(
	w http.ResponseWriter, r *http.Request, sub, clientID string, status int, errorCode string,
) {
	h.emit(
		r.Context(), audit.Event{
			EventType: "session.end_session",
			Outcome:   audit.OutcomeFailure,
			Sub:       sub,
			ClientID:  clientID,
			ErrorCode: errorCode,
		},
	)
	http.Error(w, errorCode, status)
}

func (h *EndSessionHandler) emit(ctx context.Context, ev audit.Event) {
	if h.audit == nil {
		return
	}
	if ev.CorrelationID == "" {
		if id := CorrelationID(ctx); id != "" {
			ev.CorrelationID = id
		}
	}
	if err := h.audit.Emit(ev); err != nil {
		h.logger.Warn("emit end_session audit failed", zap.Error(err))
	}
}

// shapeOnlyIDTokenValidator accepts any value with three non-empty
// dot-separated segments. It is the default validator used by the contract
// test surface; production wires a JWKS-aware validator that extracts the
// sub + aud/azp claims after signature verification.
type shapeOnlyIDTokenValidator struct{}

// Validate implements IDTokenValidator with a shape-only check.
func (shapeOnlyIDTokenValidator) Validate(_ context.Context, token string) (string, string, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return "", "", errors.New("malformed id_token_hint")
	}
	for _, segment := range parts {
		if strings.TrimSpace(segment) == "" {
			return "", "", errors.New("malformed id_token_hint segment")
		}
	}
	return "", "", nil
}
