package challenge

import (
	"context"
	"errors"
)

type identityHintContextKey struct{}

// SessionAssurance is the live Kratos session behind a login: its identity and authenticator assurance level.
type SessionAssurance struct {
	IdentityID string
	AAL        string
}

// IdentityHintProvider reads the live Kratos session of the browser completing a login.
type IdentityHintProvider interface {
	// IdentityHint returns the session's identity and assurance level.
	IdentityHint(ctx context.Context) (SessionAssurance, error)
}

type identityHintFunc func(context.Context) (SessionAssurance, error)

// IdentityHint proxies the call to the wrapped function.
func (f identityHintFunc) IdentityHint(ctx context.Context) (SessionAssurance, error) {
	return f(ctx)
}

// IdentityHintFunc wraps a function to satisfy IdentityHintProvider.
func IdentityHintFunc(fn func(context.Context) (SessionAssurance, error)) IdentityHintProvider {
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
	ErrIdentitySessionRequired     = errors.New("identity session required")
	ErrIdentitySessionInvalid      = errors.New("identity session invalid")
	ErrIdentitySessionAAL2Required = errors.New("identity session requires a second factor")
)
