// Package requestid owns the request-scoped correlation id so every package reads the same value.
package requestid

import "context"

type contextKey struct{}

// With returns a context carrying the correlation id.
func With(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

// FromContext returns the correlation id, or "" when none was set.
func FromContext(ctx context.Context) string {
	id, _ := ctx.Value(contextKey{}).(string)
	return id
}
