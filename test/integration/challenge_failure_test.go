package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/maintenance"
	"github.com/alkem-io/oidc-service/internal/server"
	"go.uber.org/zap"
)

func TestLoginHydraFailureReturnsServerError(t *testing.T) {
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/login?login_challenge=hydra-error", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d for hydra failure, got %d", http.StatusInternalServerError, rec.Code)
	}

	assertErrorPayload(t, rec.Body.Bytes())
}

func TestLoginMissingTraitsReturnsBadRequest(t *testing.T) {
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/login?login_challenge=missing-traits", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d for missing traits, got %d", http.StatusBadRequest, rec.Code)
	}

	payload := assertErrorPayload(t, rec.Body.Bytes())
	if _, ok := payload["missingTraits"]; !ok {
		t.Fatal("expected missingTraits field in error payload")
	}
}

func TestMaintenanceModeShortCircuitsChallenges(t *testing.T) {
	retry := 30 * time.Second
	maint := maintenance.NewState(config.MaintenanceState{Enabled: true, RetryAfter: &retry})

	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maint,
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/login?login_challenge=any", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d during maintenance, got %d", http.StatusServiceUnavailable, rec.Code)
	}

	if value := rec.Header().Get("Retry-After"); value == "" {
		t.Fatal("expected Retry-After header when maintenance RetryAfter configured")
	}

	payload := assertErrorPayload(t, rec.Body.Bytes())
	if payload["error"] != "maintenance_mode" {
		t.Fatalf("expected maintenance_mode error code, got %v", payload["error"])
	}
}

func assertErrorPayload(t *testing.T, data []byte) map[string]any {
	t.Helper()

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("response was not valid json: %v", err)
	}

	required := []string{"error", "message", "challengeId"}
	for _, key := range required {
		if _, ok := payload[key]; !ok {
			t.Fatalf("expected error payload to include %q", key)
		}
	}

	return payload
}
