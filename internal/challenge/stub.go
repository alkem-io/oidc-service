package challenge

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// NewStubService returns a deterministic implementation used in tests and local development.
func NewStubService() Service {
	return &stubService{}
}

type stubService struct{}

// ResolveLogin returns deterministic responses for login flows in stub mode.
func (s *stubService) ResolveLogin(ctx context.Context, challengeID string) (*Resolution, error) {
	return s.resolve(ctx, "login", challengeID)
}

// ResolveConsent returns deterministic responses for consent flows in stub mode.
func (s *stubService) ResolveConsent(ctx context.Context, challengeID string) (*Resolution, error) {
	return s.resolve(ctx, "consent", challengeID)
}

// ResolveLogout returns deterministic responses for logout flows in stub mode.
func (s *stubService) ResolveLogout(ctx context.Context, challengeID string) (*Resolution, error) {
	return s.resolve(ctx, "logout", challengeID)
}

// Readiness reports synthetic health data for the stub implementation.
func (s *stubService) Readiness(_ context.Context) ReadinessState {
	return ReadinessState{
		Status:  "ready",
		Hydra:   "ok",
		Kratos:  "ok",
		Version: "stub",
	}
}

func (s *stubService) resolve(_ context.Context, flow string, challengeID string) (*Resolution, error) {
	switch challengeID {
	case "":
		return nil, NewError(
			http.StatusBadRequest, "missing_challenge", fmt.Sprintf("%s challenge is required", flow), challengeID, nil,
		)
	case "hydra-error":
		return nil, NewHydraFailureError(challengeID, "hydra upstream returned an error")
	case "missing-traits":
		if flow == "login" {
			// login flow with missing traits should return 400 per tests
			return nil, NewMissingTraitsError(challengeID, []string{"traits.email", "traits.display_name"})
		}
		// consent flow should still succeed (graceful degradation)
		return &Resolution{RedirectURL: s.buildRedirect(flow, challengeID)}, nil
	case "invalid-traits":
		// consent flow should still succeed; login not covered in tests
		return &Resolution{RedirectURL: s.buildRedirect(flow, challengeID)}, nil
	case "missing-first-name", "missing-last-name", "empty-name-object", "null-name-fields":
		// Partial profile data scenarios should still succeed
		return &Resolution{RedirectURL: s.buildRedirect(flow, challengeID)}, nil
	case "token-size-limit":
		// Simulate a Hydra failure due to oversized tokens
		return nil, NewHydraFailureError(challengeID, "hydra rejected token (size limit)")
	case "test", "email-verified", "email-unverified", "no-email", "multiple-emails",
		"terms-accepted", "terms-not-accepted", "no-terms-trait",
		"terms-string-true", "terms-string-false", "terms-invalid-value", "terms-null-value",
		"profile-claims", "partial-profile", "no-profile", "utf8-names", "empty-names",
		"id-token-profile-claims":
		return &Resolution{RedirectURL: s.buildRedirect(flow, challengeID)}, nil
	}

	// Pattern-based scenarios
	if strings.HasPrefix(challengeID, "kratos-") {
		return nil, NewKratosFailureError(challengeID, "kratos upstream failure")
	}

	return nil, NewInvalidChallengeError(challengeID)
}

func (s *stubService) buildRedirect(flow, challengeID string) string {
	values := url.Values{}
	values.Set("flow", flow)
	values.Set("challenge", challengeID)
	return fmt.Sprintf("https://oidc.stub.local/callback?%s", values.Encode())
}
