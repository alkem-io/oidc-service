package support

import (
	"context"
	"errors"
	"net/http"

	hydraAdmin "github.com/ory/hydra-client-go/v2"

	"github.com/alkem-io/oidc-service/internal/alkemio"
	"github.com/alkem-io/oidc-service/internal/challenge"
)

// HydraClientStub provides a configurable Hydra client implementation for tests.
type HydraClientStub struct {
	GetLoginFunc      func(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2LoginRequest, *http.Response, error)
	AcceptLoginFunc   func(ctx context.Context, challengeID string, body *hydraAdmin.AcceptOAuth2LoginRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error)
	GetConsentFunc    func(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error)
	AcceptConsentFunc func(ctx context.Context, challengeID string, body *hydraAdmin.AcceptOAuth2ConsentRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error)
}

func (s *HydraClientStub) GetLoginRequest(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2LoginRequest, *http.Response, error) {
	if s == nil || s.GetLoginFunc == nil {
		return nil, nil, errors.New("get login request not stubbed")
	}
	return s.GetLoginFunc(ctx, challengeID)
}

func (s *HydraClientStub) AcceptLoginRequest(ctx context.Context, challengeID string, body *hydraAdmin.AcceptOAuth2LoginRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
	if s == nil || s.AcceptLoginFunc == nil {
		return nil, nil, errors.New("accept login request not stubbed")
	}
	return s.AcceptLoginFunc(ctx, challengeID, body)
}

func (s *HydraClientStub) GetConsentRequest(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
	if s == nil || s.GetConsentFunc == nil {
		return nil, nil, errors.New("get consent request not stubbed")
	}
	return s.GetConsentFunc(ctx, challengeID)
}

func (s *HydraClientStub) AcceptConsentRequest(ctx context.Context, challengeID string, body *hydraAdmin.AcceptOAuth2ConsentRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
	if s == nil || s.AcceptConsentFunc == nil {
		return nil, nil, errors.New("accept consent request not stubbed")
	}
	return s.AcceptConsentFunc(ctx, challengeID, body)
}

// IdentityFetcherStub provides a configurable IdentityFetcher implementation for tests.
type IdentityFetcherStub struct {
	FetchFunc func(ctx context.Context, identityID string) (*challenge.IdentityProfile, error)
}

func (s IdentityFetcherStub) Fetch(ctx context.Context, identityID string) (*challenge.IdentityProfile, error) {
	if s.FetchFunc == nil {
		return nil, errors.New("identity fetch not stubbed")
	}
	return s.FetchFunc(ctx, identityID)
}

// AlkemioResolverStub provides a configurable Alkemio resolver implementation for tests.
type AlkemioResolverStub struct {
	ResolveFunc func(ctx context.Context, authenticationID string) (*alkemio.IdentityMapping, error)
}

func (s AlkemioResolverStub) Resolve(ctx context.Context, authenticationID string) (*alkemio.IdentityMapping, error) {
	if s.ResolveFunc == nil {
		return nil, errors.New("alkemio resolve not stubbed")
	}
	return s.ResolveFunc(ctx, authenticationID)
}
