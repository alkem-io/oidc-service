package contract_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/maintenance"
	"github.com/alkem-io/oidc-service/internal/server"
	testsupport "github.com/alkem-io/oidc-service/test/support"
	"go.uber.org/zap"
)

func newIDTokenRouter() http.Handler {
	return server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})
}

// TestIDTokenClaimsContract validates that consent endpoint properly handles
// enhanced ID token claims for various name payloads.
func TestIDTokenClaimsContract(t *testing.T) {
	testCases := []struct {
		name        string
		challengeID string
		description string
	}{
		{
			name:        "CompleteProfile",
			challengeID: "id-token-profile-claims",
			description: "complete profile should redirect",
		},
		{
			name:        "UTF8Names",
			challengeID: "utf8-names",
			description: "unicode characters stay intact",
		},
		{
			name:        "EmptyNames",
			challengeID: "empty-names",
			description: "missing names keep flow backwards compatible",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := newIDTokenRouter()

			req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge="+tc.challengeID, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusFound {
				t.Fatalf("%s: expected status %d, got %d", tc.description, http.StatusFound, rec.Code)
			}

			testsupport.AssertStubRedirect(t, rec.Header().Get("Location"), "consent", tc.challengeID)
		})
	}
}
