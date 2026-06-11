package integration_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/maintenance"
	"github.com/alkem-io/oidc-service/internal/server"
)

// T075 integration — exercises the logout handler through the FULL router
// stack (route mounted at `/oidc/logout` + `/v1/oidc/logout` per
// `internal/server/router.go:137`). Verifies the wire-level mapping of
// upstream Hydra failures to HTTP status codes the BFF can interpret.

// TestLogoutRouteMissingChallengeViaRouter — wire-level invariant: the
// router-mounted handler returns 400 for a missing challenge, both at the
// canonical and v1-prefixed paths.
func TestLogoutRouteMissingChallengeViaRouter(t *testing.T) {
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})

	for _, path := range []string{"/oidc/logout", "/v1/oidc/logout"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("path %s: expected 400 for missing logout_challenge, got %d", path, rec.Code)
		}
	}
}

// TestLogoutRouteHydraFailureViaRouter — Hydra admin failure mid-flow MUST
// be mapped to a 5xx response (per existing challenge-failure mapping in
// `WriteChallengeError`). Caller MUST NOT see a 302 with stale data.
func TestLogoutRouteHydraFailureViaRouter(t *testing.T) {
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/logout?logout_challenge=hydra-error", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusFound {
		t.Fatalf("hydra-error logout MUST NOT 302; got 302 with Location=%s", rec.Header().Get("Location"))
	}
	if rec.Code < 500 || rec.Code >= 600 {
		// Allow 502 or 503 mapping; reject anything outside 5xx.
		t.Fatalf("expected 5xx for hydra failure, got %d", rec.Code)
	}
}

// TestLogoutRouteHappyPathViaRouter — valid challenge resolves through the
// stub service and 302s through the router. Pins the route → handler →
// service wiring intact.
func TestLogoutRouteHappyPathViaRouter(t *testing.T) {
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})

	req := httptest.NewRequest(http.MethodGet, "/oidc/logout?logout_challenge=test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("happy path: expected 302, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "flow=logout") {
		t.Fatalf("Location must echo flow=logout, got %s", loc)
	}
	if !strings.Contains(loc, "challenge=test") {
		t.Fatalf("Location must echo challenge id, got %s", loc)
	}
}
