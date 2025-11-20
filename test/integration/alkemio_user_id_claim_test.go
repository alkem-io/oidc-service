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

func TestConsentEndpointAddsAlkemioUserIDClaim(t *testing.T) {
	consent := hydraAdmin.NewOAuth2ConsentRequest("integration-consent")
	consent.SetSubject("kratos-identity")
	consent.SetContext(map[string]any{"identity_id": "kratos-identity"})

	var (
		capturedAccess map[string]any
		capturedID     map[string]any
	)

	hydraStub := &testsupport.HydraClientStub{
		GetConsentFunc: func(_ context.Context, challengeID string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
			if challengeID != "integration-consent" {
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
		ResolveFunc: func(_ context.Context, authenticationID string) (string, error) {
			return "alkemio-user-123", nil
		},
	}

	svc, err := challenge.NewService(challenge.Options{Hydra: hydraStub, Identity: identityStub, Alkemio: resolverStub})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
		Challenge:   svc,
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=integration-consent", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc == "" {
		t.Fatal("expected redirect location")
	}

	if capturedAccess["alkemio_user_id"] != "alkemio-user-123" {
		t.Fatalf("access token missing alkemio_user_id: %v", capturedAccess)
	}
	if capturedID["alkemio_user_id"] != "alkemio-user-123" {
		t.Fatalf("id token missing alkemio_user_id: %v", capturedID)
	}
}

func TestConsentEndpointFailsWhenIdentityMissing(t *testing.T) {
	consent := hydraAdmin.NewOAuth2ConsentRequest("integration-missing")
	consent.SetSubject("kratos-identity")

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
				Email:       "user@example.com",
				DisplayName: "Example User",
			}, nil
		},
	}

	resolverStub := testsupport.AlkemioResolverStub{
		ResolveFunc: func(_ context.Context, authenticationID string) (string, error) {
			return "", alkemio.ErrNotFound
		},
	}

	svc, err := challenge.NewService(challenge.Options{Hydra: hydraStub, Identity: identityStub, Alkemio: resolverStub})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
		Challenge:   svc,
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=integration-missing", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["error"] != "alkemio_identity_missing" {
		t.Fatalf("unexpected error payload: %v", payload)
	}
}

func TestConsentEndpointFailsOnResolverError(t *testing.T) {
	consent := hydraAdmin.NewOAuth2ConsentRequest("integration-error")
	consent.SetSubject("kratos-identity")
	consent.SetContext(map[string]any{"identity_id": "kratos-identity"})

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
			return &challenge.IdentityProfile{ID: identityID, Email: "user@example.com"}, nil
		},
	}

	var resolverCalls int
	resolverStub := testsupport.AlkemioResolverStub{
		ResolveFunc: func(_ context.Context, authenticationID string) (string, error) {
			resolverCalls++
			return "", fmt.Errorf("alkemio server returned 503 for %s", authenticationID)
		},
	}

	svc, err := challenge.NewService(challenge.Options{Hydra: hydraStub, Identity: identityStub, Alkemio: resolverStub})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
		Challenge:   svc,
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=integration-error", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["error"] != "alkemio_resolution_failed" {
		t.Fatalf("unexpected error response: %v", payload)
	}
	if resolverCalls != 1 {
		t.Fatalf("expected resolver to be called once, got %d", resolverCalls)
	}
}

func TestConsentEndpointLogsResolverTimeout(t *testing.T) {
	consent := hydraAdmin.NewOAuth2ConsentRequest("integration-timeout")
	consent.SetSubject("kratos-identity-123456")
	consent.SetContext(map[string]any{"identity_id": "kratos-identity-123456"})

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
			return &challenge.IdentityProfile{ID: identityID, Email: "user@example.com"}, nil
		},
	}

	logger := &testsupport.LoggerStub{}
	resolverStub := testsupport.AlkemioResolverStub{
		ResolveFunc: func(_ context.Context, authenticationID string) (string, error) {
			return "", context.DeadlineExceeded
		},
	}

	svc, err := challenge.NewService(challenge.Options{Hydra: hydraStub, Identity: identityStub, Alkemio: resolverStub, Logger: logger})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
		Challenge:   svc,
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=integration-timeout", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, rec.Code)
	}

	warnEntries := logger.EntriesByLevel("warn")
	if len(warnEntries) != 1 {
		t.Fatalf("expected 1 warn entry, got %d", len(warnEntries))
	}
	entry := warnEntries[0]
	if entry.Fields["error_type"] != "timeout" {
		t.Fatalf("expected timeout error_type, got %v", entry.Fields["error_type"])
	}
	if entry.Fields["identity_id"] == "kratos-identity-123456" {
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
		ResolveFunc: func(context.Context, string) (string, error) {
			t.Fatal("resolver should not be called during maintenance")
			return "", nil
		},
	}

	svc, err := challenge.NewService(challenge.Options{Hydra: hydraStub, Identity: identityStub, Alkemio: resolverStub})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintState,
		Challenge:   svc,
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=integration-maintenance", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["error"] != "maintenance_mode" {
		t.Fatalf("unexpected error payload: %v", payload)
	}
}
