package contract_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/maintenance"
	"github.com/alkem-io/oidc-service/internal/server"
)

// FR-036a contract — oidc-service exposes /health/live and /health/ready.
//   - /health/live: 200, {status:"ok"}; no dep calls
//   - /health/ready: 200 with {status:"ok", checks:{kratos, hydra}} when both
//     admin planes respond; 503 with {status:"unhealthy", checks{...}} when
//     either dep is unreachable.
//
// This test pins the response shape and the fail-closed behaviour. It does
// NOT assert specific status codes from real Hydra/Kratos — for that, see
// `internal/server/health_test.go` which exercises the per-dep paths with
// stub upstreams.
func TestHealthEndpointsMatchContract(t *testing.T) {
	// Stub admin endpoints so the readiness probe returns the healthy shape.
	stubAdmin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer stubAdmin.Close()

	maint := maintenance.NewState(config.MaintenanceState{})
	router := server.NewRouter(server.Options{
		Logger:         zap.NewNop(),
		Maintenance:    maint,
		HydraAdminURL:  stubAdmin.URL,
		KratosAdminURL: stubAdmin.URL,
	})

	// Liveness — handler responsive only, MUST always be 200.
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
	if status, ok := livePayload["status"].(string); !ok || status != "ok" {
		t.Fatalf("expected liveness payload to contain status 'ok', got %v", livePayload["status"])
	}

	// Readiness — both deps up → 200 with the FR-036a shape.
	readyReq := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	readyRec := httptest.NewRecorder()
	router.ServeHTTP(readyRec, readyReq)

	if readyRec.Code != http.StatusOK {
		t.Fatalf("expected readiness status %d, got %d", http.StatusOK, readyRec.Code)
	}

	var readyPayload map[string]any
	if err := json.Unmarshal(readyRec.Body.Bytes(), &readyPayload); err != nil {
		t.Fatalf("failed to parse readiness response: %v", err)
	}

	if status, ok := readyPayload["status"].(string); !ok || status != "ok" {
		t.Fatalf("expected readiness status 'ok', got %v", readyPayload["status"])
	}

	checks, ok := readyPayload["checks"].(map[string]any)
	if !ok {
		t.Fatalf("expected readiness payload to include 'checks' map, got %v", readyPayload["checks"])
	}
	for _, dep := range []string{"kratos", "hydra"} {
		entry, ok := checks[dep].(map[string]any)
		if !ok {
			t.Fatalf("expected checks.%s entry, got %v", dep, checks[dep])
		}
		if status, _ := entry["status"].(string); status != "ok" {
			t.Fatalf("expected checks.%s.status='ok', got %v", dep, entry["status"])
		}
	}

	// Readiness with unconfigured upstreams → 503 (fail-closed).
	closedRouter := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maint,
	})
	failReq := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	failRec := httptest.NewRecorder()
	closedRouter.ServeHTTP(failRec, failReq)
	if failRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected fail-closed 503 with no upstream config, got %d", failRec.Code)
	}
}
