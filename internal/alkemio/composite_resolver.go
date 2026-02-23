package alkemio

import (
	"context"
	"errors"

	"go.uber.org/zap"
)

// Resolver defines the interface for resolving Alkemio identity mappings.
type Resolver interface {
	// Resolve looks up the Alkemio ActorID for the given authenticationID.
	Resolve(ctx context.Context, authenticationID string) (*IdentityMapping, error)
}

// CompositeResolver tries the database first, falling back to the API resolver.
type CompositeResolver struct {
	db     Resolver
	api    Resolver
	logger *zap.Logger
}

// CompositeConfig holds the dependencies for creating a CompositeResolver.
type CompositeConfig struct {
	Database Resolver
	API      Resolver
	Logger   *zap.Logger
}

// NewCompositeResolver creates a resolver that tries DB first, then falls back to API.
func NewCompositeResolver(cfg CompositeConfig) (*CompositeResolver, error) {
	if cfg.API == nil {
		return nil, errors.New("API resolver is required")
	}

	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &CompositeResolver{
		db:     cfg.Database,
		api:    cfg.API,
		logger: logger,
	}, nil
}

// Resolve attempts to resolve the identity from the database first.
// If the database lookup fails or returns no result, it falls back to the API.
func (r *CompositeResolver) Resolve(ctx context.Context, authenticationID string) (*IdentityMapping, error) {
	if r.db != nil {
		mapping, err := r.db.Resolve(ctx, authenticationID)
		if err == nil {
			return mapping, nil
		}

		if errors.Is(err, ErrDBNotFound) {
			r.logger.Debug(
				"identity not found in database, falling back to API",
				zap.String("authentication_id", maskUUID(authenticationID)),
			)
		} else {
			r.logger.Warn(
				"database lookup failed, falling back to API",
				zap.String("authentication_id", maskUUID(authenticationID)),
				zap.Error(err),
			)
		}
	}

	return r.api.Resolve(ctx, authenticationID)
}

func maskUUID(id string) string {
	if len(id) <= 8 {
		return "***"
	}
	return id[:8] + "..."
}
