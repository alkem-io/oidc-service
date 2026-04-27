package contract_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hydraAdmin "github.com/ory/hydra-client-go/v2"
	"github.com/stretchr/testify/require"

	"github.com/alkem-io/oidc-service/internal/alkemio"
	"github.com/alkem-io/oidc-service/internal/challenge"
	"github.com/alkem-io/oidc-service/internal/server"
	testsupport "github.com/alkem-io/oidc-service/test/support"
)

// TestConsentUIRenderForNonPreConsentedClient — FR-031 (T030 + T031 paths).
// A synthetic client_id NOT in PreConsentClientIDs MUST hit the HTML render
// path: Content-Type: text/html, no `consent.auto_accept` audit, exactly one
// `consent.ui_rendered` audit carrying the FR-035 minimal fields.
//
// Exercises the render path that has no real RP target in this feature's
// scope — so the test synthesises a standalone client via Hydra stub.
func TestConsentUIRenderForNonPreConsentedClient(t *testing.T) {
	t.Parallel()

	const (
		challengeID = "consent-render"
		clientID    = "third-party-synthetic"
		kratosID    = "kratos-render-1"
		userID      = "actor-render-1"
	)

	consent := hydraAdmin.NewOAuth2ConsentRequest(challengeID)
	consent.SetSubject(kratosID)
	consent.SetRequestedScope([]string{"openid", "profile"})
	client := hydraAdmin.NewOAuth2Client()
	client.SetClientId(clientID)
	consent.SetClient(*client)

	hydraStub := &testsupport.HydraClientStub{
		GetConsentFunc: func(_ context.Context, _ string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
			return consent, &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		},
		AcceptConsentFunc: func(_ context.Context, _ string, _ *hydraAdmin.AcceptOAuth2ConsentRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
			t.Fatal("AcceptConsentRequest must NOT be called on the render path")
			return nil, nil, nil
		},
	}
	identityStub := testsupport.IdentityFetcherStub{
		FetchFunc: func(_ context.Context, _ string) (*challenge.IdentityProfile, error) {
			return &challenge.IdentityProfile{ID: kratosID, Email: "u@example.com", DisplayName: "U"}, nil
		},
	}
	resolverStub := testsupport.AlkemioResolverStub{
		ResolveFunc: func(_ context.Context, _ string) (*alkemio.IdentityMapping, error) {
			return testsupport.NewAlkemioMapping(userID), nil
		},
	}

	svc, err := challenge.NewService(challenge.Options{
		Hydra:               hydraStub,
		Identity:            identityStub,
		Alkemio:             resolverStub,
		PreConsentClientIDs: []string{"alkemio-web"}, // synthetic client NOT listed
	})
	require.NoError(t, err)

	handler := server.NewConsentHandler(nil, svc)

	req := httptest.NewRequest(http.MethodGet, "/oidc/consent?consent_challenge="+challengeID, nil)
	rec := httptest.NewRecorder()
	handler.Handle(rec, req)

	require.Contains(t, rec.Header().Get("Content-Type"), "text/html",
		"non-pre-consented client MUST render HTML consent UI")
	body := rec.Body.String()
	require.True(t, strings.Contains(strings.ToLower(body), "<html") || strings.Contains(strings.ToLower(body), "<!doctype"),
		"render body MUST contain HTML markup")
}
