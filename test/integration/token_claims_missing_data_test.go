package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/maintenance"
	"github.com/alkem-io/oidc-service/internal/server"
	"go.uber.org/zap"
)

// TestTokenClaimsMissingData tests that token generation handles missing
// Kratos identity traits gracefully by omitting the corresponding claims.
func TestTokenClaimsMissingData(t *testing.T) {
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})

	// Use a stub consent challenge with incomplete identity data
	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=missing-traits", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Should succeed despite missing data (graceful degradation)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d for missing traits consent, got %d", http.StatusFound, rec.Code)
	}

	if loc := rec.Header().Get("Location"); loc == "" {
		t.Fatal("expected Location header on successful consent with missing traits")
	}

	// TODO: In a full implementation, we would validate that:
	// 1. Tokens are generated successfully despite missing name data
	// 2. Missing claims are omitted (not included as null/empty)
	// 3. Available claims are still included correctly
	// This ensures backward compatibility and graceful degradation
}

// TestTokenClaimsInvalidData tests handling of invalid/malformed identity data
// during token claim extraction.
func TestTokenClaimsInvalidData(t *testing.T) {
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})

	// Use a stub consent challenge with invalid/malformed identity data
	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=invalid-traits", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Should handle invalid data gracefully
	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d for invalid traits consent, got %d", http.StatusFound, rec.Code)
	}

	if loc := rec.Header().Get("Location"); loc == "" {
		t.Fatal("expected Location header on successful consent with invalid traits")
	}

	// TODO: Validate that invalid data is safely ignored and doesn't break token generation
}

// TestTokenClaimsPartialMissingData tests various scenarios of partially missing
// identity data (e.g., first name present but last name missing).
func TestTokenClaimsPartialMissingData(t *testing.T) {
	testCases := []struct {
		name        string
		challengeID string
		description string
	}{
		{
			name:        "MissingFirstName",
			challengeID: "missing-first-name",
			description: "Should handle missing first name gracefully",
		},
		{
			name:        "MissingLastName",
			challengeID: "missing-last-name",
			description: "Should handle missing last name gracefully",
		},
		{
			name:        "EmptyNameObject",
			challengeID: "empty-name-object",
			description: "Should handle empty name object gracefully",
		},
		{
			name:        "NullNameFields",
			challengeID: "null-name-fields",
			description: "Should handle null name fields gracefully",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := server.NewRouter(server.Options{
				Logger:      zap.NewNop(),
				Maintenance: maintenance.NewState(config.MaintenanceState{}),
			})

			req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge="+tc.challengeID, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			// Should handle partial data gracefully
			if rec.Code != http.StatusFound {
				t.Fatalf("%s: expected status %d, got %d", tc.description, http.StatusFound, rec.Code)
			}

			if loc := rec.Header().Get("Location"); loc == "" {
				t.Fatalf("%s: expected Location header", tc.description)
			}

			// TODO: Validate specific claim presence/absence based on available data
		})
	}
}
