package support

import "github.com/alkem-io/oidc-service/internal/alkemio"

// NewAlkemioMapping returns a resolver mapping populated with the supplied identifiers.
func NewAlkemioMapping(userID, agentID string) *alkemio.IdentityMapping {
	return &alkemio.IdentityMapping{UserID: userID, AgentID: agentID}
}
