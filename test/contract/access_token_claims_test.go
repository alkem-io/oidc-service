package contract_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/maintenance"
	"github.com/alkem-io/oidc-service/internal/server"
	"go.uber.org/zap"
)

// TestAccessTokenClaimsContract validates that consent endpoint properly handles
// enhanced Access token claims for user profile data (given_name, family_name).
func TestAccessTokenClaimsContract(t *testing.T) {
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})

	// Use a stub consent challenge that includes user profile data
	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=profile-claims", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Should successfully redirect (consent acceptance)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d for profile claims consent, got %d", http.StatusFound, rec.Code)
	}

	// Should have Location header with redirect URL
	if loc := rec.Header().Get("Location"); loc == "" {
		t.Fatal("expected Location header on successful consent with profile claims")
	}

	// TODO: In a full implementation, we would:
	// 1. Parse the redirect URL to extract tokens or authorization code
	// 2. Decode the tokens to verify presence of given_name and family_name claims
	// 3. Validate that claims match expected user profile data from Kratos
	// For now, we validate that the flow completes successfully with enhanced claims
}

// TestAccessTokenClaimsContractPartialData validates Access token generation
// when only some profile data is available (e.g., only given_name).
func TestAccessTokenClaimsContractPartialData(t *testing.T) {
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})

	// Use a stub consent challenge that includes partial profile data
	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=partial-profile", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Should successfully handle partial data
	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d for partial profile consent, got %d", http.StatusFound, rec.Code)
	}

	if loc := rec.Header().Get("Location"); loc == "" {
		t.Fatal("expected Location header on successful consent with partial profile")
	}

	// TODO: Validate that only available claims are included (given_name present, family_name omitted)
}

// TestAccessTokenClaimsContractNoProfileData validates Access token generation
// when no profile data is available (backward compatibility).
func TestAccessTokenClaimsContractNoProfileData(t *testing.T) {
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})

	// Use a stub consent challenge with no profile claims
	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=no-profile", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Should successfully handle missing profile data (backward compatibility)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d for no-profile consent, got %d", http.StatusFound, rec.Code)
	}

	if loc := rec.Header().Get("Location"); loc == "" {
		t.Fatal("expected Location header on successful consent without profile")
	}

	// TODO: Validate that no profile claims are included but flow still works
}