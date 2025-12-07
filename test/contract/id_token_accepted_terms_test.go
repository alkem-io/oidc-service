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

func newTestRouter() http.Handler {
	return server.NewRouter(server.Options{
		Logger:      zap.NewNop(),
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
	})
}

// TestIDTokenAcceptedTermsStates validates that consent endpoint properly handles
// accepted_terms claim permutations before tokens exist for decoding.
func TestIDTokenAcceptedTermsStates(t *testing.T) {
	testCases := []struct {
		name        string
		challengeID string
		description string
	}{
		{
			name:        "Accepted",
			challengeID: "terms-accepted",
			description: "accepted_terms true should redirect",
		},
		{
			name:        "Rejected",
			challengeID: "terms-not-accepted",
			description: "accepted_terms false should still redirect",
		},
		{
			name:        "Missing",
			challengeID: "no-terms-trait",
			description: "missing trait keeps flow backwards compatible",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := newTestRouter()

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

// TestIDTokenAcceptedTermsInvalidContract validates handling of invalid accepted_terms values.
func TestIDTokenAcceptedTermsInvalidContract(t *testing.T) {
	testCases := []struct {
		name        string
		challengeID string
		description string
	}{
		{
			name:        "StringTrue",
			challengeID: "terms-string-true",
			description: "Should handle accepted_terms: 'true' string",
		},
		{
			name:        "StringFalse",
			challengeID: "terms-string-false",
			description: "Should handle accepted_terms: 'false' string",
		},
		{
			name:        "InvalidValue",
			challengeID: "terms-invalid-value",
			description: "Should handle invalid accepted_terms value",
		},
		{
			name:        "NullValue",
			challengeID: "terms-null-value",
			description: "Should handle null accepted_terms value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := newTestRouter()

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
