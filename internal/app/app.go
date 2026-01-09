// Package app provides the application lifecycle management for the OIDC service.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/alkemio"
	"github.com/alkem-io/oidc-service/internal/challenge"
	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/hydra"
	"github.com/alkem-io/oidc-service/internal/kratos"
	"github.com/alkem-io/oidc-service/internal/maintenance"
	"github.com/alkem-io/oidc-service/internal/server"
	"github.com/alkem-io/oidc-service/internal/webhook"
	"github.com/alkem-io/oidc-service/pkg/telemetry"
)

const (
	defaultAddr         = ":8080"
	shutdownGracePeriod = 15 * time.Second
)

// App encapsulates the OIDC service application lifecycle.
type App struct {
	logger  *zap.Logger
	server  *http.Server
	closers []io.Closer
}

// New constructs the application with all dependencies wired.
func New(cfg *config.ServiceConfig) (*App, error) {
	logger, err := telemetry.NewLogger(cfg.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("initialise logger: %w", err)
	}

	app := &App{logger: logger}

	oryClients, err := newOryClients(cfg)
	if err != nil {
		return nil, err
	}

	identityResolver, dbCloser, err := newIdentityResolver(cfg, logger)
	if err != nil {
		return nil, err
	}
	if dbCloser != nil {
		app.closers = append(app.closers, dbCloser)
	}

	challengeSvc, err := newChallengeService(cfg, logger, oryClients, identityResolver)
	if err != nil {
		return nil, err
	}

	app.server = newHTTPServer(cfg, logger, oryClients, challengeSvc, identityResolver)

	return app, nil
}

// Run starts the HTTP server and blocks until shutdown signal is received.
func (a *App) Run() error {
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	go a.handleShutdown(shutdownCh)

	a.logger.Info("starting oidc service", zap.String("addr", defaultAddr))

	err := a.server.ListenAndServe()

	// Clean up signal subscription and unblock handleShutdown goroutine
	signal.Stop(shutdownCh)
	close(shutdownCh)

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server: %w", err)
	}

	return nil
}

// Close performs graceful shutdown of all resources.
// Resources are closed in reverse order (like defer semantics).
func (a *App) Close() error {
	var errs []error

	// Close resources in reverse order for proper teardown
	for i := len(a.closers) - 1; i >= 0; i-- {
		if err := a.closers[i].Close(); err != nil {
			errs = append(errs, err)
		}
	}

	// Sync logger, ignoring spurious errors from stdout/stderr
	if err := a.logger.Sync(); err != nil && !isSyncIgnorable(err) {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

// isSyncIgnorable returns true for errors that commonly occur when syncing
// zap loggers to stdout/stderr (e.g., on Linux terminals).
func isSyncIgnorable(err error) bool {
	// EINVAL and ENOTTY occur when syncing to terminals
	return errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.ENOTTY)
}

func (a *App) handleShutdown(shutdownCh <-chan os.Signal) {
	sig, ok := <-shutdownCh
	if !ok {
		// Channel was closed, server already stopped
		return
	}
	a.logger.Info("received shutdown signal", zap.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), shutdownGracePeriod)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		a.logger.Error("server shutdown failed", zap.Error(err))
	}
}

// oryClients groups Ory-related clients (Hydra, Kratos).
type oryClients struct {
	hydra           *hydra.Client
	kratos          *kratos.Client
	sessionResolver *kratos.SessionResolver
}

func newOryClients(cfg *config.ServiceConfig) (*oryClients, error) {
	hydraClient, err := hydra.NewClient(hydra.Config{
		AdminURL:  cfg.HydraAdminURL,
		AuthToken: cfg.AuthToken,
	})
	if err != nil {
		return nil, fmt.Errorf("configure hydra client: %w", err)
	}

	kratosClient, err := kratos.NewClient(kratos.Config{
		AdminURL:  cfg.KratosAdminURL,
		AuthToken: cfg.AuthToken,
	})
	if err != nil {
		return nil, fmt.Errorf("configure kratos client: %w", err)
	}

	sessionResolver, err := kratos.NewSessionResolver(kratos.SessionConfig{
		PublicURL:  cfg.KratosPublicURL,
		Timeout:    cfg.ReadinessTimeout,
		CookieName: cfg.KratosSessionCookie,
	})
	if err != nil {
		return nil, fmt.Errorf("configure kratos session resolver: %w", err)
	}

	return &oryClients{
		hydra:           hydraClient,
		kratos:          kratosClient,
		sessionResolver: sessionResolver,
	}, nil
}

func newIdentityResolver(cfg *config.ServiceConfig, logger *zap.Logger) (*alkemio.CompositeResolver, io.Closer, error) {
	apiResolver, err := alkemio.NewIdentityResolver(alkemio.Config{
		BaseURL:     cfg.AlkemioServerURL,
		ResolvePath: cfg.AlkemioResolvePath,
		Timeout:     cfg.IdentityTimeout,
		MaxRetries:  cfg.IdentityMaxRetries,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("configure alkemio identity resolver: %w", err)
	}

	dbPool, dbResolver, err := newDatabaseResolver(cfg, logger)
	if err != nil {
		// Non-fatal: log and continue with API-only mode
		logger.Warn("database pool initialization failed, using API-only resolution", zap.Error(err))
	}

	composite, err := alkemio.NewCompositeResolver(alkemio.CompositeConfig{
		Database: dbResolver,
		API:      apiResolver,
		Logger:   logger,
	})
	if err != nil {
		if dbPool != nil {
			_ = dbPool.Close()
		}
		return nil, nil, fmt.Errorf("configure composite identity resolver: %w", err)
	}

	return composite, dbPool, nil
}

// poolCloser wraps pgxpool.Pool to implement io.Closer.
type poolCloser struct {
	pool *pgxpool.Pool
}

// Close closes the underlying connection pool.
func (p *poolCloser) Close() error {
	p.pool.Close()
	return nil
}

func newDatabaseResolver(cfg *config.ServiceConfig, logger *zap.Logger) (io.Closer, *alkemio.DatabaseResolver, error) {
	dbPool, err := alkemio.NewDatabasePool(context.Background(), alkemio.DatabaseConfig{
		DSN:     cfg.DatabaseDSN(),
		Timeout: cfg.DatabaseTimeout,
	}, logger)
	if err != nil {
		return nil, nil, err
	}

	return &poolCloser{pool: dbPool}, alkemio.NewDatabaseResolver(dbPool, logger), nil
}

func newChallengeService(
	cfg *config.ServiceConfig,
	logger *zap.Logger,
	ory *oryClients,
	identityResolver *alkemio.CompositeResolver,
) (challenge.Service, error) {
	hydraProbe, err := challenge.NewHTTPReadinessProbe(
		cfg.HydraAdminURL, "/health/ready", "/version", cfg.AuthToken, cfg.ReadinessTimeout,
	)
	if err != nil {
		return nil, fmt.Errorf("configure hydra readiness probe: %w", err)
	}

	kratosProbe, err := challenge.NewHTTPReadinessProbe(
		cfg.KratosAdminURL, "/admin/health/ready", "/version", cfg.AuthToken, cfg.ReadinessTimeout,
	)
	if err != nil {
		return nil, fmt.Errorf("configure kratos readiness probe: %w", err)
	}

	hydraOAuthClient, err := challenge.NewHydraOAuth2Client(ory.hydra.Admin().OAuth2API)
	if err != nil {
		return nil, fmt.Errorf("configure hydra oauth client: %w", err)
	}

	svc, err := challenge.NewService(challenge.Options{
		Hydra:            hydraOAuthClient,
		Identity:         challenge.NewIdentityMapper(ory.kratos.Admin().IdentityAPI),
		Alkemio:          identityResolver,
		HydraProbe:       hydraProbe,
		KratosProbe:      kratosProbe,
		ReadinessTimeout: cfg.ReadinessTimeout,
		Logger:           challenge.NewZapLoggerAdapter(logger),
	})
	if err != nil {
		return nil, fmt.Errorf("configure challenge service: %w", err)
	}

	return svc, nil
}

func newHTTPServer(
	cfg *config.ServiceConfig,
	logger *zap.Logger,
	ory *oryClients,
	challengeSvc challenge.Service,
	identityResolver *alkemio.CompositeResolver,
) *http.Server {
	maint := maintenance.NewState(cfg.Maintenance())

	kratosAdmin := webhook.NewKratosAdminClient(ory.kratos.Admin())
	webhookHandler, err := webhook.NewHandler(webhook.HandlerConfig{
		Resolver: identityResolver,
		Kratos:   kratosAdmin,
		Logger:   logger,
	})
	if err != nil {
		logger.Warn("failed to create webhook handler", zap.Error(err))
	}

	handler := server.NewRouter(server.Options{
		Logger:             logger,
		Maintenance:        maint,
		Challenge:          challengeSvc,
		SessionResolver:    ory.sessionResolver,
		SessionCookie:      cfg.KratosSessionCookie,
		KratosBrowserURL:   cfg.KratosBrowserURL,
		LoginReturnBaseURL: cfg.LoginReturnBaseURL,
		WebhookHandler:     webhookHandler,
	})

	return &http.Server{
		Addr:              defaultAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}
