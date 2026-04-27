package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/audit"
	"github.com/alkem-io/oidc-service/internal/challenge"
)

// RefreshActorIDResolver re-resolves `alkemio_actor_id` for a given Kratos
// identity at refresh-token exchange time. Production wires
// challenge.RefreshResolver; tests substitute a deterministic fake.
type RefreshActorIDResolver interface {
	Resolve(ctx context.Context, identityID string, grantedScope []string) (string, error)
}

// TokenHookHandler implements Hydra's `oauth2.token_hook` (FR-006 / FR-006a).
// On refresh-token exchange Hydra POSTs the request payload here; the handler
// re-resolves `alkemio_actor_id` and either:
//
//   - 200 with `{ "session": { "id_token": { "alkemio_actor_id": "..." } } }`
//     when the claim resolves (or scope omits `alkemio`)
//   - 403 with `{ "error": "temporarily_unavailable" }` when re-resolution
//     fails — Hydra MUST surface this to the RP and MUST NOT rotate the
//     refresh-token family
type TokenHookHandler struct {
	logger   *zap.Logger
	audit    *audit.Emitter
	resolver RefreshActorIDResolver
}

// NewTokenHookHandler constructs the handler. A nil resolver wires
// challenge.NewRefreshResolver() (deterministic stub) so the route is
// always serviceable.
func NewTokenHookHandler(logger *zap.Logger, emitter *audit.Emitter, resolver RefreshActorIDResolver) *TokenHookHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	if resolver == nil {
		resolver = challenge.NewRefreshResolver()
	}
	return &TokenHookHandler{logger: logger, audit: emitter, resolver: resolver}
}

// hookRequest mirrors the documented Hydra token-hook payload. Only the
// fields the resolver needs are decoded; any extra fields are ignored.
type hookRequest struct {
	Subject       string   `json:"subject"`
	ClientID      string   `json:"client_id"`
	GrantedScopes []string `json:"granted_scopes"`
}

// hookResponse is the documented Hydra token-hook success body. Only the
// id_token branch is populated — access-token claims pass through.
type hookResponse struct {
	Session hookSession `json:"session"`
}

type hookSession struct {
	IDToken map[string]any `json:"id_token,omitempty"`
}

// ServeHTTP implements http.Handler.
func (h *TokenHookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req hookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	actorID, err := h.resolver.Resolve(r.Context(), req.Subject, req.GrantedScopes)
	if err != nil {
		if errors.Is(err, challenge.ErrRefreshTemporarilyUnavailable) {
			h.emit(r.Context(), audit.Event{
				EventType: "refresh.missing_alkemio_actor_id",
				Outcome:   audit.OutcomeFailure,
				Sub:       req.Subject,
				ClientID:  req.ClientID,
				ErrorCode: "temporarily_unavailable",
			})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "temporarily_unavailable"})
			return
		}
		h.logger.Warn("refresh resolver returned unexpected error", zap.Error(err))
		http.Error(w, "internal_error", http.StatusInternalServerError)
		return
	}

	resp := hookResponse{Session: hookSession{IDToken: map[string]any{}}}
	if actorID != "" {
		resp.Session.IDToken["alkemio_actor_id"] = actorID
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Warn("encode token-hook response failed", zap.Error(err))
	}
}

func (h *TokenHookHandler) emit(ctx context.Context, ev audit.Event) {
	if h.audit == nil {
		return
	}
	if ev.CorrelationID == "" {
		if id := CorrelationID(ctx); id != "" {
			ev.CorrelationID = id
		}
	}
	if err := h.audit.Emit(ev); err != nil {
		h.logger.Warn("emit token-hook audit failed", zap.Error(err))
	}
}
