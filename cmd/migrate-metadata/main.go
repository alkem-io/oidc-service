// Command migrate-metadata backfills metadata_public on all Kratos identities.
//
// It lists every identity via the Kratos Admin API, resolves the Alkemio
// actor ID for each one, and patches metadata_public with alkemio_actor_id
// and alkemio_user_id. The operation is idempotent — safe to run multiple times.
//
// Uses the same OIDC_* environment variables as the main service.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	kratosClient "github.com/ory/client-go"
	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/alkemio"
	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/kratos"
	"github.com/alkem-io/oidc-service/internal/webhook"
	"github.com/alkem-io/oidc-service/pkg/telemetry"
)

const (
	pageSize    = 100
	concurrency = 5
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("load configuration: %v", err)
		return 1
	}

	logger, err := telemetry.NewLogger(cfg.LogLevel)
	if err != nil {
		log.Printf("initialise logger: %v", err)
		return 1
	}
	defer func() { _ = logger.Sync() }()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	kratosClient, err := kratos.NewClient(kratos.Config{
		AdminURL:  cfg.KratosAdminURL,
		AuthToken: cfg.AuthToken,
	})
	if err != nil {
		logger.Error("configure kratos client", zap.Error(err))
		return 1
	}

	kratosAdmin, err := webhook.NewKratosAdminClient(kratosClient.Admin())
	if err != nil {
		logger.Error("configure kratos admin client", zap.Error(err))
		return 1
	}

	resolver, closer, err := newResolver(ctx, cfg, logger)
	if err != nil {
		logger.Error("configure identity resolver", zap.Error(err))
		return 1
	}
	if closer != nil {
		defer func() { _ = closer.Close() }()
	}

	if err := migrate(ctx, logger, kratosAdmin, resolver); err != nil {
		logger.Error("migration failed", zap.Error(err))
		return 1
	}

	return 0
}

type closerFunc func() error

func (f closerFunc) Close() error { return f() }

func newResolver(ctx context.Context, cfg *config.ServiceConfig, logger *zap.Logger) (*alkemio.DatabaseResolver, closerFunc, error) {
	pool, err := alkemio.NewDatabasePool(ctx, alkemio.DatabaseConfig{
		DSN:     cfg.DatabaseDSN(),
		Timeout: cfg.DatabaseTimeout,
	}, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("database connection required for migration: %w", err)
	}

	resolver := alkemio.NewDatabaseResolver(pool, logger)
	closer := closerFunc(func() error { pool.Close(); return nil })

	return resolver, closer, nil
}

type migrationResult struct {
	identityID string
	err        error
}

type failedIdentity struct {
	id  string
	err string
}

func migrate(ctx context.Context, logger *zap.Logger, admin *webhook.KratosAdminClient, resolver *alkemio.DatabaseResolver) error {
	start := time.Now()

	var total, patched, skipped, failed atomic.Int64

	work := make(chan string, concurrency*2)
	results := make(chan migrationResult, concurrency*2)

	var wg sync.WaitGroup
	for range concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for identityID := range work {
				err := patchIdentity(ctx, admin, resolver, identityID)
				results <- migrationResult{identityID: identityID, err: err}
			}
		}()
	}

	// Collect results in a separate goroutine.
	var mu sync.Mutex
	var failures []failedIdentity

	var resultWg sync.WaitGroup
	resultWg.Add(1)
	go func() {
		defer resultWg.Done()
		for r := range results {
			if r.err != nil {
				if r.err == errAlreadyPatched {
					skipped.Add(1)
					logger.Debug("skipped (already patched)", zap.String("identity_id", maskID(r.identityID)))
				} else if r.err == errNotInDB {
					skipped.Add(1)
					logger.Info("skipped (no alkemio user)", zap.String("identity_id", r.identityID))
				} else {
					failed.Add(1)
					logger.Error("patch failed",
						zap.String("identity_id", r.identityID),
						zap.Error(r.err),
					)
					mu.Lock()
					failures = append(failures, failedIdentity{id: r.identityID, err: r.err.Error()})
					mu.Unlock()
				}
			} else {
				patched.Add(1)
				logger.Info("patched", zap.String("identity_id", maskID(r.identityID)))
			}
		}
	}()

	// Paginate through all identities.
	var pageToken string
	for {
		if ctx.Err() != nil {
			break
		}

		page, err := admin.ListIdentities(ctx, pageSize, pageToken)
		if err != nil {
			// Close work channel to stop workers, then wait.
			close(work)
			wg.Wait()
			close(results)
			resultWg.Wait()
			return fmt.Errorf("list identities: %w", err)
		}

		for _, identity := range page.Identities {
			total.Add(1)
			select {
			case work <- identity.Id:
			case <-ctx.Done():
			}
		}

		if page.NextToken == "" {
			break
		}
		pageToken = page.NextToken
	}

	close(work)
	wg.Wait()
	close(results)
	resultWg.Wait()

	logger.Info("migration complete",
		zap.Int64("total", total.Load()),
		zap.Int64("patched", patched.Load()),
		zap.Int64("skipped", skipped.Load()),
		zap.Int64("failed", failed.Load()),
		zap.Duration("elapsed", time.Since(start)),
	)

	if len(failures) > 0 {
		logger.Error("failed identities summary")
		for _, f := range failures {
			logger.Error("  failed identity",
				zap.String("identity_id", f.id),
				zap.String("reason", f.err),
			)
		}
		return fmt.Errorf("%d identities failed to patch", len(failures))
	}

	return nil
}

var (
	errAlreadyPatched = fmt.Errorf("already patched")
	errNotInDB        = fmt.Errorf("not in alkemio database")
)

func patchIdentity(ctx context.Context, admin *webhook.KratosAdminClient, resolver *alkemio.DatabaseResolver, identityID string) error {
	mapping, err := resolver.Resolve(ctx, identityID)
	if err != nil {
		if errors.Is(err, alkemio.ErrDBNotFound) {
			return errNotInDB
		}
		return fmt.Errorf("resolve: %w", err)
	}

	patches := []kratosClient.JsonPatch{
		{
			Op:   "add",
			Path: "/metadata_public",
			Value: map[string]interface{}{
				"alkemio_actor_id": mapping.ActorID,
				"alkemio_user_id":  mapping.ActorID,
			},
		},
	}

	if err := admin.PatchIdentity(ctx, identityID, patches); err != nil {
		return fmt.Errorf("patch: %w", err)
	}

	return nil
}

func maskID(id string) string {
	if len(id) <= 8 {
		return "***"
	}
	return id[:8] + "..."
}
