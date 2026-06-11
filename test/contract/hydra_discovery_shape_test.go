package contract_test

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHydraDiscoveryPosture — FR-001/002/003/010/011 (T027b).
// Asserts the platform-mandated OIDC discovery shape against a live Hydra.
// Skips when HYDRA_PUBLIC_URL is not set so local dev runs green without
// bringing Hydra up.
func TestHydraDiscoveryPosture(t *testing.T) {
	t.Parallel()

	public := os.Getenv("HYDRA_PUBLIC_URL")
	if public == "" {
		t.Skip("HYDRA_PUBLIC_URL not set; skipping Hydra discovery contract test")
	}

	resp, err := http.Get(strings.TrimRight(public, "/") + "/.well-known/openid-configuration") //nolint:gosec // G704: target comes from HYDRA_PUBLIC_URL test env
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// FR: discovery responses SHOULD carry a Cache-Control with max-age=300.
	cacheCtrl := strings.ToLower(resp.Header.Get("Cache-Control"))
	require.Contains(t, cacheCtrl, "max-age=300",
		"discovery response MUST set Cache-Control max-age=300")

	var doc map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

	// FR-002 — PKCE S256 only; `plain` MUST be absent from supported methods.
	methods := stringSlice(doc["code_challenge_methods_supported"])
	require.Equal(t, []string{"S256"}, methods, "code_challenge_methods_supported must be exactly [S256]")

	// FR-003 — only authorization_code + refresh_token at /token.
	grants := stringSlice(doc["grant_types_supported"])
	require.ElementsMatch(t, []string{"authorization_code", "refresh_token"}, grants,
		"grant_types_supported must be exactly [authorization_code, refresh_token]")
	for _, forbidden := range []string{"client_credentials", "password", "implicit", "urn:ietf:params:oauth:grant-type:token-exchange"} {
		require.NotContains(t, grants, forbidden, "grant %s must NOT be advertised", forbidden)
	}

	// FR-011 — token_endpoint_auth_methods ⊆ {client_secret_basic, none}.
	authMethods := stringSlice(doc["token_endpoint_auth_methods_supported"])
	for _, m := range authMethods {
		require.Contains(t, []string{"client_secret_basic", "none"}, m,
			"token_endpoint_auth_method %s must not be advertised", m)
	}
	for _, forbidden := range []string{"client_secret_post", "private_key_jwt", "client_secret_jwt"} {
		require.NotContains(t, authMethods, forbidden)
	}

	// FR-001 — RS256 signing only.
	sigAlgs := stringSlice(doc["id_token_signing_alg_values_supported"])
	require.Equal(t, []string{"RS256"}, sigAlgs, "id_token_signing_alg_values_supported must be exactly [RS256]")

	// FR-010 — end_session_endpoint MUST be advertised.
	endSession, ok := doc["end_session_endpoint"].(string)
	require.True(t, ok && endSession != "", "end_session_endpoint MUST be present in discovery")

	// JWKS Cache-Control — 1 day.
	jwksURI, ok := doc["jwks_uri"].(string)
	require.True(t, ok && jwksURI != "")
	jwksResp, err := http.Get(jwksURI) //nolint:gosec // G107: jwks_uri comes from the trusted test Hydra discovery doc
	require.NoError(t, err)
	defer func() { _ = jwksResp.Body.Close() }()
	jwksCache := strings.ToLower(jwksResp.Header.Get("Cache-Control"))
	require.Contains(t, jwksCache, "max-age=86400",
		"JWKS response MUST set Cache-Control max-age=86400")
}

func stringSlice(v any) []string {
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
