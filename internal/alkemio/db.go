package alkemio

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// DatabaseConfig holds the settings for connecting to the PostgreSQL database.
// DSN is the connection string and Timeout specifies the connection timeout.
type DatabaseConfig struct {
	DSN     string
	Timeout time.Duration
}

// NewDatabasePool creates a connection pool to the PostgreSQL database.
// The pool is verified with a ping before being returned.
// The caller is responsible for closing the pool when done (call pool.Close()).
func NewDatabasePool(ctx context.Context, cfg DatabaseConfig, logger *zap.Logger) (*pgxpool.Pool, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

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

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database health check failed: %w", err)
	}

	logger.Info("database pool created",
		zap.String("host", poolConfig.ConnConfig.Host),
		zap.Uint16("port", poolConfig.ConnConfig.Port),
		zap.String("database", poolConfig.ConnConfig.Database),
	)

	return pool, nil
}
