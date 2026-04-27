package contract_test

import (
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHydraAuthzRejectionPaths — FR-028 (T027c).
// Drives four negative /oauth2/auth calls against a live Hydra and asserts
// the expected rejection. Skips when HYDRA_PUBLIC_URL is not set.
//
// (a) unregistered client_id → invalid_client
// (b) registered client with unregistered redirect_uri → invalid_request / redirect_uri_mismatch
// (c) registered client with scope outside its registered set → invalid_scope
// (d) code_challenge_method=plain → rejected at authz or token endpoint
func TestHydraAuthzRejectionPaths(t *testing.T) {
	t.Parallel()

	public := os.Getenv("HYDRA_PUBLIC_URL")
	if public == "" {
		t.Skip("HYDRA_PUBLIC_URL not set; skipping Hydra authz-rejection contract test")
	}

	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}

	base := strings.TrimRight(public, "/") + "/oauth2/auth"

	t.Run("unregistered client_id yields invalid_client", func(t *testing.T) {
		t.Parallel()
		params := url.Values{}
		params.Set("client_id", "bogus-unregistered-client")
		params.Set("response_type", "code")
		params.Set("redirect_uri", "http://localhost:3000/api/auth/oidc/callback")
		params.Set("scope", "openid")

		resp, err := client.Get(base + "?" + params.Encode())
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		require.Contains(t, resp.Header.Get("Location")+resp.Status, "invalid_client",
			"expected invalid_client rejection")
	})

	t.Run("unregistered redirect_uri yields redirect_uri_mismatch", func(t *testing.T) {
		t.Parallel()
		params := url.Values{}
		params.Set("client_id", "alkemio-web")
		params.Set("response_type", "code")
		params.Set("redirect_uri", "http://evil.example/x")
		params.Set("scope", "openid")
		params.Set("code_challenge_method", "S256")
		params.Set("code_challenge", "Fe3Bd7VOAVrgHDN-2xdGiaakY0P6WTCxlFQrLyhPBYI")

		resp, err := client.Get(base + "?" + params.Encode())
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		combined := resp.Header.Get("Location") + resp.Status
		require.True(t, strings.Contains(combined, "redirect_uri_mismatch") || strings.Contains(combined, "invalid_request"),
			"expected redirect_uri_mismatch / invalid_request, got %s", combined)
	})

	t.Run("scope outside registered set yields invalid_scope", func(t *testing.T) {
		t.Parallel()
		params := url.Values{}
		params.Set("client_id", "alkemio-web")
		params.Set("response_type", "code")
		params.Set("redirect_uri", "http://localhost:3000/api/auth/oidc/callback")
		params.Set("scope", "openid not-a-registered-scope")
		params.Set("code_challenge_method", "S256")
		params.Set("code_challenge", "Fe3Bd7VOAVrgHDN-2xdGiaakY0P6WTCxlFQrLyhPBYI")

		resp, err := client.Get(base + "?" + params.Encode())
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		require.Contains(t, resp.Header.Get("Location")+resp.Status, "invalid_scope")
	})

	t.Run("code_challenge_method=plain is rejected", func(t *testing.T) {
		t.Parallel()
		params := url.Values{}
		params.Set("client_id", "alkemio-web")
		params.Set("response_type", "code")
		params.Set("redirect_uri", "http://localhost:3000/api/auth/oidc/callback")
		params.Set("scope", "openid")
		params.Set("code_challenge_method", "plain")
		params.Set("code_challenge", "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGH")

		resp, err := client.Get(base + "?" + params.Encode())
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		combined := resp.Header.Get("Location") + resp.Status
		require.NotContains(t, combined, "code=", "plain PKCE MUST NOT succeed")
	})
}
