package support

import (
	"encoding/json"
	"testing"

	kratosclient "github.com/ory/client-go"

	"github.com/alkem-io/oidc-service/internal/challenge"
)

// BuildIdentityFromJSON constructs a minimal Kratos identity by unmarshalling only the
// trait fields needed for token-claim extraction. It ignores validation required in
// full Kratos objects so tests can focus on claim behavior.
func BuildIdentityFromJSON(t *testing.T, identityJSON string) *kratosclient.Identity {
	t.Helper()
	type minimal struct {
		Traits              map[string]any   `json:"traits"`
		VerifiableAddresses []map[string]any `json:"verifiable_addresses"`
	}
	var payload minimal
	if err := json.Unmarshal([]byte(identityJSON), &payload); err != nil {
		t.Fatalf("failed to parse identity JSON: %v", err)
	}
	identity := &kratosclient.Identity{}
	if payload.Traits != nil {
		identity.Traits = payload.Traits
	}
	if len(payload.VerifiableAddresses) > 0 {
		addresses := make([]kratosclient.VerifiableIdentityAddress, 0, len(payload.VerifiableAddresses))
		for _, raw := range payload.VerifiableAddresses {
			addr := kratosclient.VerifiableIdentityAddress{}
			if value, ok := raw["value"].(string); ok {
				addr.Value = value
			}
			if via, ok := raw["via"].(string); ok {
				addr.Via = via
			}
			if verified, ok := raw["verified"].(bool); ok {
				addr.Verified = verified
			}
			addresses = append(addresses, addr)
		}
		identity.VerifiableAddresses = addresses
	}
	return identity
}

// ExtractTokenClaimsFromJSON is a convenience wrapper that builds an identity from
// JSON and runs it through the real token-claim extraction pipeline used by the
// service. It panics the test immediately if extraction unexpectedly returns nil.
func ExtractTokenClaimsFromJSON(t *testing.T, identityJSON string) *challenge.TokenClaims {
	t.Helper()
	identity := BuildIdentityFromJSON(t, identityJSON)
	claims := challenge.TestExtractTokenClaims(identity)
	if claims == nil {
		t.Fatalf("claims extraction returned nil for identity payload: %s", identityJSON)
	}
	return claims
}
