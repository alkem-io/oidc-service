package support

import "github.com/alkem-io/oidc-service/internal/alkemio"

// NewAlkemioMapping returns a resolver mapping populated with the supplied actor identifier.
func NewAlkemioMapping(actorID string) *alkemio.IdentityMapping {
	return &alkemio.IdentityMapping{ActorID: actorID}
}
