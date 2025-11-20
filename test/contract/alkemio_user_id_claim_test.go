package contract_test

import (
	"context"
	"net/http"
	"testing"

	hydraAdmin "github.com/ory/hydra-client-go/v2"

	"github.com/alkem-io/oidc-service/internal/alkemio"
	"github.com/alkem-io/oidc-service/internal/challenge"
	testsupport "github.com/alkem-io/oidc-service/test/support"
)

func TestConsentAddsAlkemioUserIDClaim(t *testing.T) {
	t.Parallel()

	consent := hydraAdmin.NewOAuth2ConsentRequest("consent-test")
	consent.SetSubject("kratos-identity")
	consent.SetContext(map[string]any{"identity_id": "kratos-identity"})

	var (
		capturedAccess map[string]any
		capturedID     map[string]any
	)

	hydraStub := &testsupport.HydraClientStub{
		GetConsentFunc: func(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
			if challengeID != "consent-test" {
				t.Fatalf("unexpected challenge id: %s", challengeID)
			}
			return consent, &http.Response{StatusCode: http.StatusOK}, nil
		},
		AcceptConsentFunc: func(ctx context.Context, challengeID string, body *hydraAdmin.AcceptOAuth2ConsentRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
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
			return hydraAdmin.NewOAuth2RedirectTo("https://app.example/callback"), &http.Response{StatusCode: http.StatusOK}, nil
		},
	}

	identityStub := testsupport.IdentityFetcherStub{
		FetchFunc: func(ctx context.Context, identityID string) (*challenge.IdentityProfile, error) {
			if identityID != "kratos-identity" {
				t.Fatalf("unexpected identity id: %s", identityID)
			}
			return &challenge.IdentityProfile{
				ID:          identityID,
				Email:       "user@example.com",
				DisplayName: "Example User",
			}, nil
		},
	}

	resolverStub := testsupport.AlkemioResolverStub{
		ResolveFunc: func(ctx context.Context, authenticationID string) (string, error) {
			if authenticationID != "kratos-identity" {
				t.Fatalf("unexpected authentication id: %s", authenticationID)
			}
			return "alkemio-user-123", nil
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

	if capturedAccess == nil {
		t.Fatal("access token claims not captured")
	}
	if capturedID == nil {
		t.Fatal("id token claims not captured")
	}

	if claim, ok := capturedAccess["alkemio_user_id"].(string); !ok || claim != "alkemio-user-123" {
		t.Fatalf("access token missing alkemio_user_id claim: %v", capturedAccess)
	}
	if claim, ok := capturedID["alkemio_user_id"].(string); !ok || claim != "alkemio-user-123" {
		t.Fatalf("id token missing alkemio_user_id claim: %v", capturedID)
	}
}

func TestConsentFailsWhenAlkemioIdentityMissing(t *testing.T) {
	t.Parallel()

	consent := hydraAdmin.NewOAuth2ConsentRequest("consent-missing")
	consent.SetSubject("kratos-identity")

	hydraStub := &testsupport.HydraClientStub{
		GetConsentFunc: func(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
			return consent, &http.Response{StatusCode: http.StatusOK}, nil
		},
		AcceptConsentFunc: func(ctx context.Context, challengeID string, body *hydraAdmin.AcceptOAuth2ConsentRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
			t.Fatal("accept consent should not be called when resolution fails")
			return nil, nil, nil
		},
	}

	identityStub := testsupport.IdentityFetcherStub{
		FetchFunc: func(ctx context.Context, identityID string) (*challenge.IdentityProfile, error) {
			return &challenge.IdentityProfile{
				ID:          identityID,
				Email:       "user@example.com",
				DisplayName: "Example User",
			}, nil
		},
	}

	resolverStub := testsupport.AlkemioResolverStub{
		ResolveFunc: func(ctx context.Context, authenticationID string) (string, error) {
			return "", alkemio.ErrNotFound
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

	challengeErr, ok := err.(challenge.Error)
	if !ok {
		t.Fatalf("expected challenge.Error, got %T", err)
	}
	if challengeErr.Code() != "alkemio_identity_missing" {
		t.Fatalf("unexpected error code: %s", challengeErr.Code())
	}
	if challengeErr.StatusCode() != http.StatusForbidden {
		t.Fatalf("unexpected status code: %d", challengeErr.StatusCode())
	}
}
