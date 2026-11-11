package contract_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Tokens of a sign-in carry its Kratos assurance level as a top-level `acr`
// claim, `aal1` or `aal2`, on the ID token and on the access token; `amr` is
// not part of the contract.
//
// TEST_ID_TOKEN and TEST_ACCESS_TOKEN are minted by the `alkemio-web` client
// through the ordinary consent path, against a Hydra whose
// `oauth2.allowed_top_level_claims` lists `acr` (without that entry Hydra
// nests the access-token claim under `ext`, and the access-token case fails).
// Each test skips when its token is not set.

func TestIDTokenCarriesACR(t *testing.T) {
	t.Parallel()
	requireTopLevelACR(t, "TEST_ID_TOKEN")
}

func TestAccessTokenCarriesTopLevelACR(t *testing.T) {
	t.Parallel()
	requireTopLevelACR(t, "TEST_ACCESS_TOKEN")
}

func requireTopLevelACR(t *testing.T, envVar string) {
	t.Helper()

	raw := os.Getenv(envVar)
	if raw == "" {
		t.Skipf("%s not set; skipping acr presence contract test", envVar)
	}

	parts := strings.Split(raw, ".")
	require.Len(t, parts, 3, "token must be a three-part JWS")
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)

	var claims map[string]any
	require.NoError(t, json.Unmarshal(payload, &claims))

	require.Contains(t, []any{"aal1", "aal2"}, claims["acr"], "token must carry a top-level `acr` of aal1 or aal2")
}

// TestACRValuesIgnoredEndToEnd drives /oauth2/auth with acr_values and asserts
// Hydra accepts the request: acr_values is not honoured, and must not be refused.
func TestACRValuesIgnoredEndToEnd(t *testing.T) {
	t.Parallel()

	public := os.Getenv("HYDRA_PUBLIC_URL")
	if public == "" {
		t.Skip("HYDRA_PUBLIC_URL not set; skipping ACR end-to-end check")
	}

	params := url.Values{}
	params.Set("client_id", "alkemio-web")
	params.Set("response_type", "code")
	params.Set("redirect_uri", "http://localhost:3000/api/auth/oidc/callback")
	params.Set("scope", "openid")
	params.Set("code_challenge_method", "S256")
	params.Set("code_challenge", "Fe3Bd7VOAVrgHDN-2xdGiaakY0P6WTCxlFQrLyhPBYI")
	params.Set("acr_values", "urn:mace:incommon:iap:silver")

	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Get(strings.TrimRight(public, "/") + "/oauth2/auth?" + params.Encode()) //nolint:gosec // G704: target comes from HYDRA_PUBLIC_URL test env
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Less(t, resp.StatusCode, 400,
		"acr_values MUST be accepted (and ignored), got %d", resp.StatusCode)
}
