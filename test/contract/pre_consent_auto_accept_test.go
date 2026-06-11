package contract_test

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	hydraAdmin "github.com/ory/hydra-client-go/v2"
	"github.com/stretchr/testify/require"

	"github.com/alkem-io/oidc-service/internal/alkemio"
	"github.com/alkem-io/oidc-service/internal/audit"
	"github.com/alkem-io/oidc-service/internal/challenge"
	testsupport "github.com/alkem-io/oidc-service/test/support"
)

// TestPreConsentAutoAcceptForAllowListedClient — FR-030 (T030).
// When the consent request's client_id is listed in PreConsentClientIDs, the
// service MUST auto-accept without rendering any UI: GrantScope ==
// RequestedScope and AcceptConsent called exactly once. Identity IS still
// resolved so the consent session carries alkemio_actor_id/email claims —
// without it the BFF cookie session lacks alkemio_actor_id and downstream
// authorization treats the user as anonymous (FR-024a).
func TestPreConsentAutoAcceptForAllowListedClient(t *testing.T) {
	t.Parallel()

	const (
		clientID    = "alkemio-web"
		challengeID = "consent-pre-accept"
		userID      = "11111111-2222-4aaa-bbbb-cccccccccccc"
		kratosID    = "22222222-3333-4aaa-bbbb-dddddddddddd"
	)

	consent := hydraAdmin.NewOAuth2ConsentRequest(challengeID)
	consent.SetSubject(kratosID)
	consent.SetRequestedScope([]string{"openid", "profile", "email", "offline_access", "alkemio"})
	client := hydraAdmin.NewOAuth2Client()
	client.SetClientId(clientID)
	consent.SetClient(*client)

	var acceptedPayload *hydraAdmin.AcceptOAuth2ConsentRequest
	identityFetched := false

	hydraStub := &testsupport.HydraClientStub{
		GetConsentFunc: func(_ context.Context, _ string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
			return consent, &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		},
		AcceptConsentFunc: func(_ context.Context, _ string, body *hydraAdmin.AcceptOAuth2ConsentRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
			acceptedPayload = body
			return hydraAdmin.NewOAuth2RedirectTo("https://app.example/callback"), &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		},
	}

	identityStub := testsupport.IdentityFetcherStub{
		FetchFunc: func(_ context.Context, _ string) (*challenge.IdentityProfile, error) {
			identityFetched = true
			return &challenge.IdentityProfile{ID: kratosID, Email: "u@example.com", DisplayName: "U"}, nil
		},
	}

	resolverStub := testsupport.AlkemioResolverStub{
		ResolveFunc: func(_ context.Context, _ string) (*alkemio.IdentityMapping, error) {
			return testsupport.NewAlkemioMapping(userID), nil
		},
	}

	var auditLog bytes.Buffer
	svc, err := challenge.NewService(challenge.Options{
		Hydra:               hydraStub,
		Identity:            identityStub,
		Alkemio:             resolverStub,
		PreConsentClientIDs: []string{clientID},
		Audit:               audit.NewEmitter(&auditLog),
	})
	require.NoError(t, err)

	_, err = svc.ResolveConsent(context.Background(), challengeID)
	require.NoError(t, err)

	require.NotNil(t, acceptedPayload, "AcceptConsentRequest must be called exactly once")
	// FR-030 — GrantScope MUST equal RequestedScope on pre-consent auto-accept.
	require.Equal(t, consent.GetRequestedScope(), acceptedPayload.GetGrantScope())
	require.True(t, identityFetched, "pre-consent MUST still resolve identity to attach token claims")
	// The auto_accept audit event is what distinguishes the pre-consent
	// short-circuit from the normal consent path (both accept in Hydra).
	require.Contains(t, auditLog.String(), `"event_type":"consent.auto_accept"`,
		"allow-listed client MUST take the pre-consent auto-accept path")
	// FR-024a — the alkemio scope is requested, so the consent session MUST
	// carry alkemio_actor_id in both token claim maps.
	session := acceptedPayload.GetSession()
	idClaims, _ := session.GetIdToken().(map[string]any)
	require.Equal(t, userID, idClaims["alkemio_actor_id"])
	accessClaims, _ := session.GetAccessToken().(map[string]any)
	require.Equal(t, userID, accessClaims["alkemio_actor_id"])
}

// TestPreConsentDoesNotApplyToUnlistedClient — a client NOT in
// PreConsentClientIDs MUST fall through to the normal consent-resolution path.
// (Non-pre-consented render behaviour is covered by T027a at the handler level.)
func TestPreConsentDoesNotApplyToUnlistedClient(t *testing.T) {
	t.Parallel()

	consent := hydraAdmin.NewOAuth2ConsentRequest("consent-other-client")
	consent.SetSubject("kratos-2002")
	consent.SetRequestedScope([]string{"openid", "profile"})
	client := hydraAdmin.NewOAuth2Client()
	client.SetClientId("third-party")
	consent.SetClient(*client)

	identityFetched := false
	hydraStub := &testsupport.HydraClientStub{
		GetConsentFunc: func(_ context.Context, _ string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
			return consent, &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		},
		AcceptConsentFunc: func(_ context.Context, _ string, _ *hydraAdmin.AcceptOAuth2ConsentRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
			return hydraAdmin.NewOAuth2RedirectTo("https://example/callback"), &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		},
	}
	identityStub := testsupport.IdentityFetcherStub{
		FetchFunc: func(_ context.Context, _ string) (*challenge.IdentityProfile, error) {
			identityFetched = true
			return &challenge.IdentityProfile{ID: "kratos-2002", Email: "u@ex", DisplayName: "U"}, nil
		},
	}
	resolverStub := testsupport.AlkemioResolverStub{
		ResolveFunc: func(_ context.Context, _ string) (*alkemio.IdentityMapping, error) {
			return testsupport.NewAlkemioMapping("33333333-4444-4aaa-bbbb-eeeeeeeeeeee"), nil
		},
	}

	var auditLog bytes.Buffer
	svc, err := challenge.NewService(challenge.Options{
		Hydra:               hydraStub,
		Identity:            identityStub,
		Alkemio:             resolverStub,
		PreConsentClientIDs: []string{"alkemio-web"}, // other client only; "third-party" not listed
		Audit:               audit.NewEmitter(&auditLog),
	})
	require.NoError(t, err)

	_, err = svc.ResolveConsent(context.Background(), "consent-other-client")
	require.NoError(t, err)

	require.True(t, identityFetched, "unlisted client MUST fall through and fetch identity")
	// Both paths accept in Hydra; the absence of the auto_accept audit event
	// proves the allow-list gate routed this client through the normal path.
	require.NotContains(t, auditLog.String(), `"event_type":"consent.auto_accept"`,
		"unlisted client MUST NOT take the pre-consent auto-accept path")
}
