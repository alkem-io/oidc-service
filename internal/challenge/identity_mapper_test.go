package challenge

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	kratosclient "github.com/ory/client-go"
)

func TestIdentityMapperFetchSuccess(t *testing.T) {
	identity := kratosclient.NewIdentity("identity-id", "default", "https://example.com/schema", map[string]any{
		"email":        "user@example.com",
		"display_name": "Example User",
	})
	identity.MetadataPublic = map[string]interface{}{
		"matrix_user_id": "@user:example.com",
	}

	mapper := NewIdentityMapperWithProvider(func(context.Context, string) (*kratosclient.Identity, *http.Response, error) {
		return identity, &http.Response{StatusCode: http.StatusOK}, nil
	})

	profile, err := mapper.Fetch(context.Background(), "identity-id")
	require.NoError(t, err)
	require.NotNil(t, profile)
	require.Equal(t, "identity-id", profile.ID)
	require.Equal(t, "user@example.com", profile.Email)
	require.Equal(t, "Example User", profile.DisplayName)
	require.Equal(t, "@user:example.com", profile.MatrixUserID)
	require.Contains(t, profile.Traits, "email")
	require.Contains(t, profile.Traits, "display_name")
	profile.Traits["email"] = "modified"
	require.Equal(t, "user@example.com", identity.GetTraits().(map[string]any)["email"]) // original map unchanged
}

func TestIdentityMapperMissingTraits(t *testing.T) {
	identity := kratosclient.NewIdentity("identity-id", "default", "https://example.com/schema", map[string]any{
		"display_name": "Example User",
	})

	mapper := NewIdentityMapperWithProvider(func(context.Context, string) (*kratosclient.Identity, *http.Response, error) {
		return identity, &http.Response{StatusCode: http.StatusOK}, nil
	})

	profile, err := mapper.Fetch(context.Background(), "identity-id")
	require.Nil(t, profile)
	var missingErr *MissingTraitsError
	require.ErrorAs(t, err, &missingErr)
	require.ElementsMatch(t, []string{"traits.email"}, missingErr.Traits)
}

func TestIdentityMapperInvalidEmail(t *testing.T) {
	identity := kratosclient.NewIdentity("identity-id", "default", "https://example.com/schema", map[string]any{
		"email":        "not-an-email",
		"display_name": "Example User",
	})

	mapper := NewIdentityMapperWithProvider(func(context.Context, string) (*kratosclient.Identity, *http.Response, error) {
		return identity, &http.Response{StatusCode: http.StatusOK}, nil
	})

	_, err := mapper.Fetch(context.Background(), "identity-id")
	var missingErr *MissingTraitsError
	require.ErrorAs(t, err, &missingErr)
	require.ElementsMatch(t, []string{"traits.email"}, missingErr.Traits)
}

func TestIdentityMapperIdentityNotFound(t *testing.T) {
	mapper := NewIdentityMapperWithProvider(func(context.Context, string) (*kratosclient.Identity, *http.Response, error) {
		return nil, &http.Response{StatusCode: http.StatusNotFound}, errors.New("not found")
	})

	_, err := mapper.Fetch(context.Background(), "missing-id")
	var notFoundErr *IdentityNotFoundError
	require.ErrorAs(t, err, &notFoundErr)
	require.Equal(t, "missing-id", notFoundErr.IdentityID)
}

func TestIdentityMapperLookupFailure(t *testing.T) {
	lookupErr := errors.New("boom")
	mapper := NewIdentityMapperWithProvider(func(context.Context, string) (*kratosclient.Identity, *http.Response, error) {
		return nil, &http.Response{StatusCode: http.StatusInternalServerError}, lookupErr
	})

	_, err := mapper.Fetch(context.Background(), "identity-id")
	var wrapped *IdentityLookupError
	require.ErrorAs(t, err, &wrapped)
	require.ErrorIs(t, wrapped, lookupErr)
}
