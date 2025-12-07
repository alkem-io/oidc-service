package challenge

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	kratosclient "github.com/ory/client-go"

	"github.com/alkem-io/oidc-service/internal/alkemio"
)

const (
	testIdentityID  = "identity-id"
	testSchemaURL   = "https://example.com/schema"
	testEmail       = "user@example.com"
	testDisplayName = "Example User"
)

func TestIdentityMapperFetchSuccess(t *testing.T) {
	identity := kratosclient.NewIdentity(testIdentityID, "default", testSchemaURL, map[string]any{
		"email":        testEmail,
		"display_name": testDisplayName,
	})
	identity.MetadataPublic = map[string]interface{}{
		"matrix_user_id": "@user:example.com",
	}

	mapper := NewIdentityMapperWithProvider(func(context.Context, string) (*kratosclient.Identity, *http.Response, error) {
		return identity, &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})

	profile, err := mapper.Fetch(context.Background(), testIdentityID)
	require.NoError(t, err)
	require.NotNil(t, profile)
	require.Equal(t, testIdentityID, profile.ID)
	require.Equal(t, testEmail, profile.Email)
	require.Equal(t, testDisplayName, profile.DisplayName)
	require.Equal(t, "@user:example.com", profile.MatrixUserID)
	require.Contains(t, profile.Traits, "email")
	require.Contains(t, profile.Traits, "display_name")
	profile.Traits["email"] = "modified"
	require.Equal(t, testEmail, identity.GetTraits().(map[string]any)["email"]) // original map unchanged
}

func TestIdentityMapperMissingTraits(t *testing.T) {
	identity := kratosclient.NewIdentity(testIdentityID, "default", testSchemaURL, map[string]any{
		"display_name": testDisplayName,
	})

	mapper := NewIdentityMapperWithProvider(func(context.Context, string) (*kratosclient.Identity, *http.Response, error) {
		return identity, &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})

	profile, err := mapper.Fetch(context.Background(), testIdentityID)
	require.Nil(t, profile)
	var missingErr *MissingTraitsError
	require.ErrorAs(t, err, &missingErr)
	require.ElementsMatch(t, []string{"traits.email"}, missingErr.Traits)
}

func TestIdentityMapperDisplayNameFallbackFromName(t *testing.T) {
	identity := kratosclient.NewIdentity(testIdentityID, "default", testSchemaURL, map[string]any{
		"email": testEmail,
		"name": map[string]any{
			"first": "Example",
			"last":  "User",
		},
	})

	mapper := NewIdentityMapperWithProvider(func(context.Context, string) (*kratosclient.Identity, *http.Response, error) {
		return identity, &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})

	profile, err := mapper.Fetch(context.Background(), testIdentityID)
	require.NoError(t, err)
	require.Equal(t, "Example User", profile.DisplayName)
}

func TestIdentityMapperInvalidEmail(t *testing.T) {
	identity := kratosclient.NewIdentity(testIdentityID, "default", testSchemaURL, map[string]any{
		"email":        "not-an-email",
		"display_name": testDisplayName,
	})

	mapper := NewIdentityMapperWithProvider(func(context.Context, string) (*kratosclient.Identity, *http.Response, error) {
		return identity, &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})

	_, err := mapper.Fetch(context.Background(), testIdentityID)
	var missingErr *MissingTraitsError
	require.ErrorAs(t, err, &missingErr)
	require.ElementsMatch(t, []string{"traits.email"}, missingErr.Traits)
}

func TestIdentityMapperIdentityNotFound(t *testing.T) {
	mapper := NewIdentityMapperWithProvider(func(context.Context, string) (*kratosclient.Identity, *http.Response, error) {
		return nil, &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody}, errors.New("not found")
	})

	_, err := mapper.Fetch(context.Background(), "missing-id")
	var notFoundErr *IdentityNotFoundError
	require.ErrorAs(t, err, &notFoundErr)
	require.Equal(t, "missing-id", notFoundErr.IdentityID)
}

func TestIdentityMapperLookupFailure(t *testing.T) {
	lookupErr := errors.New("boom")
	mapper := NewIdentityMapperWithProvider(func(context.Context, string) (*kratosclient.Identity, *http.Response, error) {
		return nil, &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}, lookupErr
	})

	_, err := mapper.Fetch(context.Background(), testIdentityID)
	var wrapped *IdentityLookupError
	require.ErrorAs(t, err, &wrapped)
	require.ErrorIs(t, wrapped, lookupErr)
}

func TestValidateAlkemioMappingSuccess(t *testing.T) {
	userID := uuid.NewString()
	agentID := uuid.NewString()
	mapping := &alkemio.IdentityMapping{
		UserID:  "  " + userID + "  ",
		AgentID: "\t" + agentID,
	}

	validated, err := validateAlkemioMapping(mapping)
	require.NoError(t, err)
	require.NotSame(t, mapping, validated)
	require.Equal(t, userID, validated.UserID)
	require.Equal(t, agentID, validated.AgentID)
}

func TestValidateAlkemioMappingMissingAgent(t *testing.T) {
	mapping := &alkemio.IdentityMapping{UserID: uuid.NewString()}

	_, err := validateAlkemioMapping(mapping)
	require.Error(t, err)
	require.ErrorContains(t, err, "missing agent id")
}

func TestValidateAlkemioMappingInvalidAgent(t *testing.T) {
	mapping := &alkemio.IdentityMapping{UserID: uuid.NewString(), AgentID: "not-a-uuid"}

	_, err := validateAlkemioMapping(mapping)
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid agent id")
}

func TestValidateAlkemioMappingMissingUser(t *testing.T) {
	mapping := &alkemio.IdentityMapping{AgentID: uuid.NewString()}

	_, err := validateAlkemioMapping(mapping)
	require.Error(t, err)
	require.ErrorContains(t, err, "missing user id")
}

func TestValidateAlkemioMappingInvalidUser(t *testing.T) {
	mapping := &alkemio.IdentityMapping{UserID: strings.Repeat("a", 5), AgentID: uuid.NewString()}

	_, err := validateAlkemioMapping(mapping)
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid user id")
}

func TestValidateAlkemioMappingMissingPayload(t *testing.T) {
	_, err := validateAlkemioMapping(nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "mapping missing")
}
