package challenge

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// NewStubService returns a deterministic implementation used in tests and local development.
func NewStubService() Service {
	return &stubService{}
}

type stubService struct{}

func (s *stubService) ResolveLogin(ctx context.Context, challengeID string) (*Resolution, error) {
	return s.resolve(ctx, "login", challengeID)
}

func (s *stubService) ResolveConsent(ctx context.Context, challengeID string) (*Resolution, error) {
	return s.resolve(ctx, "consent", challengeID)
}

func (s *stubService) Readiness(ctx context.Context) ReadinessState {
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
		return nil, NewError(http.StatusBadRequest, "missing_challenge", fmt.Sprintf("%s challenge is required", flow), challengeID, nil)
	case "hydra-error":
		return nil, NewHydraFailureError(challengeID, "hydra upstream returned an error")
	case "missing-traits":
		return nil, NewMissingTraitsError(challengeID, []string{"traits.email", "traits.display_name"})
	case "test":
		return &Resolution{RedirectURL: s.buildRedirect(flow, challengeID)}, nil
	case "email-verified", "email-unverified", "no-email", "multiple-emails":
		return &Resolution{RedirectURL: s.buildRedirect(flow, challengeID)}, nil
	case "terms-accepted", "terms-not-accepted", "no-terms-trait":
		return &Resolution{RedirectURL: s.buildRedirect(flow, challengeID)}, nil
	case "terms-string-true", "terms-string-false", "terms-invalid-value", "terms-null-value":
		return &Resolution{RedirectURL: s.buildRedirect(flow, challengeID)}, nil
	case "profile-claims", "partial-profile", "no-profile", "utf8-names", "empty-names":
		return &Resolution{RedirectURL: s.buildRedirect(flow, challengeID)}, nil
	case "id-token-profile-claims":
		return &Resolution{RedirectURL: s.buildRedirect(flow, challengeID)}, nil
	default:
		return nil, NewInvalidChallengeError(challengeID)
	}
}

func (s *stubService) buildRedirect(flow, challengeID string) string {
	values := url.Values{}
	values.Set("flow", flow)
	values.Set("challenge", challengeID)
	return fmt.Sprintf("https://oidc.stub.local/callback?%s", values.Encode())
}
