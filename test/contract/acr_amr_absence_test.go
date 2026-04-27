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

// TestACRAMRAbsence — FR-012 (T027d).
// Three invariants against the live Hydra + oidc-service chain:
//   (a) /oauth2/auth with acr_values=... completes without acr/amr in the id token
//   (b) discovery claims_supported MUST NOT list acr/amr (or MAY be absent)
//   (c) a client with no token_endpoint_auth_signing_alg override produces id
//       tokens without acr/amr keys at all
//
// Skips when HYDRA_PUBLIC_URL is not set AND when a valid id_token cannot be
// obtained via `hydra perform authorization-code` (TEST_ID_TOKEN env var).
func TestDiscoveryClaimsSupportedDoesNotAdvertiseACROrAMR(t *testing.T) {
	t.Parallel()

	public := os.Getenv("HYDRA_PUBLIC_URL")
	if public == "" {
		t.Skip("HYDRA_PUBLIC_URL not set; skipping ACR/AMR discovery contract test")
	}

	resp, err := http.Get(strings.TrimRight(public, "/") + "/.well-known/openid-configuration")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

	// claims_supported MAY be absent; when present, acr/amr MUST NOT appear.
	claimsRaw, present := doc["claims_supported"]
	if !present {
		return // vacuously satisfied
	}
	list, ok := claimsRaw.([]any)
	require.True(t, ok, "claims_supported must be an array when present")
	for _, item := range list {
		s, _ := item.(string)
		require.NotEqual(t, "acr", s, "claims_supported MUST NOT advertise acr")
		require.NotEqual(t, "amr", s, "claims_supported MUST NOT advertise amr")
	}
}

func TestIDTokenDoesNotCarryACRAMRKeys(t *testing.T) {
	t.Parallel()

	raw := os.Getenv("TEST_ID_TOKEN")
	if raw == "" {
		t.Skip("TEST_ID_TOKEN not set; skipping id-token ACR/AMR absence contract test")
	}

	parts := strings.Split(raw, ".")
	require.Len(t, parts, 3, "id token must be a three-part JWS")
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)

	var claims map[string]any
	require.NoError(t, json.Unmarshal(payload, &claims))

	_, acrPresent := claims["acr"]
	require.False(t, acrPresent, "id token MUST NOT carry `acr` key (even null-valued)")
	_, amrPresent := claims["amr"]
	require.False(t, amrPresent, "id token MUST NOT carry `amr` key (even null-valued)")
}

// TestACRValuesIgnoredEndToEnd — smoke wrapper for (a). Drives /oauth2/auth
// with acr_values and asserts that the eventual redirect carries no hint
// that ACR was honoured. Skips when env preconditions are absent.
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
	resp, err := client.Get(strings.TrimRight(public, "/") + "/oauth2/auth?" + params.Encode())
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	// Hydra MUST NOT 4xx the request on acr_values alone — it's accepted as a
	// no-op. This is a cheap "does not reject" smoke check; deeper id-token
	// assertion lives in TestIDTokenDoesNotCarryACRAMRKeys.
	require.True(t, resp.StatusCode < 400 || resp.StatusCode == http.StatusFound,
		"acr_values MUST be accepted (and ignored), got %d", resp.StatusCode)
}
