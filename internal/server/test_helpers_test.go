package server

import (
	"context"
	"net/http"

	"github.com/alkem-io/oidc-service/internal/challenge"
)

type challengeServiceStub struct {
	resolveLogin   func(ctx context.Context, challengeID string) (*challenge.Resolution, error)
	resolveConsent func(ctx context.Context, challengeID string) (*challenge.Resolution, error)
	readiness      func(ctx context.Context) challenge.ReadinessState
}

func (s *challengeServiceStub) ResolveLogin(ctx context.Context, challengeID string) (*challenge.Resolution, error) {
	if s.resolveLogin == nil {
		return nil, challenge.NewError(http.StatusNotImplemented, "not_implemented", "resolve login not implemented", challengeID, nil)
	}
	return s.resolveLogin(ctx, challengeID)
}

func (s *challengeServiceStub) ResolveConsent(ctx context.Context, challengeID string) (*challenge.Resolution, error) {
	if s.resolveConsent == nil {
		return nil, challenge.NewError(http.StatusNotImplemented, "not_implemented", "resolve consent not implemented", challengeID, nil)
	}
	return s.resolveConsent(ctx, challengeID)
}

func (s *challengeServiceStub) Readiness(ctx context.Context) challenge.ReadinessState {
	if s.readiness == nil {
		return challenge.ReadinessState{}
	}
	return s.readiness(ctx)
}
