package challenge

import "context"

type correlationIDContextKey struct{}

// WithCorrelationID attaches the inbound request's correlation id so the records the service writes join its request.
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationIDContextKey{}, correlationID)
}

func correlationIDFromContext(ctx context.Context) string {
	correlationID, _ := ctx.Value(correlationIDContextKey{}).(string)
	return correlationID
}
