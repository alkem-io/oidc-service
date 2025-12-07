package challenge

import (
	"context"
	"errors"
)

type identityHintContextKey struct{}

// IdentityHintProvider supplies a fallback identity identifier when Hydra omits the subject.
type IdentityHintProvider interface {
	// IdentityHint returns the identity ID from the provider.
	IdentityHint(ctx context.Context) (string, error)
}

type identityHintFunc func(context.Context) (string, error)

// IdentityHint proxies the call to the wrapped function.
func (f identityHintFunc) IdentityHint(ctx context.Context) (string, error) {
	return f(ctx)
}

// IdentityHintFunc wraps a function to satisfy IdentityHintProvider.
func IdentityHintFunc(fn func(context.Context) (string, error)) IdentityHintProvider {
	if fn == nil {
		return nil
	}
	return identityHintFunc(fn)
}

// WithIdentityHintProvider attaches an identity hint provider to the context for downstream resolution.
func WithIdentityHintProvider(ctx context.Context, provider IdentityHintProvider) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if provider == nil {
		return ctx
	}
	return context.WithValue(ctx, identityHintContextKey{}, provider)
}

// IdentityHintProviderFromContext extracts the hint provider from context when available.
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
