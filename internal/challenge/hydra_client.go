package challenge

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	hydraAdmin "github.com/ory/hydra-client-go/v2"
)

// hydraOAuth2Client adapts the generated Hydra OAuth2 API to the domain interface.
type hydraOAuth2Client struct {
	api hydraAdmin.OAuth2API
}

// NewHydraOAuth2Client wraps the provided Hydra OAuth2 API implementation.
func NewHydraOAuth2Client(api hydraAdmin.OAuth2API) (HydraClient, error) {
	if api == nil {
		return nil, errors.New("hydra oauth2 api is required")
	}
	return &hydraOAuth2Client{api: api}, nil
}

// GetLoginRequest fetches the login challenge metadata from Hydra.
func (c *hydraOAuth2Client) GetLoginRequest(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2LoginRequest, *http.Response, error) {
	req := c.api.GetOAuth2LoginRequest(ctx).LoginChallenge(challengeID)
	return req.Execute()
}

// AcceptLoginRequest finalizes the login challenge with the supplied payload.
func (c *hydraOAuth2Client) AcceptLoginRequest(ctx context.Context, challengeID string, body *hydraAdmin.AcceptOAuth2LoginRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
	if body == nil {
		return nil, nil, fmt.Errorf("accept login request body is required")
	}
	req := c.api.AcceptOAuth2LoginRequest(ctx).LoginChallenge(challengeID).AcceptOAuth2LoginRequest(*body)
	return req.Execute()
}

// GetConsentRequest fetches the consent challenge metadata from Hydra.
func (c *hydraOAuth2Client) GetConsentRequest(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2ConsentRequest, *http.Response, error) {
	req := c.api.GetOAuth2ConsentRequest(ctx).ConsentChallenge(challengeID)
	return req.Execute()
}

// AcceptConsentRequest finalizes the consent challenge with the supplied payload.
func (c *hydraOAuth2Client) AcceptConsentRequest(ctx context.Context, challengeID string, body *hydraAdmin.AcceptOAuth2ConsentRequest) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
	if body == nil {
		return nil, nil, fmt.Errorf("accept consent request body is required")
	}
	req := c.api.AcceptOAuth2ConsentRequest(ctx).ConsentChallenge(challengeID).AcceptOAuth2ConsentRequest(*body)
	return req.Execute()
}

// GetLogoutRequest fetches the logout challenge metadata from Hydra.
func (c *hydraOAuth2Client) GetLogoutRequest(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2LogoutRequest, *http.Response, error) {
	req := c.api.GetOAuth2LogoutRequest(ctx).LogoutChallenge(challengeID)
	return req.Execute()
}

// AcceptLogoutRequest finalizes the logout challenge and returns Hydra's post-logout redirect.
func (c *hydraOAuth2Client) AcceptLogoutRequest(ctx context.Context, challengeID string) (*hydraAdmin.OAuth2RedirectTo, *http.Response, error) {
	req := c.api.AcceptOAuth2LogoutRequest(ctx).LogoutChallenge(challengeID)
	return req.Execute()
}
