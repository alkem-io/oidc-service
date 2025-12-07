package contract_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/maintenance"
	"github.com/alkem-io/oidc-service/internal/server"
)

func TestLoginContractRedirectsOnSuccess(t *testing.T) {
	router := server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/login?login_challenge=test", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, rec.Code)
	}

	if loc := rec.Header().Get("Location"); loc == "" {
		t.Fatal("expected Location header on redirect response")
	}
}
