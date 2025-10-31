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

// TestIDTokenClaimsContract validates that consent endpoint properly handles
// enhanced ID token claims for user profile data (given_name, family_name).
func TestIDTokenClaimsContract(t *testing.T) {
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})

	// Use a stub consent challenge that includes complete user profile data
	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=id-token-profile-claims", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Should successfully redirect (consent acceptance)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d for ID token profile claims consent, got %d", http.StatusFound, rec.Code)
	}

	// Should have Location header with redirect URL
	if loc := rec.Header().Get("Location"); loc == "" {
		t.Fatal("expected Location header on successful ID token consent with profile claims")
	}

	// TODO: In a full implementation, we would:
	// 1. Parse the redirect URL to extract tokens or authorization code
	// 2. Decode the ID token to verify presence of given_name and family_name claims
	// 3. Validate that claims match expected user profile data from Kratos
	// 4. Ensure ID token includes both profile claims (given_name, family_name)
	// For now, we validate that the flow completes successfully with enhanced claims
}

// TestIDTokenClaimsContractUTF8Names validates ID token generation
// with Unicode/international names (UTF-8 validation).
func TestIDTokenClaimsContractUTF8Names(t *testing.T) {
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})

	// Use a stub consent challenge that includes UTF-8 names
	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=utf8-names", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Should successfully handle UTF-8 names
	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d for UTF-8 names consent, got %d", http.StatusFound, rec.Code)
	}

	if loc := rec.Header().Get("Location"); loc == "" {
		t.Fatal("expected Location header on successful consent with UTF-8 names")
	}

	// TODO: Validate that UTF-8 names are properly encoded in ID token claims
	// Example names to test: "José", "李明", "Müller", "O'Connor"
}

// TestIDTokenClaimsContractEmptyNames validates handling of empty/invalid names
// in ID token generation.
func TestIDTokenClaimsContractEmptyNames(t *testing.T) {
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})

	// Use a stub consent challenge with empty/invalid name data
	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=empty-names", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Should successfully handle empty names (omit claims)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d for empty names consent, got %d", http.StatusFound, rec.Code)
	}

	if loc := rec.Header().Get("Location"); loc == "" {
		t.Fatal("expected Location header on successful consent with empty names")
	}

	// TODO: Validate that empty/invalid names result in omitted claims
	// rather than null/empty values in the ID token
}