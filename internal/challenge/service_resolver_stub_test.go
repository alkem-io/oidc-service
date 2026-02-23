package challenge

import (
	"context"

	"github.com/alkem-io/oidc-service/internal/alkemio"
)

const (
	defaultAlkemioActorID = "0f8fad5b-d9cb-469f-a165-70867728950e"
)

type alkemioResolverStub struct {
	resolve func(ctx context.Context, authenticationID string) (*alkemio.IdentityMapping, error)
}

func (s alkemioResolverStub) Resolve(ctx context.Context, authenticationID string) (*alkemio.IdentityMapping, error) {
	if s.resolve != nil {
		return s.resolve(ctx, authenticationID)
	}
	return &alkemio.IdentityMapping{ActorID: defaultAlkemioActorID}, nil
}
