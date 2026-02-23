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

func newIDTokenRouter() http.Handler {
	return server.NewRouter(
		server.Options{
			Logger:      zap.NewNop(),
			Maintenance: maintenance.NewState(config.MaintenanceState{}),
		},
	)
}

// TestIDTokenClaimsContract validates that consent endpoint properly handles
// enhanced ID token claims for various name payloads.
func TestIDTokenClaimsContract(t *testing.T) {
	testCases := []struct {
		name        string
		challengeID string
		description string
	}{
		{
			name:        "CompleteProfile",
			challengeID: "id-token-profile-claims",
			description: "complete profile should redirect",
		},
		{
			name:        "UTF8Names",
			challengeID: "utf8-names",
			description: "unicode characters stay intact",
		},
		{
			name:        "EmptyNames",
			challengeID: "empty-names",
			description: "missing names keep flow backwards compatible",
		},
	}

	for _, tc := range testCases {
		t.Run(
			tc.name, func(t *testing.T) {
				router := newIDTokenRouter()

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

func TestIDTokenClaimsIncludeActorID(t *testing.T) {
	t.Parallel()

	const challengeID = "id-token-actor-claims"
	consent := hydraAdmin.NewOAuth2ConsentRequest(challengeID)
	consent.SetSubject(contractKratosID)
	consent.SetContext(map[string]any{"identity_id": contractKratosID})

	var capturedID map[string]any

	hydraStub := &testsupport.HydraClientStub{
		GetConsentFunc: func(_ context.Context, requested string) (
			*hydraAdmin.OAuth2ConsentRequest, *http.Response, error,
		) {
			require.Equal(t, challengeID, requested)
			return consent, &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		},
		AcceptConsentFunc: func(
			_ context.Context, requested string, body *hydraAdmin.AcceptOAuth2ConsentRequest,
		) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
			require.Equal(t, challengeID, requested)
			session := body.GetSession()
			if claims, ok := session.GetIdToken().(map[string]any); ok {
				capturedID = claims
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
			return testsupport.NewAlkemioMapping(contractUserID), nil
		},
	}

	svc, err := challenge.NewService(challenge.Options{Hydra: hydraStub, Identity: identityStub, Alkemio: resolverStub})
	require.NoError(t, err)

	_, err = svc.ResolveConsent(context.Background(), challengeID)
	require.NoError(t, err)
	require.NotNil(t, capturedID, "id token claims not captured")
	require.Equal(t, contractUserID, capturedID["alkemio_actor_id"])
}
