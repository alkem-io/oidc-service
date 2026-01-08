package alkemio

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DatabaseConfig holds the settings for connecting to the PostgreSQL database.
type DatabaseConfig struct {
	DSN     string
	Timeout time.Duration
}

// NewDatabasePool creates a connection pool to the PostgreSQL database.
func NewDatabasePool(ctx context.Context, cfg DatabaseConfig) (*pgxpool.Pool, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("database DSN is required")
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	if cfg.Timeout > 0 {
		poolConfig.ConnConfig.ConnectTimeout = cfg.Timeout
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	return pool, nil
}
