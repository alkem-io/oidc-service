package contract_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/challenge"
	"github.com/alkem-io/oidc-service/internal/server"
)

// T075 — Contract tests for the Hydra logout-challenge handler. Pin the
// FR-010 plumbing landed in T070 (`internal/server/logout_handler.go` +
// router mount at `/oidc/logout` + `/v1/oidc/logout`). The handler MUST:
//   - 400 when `logout_challenge` query param is absent or empty
//   - 302 with `Location` from `service.ResolveLogout` for a valid challenge
//   - propagate ResolveLogout errors via WriteChallengeError mapping

// TestLogoutHandlerRejectsMissingChallenge — FR-010 / T070 / T075.
// Without a `logout_challenge` query parameter the handler MUST return 400
// `missing_challenge` and MUST NOT 302.
func TestLogoutHandlerRejectsMissingChallenge(t *testing.T) {
	t.Parallel()

	handler := server.NewLogoutHandler(zap.NewNop(), challenge.NewStubService())

	req := httptest.NewRequest(http.MethodGet, "/oidc/logout", nil)
	rec := httptest.NewRecorder()
	handler.Handle(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code, "missing logout_challenge must return 400")
	require.NotEqual(t, http.StatusFound, rec.Code, "missing logout_challenge MUST NOT 302")
	require.Contains(t, rec.Body.String(), "missing_challenge")
}

// TestLogoutHandlerRejectsBlankChallenge — whitespace-only challenge is
// equivalent to missing per the handler's TrimSpace check.
func TestLogoutHandlerRejectsBlankChallenge(t *testing.T) {
	t.Parallel()

	handler := server.NewLogoutHandler(zap.NewNop(), challenge.NewStubService())

	req := httptest.NewRequest(http.MethodGet, "/oidc/logout?logout_challenge=%20%20%20", nil)
	rec := httptest.NewRecorder()
	handler.Handle(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code, "blank logout_challenge must return 400")
}

// TestLogoutHandlerResolvesValidChallengeTo302 — happy path. With a stub
// service and a valid challenge id, the handler 302s to the redirect URL
// the service returned.
func TestLogoutHandlerResolvesValidChallengeTo302(t *testing.T) {
	t.Parallel()

	handler := server.NewLogoutHandler(zap.NewNop(), challenge.NewStubService())

	req := httptest.NewRequest(http.MethodGet, "/oidc/logout?logout_challenge=test", nil)
	rec := httptest.NewRecorder()
	handler.Handle(rec, req)

	require.Equal(t, http.StatusFound, rec.Code, "valid logout_challenge must 302")
	location := rec.Header().Get("Location")
	require.NotEmpty(t, location, "302 must carry a Location header")
	// Stub redirect format: `https://oidc.stub.local/callback?flow=logout&challenge=<id>`
	require.True(t, strings.HasPrefix(location, "https://oidc.stub.local/callback?"),
		"Location must come from ResolveLogout (stub redirect format)")
	require.Contains(t, location, "flow=logout", "Location must carry flow=logout")
	require.Contains(t, location, "challenge=test", "Location must echo challenge id")
}

// TestLogoutHandlerPropagatesHydraFailure — when the underlying service
// returns a Hydra failure (network/upstream error), the handler MUST surface
// it via WriteChallengeError instead of attempting a 302 with stale data.
func TestLogoutHandlerPropagatesHydraFailure(t *testing.T) {
	t.Parallel()

	handler := server.NewLogoutHandler(zap.NewNop(), challenge.NewStubService())

	// `hydra-error` is a stub-recognized challenge id that triggers
	// NewHydraFailureError per `internal/challenge/stub.go:50`.
	req := httptest.NewRequest(http.MethodGet, "/oidc/logout?logout_challenge=hydra-error", nil)
	rec := httptest.NewRecorder()
	handler.Handle(rec, req)

	require.NotEqual(t, http.StatusFound, rec.Code, "hydra failure MUST NOT 302")
	require.True(t, rec.Code >= 500 && rec.Code < 600 || rec.Code == http.StatusBadGateway,
		"hydra failure should map to 5xx/502, got %d", rec.Code)
}
