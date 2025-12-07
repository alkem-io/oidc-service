package contract_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	hydraAdmin "github.com/ory/hydra-client-go/v2"
	"github.com/stretchr/testify/require"

	"github.com/alkem-io/oidc-service/internal/alkemio"
	"github.com/alkem-io/oidc-service/internal/challenge"
	testsupport "github.com/alkem-io/oidc-service/test/support"
)

const (
	contractKratosID         = "a9b6d4c2-68c2-4c60-920e-0f2b77e9c123"
	contractUserID           = "cb658e61-0901-46b7-b1ce-91b6ec1345af"
	contractAgentID          = "2c1c9b1f-b6e5-4c42-bc6c-4a4c934e6712"
	contractConsentChallenge = "consent-test"
)

func TestConsentAddsAlkemioUserIDClaim(t *testing.T) {
	t.Parallel()

	consent := hydraAdmin.NewOAuth2ConsentRequest(contractConsentChallenge)
	consent.SetSubject(contractKratosID)
	consent.SetContext(map[string]any{"identity_id": contractKratosID})

	var (
		capturedAccess map[string]any
		capturedID     map[string]any
	)

	hydraStub := &testsupport.HydraClientStub{
		GetConsentFunc: func(_ context.Context, challengeID string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
			require.Equal(t, contractConsentChallenge, challengeID, "unexpected challenge id")
			return consent, &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		},
		AcceptConsentFunc: func(_ context.Context, _ string, body *hydraAdmin.AcceptOAuth2ConsentRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
			session := body.GetSession()
			if token := session.GetAccessToken(); token != nil {
				if claims, ok := token.(map[string]any); ok {
					capturedAccess = claims
				}
			}
			if token := session.GetIdToken(); token != nil {
				if claims, ok := token.(map[string]any); ok {
					capturedID = claims
				}
			}
			return hydraAdmin.NewOAuth2RedirectTo("https://app.example/callback"), &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		},
	}

	identityStub := testsupport.IdentityFetcherStub{
		FetchFunc: func(_ context.Context, identityID string) (*challenge.IdentityProfile, error) {
			require.Equal(t, contractKratosID, identityID, "unexpected identity id")
			return &challenge.IdentityProfile{
				ID:          identityID,
				Email:       "user@example.com",
				DisplayName: "Example User",
			}, nil
		},
	}

	resolverStub := testsupport.AlkemioResolverStub{
		ResolveFunc: func(_ context.Context, authenticationID string) (*alkemio.IdentityMapping, error) {
			require.Equal(t, contractKratosID, authenticationID, "unexpected authentication id")
			return testsupport.NewAlkemioMapping(contractUserID, contractAgentID), nil
		},
	}

	svc, err := challenge.NewService(challenge.Options{
		Hydra:    hydraStub,
		Identity: identityStub,
		Alkemio:  resolverStub,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if _, err := svc.ResolveConsent(context.Background(), "consent-test"); err != nil {
		t.Fatalf("resolve consent: %v", err)
	}

	require.NotNil(t, capturedAccess, "access token claims not captured")
	require.NotNil(t, capturedID, "id token claims not captured")
	requireStringClaim(t, capturedAccess, "alkemio_user_id", contractUserID)
	requireStringClaim(t, capturedAccess, "agent_id", contractAgentID)
	requireStringClaim(t, capturedID, "alkemio_user_id", contractUserID)
	requireStringClaim(t, capturedID, "agent_id", contractAgentID)
}

func TestConsentFailsWhenAlkemioIdentityMissing(t *testing.T) {
	t.Parallel()

	consent := hydraAdmin.NewOAuth2ConsentRequest("consent-missing")
	consent.SetSubject(contractKratosID)

	hydraStub := &testsupport.HydraClientStub{
		GetConsentFunc: func(_ context.Context, _ string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
			return consent, &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		},
		AcceptConsentFunc: func(_ context.Context, _ string, _ *hydraAdmin.AcceptOAuth2ConsentRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
			t.Fatal("accept consent should not be called when resolution fails")
			return nil, nil, nil
		},
	}

	identityStub := testsupport.IdentityFetcherStub{
		FetchFunc: func(_ context.Context, identityID string) (*challenge.IdentityProfile, error) {
			return &challenge.IdentityProfile{
				ID:          identityID,
				Email:       "user@example.com",
				DisplayName: "Example User",
			}, nil
		},
	}

	resolverStub := testsupport.AlkemioResolverStub{
		ResolveFunc: func(_ context.Context, _ string) (*alkemio.IdentityMapping, error) {
			return nil, alkemio.ErrNotFound
		},
	}

	svc, err := challenge.NewService(challenge.Options{
		Hydra:    hydraStub,
		Identity: identityStub,
		Alkemio:  resolverStub,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, err = svc.ResolveConsent(context.Background(), "consent-missing")
	if err == nil {
		t.Fatal("expected error when alkemio identity missing")
	}

	var challengeErr challenge.Error
	if !errors.As(err, &challengeErr) {
		t.Fatalf("expected challenge.Error, got %T", err)
	}
	if challengeErr.Code() != "alkemio_identity_missing" {
		t.Fatalf("unexpected error code: %s", challengeErr.Code())
	}
	if challengeErr.StatusCode() != http.StatusForbidden {
		t.Fatalf("unexpected status code: %d", challengeErr.StatusCode())
	}
}

func requireStringClaim(t *testing.T, claims map[string]any, key, expected string) {
	t.Helper()
	value, ok := claims[key].(string)
	require.True(t, ok, "claim %s missing", key)
	require.Equal(t, expected, value, "unexpected claim value for %s", key)
}
