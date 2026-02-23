package challenge

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	kratosclient "github.com/ory/client-go"

	"github.com/alkem-io/oidc-service/internal/alkemio"
)

// IdentityProviderFunc fetches a Kratos identity and exposes the raw HTTP response for error inspection.
type IdentityProviderFunc func(ctx context.Context, identityID string) (*kratosclient.Identity, *http.Response, error)

// IdentityMapper retrieves identity records from Kratos and converts them into domain profiles.
type IdentityMapper struct {
	fetch IdentityProviderFunc
}

var errInvalidAlkemioMapping = errors.New("invalid alkemio identity mapping")

// NewIdentityMapper constructs an IdentityMapper backed by the given Kratos API implementation.
func NewIdentityMapper(api kratosclient.IdentityAPI) *IdentityMapper {
	return &IdentityMapper{
		fetch: func(ctx context.Context, identityID string) (*kratosclient.Identity, *http.Response, error) {
			identity, resp, err := api.GetIdentity(ctx, identityID).Execute()
			return identity, resp, err
		},
	}
}

// NewIdentityMapperWithProvider builds an IdentityMapper using the supplied provider function.
func NewIdentityMapperWithProvider(provider IdentityProviderFunc) *IdentityMapper {
	if provider == nil {
		panic("identity provider func is required")
	}
	return &IdentityMapper{fetch: provider}
}

// MissingTraitsError indicates Kratos returned an identity without the required traits.
type MissingTraitsError struct {
	Traits []string
}

// Error satisfies the error interface for MissingTraitsError.
func (e *MissingTraitsError) Error() string {
	return fmt.Sprintf("missing required traits: %s", strings.Join(e.Traits, ", "))
}

// IdentityNotFoundError represents a 404 response from Kratos.
type IdentityNotFoundError struct {
	IdentityID string
}

// Error satisfies the error interface for IdentityNotFoundError.
func (e *IdentityNotFoundError) Error() string {
	return fmt.Sprintf("kratos identity %q not found", e.IdentityID)
}

// IdentityLookupError wraps unexpected failures reaching Kratos.
type IdentityLookupError struct {
	Err error
}

// Error satisfies the error interface for IdentityLookupError.
func (e *IdentityLookupError) Error() string {
	return fmt.Sprintf("kratos identity lookup failed: %v", e.Err)
}

// Unwrap exposes the underlying lookup error for errors.Is/As support.
func (e *IdentityLookupError) Unwrap() error {
	return e.Err
}

// Fetch retrieves the identity profile for the supplied identifier, validating required traits.
func (m *IdentityMapper) Fetch(ctx context.Context, identityID string) (*IdentityProfile, error) {
	if m == nil || m.fetch == nil {
		return nil, errors.New("identity mapper not initialised")
	}

	id := strings.TrimSpace(identityID)
	if id == "" {
		return nil, errors.New("identity id is required")
	}

	identity, resp, err := m.fetch(ctx, id)
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, &IdentityNotFoundError{IdentityID: id}
		}
		return nil, &IdentityLookupError{Err: err}
	}

	profile, missingErr := buildIdentityProfile(identity)
	if missingErr != nil {
		return nil, missingErr
	}

	return profile, nil
}

func buildIdentityProfile(identity *kratosclient.Identity) (*IdentityProfile, error) {
	if identity == nil {
		return nil, &IdentityLookupError{Err: errors.New("identity payload was empty")}
	}

	traits, ok := identity.GetTraits().(map[string]interface{})
	if !ok {
		return nil, &MissingTraitsError{Traits: requiredTraitKeys()}
	}

	missing := make([]string, 0, 2)

	email := extractString(traits["email"])
	if email == "" || !isValidEmail(email) {
		missing = append(missing, "traits.email")
	}

	displayName := deriveDisplayName(traits)
	if displayName == "" {
		missing = append(missing, "traits.display_name")
	}

	if len(missing) > 0 {
		return nil, &MissingTraitsError{Traits: dedupe(missing)}
	}

	profile := &IdentityProfile{
		ID:          identity.GetId(),
		Email:       email,
		DisplayName: displayName,
		Traits:      cloneMap(traits),
		TokenClaims: extractTokenClaims(identity),
	}

	if meta := identity.GetMetadataPublic(); meta != nil {
		if value, ok := meta["matrix_user_id"]; ok {
			if matrixID := extractString(value); matrixID != "" {
				profile.MatrixUserID = matrixID
			}
		}
	}

	return profile, nil
}

func requiredTraitKeys() []string {
	return []string{"traits.email", "traits.display_name"}
}

func extractString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return ""
	}
}

func deriveDisplayName(traits map[string]interface{}) string {
	if traits == nil {
		return ""
	}

	if value := extractString(traits["display_name"]); value != "" {
		return value
	}

	if nameValue, ok := traits["name"]; ok {
		if name, ok := nameValue.(map[string]any); ok {
			first := extractString(name["first"])
			last := extractString(name["last"])
			if combined := strings.TrimSpace(strings.TrimSpace(first + " " + last)); combined != "" {
				return combined
			}
			if first != "" {
				return first
			}
			if last != "" {
				return last
			}
		}
	}

	return ""
}

func isValidEmail(value string) bool {
	if value == "" {
		return false
	}
	if _, err := mail.ParseAddress(value); err != nil {
		return false
	}
	return true
}

func cloneMap(src map[string]interface{}) map[string]any {
	if src == nil {
		return nil
	}
	clone := make(map[string]any, len(src))
	for key, value := range src {
		clone[key] = value
	}
	return clone
}

func dedupe(values []string) []string {
	if len(values) <= 1 {
		return values
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

// validateAlkemioMapping ensures resolver payloads include a canonical UUID for the actor id.
func validateAlkemioMapping(mapping *alkemio.IdentityMapping) (*alkemio.IdentityMapping, error) {
	if mapping == nil {
		return nil, invalidMappingError("mapping missing")
	}

	actorID := strings.TrimSpace(mapping.ActorID)
	if actorID == "" {
		return nil, invalidMappingError("missing actor id")
	}
	if _, err := uuid.Parse(actorID); err != nil {
		return nil, invalidMappingError("invalid actor id")
	}

	return &alkemio.IdentityMapping{ActorID: actorID}, nil
}

func invalidMappingError(reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return errInvalidAlkemioMapping
	}
	return fmt.Errorf("%w: %s", errInvalidAlkemioMapping, reason)
}

// extractTokenClaims builds TokenClaims from Kratos identity data.
func extractTokenClaims(identity *kratosclient.Identity) *TokenClaims {
	if identity == nil {
		return &TokenClaims{}
	}

	traits, ok := identity.GetTraits().(map[string]interface{})
	if !ok {
		return &TokenClaims{}
	}

	claims := &TokenClaims{}

	// Extract name claims
	if givenName := extractGivenNameClaim(traits); givenName != "" {
		claims.GivenName = &givenName
	}
	if familyName := extractFamilyNameClaim(traits); familyName != "" {
		claims.FamilyName = &familyName
	}

	// Extract compliance claims
	if emailVerified := extractEmailVerifiedClaim(identity); emailVerified != nil {
		claims.EmailVerified = emailVerified
	}
	if acceptedTerms := extractAcceptedTermsClaim(traits); acceptedTerms != nil {
		claims.AcceptedTerms = acceptedTerms
	}

	return claims
}

// extractGivenNameClaim extracts and validates the given name from traits.name.first.
func extractGivenNameClaim(traits map[string]interface{}) string {
	return extractNameFromTraits(traits, "first")
}

// extractFamilyNameClaim extracts and validates the family name from traits.name.last.
func extractFamilyNameClaim(traits map[string]interface{}) string {
	return extractNameFromTraits(traits, "last")
}

// extractNameFromTraits is a helper function to extract and validate name fields consistently.
func extractNameFromTraits(traits map[string]interface{}, field string) string {
	if traits == nil {
		return ""
	}

	nameValue, ok := traits["name"]
	if !ok {
		return ""
	}

	nameMap, ok := nameValue.(map[string]any)
	if !ok {
		return ""
	}

	nameField := extractString(nameMap[field])
	if nameField == "" {
		return ""
	}

	// Validate UTF-8 and reasonable length
	if !isValidUTF8String(nameField) || len(nameField) > 255 {
		return ""
	}

	return nameField
}

// extractEmailVerifiedClaim determines email verification status from verifiable_addresses.
func extractEmailVerifiedClaim(identity *kratosclient.Identity) *bool {
	if identity == nil {
		return nil
	}

	addresses := identity.GetVerifiableAddresses()
	if len(addresses) == 0 {
		return nil
	}

	// Filter to email addresses only and check verification status
	hasEmailAddresses := false
	for _, addr := range addresses {
		// Only consider email addresses (skip SMS, etc.)
		if addr.GetVia() == "email" {
			hasEmailAddresses = true
			if addr.GetVerified() {
				verified := true
				return &verified
			}
		}
	}

	// If no email addresses found, omit the claim
	if !hasEmailAddresses {
		return nil
	}

	// If we have email addresses but none are verified
	verified := false
	return &verified
}

// extractAcceptedTermsClaim extracts the boolean terms acceptance from traits.accepted_terms.
func extractAcceptedTermsClaim(traits map[string]interface{}) *bool {
	if traits == nil {
		return nil
	}

	termsValue, ok := traits["accepted_terms"]
	if !ok {
		return nil
	}

	switch v := termsValue.(type) {
	case bool:
		return &v
	case string:
		// Handle string representations of boolean
		lower := strings.ToLower(strings.TrimSpace(v))
		switch lower {
		case "true", "1", "yes":
			result := true
			return &result
		case "false", "0", "no":
			result := false
			return &result
		}
	}

	return nil
}

// isValidUTF8String checks if a string is valid UTF-8 and contains printable characters.
func isValidUTF8String(s string) bool {
	if s == "" {
		return false
	}

	// Check if it's valid UTF-8
	if !utf8.ValidString(s) {
		return false
	}

	// Check if all runes are allowed
	for _, r := range s {
		if !isAllowedRune(r) {
			return false
		}
	}

	return true
}

// isAllowedRune checks if a rune is allowed in name fields.
func isAllowedRune(r rune) bool {
	// Allow Unicode letters for international names via unicode.IsLetter,
	// and keep existing allowances for digits and select punctuation.
	return unicode.IsLetter(r) ||
		(r >= '0' && r <= '9') ||
		r == ' ' || r == '-' || r == '\'' || r == '.'
}
