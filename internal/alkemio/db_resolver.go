package alkemio

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/alkemio/queries"
)

// ErrDBNotFound signals that no user mapping exists in the database.
var ErrDBNotFound = errors.New("user not found in database")

// DBTX is the interface required by SQLC queries (satisfied by pgxpool.Pool).
type DBTX interface {
	queries.DBTX
}

// DatabaseResolver resolves Alkemio identity from the PostgreSQL database.
type DatabaseResolver struct {
	queries *queries.Queries
	logger  *zap.Logger
}

// NewDatabaseResolver creates a resolver that queries the database for identity mappings.
func NewDatabaseResolver(pool *pgxpool.Pool, logger *zap.Logger) *DatabaseResolver {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &DatabaseResolver{
		queries: queries.New(pool),
		logger:  logger,
	}
}

// Resolve looks up the Alkemio ActorID for the given authenticationID.
func (r *DatabaseResolver) Resolve(ctx context.Context, authenticationID string) (*IdentityMapping, error) {
	if r == nil || r.queries == nil {
		return nil, errors.New("database resolver is not initialized")
	}

	id := strings.TrimSpace(authenticationID)
	r.logger.Debug(
		"resolving identity from database",
		zap.String("authentication_id", maskUUID(id)),
	)

	if id == "" {
		r.logger.Debug("authentication id validation failed: empty after trim")
		return nil, errors.New("authentication id is required")
	}

	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		r.logger.Debug(
			"authentication id validation failed: invalid uuid",
			zap.String("authentication_id", maskUUID(id)),
			zap.Error(err),
		)
		return nil, fmt.Errorf("authentication id must be a valid uuid: %w", err)
	}

	r.logger.Debug(
		"parsed authentication id",
		zap.String("parsed_uuid", maskUUID(parsedUUID.String())),
	)

	authUUID := pgtype.UUID{
		Bytes: parsedUUID,
		Valid: true,
	}

	r.logger.Debug(
		"querying database for user",
		zap.String("authentication_id", maskUUID(id)),
	)

	row, err := r.queries.GetUserByAuthenticationID(ctx, authUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Debug(
				"user not found in database",
				zap.String("authentication_id", maskUUID(id)),
			)
			return nil, ErrDBNotFound
		}
		r.logger.Error(
			"database query failed",
			zap.String("authentication_id", maskUUID(id)),
			zap.Error(err),
		)
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	if !row.Valid {
		r.logger.Debug(
			"database returned invalid row: missing id",
			zap.String("authentication_id", maskUUID(id)),
		)
		return nil, ErrDBNotFound
	}

	actorID := uuidToString(row)

	if actorID == "" {
		r.logger.Debug(
			"uuid conversion returned empty string",
			zap.String("authentication_id", maskUUID(id)),
		)
		return nil, ErrDBNotFound
	}

	if !isUUID(actorID) {
		r.logger.Error(
			"database returned invalid actor id format",
			zap.String("authentication_id", maskUUID(id)),
			zap.String("actor_id", maskUUID(actorID)),
		)
		return nil, fmt.Errorf("database returned invalid actor id")
	}

	r.logger.Info(
		"identity resolved from database",
		zap.String("authentication_id", maskUUID(id)),
		zap.String("actor_id", maskUUID(actorID)),
	)

	return &IdentityMapping{
		ActorID: actorID,
	}, nil
}

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}
