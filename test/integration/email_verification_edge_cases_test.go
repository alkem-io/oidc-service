package integration_test

import (
	"encoding/json"
	"testing"

	kratosclient "github.com/ory/client-go"

	"github.com/alkem-io/oidc-service/internal/challenge"
)

// helper: build a minimal kratos Identity from JSON without triggering strict required-field checks
func buildIdentityFromJSON(t *testing.T, identityJSON string) *kratosclient.Identity {
	t.Helper()
	// Unmarshal only the fields we need using a minimal struct
	type minimal struct {
		Traits              map[string]interface{}   `json:"traits"`
		VerifiableAddresses []map[string]interface{} `json:"verifiable_addresses"`
	}
	var m minimal
	if err := json.Unmarshal([]byte(identityJSON), &m); err != nil {
		t.Fatalf("Failed to parse identity JSON: %v", err)
	}
	id := &kratosclient.Identity{}
	if m.Traits != nil {
		id.Traits = m.Traits
	}
	if len(m.VerifiableAddresses) > 0 {
		addrs := make([]kratosclient.VerifiableIdentityAddress, 0, len(m.VerifiableAddresses))
		for _, v := range m.VerifiableAddresses {
			addr := kratosclient.VerifiableIdentityAddress{}
			if val, ok := v["value"].(string); ok {
				addr.Value = val
			}
			if via, ok := v["via"].(string); ok {
				addr.Via = via
			}
			if verified, ok := v["verified"].(bool); ok {
				addr.Verified = verified
			}
			addrs = append(addrs, addr)
		}
		id.VerifiableAddresses = addrs
	}
	return id
}

// TestEmailVerificationEdgeCases tests edge cases in email verification claim extraction.
func TestEmailVerificationEdgeCases(t *testing.T) {
	testCases := []struct {
		name         string
		identityJSON string
		expected     bool
		shouldOmit   bool
		description  string
	}{
		{
			name: "SingleVerifiedEmail",
			identityJSON: `{
				"id": "test-user-1",
				"schema_id": "default",
				"state": "active",
				"traits": {
					"email": "user@example.com"
				},
				"verifiable_addresses": [
					{
						"value": "user@example.com",
						"verified": true,
						"via": "email"
					}
				]
			}`,
			expected:    true,
			shouldOmit:  false,
			description: "Single verified email should return true",
		},
		{
			name: "SingleUnverifiedEmail",
			identityJSON: `{
				"traits": {
					"email": "user@example.com"
				},
				"verifiable_addresses": [
					{
						"value": "user@example.com",
						"verified": false,
						"via": "email"
					}
				]
			}`,
			expected:    false,
			shouldOmit:  false,
			description: "Single unverified email should return false",
		},
		{
			name: "MultipleEmailsAllVerified",
			identityJSON: `{
				"traits": {
					"email": "user@example.com"
				},
				"verifiable_addresses": [
					{
						"value": "user@example.com",
						"verified": true,
						"via": "email"
					},
					{
						"value": "user2@example.com",
						"verified": true,
						"via": "email"
					}
				]
			}`,
			expected:    true,
			shouldOmit:  false,
			description: "Multiple emails all verified should return true",
		},
		{
			name: "MultipleEmailsSomeVerified",
			identityJSON: `{
				"traits": {
					"email": "user@example.com"
				},
				"verifiable_addresses": [
					{
						"value": "user@example.com",
						"verified": true,
						"via": "email"
					},
					{
						"value": "user2@example.com",
						"verified": false,
						"via": "email"
					}
				]
			}`,
			expected:    true,
			shouldOmit:  false,
			description: "Multiple emails with some verified should return true (OR logic)",
		},
		{
			name: "MultipleEmailsNoneVerified",
			identityJSON: `{
				"traits": {
					"email": "user@example.com"
				},
				"verifiable_addresses": [
					{
						"value": "user@example.com",
						"verified": false,
						"via": "email"
					},
					{
						"value": "user2@example.com",
						"verified": false,
						"via": "email"
					}
				]
			}`,
			expected:    false,
			shouldOmit:  false,
			description: "Multiple emails none verified should return false",
		},
		{
			name: "EmailInTraitsButNoVerifiableAddresses",
			identityJSON: `{
				"traits": {
					"email": "user@example.com"
				},
				"verifiable_addresses": []
			}`,
			expected:    false,
			shouldOmit:  true,
			description: "Email in traits but no verifiable addresses should omit claim",
		},
		{
			name: "NoEmailInTraitsButVerifiableAddresses",
			identityJSON: `{
				"traits": {
					"name": "John Doe"
				},
				"verifiable_addresses": [
					{
						"value": "user@example.com",
						"verified": true,
						"via": "email"
					}
				]
			}`,
			expected:    true,
			shouldOmit:  false,
			description: "No email in traits but verifiable addresses should still set claim based on verified addresses",
		},
		{
			name: "EmptyVerifiableAddresses",
			identityJSON: `{
				"traits": {
					"email": "user@example.com"
				}
			}`,
			expected:    false,
			shouldOmit:  true,
			description: "Missing verifiable_addresses field should omit claim",
		},
		{
			name: "NonEmailVerifiableAddresses",
			identityJSON: `{
				"traits": {
					"email": "user@example.com"
				},
				"verifiable_addresses": [
					{
						"value": "+1234567890",
						"verified": true,
						"via": "sms"
					}
				]
			}`,
			expected:    false,
			shouldOmit:  true,
			description: "Non-email verifiable addresses should omit claim",
		},
		{
			name: "MissingVerifiedField",
			identityJSON: `{
				"traits": {
					"email": "user@example.com"
				},
				"verifiable_addresses": [
					{
						"value": "user@example.com",
						"via": "email"
					}
				]
			}`,
			expected:    false,
			shouldOmit:  false,
			description: "Missing verified field should default to false",
		},
		{
			name: "InvalidEmailFormat",
			identityJSON: `{
				"traits": {
					"email": "invalid-email"
				},
				"verifiable_addresses": [
					{
						"value": "invalid-email",
						"verified": true,
						"via": "email"
					}
				]
			}`,
			expected:    true,
			shouldOmit:  false,
			description: "Invalid email format but verified should still return true",
		},
	}

	for _, tc := range testCases {
		t.Run(
			tc.name, func(t *testing.T) {
				// Build identity object from JSON (avoids strict required-field unmarshalling)
				identity := buildIdentityFromJSON(t, tc.identityJSON)

				// Call the actual extraction helper to get token claims
				claims := challenge.TestExtractTokenClaims(identity)

				// Defensive check to avoid nil-dereference on claims
				if claims == nil {
					t.Fatalf("%s: claims extraction returned nil (unexpected)", tc.description)
				}

				if tc.shouldOmit {
					if claims.EmailVerified != nil {
						t.Errorf(
							"%s: expected email_verified claim to be omitted, but got %v", tc.description,
							*claims.EmailVerified,
						)
					}
				} else {
					if claims.EmailVerified == nil {
						t.Errorf("%s: expected email_verified claim to be present, but it was omitted", tc.description)
					} else if *claims.EmailVerified != tc.expected {
						t.Errorf(
							"%s: expected email_verified=%v, got %v", tc.description, tc.expected,
							*claims.EmailVerified,
						)
					}
				}
			},
		)
	}
}

// TestAcceptedTermsEdgeCases tests edge cases in accepted terms claim extraction.
func TestAcceptedTermsEdgeCases(t *testing.T) {
	testCases := []struct {
		name         string
		identityJSON string
		expected     bool
		shouldOmit   bool
		description  string
	}{
		{
			name: "AcceptedTermsTrue",
			identityJSON: `{
				"traits": {
					"email": "user@example.com",
					"accepted_terms": true
				}
			}`,
			expected:    true,
			shouldOmit:  false,
			description: "accepted_terms: true should return true",
		},
		{
			name: "AcceptedTermsFalse",
			identityJSON: `{
				"traits": {
					"email": "user@example.com", 
					"accepted_terms": false
				}
			}`,
			expected:    false,
			shouldOmit:  false,
			description: "accepted_terms: false should return false",
		},
		{
			name: "AcceptedTermsStringTrue",
			identityJSON: `{
				"traits": {
					"email": "user@example.com",
					"accepted_terms": "true"
				}
			}`,
			expected:    true,
			shouldOmit:  false,
			description: "accepted_terms: 'true' string should be parsed as true",
		},
		{
			name: "AcceptedTermsStringFalse",
			identityJSON: `{
				"traits": {
					"email": "user@example.com",
					"accepted_terms": "false"
				}
			}`,
			expected:    false,
			shouldOmit:  false,
			description: "accepted_terms: 'false' string should be parsed as false",
		},
		{
			name: "AcceptedTermsNumber",
			identityJSON: `{
				"traits": {
					"email": "user@example.com",
					"accepted_terms": 1
				}
			}`,
			expected:    false,
			shouldOmit:  true,
			description: "accepted_terms: 1 number should omit claim (type safety)",
		},
		{
			name: "AcceptedTermsNull",
			identityJSON: `{
				"traits": {
					"email": "user@example.com",
					"accepted_terms": null
				}
			}`,
			expected:    false,
			shouldOmit:  true,
			description: "accepted_terms: null should omit claim",
		},
		{
			name: "MissingAcceptedTerms",
			identityJSON: `{
				"traits": {
					"email": "user@example.com"
				}
			}`,
			expected:    false,
			shouldOmit:  true,
			description: "Missing accepted_terms should omit claim",
		},
		{
			name: "EmptyTraits",
			identityJSON: `{
				"traits": {}
			}`,
			expected:    false,
			shouldOmit:  true,
			description: "Empty traits should omit claim",
		},
		{
			name:         "MissingTraits",
			identityJSON: `{}`,
			expected:     false,
			shouldOmit:   true,
			description:  "Missing traits should omit claim",
		},
	}

	for _, tc := range testCases {
		t.Run(
			tc.name, func(t *testing.T) {
				// Build identity object from JSON
				identity := buildIdentityFromJSON(t, tc.identityJSON)

				claims := challenge.TestExtractTokenClaims(identity)

				// Defensive check to avoid nil-dereference on claims
				if claims == nil {
					t.Fatalf("%s: claims extraction returned nil (unexpected)", tc.description)
				}

				if tc.shouldOmit {
					if claims.AcceptedTerms != nil {
						t.Errorf(
							"%s: expected accepted_terms claim to be omitted, but got %v", tc.description,
							*claims.AcceptedTerms,
						)
					}
				} else {
					if claims.AcceptedTerms == nil {
						t.Errorf("%s: expected accepted_terms claim to be present, but it was omitted", tc.description)
					} else if *claims.AcceptedTerms != tc.expected {
						t.Errorf(
							"%s: expected accepted_terms=%v, got %v", tc.description, tc.expected,
							*claims.AcceptedTerms,
						)
					}
				}
			},
		)
	}
}
