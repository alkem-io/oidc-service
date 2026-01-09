package webhook

// Request represents the Kratos webhook payload.
type Request struct {
	IdentityID string `json:"identity_id"`
}

// Response represents the response to Kratos.
type Response struct {
	Identity IdentityUpdate `json:"identity"`
}

// IdentityUpdate contains fields Kratos will merge into the identity.
type IdentityUpdate struct {
	MetadataPublic *MetadataPublic `json:"metadata_public,omitempty"`
}

// MetadataPublic contains Alkemio identity claims.
type MetadataPublic struct {
	AlkemioActorID string `json:"alkemio_actor_id"`
	AlkemioAgentID string `json:"alkemio_agent_id"`
}

// ErrorResponse represents an error response to Kratos.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
