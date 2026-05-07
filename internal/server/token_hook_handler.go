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

// TokenHookHandler implements Hydra's `oauth2.token_hook`. Hydra v2 calls this
// on EVERY token issuance (auth-code AND refresh) and **REPLACES** the session
// payload with whatever this handler returns. The handler MUST therefore echo
// the input `session.access_token` and `session.id_token` claims through
// verbatim — emitting empty maps would wipe the consent-time claim payload
// set by `resolveConsent` → `SetAccessToken` / `SetIdToken`.
//
// On top of the echo the handler injects `alkemio_actor_id` into both tokens
// when the resolver returns a non-empty value. This covers the FR-006 / FR-006a
// re-resolution path: when the refresh resolver finds the claim (originally
// or after a single re-stamp), we ship it; when the resolver is the
// deterministic stub or scope omits `alkemio`, the resolver returns "" + nil
// and we still echo the existing session unchanged.
//
//   - 200 with the echoed session (+ alkemio_actor_id when resolved)
//   - 403 with `{ "error": "temporarily_unavailable" }` on
//     `ErrRefreshTemporarilyUnavailable` — Hydra MUST surface this to the RP
//     and MUST NOT rotate the refresh-token family (FR-006a).
//
// Stage-1 exit log finding L: the prior implementation decoded the request at
// the wrong level (`subject`/`client_id`/`granted_scopes` at top level instead
// of under `session.*` / `requester.*`) AND returned only
// `{session:{id_token:{alkemio_actor_id?}}}` with no access-token echo, so
// Hydra REPLACED the consent-time session with an empty one and the issued
// JWTs shipped without `alkemio_actor_id`/`email`/`display_name`/etc.
type TokenHookHandler struct {
	logger   *zap.Logger
	audit    *audit.Emitter
	resolver RefreshActorIDResolver
}

// NewTokenHookHandler constructs the handler. A nil resolver wires
// `challenge.NewRefreshResolver()` (deterministic stub) so the route is
// always serviceable; production must pass a resolver wired with real
// Kratos+alkemio readers (see Stage-1 exit log finding L follow-up).
func NewTokenHookHandler(logger *zap.Logger, emitter *audit.Emitter, resolver RefreshActorIDResolver) *TokenHookHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	if resolver == nil {
		resolver = challenge.NewRefreshResolver()
	}
	return &TokenHookHandler{logger: logger, audit: emitter, resolver: resolver}
}

// hookRequest mirrors the Hydra v2 `oauth2.token_hook` payload. Hydra emits
// the existing session under `session.*` and the request metadata under
// `requester.*`. Only the fields the handler needs are decoded; extras are
// ignored.
type hookRequest struct {
	Session   hookInputSession `json:"session"`
	Requester hookRequester    `json:"requester"`
}

type hookInputSession struct {
	AccessToken map[string]any `json:"access_token,omitempty"`
	IDToken     map[string]any `json:"id_token,omitempty"`
	Subject     string         `json:"subject"`
}

type hookRequester struct {
	ClientID        string   `json:"client_id"`
	GrantedScopes   []string `json:"granted_scopes"`
	GrantedAudience []string `json:"granted_audience"`
	GrantTypes      []string `json:"grant_types"`
}

// hookResponse is the Hydra token-hook success body. `omitempty` on the
// claim maps means an empty echo (no consent-time claims, no resolver
// injection) serializes as `{session:{}}`, which Hydra interprets as
// "no change to session" — strictly safer than emitting `{session:{id_token:{}}}`.
type hookResponse struct {
	Session hookSession `json:"session"`
}

type hookSession struct {
	AccessToken map[string]any `json:"access_token,omitempty"`
	IDToken     map[string]any `json:"id_token,omitempty"`
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

	// Default response: echo the input session unchanged. Hydra REPLACES
	// session from hook response — emitting empty maps without the echo
	// wipes consent-time claims (Stage-1 finding L).
	accessTokenClaims := cloneClaims(req.Session.AccessToken)
	idTokenClaims := cloneClaims(req.Session.IDToken)

	actorID, err := h.resolver.Resolve(r.Context(), req.Session.Subject, req.Requester.GrantedScopes)
	if err != nil {
		if errors.Is(err, challenge.ErrRefreshTemporarilyUnavailable) {
			h.emit(r.Context(), audit.Event{
				EventType: "refresh.missing_alkemio_actor_id",
				Outcome:   audit.OutcomeFailure,
				Sub:       req.Session.Subject,
				ClientID:  req.Requester.ClientID,
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

	// Inject alkemio_actor_id into both tokens only when the resolver
	// returned a value. Resolver returns "" + nil for scope without
	// "alkemio" or when the deterministic stub doesn't have a magic
	// identity id; in those cases echo the existing claims as-is.
	if actorID != "" {
		accessTokenClaims["alkemio_actor_id"] = actorID
		idTokenClaims["alkemio_actor_id"] = actorID
	}

	resp := hookResponse{
		Session: hookSession{
			AccessToken: accessTokenClaims,
			IDToken:     idTokenClaims,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Warn("encode token-hook response failed", zap.Error(err))
	}
}

// cloneClaims deep-copies the top-level keys of a claims map. Returns an
// empty (but non-nil) map when input is nil so downstream injection can
// `claims["k"] = v` safely without nil-map panic.
func cloneClaims(in map[string]any) map[string]any {
	out := make(map[string]any, len(in)+1)
	for k, v := range in {
		out[k] = v
	}
	return out
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
