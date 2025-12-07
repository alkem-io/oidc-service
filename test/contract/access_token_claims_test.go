package contract_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	hydraAdmin "github.com/ory/hydra-client-go/v2"
	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/alkemio"
	"github.com/alkem-io/oidc-service/internal/challenge"
	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/maintenance"
	"github.com/alkem-io/oidc-service/internal/server"
	testsupport "github.com/alkem-io/oidc-service/test/support"
)

// TestAccessTokenClaimsContract validates that the consent handler routes
// profile-claim scenarios through the Hydra stub and preserves challenge IDs.
func TestAccessTokenClaimsContract(t *testing.T) {
	router := server.NewRouter(
		server.Options{
			Logger:      zap.NewNop(),
			Maintenance: maintenance.NewState(config.MaintenanceState{}),
		},
	)

	testCases := []struct {
		name        string
		challengeID string
		description string
	}{
		{
			name:        "CompleteProfile",
			challengeID: "profile-claims",
			description: "full profile data should redirect via stub",
		},
		{
			name:        "PartialProfile",
			challengeID: "partial-profile",
			description: "partial profile data still succeeds",
		},
		{
			name:        "NoProfile",
			challengeID: "no-profile",
			description: "absence of profile data keeps flow compatible",
		},
	}

	for _, tc := range testCases {
		t.Run(
			tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge="+tc.challengeID, nil)
				rec := httptest.NewRecorder()

				router.ServeHTTP(rec, req)

				if rec.Code != http.StatusFound {
					t.Fatalf("%s: expected status %d, got %d", tc.description, http.StatusFound, rec.Code)
				}

				testsupport.AssertStubRedirect(t, rec.Header().Get("Location"), "consent", tc.challengeID)
			},
		)
	}
}

func TestAccessTokenClaimsIncludeAgentID(t *testing.T) {
	t.Parallel()

	const challengeID = "access-token-agent-claims"
	consent := hydraAdmin.NewOAuth2ConsentRequest(challengeID)
	consent.SetSubject(contractKratosID)
	consent.SetContext(map[string]any{"identity_id": contractKratosID})

	var capturedAccess map[string]any

	hydraStub := &testsupport.HydraClientStub{
		GetConsentFunc: func(_ context.Context, requested string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
			require.Equal(t, challengeID, requested)
			return consent, &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		},
		AcceptConsentFunc: func(_ context.Context, requested string, body *hydraAdmin.AcceptOAuth2ConsentRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
			require.Equal(t, challengeID, requested)
			session := body.GetSession()
			if claims, ok := session.GetAccessToken().(map[string]any); ok {
				capturedAccess = claims
			}
			return hydraAdmin.NewOAuth2RedirectTo("https://app.example/callback"), &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		},
	}

	identityStub := testsupport.IdentityFetcherStub{
		FetchFunc: func(_ context.Context, identityID string) (*challenge.IdentityProfile, error) {
			require.Equal(t, contractKratosID, identityID)
			return &challenge.IdentityProfile{ID: identityID, Email: "user@example.com"}, nil
		},
	}

	resolverStub := testsupport.AlkemioResolverStub{
		ResolveFunc: func(_ context.Context, authenticationID string) (*alkemio.IdentityMapping, error) {
			require.Equal(t, contractKratosID, authenticationID)
			return testsupport.NewAlkemioMapping(contractUserID, contractAgentID), nil
		},
	}

	svc, err := challenge.NewService(challenge.Options{Hydra: hydraStub, Identity: identityStub, Alkemio: resolverStub})
	require.NoError(t, err)

	_, err = svc.ResolveConsent(context.Background(), challengeID)
	require.NoError(t, err)
	require.NotNil(t, capturedAccess, "access token claims not captured")
	require.Equal(t, contractAgentID, capturedAccess["agent_id"])
}
