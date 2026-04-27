package challenge

import (
	"context"
	"errors"
)

// ErrRefreshTemporarilyUnavailable signals FR-006/006a: the refresh
// re-resolution was unable to surface `alkemio_actor_id` for a session whose
// scope requires it. Callers MUST propagate this to Hydra as
// `temporarily_unavailable` and MUST NOT rotate the refresh-token family.
var ErrRefreshTemporarilyUnavailable = errors.New("refresh: alkemio_actor_id still absent after re-resolution")

// RefreshResolver re-resolves the Kratos identity at refresh-token exchange
// time. It reads `metadata_public.alkemio_actor_id` from Kratos; on claim
// absent + `alkemio` scope it POSTs /rest/internal/identity/resolve once
// against alkemio-server and re-reads. Still-absent or stamp-fail returns
// ErrRefreshTemporarilyUnavailable and the caller emits a
// `refresh.missing_alkemio_actor_id` failure audit. FR-006 / FR-006a.
type RefreshResolver struct {
	// T033 — fields TBD at impl time (Kratos admin client, alkemio-server
	// re-stamp client, Logger). Structure left open so the test surface
	// can drive the final shape.
}

// NewRefreshResolver constructs a resolver. T033 — impl pending.
func NewRefreshResolver() *RefreshResolver {
	return &RefreshResolver{}
}

// Resolve attempts to return the alkemio_actor_id for the given Kratos
// identity, calling the alkemio-server identity re-stamp endpoint once on
// cache miss. Returns ErrRefreshTemporarilyUnavailable when the claim is
// still absent after re-resolution.
func (r *RefreshResolver) Resolve(_ context.Context, _ string, _ []string) (string, error) {
	return "", ErrRefreshTemporarilyUnavailable
}
