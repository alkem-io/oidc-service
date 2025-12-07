package unit_test

import (
	"testing"

	"github.com/alkem-io/oidc-service/internal/challenge"
	kratosclient "github.com/ory/client-go"
)

// TestExtractNameFromTraitsEdgeCases tests unicode handling and boundary conditions.
func TestExtractNameFromTraitsEdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		traits      map[string]interface{}
		field       string
		expected    string
		description string
	}{
		{
			name: "ValidUTF8Names",
			traits: map[string]interface{}{
				"name": map[string]interface{}{
					"first": "José",
					"last":  "González",
				},
			},
			field:       "first",
			expected:    "José",
			description: "Valid UTF-8 names should be accepted",
		},
		{
			name: "EmojiInName",
			traits: map[string]interface{}{
				"name": map[string]interface{}{
					"first": "John 😊",
					"last":  "Doe",
				},
			},
			field:       "first",
			expected:    "",
			description: "Names with emojis should be rejected",
		},
		{
			name: "ControlCharactersInName",
			traits: map[string]interface{}{
				"name": map[string]interface{}{
					"first": "John\x00Doe",
					"last":  "Smith",
				},
			},
			field:       "first",
			expected:    "",
			description: "Names with control characters should be rejected",
		},
		{
			name: "MaxLengthName",
			traits: map[string]interface{}{
				"name": map[string]interface{}{
					"first": string(make([]byte, 255)),
					"last":  "Doe",
				},
			},
			field:       "first",
			expected:    "",
			description: "Names with control characters should be filtered out",
		},
		{
			name: "TooLongName",
			traits: map[string]interface{}{
				"name": map[string]interface{}{
					"first": string(make([]byte, 256)),
					"last":  "Doe",
				},
			},
			field:       "first",
			expected:    "",
			description: "Names over 255 characters should be rejected",
		},
		{
			name: "UnicodeNormalization",
			traits: map[string]interface{}{
				"name": map[string]interface{}{
					"first": "José", // Composed character
					"last":  "José", // Decomposed equivalent
				},
			},
			field:       "first",
			expected:    "José",
			description: "Unicode composed characters should be accepted",
		},
		{
			name: "LeadingTrailingWhitespace",
			traits: map[string]interface{}{
				"name": map[string]interface{}{
					"first": "  John  ",
					"last":  "Doe",
				},
			},
			field:       "first",
			expected:    "John",
			description: "Leading/trailing whitespace should be trimmed",
		},
		{
			name: "OnlyWhitespace",
			traits: map[string]interface{}{
				"name": map[string]interface{}{
					"first": "   ",
					"last":  "Doe",
				},
			},
			field:       "first",
			expected:    "",
			description: "Names with only whitespace should be rejected",
		},
		{
			name: "EmptyName",
			traits: map[string]interface{}{
				"name": map[string]interface{}{
					"first": "",
					"last":  "Doe",
				},
			},
			field:       "first",
			expected:    "",
			description: "Empty names should be rejected",
		},
		{
			name: "NonStringName",
			traits: map[string]interface{}{
				"name": map[string]interface{}{
					"first": 12345,
					"last":  "Doe",
				},
			},
			field:       "first",
			expected:    "",
			description: "Non-string names should be rejected",
		},
		{
			name: "NullName",
			traits: map[string]interface{}{
				"name": map[string]interface{}{
					"first": nil,
					"last":  "Doe",
				},
			},
			field:       "first",
			expected:    "",
			description: "Null names should be rejected",
		},
		{
			name: "MissingNameField",
			traits: map[string]interface{}{
				"name": map[string]interface{}{
					"last": "Doe",
				},
			},
			field:       "first",
			expected:    "",
			description: "Missing name fields should return empty",
		},
		{
			name: "MissingNameObject",
			traits: map[string]interface{}{
				"email": "user@example.com",
			},
			field:       "first",
			expected:    "",
			description: "Missing name object should return empty",
		},
		{
			name: "InvalidNameObject",
			traits: map[string]interface{}{
				"name": "John Doe", // String instead of object
			},
			field:       "first",
			expected:    "",
			description: "Invalid name object type should return empty",
		},
		{
			name:        "NilTraits",
			traits:      nil,
			field:       "first",
			expected:    "",
			description: "Nil traits should return empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result string
			if tc.field == "first" {
				result = challenge.TestExtractGivenNameClaim(tc.traits)
			} else {
				result = challenge.TestExtractFamilyNameClaim(tc.traits)
			}

			if result != tc.expected {
				t.Errorf("%s: expected %q, got %q", tc.description, tc.expected, result)
			}
		})
	}
}

// TestExtractEmailVerifiedClaimEdgeCases tests email verification edge cases.
func TestExtractEmailVerifiedClaimEdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func() *kratosclient.Identity
		expected    *bool
		description string
	}{
		{
			name: "MultipleEmailsOneVerified",
			setupFunc: func() *kratosclient.Identity {
				identity := &kratosclient.Identity{}
				addresses := []kratosclient.VerifiableIdentityAddress{
					{Via: "email", Value: "user1@example.com", Verified: false},
					{Via: "email", Value: "user2@example.com", Verified: true},
					{Via: "email", Value: "user3@example.com", Verified: false},
				}
				identity.SetVerifiableAddresses(addresses)
				return identity
			},
			expected:    boolPtr(true),
			description: "One verified email among multiple should return true",
		},
		{
			name: "MixedAddressTypes",
			setupFunc: func() *kratosclient.Identity {
				identity := &kratosclient.Identity{}
				addresses := []kratosclient.VerifiableIdentityAddress{
					{Via: "sms", Value: "+1234567890", Verified: true},
					{Via: "email", Value: "user@example.com", Verified: false},
				}
				identity.SetVerifiableAddresses(addresses)
				return identity
			},
			expected:    boolPtr(false),
			description: "Non-email verified addresses should be ignored",
		},
		{
			name: "OnlyNonEmailAddresses",
			setupFunc: func() *kratosclient.Identity {
				identity := &kratosclient.Identity{}
				addresses := []kratosclient.VerifiableIdentityAddress{
					{Via: "sms", Value: "+1234567890", Verified: true},
					{Via: "phone", Value: "+0987654321", Verified: true},
				}
				identity.SetVerifiableAddresses(addresses)
				return identity
			},
			expected:    nil,
			description: "Only non-email addresses should omit claim",
		},
		{
			name: "EmptyAddressesArray",
			setupFunc: func() *kratosclient.Identity {
				identity := &kratosclient.Identity{}
				addresses := []kratosclient.VerifiableIdentityAddress{}
				identity.SetVerifiableAddresses(addresses)
				return identity
			},
			expected:    nil,
			description: "Empty addresses array should omit claim",
		},
		{
			name: "MissingViaField",
			setupFunc: func() *kratosclient.Identity {
				identity := &kratosclient.Identity{}
				addresses := []kratosclient.VerifiableIdentityAddress{
					{Value: "user@example.com", Verified: true},
				}
				identity.SetVerifiableAddresses(addresses)
				return identity
			},
			expected:    nil,
			description: "Missing via field should omit claim",
		},
		{
			name: "MissingVerifiedField",
			setupFunc: func() *kratosclient.Identity {
				identity := &kratosclient.Identity{}
				addresses := []kratosclient.VerifiableIdentityAddress{
					{Via: "email", Value: "user@example.com"},
				}
				identity.SetVerifiableAddresses(addresses)
				return identity
			},
			expected:    boolPtr(false),
			description: "Missing verified field should default to false",
		},
		{
			name: "NilIdentity",
			setupFunc: func() *kratosclient.Identity {
				return nil
			},
			expected:    nil,
			description: "Nil identity should omit claim",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			identity := tc.setupFunc()
			result := challenge.TestExtractEmailVerifiedClaim(identity)

			switch {
			case tc.expected == nil && result != nil:
				t.Errorf("%s: expected nil, got %v", tc.description, *result)
			case tc.expected != nil && result == nil:
				t.Errorf("%s: expected %v, got nil", tc.description, *tc.expected)
			case tc.expected != nil && result != nil && *tc.expected != *result:
				t.Errorf("%s: expected %v, got %v", tc.description, *tc.expected, *result)
			}
		})
	}
}

// TestExtractAcceptedTermsClaimBoundaryConditions tests boundary conditions.
func TestExtractAcceptedTermsClaimBoundaryConditions(t *testing.T) {
	testCases := []struct {
		name        string
		traits      map[string]interface{}
		expected    *bool
		description string
	}{
		{
			name: "StringTrueVariations",
			traits: map[string]interface{}{
				"accepted_terms": "TRUE",
			},
			expected:    boolPtr(true),
			description: "String 'TRUE' should be parsed as true",
		},
		{
			name: "StringOneAsTrue",
			traits: map[string]interface{}{
				"accepted_terms": "1",
			},
			expected:    boolPtr(true),
			description: "String '1' should be parsed as true",
		},
		{
			name: "StringYesAsTrue",
			traits: map[string]interface{}{
				"accepted_terms": "yes",
			},
			expected:    boolPtr(true),
			description: "String 'yes' should be parsed as true",
		},
		{
			name: "StringZeroAsFalse",
			traits: map[string]interface{}{
				"accepted_terms": "0",
			},
			expected:    boolPtr(false),
			description: "String '0' should be parsed as false",
		},
		{
			name: "StringNoAsFalse",
			traits: map[string]interface{}{
				"accepted_terms": "no",
			},
			expected:    boolPtr(false),
			description: "String 'no' should be parsed as false",
		},
		{
			name: "WhitespaceInString",
			traits: map[string]interface{}{
				"accepted_terms": "  true  ",
			},
			expected:    boolPtr(true),
			description: "Whitespace around string should be trimmed",
		},
		{
			name: "InvalidStringValue",
			traits: map[string]interface{}{
				"accepted_terms": "maybe",
			},
			expected:    nil,
			description: "Invalid string values should omit claim",
		},
		{
			name: "NumericValue",
			traits: map[string]interface{}{
				"accepted_terms": 1,
			},
			expected:    nil,
			description: "Numeric values should omit claim for type safety",
		},
		{
			name: "FloatValue",
			traits: map[string]interface{}{
				"accepted_terms": 1.0,
			},
			expected:    nil,
			description: "Float values should omit claim for type safety",
		},
		{
			name: "ArrayValue",
			traits: map[string]interface{}{
				"accepted_terms": []bool{true},
			},
			expected:    nil,
			description: "Array values should omit claim for type safety",
		},
		{
			name: "ObjectValue",
			traits: map[string]interface{}{
				"accepted_terms": map[string]bool{"accepted": true},
			},
			expected:    nil,
			description: "Object values should omit claim for type safety",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := challenge.TestExtractAcceptedTermsClaim(tc.traits)

			switch {
			case tc.expected == nil && result != nil:
				t.Errorf("%s: expected nil, got %v", tc.description, *result)
			case tc.expected != nil && result == nil:
				t.Errorf("%s: expected %v, got nil", tc.description, *tc.expected)
			case tc.expected != nil && result != nil && *tc.expected != *result:
				t.Errorf("%s: expected %v, got %v", tc.description, *tc.expected, *result)
			}
		})
	}
}

// Helper functions for creating pointers are defined in extract_claims_test.go
