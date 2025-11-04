package challenge

import kratosclient "github.com/ory/client-go"

// TestExtractEmailVerifiedClaim exposes extractEmailVerifiedClaim for unit testing.
func TestExtractEmailVerifiedClaim(identity *kratosclient.Identity) *bool {
	return extractEmailVerifiedClaim(identity)
}

// TestExtractAcceptedTermsClaim exposes extractAcceptedTermsClaim for unit testing.
func TestExtractAcceptedTermsClaim(traits map[string]interface{}) *bool {
	return extractAcceptedTermsClaim(traits)
}

// TestExtractGivenNameClaim exposes extractGivenNameClaim for unit testing.
func TestExtractGivenNameClaim(traits map[string]interface{}) string {
	return extractGivenNameClaim(traits)
}

// TestExtractFamilyNameClaim exposes extractFamilyNameClaim for unit testing.
func TestExtractFamilyNameClaim(traits map[string]interface{}) string {
	return extractFamilyNameClaim(traits)
}

// TestExtractTokenClaims exposes extractTokenClaims for unit testing.
func TestExtractTokenClaims(identity *kratosclient.Identity) *TokenClaims {
	return extractTokenClaims(identity)
}