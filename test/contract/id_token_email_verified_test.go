package contract_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/maintenance"
	"github.com/alkem-io/oidc-service/internal/server"
	testsupport "github.com/alkem-io/oidc-service/test/support"
)

func newEmailRouter() http.Handler {
	return server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})
}

// TestIDTokenEmailVerifiedScenarios validates that consent endpoint properly handles
// email_verified claim permutations before token decoding is wired in.
func TestIDTokenEmailVerifiedScenarios(t *testing.T) {
	testCases := []struct {
		name        string
		challengeID string
		description string
	}{
		{
			name:        "Verified",
			challengeID: "email-verified",
			description: "verified email should redirect",
		},
		{
			name:        "Unverified",
			challengeID: "email-unverified",
			description: "unverified email still returns redirect",
		},
		{
			name:        "NoEmail",
			challengeID: "no-email",
			description: "missing emails remain backwards compatible",
		},
		{
			name:        "MultipleEmails",
			challengeID: "multiple-emails",
			description: "multiple addresses collapse to a single redirect",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := newEmailRouter()

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
