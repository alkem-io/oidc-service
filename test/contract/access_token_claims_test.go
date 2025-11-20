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

// TestAccessTokenClaimsContract validates that the consent handler routes
// profile-claim scenarios through the Hydra stub and preserves challenge IDs.
func TestAccessTokenClaimsContract(t *testing.T) {
	router := server.NewRouter(
		server.Options{
			Logger:      zap.NewNop(),
			Maintenance: maintenance.NewState(config.MaintenanceState{}),
		},
	)

	testCases := []struct {
		name        string
		challengeID string
		description string
	}{
		{
			name:        "CompleteProfile",
			challengeID: "profile-claims",
			description: "full profile data should redirect via stub",
		},
		{
			name:        "PartialProfile",
			challengeID: "partial-profile",
			description: "partial profile data still succeeds",
		},
		{
			name:        "NoProfile",
			challengeID: "no-profile",
			description: "absence of profile data keeps flow compatible",
		},
	}

	for _, tc := range testCases {
		t.Run(
			tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge="+tc.challengeID, nil)
				rec := httptest.NewRecorder()

				router.ServeHTTP(rec, req)

				if rec.Code != http.StatusFound {
					t.Fatalf("%s: expected status %d, got %d", tc.description, http.StatusFound, rec.Code)
				}

				testsupport.AssertStubRedirect(t, rec.Header().Get("Location"), "consent", tc.challengeID)
			},
		)
	}
}
