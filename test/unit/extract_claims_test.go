package unit_test

import (
	"testing"

	"github.com/alkem-io/oidc-service/internal/challenge"
	kratosclient "github.com/ory/client-go"
)

// TestExtractEmailVerifiedClaimUnit tests the extractEmailVerifiedClaim function directly.
func TestExtractEmailVerifiedClaimUnit(t *testing.T) {
	testCases := []struct {
		name        string
		identity    *kratosclient.Identity
		expected    *bool
		description string
	}{
		{
			name: "SingleVerifiedEmail",
			identity: &kratosclient.Identity{
				VerifiableAddresses: []kratosclient.VerifiableIdentityAddress{
					{
						Value:    "user@example.com",
						Verified: true,
						Via:      "email",
					},
				},
			},
			expected:    boolPtr(true),
			description: "Single verified email should return true",
		},
		{
			name: "SingleUnverifiedEmail",
			identity: &kratosclient.Identity{
				VerifiableAddresses: []kratosclient.VerifiableIdentityAddress{
					{
						Value:    "user@example.com",
						Verified: false,
						Via:      "email",
					},
				},
			},
			expected:    boolPtr(false),
			description: "Single unverified email should return false",
		},
		{
			name: "MultipleEmailsSomeVerified",
			identity: &kratosclient.Identity{
				VerifiableAddresses: []kratosclient.VerifiableIdentityAddress{
					{
						Value:    "user@example.com",
						Verified: true,
						Via:      "email",
					},
					{
						Value:    "user2@example.com",
						Verified: false,
						Via:      "email",
					},
				},
			},
			expected:    boolPtr(true),
			description: "Multiple emails with some verified should return true (OR logic)",
		},
		{
			name: "MultipleEmailsNoneVerified",
			identity: &kratosclient.Identity{
				VerifiableAddresses: []kratosclient.VerifiableIdentityAddress{
					{
						Value:    "user@example.com",
						Verified: false,
						Via:      "email",
					},
					{
						Value:    "user2@example.com",
						Verified: false,
						Via:      "email",
					},
				},
			},
			expected:    boolPtr(false),
			description: "Multiple emails none verified should return false",
		},
		{
			name: "EmptyVerifiableAddresses",
			identity: &kratosclient.Identity{
				VerifiableAddresses: []kratosclient.VerifiableIdentityAddress{},
			},
			expected:    nil,
			description: "Empty verifiable addresses should return nil",
		},
		{
			name: "NonEmailVerifiableAddresses",
			identity: &kratosclient.Identity{
				VerifiableAddresses: []kratosclient.VerifiableIdentityAddress{
					{
						Value:    "+1234567890",
						Verified: true,
						Via:      "sms",
					},
				},
			},
			expected:    nil,
			description: "Non-email verifiable addresses should return nil (omit claim)",
		},
		{
			name:        "NilIdentity",
			identity:    nil,
			expected:    nil,
			description: "Nil identity should return nil",
		},
	}

	for _, tc := range testCases {
		t.Run(
			tc.name, func(t *testing.T) {
				// Note: This test requires access to the extractEmailVerifiedClaim function
				// In a real implementation, this function should be exported or
				// we should have a test helper that allows testing internal functions

				// For now, we're documenting the expected behavior
				// TODO: Implement actual function call once it's accessible

				result := challenge.TestExtractEmailVerifiedClaim(tc.identity)

				if tc.expected == nil && result != nil {
					t.Errorf("%s: expected nil, got %v", tc.description, *result)
				} else if tc.expected != nil && result == nil {
					t.Errorf("%s: expected %v, got nil", tc.description, *tc.expected)
				} else if tc.expected != nil && result != nil && *tc.expected != *result {
					t.Errorf("%s: expected %v, got %v", tc.description, *tc.expected, *result)
				}
			},
		)
	}
}

// TestExtractAcceptedTermsClaimUnit tests the extractAcceptedTermsClaim function directly.
func TestExtractAcceptedTermsClaimUnit(t *testing.T) {
	testCases := []struct {
		name        string
		traits      map[string]interface{}
		expected    *bool
		description string
	}{
		{
			name: "AcceptedTermsTrue",
			traits: map[string]interface{}{
				"email":          "user@example.com",
				"accepted_terms": true,
			},
			expected:    boolPtr(true),
			description: "accepted_terms: true should return true",
		},
		{
			name: "AcceptedTermsFalse",
			traits: map[string]interface{}{
				"email":          "user@example.com",
				"accepted_terms": false,
			},
			expected:    boolPtr(false),
			description: "accepted_terms: false should return false",
		},
		{
			name: "AcceptedTermsStringTrue",
			traits: map[string]interface{}{
				"email":          "user@example.com",
				"accepted_terms": "true",
			},
			expected:    boolPtr(true),
			description: "accepted_terms: 'true' string should return true",
		},
		{
			name: "AcceptedTermsStringFalse",
			traits: map[string]interface{}{
				"email":          "user@example.com",
				"accepted_terms": "false",
			},
			expected:    boolPtr(false),
			description: "accepted_terms: 'false' string should return false",
		},
		{
			name: "AcceptedTermsInvalidString",
			traits: map[string]interface{}{
				"email":          "user@example.com",
				"accepted_terms": "maybe",
			},
			expected:    nil,
			description: "accepted_terms: 'maybe' invalid string should return nil",
		},
		{
			name: "AcceptedTermsNumber",
			traits: map[string]interface{}{
				"email":          "user@example.com",
				"accepted_terms": 1,
			},
			expected:    nil,
			description: "accepted_terms: 1 number should return nil (type safety)",
		},
		{
			name: "MissingAcceptedTerms",
			traits: map[string]interface{}{
				"email": "user@example.com",
			},
			expected:    nil,
			description: "Missing accepted_terms should return nil",
		},
		{
			name:        "NilTraits",
			traits:      nil,
			expected:    nil,
			description: "Nil traits should return nil",
		},
	}

	for _, tc := range testCases {
		t.Run(
			tc.name, func(t *testing.T) {
				// Note: This test requires access to the extractAcceptedTermsClaim function
				// TODO: Implement actual function call once it's accessible

				result := challenge.TestExtractAcceptedTermsClaim(tc.traits)

				if tc.expected == nil && result != nil {
					t.Errorf("%s: expected nil, got %v", tc.description, *result)
				} else if tc.expected != nil && result == nil {
					t.Errorf("%s: expected %v, got nil", tc.description, *tc.expected)
				} else if tc.expected != nil && result != nil && *tc.expected != *result {
					t.Errorf("%s: expected %v, got %v", tc.description, *tc.expected, *result)
				}
			},
		)
	}
}

// Helper functions for creating pointers
func boolPtr(b bool) *bool {
	return &b
}
