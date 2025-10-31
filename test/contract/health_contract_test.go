package contract_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/maintenance"
	"github.com/alkem-io/oidc-service/internal/server"
	"go.uber.org/zap"
)

func TestHealthEndpointsMatchContract(t *testing.T) {
	maint := maintenance.NewState(config.MaintenanceState{})
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maint,
	})

	readyReq := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	readyRec := httptest.NewRecorder()
	router.ServeHTTP(readyRec, readyReq)

	if readyRec.Code != http.StatusOK {
		t.Fatalf("expected readiness status %d, got %d", http.StatusOK, readyRec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(readyRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse readiness response: %v", err)
	}

	required := []string{"status", "hydra", "kratos", "maintenance"}
	for _, key := range required {
		if _, ok := payload[key]; !ok {
			t.Fatalf("expected readiness payload to include %q", key)
		}
	}

	liveReq := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	liveRec := httptest.NewRecorder()
	router.ServeHTTP(liveRec, liveReq)

	if liveRec.Code != http.StatusOK {
		t.Fatalf("expected liveness status %d, got %d", http.StatusOK, liveRec.Code)
	}

	var livePayload map[string]any
	if err := json.Unmarshal(liveRec.Body.Bytes(), &livePayload); err != nil {
		t.Fatalf("failed to parse liveness response: %v", err)
	}

	if status, ok := livePayload["status"].(string); !ok || status != "alive" {
		t.Fatalf("expected liveness payload to contain status 'alive', got %v", livePayload["status"])
	}
}
