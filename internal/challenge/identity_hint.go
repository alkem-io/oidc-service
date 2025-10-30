package challenge

import (
	"context"
	"errors"
)

type identityHintContextKey struct{}

// IdentityHintProvider supplies a fallback identity identifier when Hydra omits the subject.
type IdentityHintProvider interface {
	IdentityHint(ctx context.Context) (string, error)
}

type identityHintFunc func(context.Context) (string, error)

func (f identityHintFunc) IdentityHint(ctx context.Context) (string, error) {
	return f(ctx)
}

// IdentityHintFunc returns an IdentityHintProvider that calls the given function.
// If fn is nil, IdentityHintFunc returns nil; otherwise the returned provider
// delegates IdentityHint calls to fn.
func IdentityHintFunc(fn func(context.Context) (string, error)) IdentityHintProvider {
	if fn == nil {
		return nil
	}
	return identityHintFunc(fn)
}

// WithIdentityHintProvider returns a copy of ctx that carries the provided IdentityHintProvider.
// If ctx is nil, context.Background() is used. If provider is nil, the original (or defaulted) context is returned unchanged.
func WithIdentityHintProvider(ctx context.Context, provider IdentityHintProvider) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if provider == nil {
		return ctx
	}
	return context.WithValue(ctx, identityHintContextKey{}, provider)
}

// IdentityHintProviderFromContext extracts the IdentityHintProvider stored in the context, if present.
// If ctx is nil or no provider is associated with the context, it returns nil.
func IdentityHintProviderFromContext(ctx context.Context) IdentityHintProvider {
	if ctx == nil {
		return nil
	}
	provider, _ := ctx.Value(identityHintContextKey{}).(IdentityHintProvider)
	return provider
}

// Sentinel errors surfaced by identity hint providers.
var (
	ErrIdentitySessionRequired = errors.New("identity session required")
	ErrIdentitySessionInvalid  = errors.New("identity session invalid")
)