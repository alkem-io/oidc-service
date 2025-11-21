package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	hydraAdmin "github.com/ory/hydra-client-go/v2"
	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/alkemio"
	"github.com/alkem-io/oidc-service/internal/challenge"
	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/maintenance"
	"github.com/alkem-io/oidc-service/internal/server"
	testsupport "github.com/alkem-io/oidc-service/test/support"
)

const (
	testUserEmail        = "user@example.com"
	testUserDisplayName  = "Example User"
	errFmtNewService     = "new service: %v"
	errFmtExpectedStatus = "expected status %d, got %d"
	errFmtDecodeResponse = "decode response: %v"
	consentPathFormat    = "/v1/oidc/consent?consent_challenge=%s"
	challengeConsentID   = "integration-consent"
	challengeMissingID   = "integration-missing"
	challengeErrorID     = "integration-error"
	challengeTimeoutID   = "integration-timeout"
	challengeMaintID     = "integration-maintenance"
	challengeInvalidID   = "integration-invalid-agent"
)

var (
	testResolverFixture = struct {
		AuthenticationID string
		UserID           string
		AgentID          string
	}{
		AuthenticationID: "8c0b7f5a-4dce-4d13-b6ad-6b2df2c1d10c",
		UserID:           "1bcf5bd1-5f3e-4f01-9125-0edc93e5f5b1",
		AgentID:          "6f4ed2d2-0ad0-4b83-8d43-8d9b9b4970b3",
	}
)

func TestConsentEndpointAddsAlkemioUserIDClaim(t *testing.T) {
	consent := hydraAdmin.NewOAuth2ConsentRequest(challengeConsentID)
	consent.SetSubject(testResolverFixture.AuthenticationID)
	consent.SetContext(map[string]any{"identity_id": testResolverFixture.AuthenticationID})

	var (
		capturedAccess map[string]any
		capturedID     map[string]any
	)

	hydraStub := &testsupport.HydraClientStub{
		GetConsentFunc: func(_ context.Context, challengeID string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
			if challengeID != challengeConsentID {
				t.Fatalf("unexpected challenge id: %s", challengeID)
			}
			return consent, &http.Response{StatusCode: http.StatusOK}, nil
		},
		AcceptConsentFunc: func(_ context.Context, challengeID string, body *hydraAdmin.AcceptOAuth2ConsentRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
			session := body.GetSession()
			if claims, ok := session.GetAccessToken().(map[string]any); ok {
				capturedAccess = claims
			}
			if claims, ok := session.GetIdToken().(map[string]any); ok {
				capturedID = claims
			}
			return hydraAdmin.NewOAuth2RedirectTo("https://app.example/callback"), &http.Response{StatusCode: http.StatusOK}, nil
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
		ResolveFunc: func(_ context.Context, authenticationID string) (*alkemio.IdentityMapping, error) {
			return testsupport.NewAlkemioMapping(testResolverFixture.UserID, testResolverFixture.AgentID), nil
		},
	}

	svc, err := challenge.NewService(challenge.Options{Hydra: hydraStub, Identity: identityStub, Alkemio: resolverStub})
	if err != nil {
		t.Fatalf(errFmtNewService, err)
	}

	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
		Challenge:   svc,
	})

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf(consentPathFormat, challengeConsentID), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf(errFmtExpectedStatus, http.StatusFound, rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc == "" {
		t.Fatal("expected redirect location")
	}

	if capturedAccess["alkemio_user_id"] != testResolverFixture.UserID {
		t.Fatalf("access token missing alkemio_user_id: %v", capturedAccess)
	}
	if capturedAccess["agent_id"] != testResolverFixture.AgentID {
		t.Fatalf("access token missing agent_id: %v", capturedAccess)
	}
	if capturedID["alkemio_user_id"] != testResolverFixture.UserID {
		t.Fatalf("id token missing alkemio_user_id: %v", capturedID)
	}
	if capturedID["agent_id"] != testResolverFixture.AgentID {
		t.Fatalf("id token missing agent_id: %v", capturedID)
	}
}

func TestConsentEndpointFailsWhenIdentityMissing(t *testing.T) {
	consent := hydraAdmin.NewOAuth2ConsentRequest(challengeMissingID)
	consent.SetSubject(testResolverFixture.AuthenticationID)

	hydraStub := &testsupport.HydraClientStub{
		GetConsentFunc: func(_ context.Context, challengeID string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
			return consent, &http.Response{StatusCode: http.StatusOK}, nil
		},
		AcceptConsentFunc: func(_ context.Context, challengeID string, body *hydraAdmin.AcceptOAuth2ConsentRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
			t.Fatal("accept consent should not be invoked on failure")
			return nil, nil, nil
		},
	}

	identityStub := testsupport.IdentityFetcherStub{
		FetchFunc: func(_ context.Context, identityID string) (*challenge.IdentityProfile, error) {
			return &challenge.IdentityProfile{
				ID:          identityID,
				Email:       testUserEmail,
				DisplayName: testUserDisplayName,
			}, nil
		},
	}

	resolverStub := testsupport.AlkemioResolverStub{
		ResolveFunc: func(_ context.Context, authenticationID string) (*alkemio.IdentityMapping, error) {
			return nil, alkemio.ErrNotFound
		},
	}

	svc, err := challenge.NewService(challenge.Options{Hydra: hydraStub, Identity: identityStub, Alkemio: resolverStub})
	if err != nil {
		t.Fatalf(errFmtNewService, err)
	}

	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
		Challenge:   svc,
	})

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf(consentPathFormat, challengeMissingID), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf(errFmtExpectedStatus, http.StatusForbidden, rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf(errFmtDecodeResponse, err)
	}
	if payload["error"] != "alkemio_identity_missing" {
		t.Fatalf("unexpected error payload: %v", payload)
	}
}

func TestConsentEndpointFailsOnResolverError(t *testing.T) {
	consent := hydraAdmin.NewOAuth2ConsentRequest(challengeErrorID)
	consent.SetSubject(testResolverFixture.AuthenticationID)
	consent.SetContext(map[string]any{"identity_id": testResolverFixture.AuthenticationID})

	hydraStub := &testsupport.HydraClientStub{
		GetConsentFunc: func(_ context.Context, challengeID string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
			return consent, &http.Response{StatusCode: http.StatusOK}, nil
		},
		AcceptConsentFunc: func(context.Context, string, *hydraAdmin.AcceptOAuth2ConsentRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
			t.Fatal("accept consent should not be called on resolver error")
			return nil, nil, nil
		},
	}

	identityStub := testsupport.IdentityFetcherStub{
		FetchFunc: func(_ context.Context, identityID string) (*challenge.IdentityProfile, error) {
			return &challenge.IdentityProfile{ID: identityID, Email: testUserEmail}, nil
		},
	}

	var resolverCalls int
	resolverStub := testsupport.AlkemioResolverStub{
		ResolveFunc: func(_ context.Context, authenticationID string) (*alkemio.IdentityMapping, error) {
			resolverCalls++
			return nil, fmt.Errorf("alkemio server returned 503 for %s", authenticationID)
		},
	}

	svc, err := challenge.NewService(challenge.Options{Hydra: hydraStub, Identity: identityStub, Alkemio: resolverStub})
	if err != nil {
		t.Fatalf(errFmtNewService, err)
	}

	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
		Challenge:   svc,
	})

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf(consentPathFormat, challengeErrorID), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf(errFmtExpectedStatus, http.StatusBadGateway, rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf(errFmtDecodeResponse, err)
	}
	if payload["error"] != "alkemio_resolution_failed" {
		t.Fatalf("unexpected error response: %v", payload)
	}
	if resolverCalls != 1 {
		t.Fatalf("expected resolver to be called once, got %d", resolverCalls)
	}
}

func TestConsentEndpointFailsWhenAgentIDInvalid(t *testing.T) {
	consent := hydraAdmin.NewOAuth2ConsentRequest(challengeInvalidID)
	consent.SetSubject(testResolverFixture.AuthenticationID)
	consent.SetContext(map[string]any{"identity_id": testResolverFixture.AuthenticationID})

	hydraStub := &testsupport.HydraClientStub{
		GetConsentFunc: func(_ context.Context, challengeID string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
			return consent, &http.Response{StatusCode: http.StatusOK}, nil
		},
		AcceptConsentFunc: func(context.Context, string, *hydraAdmin.AcceptOAuth2ConsentRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
			t.Fatal("accept consent should not be called when mapping invalid")
			return nil, nil, nil
		},
	}

	identityStub := testsupport.IdentityFetcherStub{
		FetchFunc: func(_ context.Context, identityID string) (*challenge.IdentityProfile, error) {
			return &challenge.IdentityProfile{ID: identityID, Email: testUserEmail}, nil
		},
	}

	resolverStub := testsupport.AlkemioResolverStub{
		ResolveFunc: func(_ context.Context, authenticationID string) (*alkemio.IdentityMapping, error) {
			return testsupport.NewAlkemioMapping(testResolverFixture.UserID, "not-a-uuid"), nil
		},
	}

	svc, err := challenge.NewService(challenge.Options{Hydra: hydraStub, Identity: identityStub, Alkemio: resolverStub})
	if err != nil {
		t.Fatalf(errFmtNewService, err)
	}

	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
		Challenge:   svc,
	})

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf(consentPathFormat, challengeInvalidID), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf(errFmtExpectedStatus, http.StatusBadGateway, rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf(errFmtDecodeResponse, err)
	}
	if payload["error"] != "alkemio_resolution_failed" {
		t.Fatalf("unexpected error response: %v", payload)
	}
}

func TestConsentEndpointLogsResolverTimeout(t *testing.T) {
	consent := hydraAdmin.NewOAuth2ConsentRequest(challengeTimeoutID)
	consent.SetSubject(testResolverFixture.AuthenticationID)
	consent.SetContext(map[string]any{"identity_id": testResolverFixture.AuthenticationID})

	hydraStub := &testsupport.HydraClientStub{
		GetConsentFunc: func(_ context.Context, challengeID string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
			return consent, &http.Response{StatusCode: http.StatusOK}, nil
		},
		AcceptConsentFunc: func(context.Context, string, *hydraAdmin.AcceptOAuth2ConsentRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
			t.Fatal("accept consent should not be called on timeout")
			return nil, nil, nil
		},
	}

	identityStub := testsupport.IdentityFetcherStub{
		FetchFunc: func(_ context.Context, identityID string) (*challenge.IdentityProfile, error) {
			return &challenge.IdentityProfile{ID: identityID, Email: testUserEmail}, nil
		},
	}

	logger := &testsupport.LoggerStub{}
	resolverStub := testsupport.AlkemioResolverStub{
		ResolveFunc: func(_ context.Context, authenticationID string) (*alkemio.IdentityMapping, error) {
			return nil, context.DeadlineExceeded
		},
	}

	svc, err := challenge.NewService(challenge.Options{Hydra: hydraStub, Identity: identityStub, Alkemio: resolverStub, Logger: logger})
	if err != nil {
		t.Fatalf(errFmtNewService, err)
	}

	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
		Challenge:   svc,
	})

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf(consentPathFormat, challengeTimeoutID), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf(errFmtExpectedStatus, http.StatusBadGateway, rec.Code)
	}

	warnEntries := logger.EntriesByLevel("warn")
	if len(warnEntries) != 1 {
		t.Fatalf("expected 1 warn entry, got %d", len(warnEntries))
	}
	entry := warnEntries[0]
	if entry.Fields["error_type"] != "timeout" {
		t.Fatalf("expected timeout error_type, got %v", entry.Fields["error_type"])
	}
	if entry.Fields["identity_id"] == testResolverFixture.AuthenticationID {
		t.Fatalf("expected masked identity id, got %v", entry.Fields["identity_id"])
	}
}

func TestMaintenanceShortCircuitsBeforeResolver(t *testing.T) {
	maintState := maintenance.NewState(config.MaintenanceState{Enabled: true, Message: "maintenance"})

	hydraStub := &testsupport.HydraClientStub{
		GetConsentFunc: func(context.Context, string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
			t.Fatal("hydra should not be called during maintenance")
			return nil, nil, nil
		},
	}

	identityStub := testsupport.IdentityFetcherStub{
		FetchFunc: func(context.Context, string) (*challenge.IdentityProfile, error) {
			t.Fatal("identity fetch should not be called during maintenance")
			return nil, nil
		},
	}

	resolverStub := testsupport.AlkemioResolverStub{
		ResolveFunc: func(context.Context, string) (*alkemio.IdentityMapping, error) {
			t.Fatal("resolver should not be called during maintenance")
			return nil, nil
		},
	}

	svc, err := challenge.NewService(challenge.Options{Hydra: hydraStub, Identity: identityStub, Alkemio: resolverStub})
	if err != nil {
		t.Fatalf(errFmtNewService, err)
	}

	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintState,
		Challenge:   svc,
	})

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf(consentPathFormat, challengeMaintID), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf(errFmtExpectedStatus, http.StatusServiceUnavailable, rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf(errFmtDecodeResponse, err)
	}
	if payload["error"] != "maintenance_mode" {
		t.Fatalf("unexpected error payload: %v", payload)
	}
}
