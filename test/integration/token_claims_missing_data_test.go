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

			assertClaim(t, tc.description, "given_name", tc.expectedGiven, claims.GivenName, idToken)
			assertClaim(t, tc.description, "family_name", tc.expectedLast, claims.FamilyName, idToken)
		})
	}
}

func assertClaim(t *testing.T, description, claimName string, expected *string, actual *string, tokenMap map[string]any) {
	t.Helper()
	if expected == nil {
		if actual != nil {
			t.Fatalf("%s: unexpected %s %q", description, claimName, *actual)
		}
		if _, ok := tokenMap[claimName]; ok {
			t.Fatalf("%s: %s should be absent in ID token", description, claimName)
		}
	} else {
		if actual == nil || *actual != *expected {
			t.Fatalf("%s: expected %s=%q, got %v", description, claimName, *expected, actual)
		}
		if idVal, ok := tokenMap[claimName].(string); !ok || idVal != *expected {
			t.Fatalf("%s: ID token %s mismatch", description, claimName)
		}
	}
}

func stringPtr(val string) *string {
	return &val
}
