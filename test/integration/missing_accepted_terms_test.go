package integration_test

import (
	"testing"

	testsupport "github.com/alkem-io/oidc-service/test/support"
)

// TestAcceptedTermsClaims ensures accepted_terms is only emitted when the trait is a strict boolean or supported string.
func TestAcceptedTermsClaims(t *testing.T) {
	testCases := []struct {
		name         string
		identityJSON string
		expected     *bool
		description  string
	}{
		{
			name: "EmptyTraits",
			identityJSON: `{
				"traits": {}
			}`,
			expected:    nil,
			description: "Empty traits omits accepted_terms",
		},
		{
			name: "OtherTraitsOnly",
			identityJSON: `{
				"traits": {
					"email": "user@example.com",
					"name": {
						"first": "Alex",
						"last": "River"
					}
				}
			}`,
			expected:    nil,
			description: "Unrelated traits should not fabricate accepted_terms",
		},
		{
			name: "NullAcceptedTerms",
			identityJSON: `{
				"traits": {
					"accepted_terms": null
				}
			}`,
			expected:    nil,
			description: "Null accepted_terms omits the claim",
		},
		{
			name: "TrueBoolean",
			identityJSON: `{
				"traits": {
					"accepted_terms": true
				}
			}`,
			expected:    boolPtr(true),
			description: "Boolean true should surface accepted_terms",
		},
		{
			name: "FalseBoolean",
			identityJSON: `{
				"traits": {
					"accepted_terms": false
				}
			}`,
			expected:    boolPtr(false),
			description: "Boolean false still surfaces the claim",
		},
		{
			name: "StringTrue",
			identityJSON: `{
				"traits": {
					"accepted_terms": "true"
				}
			}`,
			expected:    boolPtr(true),
			description: "String true coerces to boolean",
		},
		{
			name: "StringNumericOne",
			identityJSON: `{
				"traits": {
					"accepted_terms": "1"
				}
			}`,
			expected:    boolPtr(true),
			description: "Numeric string one coerces to true",
		},
		{
			name: "StringFalse",
			identityJSON: `{
				"traits": {
					"accepted_terms": "false"
				}
			}`,
			expected:    boolPtr(false),
			description: "String false coerces to boolean",
		},
		{
			name: "UnsupportedType",
			identityJSON: `{
				"traits": {
					"accepted_terms": 123
				}
			}`,
			expected:    nil,
			description: "Unsupported numbers omit the claim",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			claims := testsupport.ExtractTokenClaimsFromJSON(t, tc.identityJSON)
			idTokenClaims := claims.ToIDTokenMap()

			if tc.expected == nil {
				if claims.AcceptedTerms != nil {
					t.Fatalf("%s: expected claim omission, got %v", tc.description, *claims.AcceptedTerms)
				}
				if _, exists := idTokenClaims["accepted_terms"]; exists {
					t.Fatalf("%s: claim leaked into ID token map", tc.description)
				}
				return
			}

			if claims.AcceptedTerms == nil {
				t.Fatalf("%s: expected claim presence", tc.description)
			}
			if *claims.AcceptedTerms != *tc.expected {
				t.Fatalf("%s: expected accepted_terms=%v, got %v", tc.description, *tc.expected, *claims.AcceptedTerms)
			}

			value, ok := idTokenClaims["accepted_terms"].(bool)
			if !ok {
				t.Fatalf("%s: expected boolean in ID token map", tc.description)
			}
			if value != *tc.expected {
				t.Fatalf("%s: ID token map mismatch expected %v got %v", tc.description, *tc.expected, value)
			}
		})
	}
}

// TestEmailVerifiedClaims ensures email_verified matches the verifiable address state.
func TestEmailVerifiedClaims(t *testing.T) {
	testCases := []struct {
		name         string
		identityJSON string
		expected     *bool
		description  string
	}{
		{
			name: "VerifiedEmail",
			identityJSON: `{
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
			expected:    boolPtr(true),
			description: "Verified email sets claim true",
		},
		{
			name: "UnverifiedEmail",
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
			expected:    boolPtr(false),
			description: "Presence but no verification yields false",
		},
		{
			name: "NoEmailAddresses",
			identityJSON: `{
				"traits": {
					"email": "user@example.com"
				}
			}`,
			expected:    nil,
			description: "Missing verifiable addresses omits claim",
		},
		{
			name: "NonEmailAddress",
			identityJSON: `{
				"traits": {
					"email": "user@example.com"
				},
				"verifiable_addresses": [
					{
						"value": "+15555555555",
						"verified": true,
						"via": "sms"
					}
				]
			}`,
			expected:    nil,
			description: "Non-email channels should not emit claim",
		},
		{
			name: "MissingVerifiedFieldDefaultsFalse",
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
			expected:    boolPtr(false),
			description: "Missing verified flag behaves like false",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			claims := testsupport.ExtractTokenClaimsFromJSON(t, tc.identityJSON)
			idTokenClaims := claims.ToIDTokenMap()

			if tc.expected == nil {
				if claims.EmailVerified != nil {
					t.Fatalf("%s: expected claim omission, got %v", tc.description, *claims.EmailVerified)
				}
				if _, exists := idTokenClaims["email_verified"]; exists {
					t.Fatalf("%s: claim leaked into ID token map", tc.description)
				}
				return
			}

			if claims.EmailVerified == nil {
				t.Fatalf("%s: expected claim presence", tc.description)
			}
			if *claims.EmailVerified != *tc.expected {
				t.Fatalf(
					"%s: expected email_verified=%v, got %v", tc.description, *tc.expected, *claims.EmailVerified,
				)
			}

			value, ok := idTokenClaims["email_verified"].(bool)
			if !ok {
				t.Fatalf("%s: expected boolean in ID token map", tc.description)
			}
			if value != *tc.expected {
				t.Fatalf("%s: ID token map mismatch expected %v got %v", tc.description, *tc.expected, value)
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}
