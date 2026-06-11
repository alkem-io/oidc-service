package contract_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alkem-io/oidc-service/internal/challenge"
)

// TestRefreshReresolveHappyPath — FR-006 (T033). When Kratos already carries
// `metadata_public.alkemio_actor_id`, Resolve returns it without calling the
// alkemio-server re-stamp endpoint. Until T033 lands the resolver stub
// returns ErrRefreshTemporarilyUnavailable — test is RED.
func TestRefreshReresolveHappyPath(t *testing.T) {
	t.Parallel()

	resolver := challenge.NewRefreshResolver()
	actorID, err := resolver.Resolve(context.Background(), "kratos-happy", []string{"openid", "alkemio"})
	require.NoError(t, err)
	require.NotEmpty(t, actorID, "resolver must return alkemio_actor_id on happy path")
}

// TestRefreshReresolveReStampsOnClaimMiss — FR-006a (T033). When Kratos has
// no claim yet, the resolver MUST POST to alkemio-server's identity resolve
// endpoint ONCE and re-read the claim. On success the returned actor id MUST
// be non-empty.
func TestRefreshReresolveReStampsOnClaimMiss(t *testing.T) {
	t.Parallel()

	resolver := challenge.NewRefreshResolver()
	actorID, err := resolver.Resolve(context.Background(), "kratos-miss-then-present", []string{"openid", "alkemio"})
	require.NoError(t, err)
	require.NotEmpty(t, actorID)
}

// TestRefreshReresolveStillAbsentYieldsTemporarilyUnavailable — FR-006a
// (T033). When the claim is still absent after re-stamp, the resolver MUST
// return ErrRefreshTemporarilyUnavailable so the caller surfaces
// `temporarily_unavailable` to Hydra and MUST NOT rotate the refresh-token
// family. The caller also emits `refresh.missing_alkemio_actor_id` failure
// audit — asserted by a separate test at the caller's level.
func TestRefreshReresolveStillAbsentYieldsTemporarilyUnavailable(t *testing.T) {
	t.Parallel()

	resolver := challenge.NewRefreshResolver()
	_, err := resolver.Resolve(context.Background(), "kratos-permanent-miss", []string{"openid", "alkemio"})
	require.Error(t, err)
	require.True(t,
		errors.Is(err, challenge.ErrRefreshTemporarilyUnavailable),
		"expected ErrRefreshTemporarilyUnavailable, got %v", err,
	)
}

// TestRefreshReresolveSkipsWhenScopeAbsent — FR-005 coupling. When the
// session's scope does NOT include `alkemio`, the resolver MUST NOT treat a
// missing claim as a failure: it returns "" + nil error so the refresh flow
// proceeds without the claim.
func TestRefreshReresolveSkipsWhenScopeAbsent(t *testing.T) {
	t.Parallel()

	resolver := challenge.NewRefreshResolver()
	actorID, err := resolver.Resolve(context.Background(), "kratos-no-alkemio-scope", []string{"openid", "profile"})
	require.NoError(t, err)
	require.Empty(t, actorID)
}
