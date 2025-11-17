package integration_test

import (
	"testing"

	testsupport "github.com/alkem-io/oidc-service/test/support"
)

// TestTokenClaimsMissingData ensures missing name traits are omitted from token payloads.
func TestTokenClaimsMissingData(t *testing.T) {
	testCases := []struct {
		name         string
		identityJSON string
		description  string
	}{
		{
			name: "NoNameObject",
			identityJSON: `{
				"traits": {
					"email": "user@example.com"
				}
			}`,
			description: "Traits without name object should omit name claims",
		},
		{
			name: "EmptyNameObject",
			identityJSON: `{
				"traits": {
					"name": {}
				}
			}`,
			description: "Empty name object should not emit claims",
		},
		{
			name: "NullNameFields",
			identityJSON: `{
				"traits": {
					"name": {
						"first": null,
						"last": null
					}
				}
			}`,
			description: "Null name fields should be ignored",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			claims := testsupport.ExtractTokenClaimsFromJSON(t, tc.identityJSON)
			idToken := claims.ToIDTokenMap()

			if claims.GivenName != nil {
				t.Fatalf("%s: expected given_name to be omitted, got %q", tc.description, *claims.GivenName)
			}
			if claims.FamilyName != nil {
				t.Fatalf("%s: expected family_name to be omitted, got %q", tc.description, *claims.FamilyName)
			}
			if _, ok := idToken["given_name"]; ok {
				t.Fatalf("%s: given_name leaked into ID token map", tc.description)
			}
			if _, ok := idToken["family_name"]; ok {
				t.Fatalf("%s: family_name leaked into ID token map", tc.description)
			}
		})
	}
}

// TestTokenClaimsInvalidData ensures malformed name data is sanitized.
func TestTokenClaimsInvalidData(t *testing.T) {
	testCases := []struct {
		name         string
		identityJSON string
		description  string
	}{
		{
			name: "NameAsString",
			identityJSON: `{
				"traits": {
					"name": "not-an-object"
				}
			}`,
			description: "Name value as string should be ignored",
		},
		{
			name: "NumericFields",
			identityJSON: `{
				"traits": {
					"name": {
						"first": 123,
						"last": 456
					}
				}
			}`,
			description: "Numeric name fields should be ignored",
		},
		{
			name: "InvalidCharacters",
			identityJSON: `{
				"traits": {
					"name": {
						"first": "\u0001control",
						"last": "Doe"
					}
				}
			}`,
			description: "Disallowed characters should drop the field",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			claims := testsupport.ExtractTokenClaimsFromJSON(t, tc.identityJSON)

			if claims.GivenName != nil {
				t.Fatalf("%s: expected given_name omission, got %q", tc.description, *claims.GivenName)
			}
			// last name may pass for invalid characters test; ensure enforcement only when applicable
			if tc.name != "InvalidCharacters" && claims.FamilyName != nil {
				t.Fatalf("%s: expected family_name omission, got %q", tc.description, *claims.FamilyName)
			}
		})
	}
}

// TestTokenClaimsPartialMissingData ensures partial name data still surfaces valid claims.
func TestTokenClaimsPartialMissingData(t *testing.T) {
	testCases := []struct {
		name          string
		identityJSON  string
		expectedGiven *string
		expectedLast  *string
		description   string
	}{
		{
			name: "FirstNameOnly",
			identityJSON: `{
				"traits": {
					"name": {
						"first": "Alice"
					}
				}
			}`,
			expectedGiven: stringPtr("Alice"),
			expectedLast:  nil,
			description:   "Only first name should populate given_name",
		},
		{
			name: "LastNameOnly",
			identityJSON: `{
				"traits": {
					"name": {
						"last": "Smith"
					}
				}
			}`,
			expectedGiven: nil,
			expectedLast:  stringPtr("Smith"),
			description:   "Only last name should populate family_name",
		},
		{
			name: "UnicodeNames",
			identityJSON: `{
				"traits": {
					"name": {
						"first": "Zoë",
						"last": "García"
					}
				}
			}`,
			expectedGiven: stringPtr("Zoë"),
			expectedLast:  stringPtr("García"),
			description:   "UTF-8 names should be preserved",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			claims := testsupport.ExtractTokenClaimsFromJSON(t, tc.identityJSON)
			idToken := claims.ToIDTokenMap()

			if tc.expectedGiven == nil {
				if claims.GivenName != nil {
					t.Fatalf("%s: unexpected given_name %q", tc.description, *claims.GivenName)
				}
				if _, ok := idToken["given_name"]; ok {
					t.Fatalf("%s: given_name should be absent in ID token", tc.description)
				}
			} else {
				if claims.GivenName == nil || *claims.GivenName != *tc.expectedGiven {
					t.Fatalf("%s: expected given_name=%q, got %v", tc.description, *tc.expectedGiven, claims.GivenName)
				}
				if idVal, ok := idToken["given_name"].(string); !ok || idVal != *tc.expectedGiven {
					t.Fatalf("%s: ID token given_name mismatch", tc.description)
				}
			}

			if tc.expectedLast == nil {
				if claims.FamilyName != nil {
					t.Fatalf("%s: unexpected family_name %q", tc.description, *claims.FamilyName)
				}
				if _, ok := idToken["family_name"]; ok {
					t.Fatalf("%s: family_name should be absent in ID token", tc.description)
				}
			} else {
				if claims.FamilyName == nil || *claims.FamilyName != *tc.expectedLast {
					t.Fatalf("%s: expected family_name=%q, got %v", tc.description, *tc.expectedLast, claims.FamilyName)
				}
				if idVal, ok := idToken["family_name"].(string); !ok || idVal != *tc.expectedLast {
					t.Fatalf("%s: ID token family_name mismatch", tc.description)
				}
			}
		})
	}
}

func stringPtr(val string) *string {
	return &val
}
