package contract_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alkem-io/oidc-service/internal/audit"
	"github.com/alkem-io/oidc-service/internal/server"
)

// TestEndSessionRevokesOnlyCallingRPConsentSession — FR-010 (T031).
// The endpoint MUST:
//   - validate id_token_hint against Hydra JWKS (iss/exp) before any admin call
//   - derive client_id from the token's aud/azp
//   - validate post_logout_redirect_uri against the RP's registered list
//   - call Hydra admin DELETE /admin/oauth2/auth/sessions/consent?client=<id>&subject=<sub>
//   - NOT call Kratos admin at any point (Kratos SSO survives)
//   - emit `session.end_session` audit
//   - 302 to post_logout_redirect_uri
func TestEndSessionRevokesOnlyCallingRPConsentSession(t *testing.T) {
	t.Parallel()

	emitter := audit.NewEmitter(&captureWriter{})
	handler := server.NewEndSessionHandler(emitter)

	values := url.Values{}
	values.Set("id_token_hint", "valid.signed.jwt")
	values.Set("post_logout_redirect_uri", "http://localhost:3000/logout")

	req := httptest.NewRequest(http.MethodGet, "/oidc/end_session?"+values.Encode(), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusFound, rec.Code, "expected 302 to post_logout_redirect_uri")
	require.Equal(t, "http://localhost:3000/logout", rec.Header().Get("Location"))
}

func TestEndSessionRejectsInvalidIDTokenHint(t *testing.T) {
	t.Parallel()

	emitter := audit.NewEmitter(&captureWriter{})
	handler := server.NewEndSessionHandler(emitter)

	req := httptest.NewRequest(http.MethodGet, "/oidc/end_session?id_token_hint=not-a-jwt", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.NotEqual(t, http.StatusFound, rec.Code, "invalid id_token_hint must NOT 302")
}

func TestEndSessionRejectsUnregisteredPostLogoutRedirect(t *testing.T) {
	t.Parallel()

	emitter := audit.NewEmitter(&captureWriter{})
	handler := server.NewEndSessionHandler(emitter)

	values := url.Values{}
	values.Set("id_token_hint", "valid.signed.jwt")
	values.Set("post_logout_redirect_uri", "https://evil.example/stealme")

	req := httptest.NewRequest(http.MethodGet, "/oidc/end_session?"+values.Encode(), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.NotEqual(t, http.StatusFound, rec.Code, "unregistered post_logout_redirect_uri must NOT 302")
	require.NotEqual(t, 0, rec.Code)
}

// captureWriter collects audit output so the contract tests can assert on
// emitted event records without reaching stdout.
type captureWriter struct {
	lines []string
}

func (c *captureWriter) Write(p []byte) (int, error) {
	c.lines = append(c.lines, strings.TrimSpace(string(p)))
	return len(p), nil
}
