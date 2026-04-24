package server

import (
	"context"
	"net/http"

	middlewarepkg "github.com/alkem-io/oidc-service/internal/middleware"
)

// CorrelationIDHeader mirrors middleware.RequestIDHeader so the server package
// has a single import surface for the correlation-id concept.
const CorrelationIDHeader = middlewarepkg.RequestIDHeader

// CorrelationID returns the request-scoped correlation id established by the
// RequestContext middleware. Returns an empty string when the context carries
// no correlation id (e.g. outside of an inbound HTTP request).
func CorrelationID(ctx context.Context) string {
	return middlewarepkg.RequestID(ctx)
}

// PropagateCorrelationID copies the inbound correlation id onto an outbound
// HTTP request so downstream hops (Kratos admin, Hydra admin, alkemio-server)
// stay on the same correlation thread. No-op when the context has no id.
func PropagateCorrelationID(ctx context.Context, req *http.Request) {
	id := CorrelationID(ctx)
	if id == "" {
		return
	}
	req.Header.Set(CorrelationIDHeader, id)
}
