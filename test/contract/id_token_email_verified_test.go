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

// TestIDTokenEmailVerifiedClaimContract validates that consent endpoint properly handles
// email_verified claim in ID tokens based on Kratos verifiable_addresses.
func TestIDTokenEmailVerifiedClaimContract(t *testing.T) {
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})

	// Use a stub consent challenge with verified email address
	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=email-verified", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Should successfully redirect (consent acceptance)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d for email verified consent, got %d", http.StatusFound, rec.Code)
	}

	// Should have Location header with redirect URL
	if loc := rec.Header().Get("Location"); loc == "" {
		t.Fatal("expected Location header on successful consent with email verified")
	}

	// TODO: In a full implementation, we would:
	// 1. Parse the redirect URL to extract tokens or authorization code
	// 2. Decode the ID token to verify presence of email_verified: true claim
	// 3. Validate that claim matches expected verification status from Kratos
	// For now, we validate that the flow completes successfully
}

// TestIDTokenEmailUnverifiedClaimContract validates handling of unverified email addresses.
func TestIDTokenEmailUnverifiedClaimContract(t *testing.T) {
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})

	// Use a stub consent challenge with unverified email address
	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=email-unverified", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Should successfully handle unverified email
	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d for email unverified consent, got %d", http.StatusFound, rec.Code)
	}

	if loc := rec.Header().Get("Location"); loc == "" {
		t.Fatal("expected Location header on successful consent with email unverified")
	}

	// TODO: Validate that email_verified: false claim is included in ID token
}

// TestIDTokenNoEmailAddressContract validates handling when no email addresses exist.
func TestIDTokenNoEmailAddressContract(t *testing.T) {
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})

	// Use a stub consent challenge with no email addresses
	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=no-email", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Should successfully handle missing email addresses
	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d for no email consent, got %d", http.StatusFound, rec.Code)
	}

	if loc := rec.Header().Get("Location"); loc == "" {
		t.Fatal("expected Location header on successful consent with no email")
	}

	// TODO: Validate that email_verified claim is omitted (not false) when no addresses exist
}

// TestIDTokenMultipleEmailsContract validates handling of multiple email addresses
// with mixed verification status.
func TestIDTokenMultipleEmailsContract(t *testing.T) {
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})

	// Use a stub consent challenge with multiple emails (some verified, some not)
	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=multiple-emails", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Should successfully handle multiple email addresses
	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d for multiple emails consent, got %d", http.StatusFound, rec.Code)
	}

	if loc := rec.Header().Get("Location"); loc == "" {
		t.Fatal("expected Location header on successful consent with multiple emails")
	}

	// TODO: Validate that email_verified is true if ANY email is verified
	// (Boolean OR logic across all verifiable_addresses)
}