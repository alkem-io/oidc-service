package challenge

import (
	"context"

	"github.com/alkem-io/oidc-service/internal/alkemio"
)

const (
	defaultAlkemioUserID  = "0f8fad5b-d9cb-469f-a165-70867728950e"
	defaultAlkemioAgentID = "3b241101-e2bb-4255-8caf-4136c566a962"
)

type alkemioResolverStub struct {
	resolve func(ctx context.Context, authenticationID string) (*alkemio.IdentityMapping, error)
}

func (s alkemioResolverStub) Resolve(ctx context.Context, authenticationID string) (*alkemio.IdentityMapping, error) {
	if s.resolve != nil {
		return s.resolve(ctx, authenticationID)
	}
	return &alkemio.IdentityMapping{UserID: defaultAlkemioUserID, AgentID: defaultAlkemioAgentID}, nil
}
