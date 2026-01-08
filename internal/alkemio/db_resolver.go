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
}

// NewDatabaseResolver creates a resolver that queries the database for identity mappings.
func NewDatabaseResolver(pool *pgxpool.Pool) *DatabaseResolver {
	return &DatabaseResolver{
		queries: queries.New(pool),
	}
}

// Resolve looks up the Alkemio UserID and AgentID for the given authenticationID.
func (r *DatabaseResolver) Resolve(ctx context.Context, authenticationID string) (*IdentityMapping, error) {
	if r == nil || r.queries == nil {
		return nil, errors.New("database resolver is not initialized")
	}

	id := strings.TrimSpace(authenticationID)
	if id == "" {
		return nil, errors.New("authentication id is required")
	}

	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("authentication id must be a valid uuid: %w", err)
	}

	authUUID := pgtype.UUID{
		Bytes: parsedUUID,
		Valid: true,
	}

	row, err := r.queries.GetUserByAuthenticationID(ctx, authUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDBNotFound
		}
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	if !row.ID.Valid || !row.AgentId.Valid {
		return nil, ErrDBNotFound
	}

	userID := uuidToString(row.ID)
	agentID := uuidToString(row.AgentId)

	if userID == "" || agentID == "" {
		return nil, ErrDBNotFound
	}

	if !isUUID(userID) {
		return nil, fmt.Errorf("database returned invalid user id")
	}
	if !isUUID(agentID) {
		return nil, fmt.Errorf("database returned invalid agent id")
	}

	return &IdentityMapping{
		UserID:  userID,
		AgentID: agentID,
	}, nil
}

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}
