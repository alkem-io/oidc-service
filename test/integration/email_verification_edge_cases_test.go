package integration_test

import (
	"encoding/json"
	"testing"

	"github.com/alkem-io/oidc-service/internal/challenge"
	kratosclient "github.com/ory/client-go"
)

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
			expected:    false,
			shouldOmit:  true,
			description: "No email in traits but verifiable addresses should omit claim",
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
		t.Run(tc.name, func(t *testing.T) {
			// Parse identity JSON
			var identity kratosclient.Identity
			if err := json.Unmarshal([]byte(tc.identityJSON), &identity); err != nil {
				t.Fatalf("Failed to parse identity JSON: %v", err)
			}

			// Test the internal function (assuming it's exported or we have access)
			// In a real implementation, this would need to be accessible for testing
			claims := challenge.TokenClaims{}

			// TODO: Call the actual extractEmailVerifiedClaim function
			// This would require either making the function exported or having a test helper
			// For now, we're documenting the expected behavior

			// Simulate the expected behavior based on the test case
			// This should be replaced with actual function calls once accessible
			if tc.shouldOmit {
				// Claim should be omitted (not present in token)
				// Verify that the claim is not set
				if claims.EmailVerified != nil {
					t.Errorf("%s: expected email_verified claim to be omitted, but got %v", tc.description, *claims.EmailVerified)
				}
			} else {
				// Claim should be present with expected value
				if claims.EmailVerified == nil {
					t.Errorf("%s: expected email_verified claim to be present, but it was omitted", tc.description)
				} else if *claims.EmailVerified != tc.expected {
					t.Errorf("%s: expected email_verified=%v, got %v", tc.description, tc.expected, *claims.EmailVerified)
				}
			}
		})
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
			expected:    false,
			shouldOmit:  true,
			description: "accepted_terms: 'true' string should omit claim (type safety)",
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
			shouldOmit:  true,
			description: "accepted_terms: 'false' string should omit claim (type safety)",
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
		t.Run(tc.name, func(t *testing.T) {
			// Parse identity JSON
			var identity kratosclient.Identity
			if err := json.Unmarshal([]byte(tc.identityJSON), &identity); err != nil {
				t.Fatalf("Failed to parse identity JSON: %v", err)
			}

			// Test the internal function (assuming it's exported or we have access)
			claims := challenge.TokenClaims{}

			// TODO: Call the actual extractAcceptedTermsClaim function
			// This would require either making the function exported or having a test helper
			// For now, we're documenting the expected behavior

			// Simulate the expected behavior based on the test case
			if tc.shouldOmit {
				// Claim should be omitted (not present in token)
				if claims.AcceptedTerms != nil {
					t.Errorf("%s: expected accepted_terms claim to be omitted, but got %v", tc.description, *claims.AcceptedTerms)
				}
			} else {
				// Claim should be present with expected value
				if claims.AcceptedTerms == nil {
					t.Errorf("%s: expected accepted_terms claim to be present, but it was omitted", tc.description)
				} else if *claims.AcceptedTerms != tc.expected {
					t.Errorf("%s: expected accepted_terms=%v, got %v", tc.description, tc.expected, *claims.AcceptedTerms)
				}
			}
		})
	}
}
