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
	contractConsentChallenge = "consent-test"
)

func TestConsentAddsAlkemioActorIDClaim(t *testing.T) {
	t.Parallel()

	consent := hydraAdmin.NewOAuth2ConsentRequest(contractConsentChallenge)
	consent.SetSubject(contractKratosID)
	consent.SetContext(map[string]any{"identity_id": contractKratosID})
	// FR-005 scope-gate (T029) — claim is emitted iff "alkemio" is in the
	// requested scope. This case requests the scope and expects the claim.
	consent.SetRequestedScope([]string{"openid", "profile", "email", "alkemio"})

	var (
		capturedAccess map[string]any
		capturedID     map[string]any
	)

	hydraStub := &testsupport.HydraClientStub{
		GetConsentFunc: func(_ context.Context, challengeID string) (
			*hydraAdmin.OAuth2ConsentRequest, *http.Response, error,
		) {
			require.Equal(t, contractConsentChallenge, challengeID, "unexpected challenge id")
			return consent, &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		},
		AcceptConsentFunc: func(
			_ context.Context, _ string, body *hydraAdmin.AcceptOAuth2ConsentRequest,
		) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
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
			return testsupport.NewAlkemioMapping(contractUserID), nil
		},
	}

	svc, err := challenge.NewService(
		challenge.Options{
			Hydra:    hydraStub,
			Identity: identityStub,
			Alkemio:  resolverStub,
		},
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if _, err := svc.ResolveConsent(context.Background(), "consent-test"); err != nil {
		t.Fatalf("resolve consent: %v", err)
	}

	require.NotNil(t, capturedAccess, "access token claims not captured")
	require.NotNil(t, capturedID, "id token claims not captured")
	requireStringClaim(t, capturedAccess, "alkemio_actor_id", contractUserID)
	requireStringClaim(t, capturedID, "alkemio_actor_id", contractUserID)
}

func TestConsentFailsWhenAlkemioIdentityMissing(t *testing.T) {
	t.Parallel()

	consent := hydraAdmin.NewOAuth2ConsentRequest("consent-missing")
	consent.SetSubject(contractKratosID)
	consent.SetRequestedScope([]string{"openid", "profile", "email", "offline_access", "alkemio"})

	hydraStub := &testsupport.HydraClientStub{
		GetConsentFunc: func(_ context.Context, _ string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
			return consent, &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		},
		AcceptConsentFunc: func(
			_ context.Context, _ string, _ *hydraAdmin.AcceptOAuth2ConsentRequest,
		) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
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

	svc, err := challenge.NewService(
		challenge.Options{
			Hydra:    hydraStub,
			Identity: identityStub,
			Alkemio:  resolverStub,
		},
	)
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

// TestConsentOmitsAlkemioClaimWhenScopeAbsent locks in FR-005 (T029): when the
// requested/granted scope list does NOT include "alkemio", neither the access
// token nor the ID token may carry `alkemio_actor_id`. Until T029 lands this
// test is RED because the impl attaches the claim unconditionally.
func TestConsentOmitsAlkemioClaimWhenScopeAbsent(t *testing.T) {
	t.Parallel()

	consent := hydraAdmin.NewOAuth2ConsentRequest("consent-no-alkemio")
	consent.SetSubject(contractKratosID)
	consent.SetContext(map[string]any{"identity_id": contractKratosID})
	consent.SetRequestedScope([]string{"openid", "profile", "email"}) // no "alkemio"

	var (
		capturedAccess map[string]any
		capturedID     map[string]any
	)

	hydraStub := &testsupport.HydraClientStub{
		GetConsentFunc: func(_ context.Context, challengeID string) (
			*hydraAdmin.OAuth2ConsentRequest, *http.Response, error,
		) {
			require.Equal(t, "consent-no-alkemio", challengeID)
			return consent, &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		},
		AcceptConsentFunc: func(
			_ context.Context, _ string, body *hydraAdmin.AcceptOAuth2ConsentRequest,
		) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
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
			return &challenge.IdentityProfile{
				ID:          identityID,
				Email:       "user@example.com",
				DisplayName: "Example User",
			}, nil
		},
	}

	resolverStub := testsupport.AlkemioResolverStub{
		ResolveFunc: func(_ context.Context, _ string) (*alkemio.IdentityMapping, error) {
			return testsupport.NewAlkemioMapping(contractUserID), nil
		},
	}

	svc, err := challenge.NewService(
		challenge.Options{
			Hydra:    hydraStub,
			Identity: identityStub,
			Alkemio:  resolverStub,
		},
	)
	require.NoError(t, err)

	_, err = svc.ResolveConsent(context.Background(), "consent-no-alkemio")
	require.NoError(t, err)

	// Both token sessions MAY be nil if the impl skips the entire claim block.
	// The invariant under test: NEITHER carries `alkemio_actor_id`.
	if capturedAccess != nil {
		_, present := capturedAccess["alkemio_actor_id"]
		require.False(t, present, "access token must not carry alkemio_actor_id when scope lacks it")
	}
	if capturedID != nil {
		_, present := capturedID["alkemio_actor_id"]
		require.False(t, present, "id token must not carry alkemio_actor_id when scope lacks it")
	}
}

func requireStringClaim(t *testing.T, claims map[string]any, key, expected string) {
	t.Helper()
	value, ok := claims[key].(string)
	require.True(t, ok, "claim %s missing", key)
	require.Equal(t, expected, value, "unexpected claim value for %s", key)
}
