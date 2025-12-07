package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/maintenance"
	"github.com/alkem-io/oidc-service/internal/server"
)

// TestKratosUnavailableTokenGeneration tests that token generation fails gracefully
// when Kratos is unavailable during consent flow.
func TestKratosUnavailableTokenGeneration(t *testing.T) {
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})

	// Use a stub challenge ID that will trigger Kratos unavailability
	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=kratos-unavailable", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d for Kratos unavailability, got %d", http.StatusInternalServerError, rec.Code)
	}

	payload := assertErrorPayload(t, rec.Body.Bytes())

	// Verify that we get a Kratos failure error
	if payload["error"] != "kratos_failure" {
		t.Fatalf("expected kratos_failure error code, got %v", payload["error"])
	}
}

// TestKratosUnavailableClaimExtraction tests handling of Kratos unavailability
// specifically during the identity trait extraction phase.
func TestKratosUnavailableClaimExtraction(t *testing.T) {
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})

	// Use a stub challenge ID that will trigger Kratos failure during identity fetch
	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/login?login_challenge=kratos-fetch-error", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d for Kratos fetch failure, got %d", http.StatusInternalServerError, rec.Code)
	}

	payload := assertErrorPayload(t, rec.Body.Bytes())

	// Verify that we get a Kratos failure error
	if payload["error"] != "kratos_failure" {
		t.Fatalf("expected kratos_failure error code, got %v", payload["error"])
	}
}

// TestKratosTimeoutTokenGeneration tests handling of Kratos timeouts during
// token claim extraction.
func TestKratosTimeoutTokenGeneration(t *testing.T) {
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})

	// Use a stub challenge ID that will trigger Kratos timeout
	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=kratos-timeout", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d for Kratos timeout, got %d", http.StatusInternalServerError, rec.Code)
	}

	payload := assertErrorPayload(t, rec.Body.Bytes())

	// Verify that we get a Kratos failure error
	if payload["error"] != "kratos_failure" {
		t.Fatalf("expected kratos_failure error code, got %v", payload["error"])
	}
}
