package challenge

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"

	kratosclient "github.com/ory/client-go"
)

// IdentityProviderFunc fetches a Kratos identity and exposes the raw HTTP response for error inspection.
type IdentityProviderFunc func(ctx context.Context, identityID string) (*kratosclient.Identity, *http.Response, error)

// IdentityMapper retrieves identity records from Kratos and converts them into domain profiles.
type IdentityMapper struct {
	fetch IdentityProviderFunc
}

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

func (e *MissingTraitsError) Error() string {
	return fmt.Sprintf("missing required traits: %s", strings.Join(e.Traits, ", "))
}

// IdentityNotFoundError represents a 404 response from Kratos.
type IdentityNotFoundError struct {
	IdentityID string
}

func (e *IdentityNotFoundError) Error() string {
	return fmt.Sprintf("kratos identity %q not found", e.IdentityID)
}

// IdentityLookupError wraps unexpected failures reaching Kratos.
type IdentityLookupError struct {
	Err error
}

func (e *IdentityLookupError) Error() string {
	return fmt.Sprintf("kratos identity lookup failed: %v", e.Err)
}

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

	displayName := extractString(traits["display_name"])
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
