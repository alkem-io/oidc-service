package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alkem-io/oidc-service/internal/alkemio"
	"github.com/alkem-io/oidc-service/internal/challenge"
	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/hydra"
	"github.com/alkem-io/oidc-service/internal/kratos"
	"github.com/alkem-io/oidc-service/internal/maintenance"
	"github.com/alkem-io/oidc-service/internal/server"
	"github.com/alkem-io/oidc-service/pkg/telemetry"
	"go.uber.org/zap"
)

const (
	defaultAddr         = ":8080"
	shutdownGracePeriod = 15 * time.Second
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fatal("load configuration", err)
	}

	logger, err := telemetry.NewLogger(cfg.LogLevel)
	if err != nil {
		fatal("initialise logger", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	maint := maintenance.NewState(cfg.Maintenance())

	hydraClient, err := hydra.NewClient(hydra.Config{
		AdminURL:  cfg.HydraAdminURL,
		AuthToken: cfg.AuthToken,
	})
	if err != nil {
		fatal("configure hydra client", err)
	}

	kratosClient, err := kratos.NewClient(kratos.Config{
		AdminURL:  cfg.KratosAdminURL,
		AuthToken: cfg.AuthToken,
	})
	if err != nil {
		fatal("configure kratos client", err)
	}

	sessionResolver, err := kratos.NewSessionResolver(kratos.SessionConfig{
		PublicURL:  cfg.KratosPublicURL,
		Timeout:    cfg.ReadinessTimeout,
		CookieName: cfg.KratosSessionCookie,
	})
	if err != nil {
		fatal("configure kratos session resolver", err)
	}

	hydraProbe, err := challenge.NewHTTPReadinessProbe(cfg.HydraAdminURL, "/health/ready", "/version", cfg.AuthToken, cfg.ReadinessTimeout)
	if err != nil {
		fatal("configure hydra readiness probe", err)
	}

	kratosProbe, err := challenge.NewHTTPReadinessProbe(cfg.KratosAdminURL, "/admin/health/ready", "/version", cfg.AuthToken, cfg.ReadinessTimeout)
	if err != nil {
		fatal("configure kratos readiness probe", err)
	}

	identityResolver, err := alkemio.NewIdentityResolver(alkemio.Config{
		BaseURL:     cfg.AlkemioServerURL,
		ResolvePath: cfg.AlkemioResolvePath,
		Timeout:     cfg.IdentityTimeout,
		MaxRetries:  cfg.IdentityMaxRetries,
	})
	if err != nil {
		fatal("configure alkemio identity resolver", err)
	}

	hydraOAuthClient, err := challenge.NewHydraOAuth2Client(hydraClient.Admin().OAuth2API)
	if err != nil {
		fatal("configure hydra oauth client", err)
	}

	challengeService, err := challenge.NewService(challenge.Options{
		Hydra:            hydraOAuthClient,
		Identity:         challenge.NewIdentityMapper(kratosClient.Admin().IdentityAPI),
		Alkemio:          identityResolver,
		HydraProbe:       hydraProbe,
		KratosProbe:      kratosProbe,
		ReadinessTimeout: cfg.ReadinessTimeout,
		Logger:           challenge.NewZapLoggerAdapter(logger),
	})
	if err != nil {
		fatal("configure challenge service", err)
	}

	handler := server.NewRouter(server.Options{
		Logger:             logger,
		Maintenance:        maint,
		Challenge:          challengeService,
		SessionResolver:    sessionResolver,
		SessionCookie:      cfg.KratosSessionCookie,
		KratosBrowserURL:   cfg.KratosBrowserURL,
		LoginReturnBaseURL: cfg.LoginReturnBaseURL,
	})

	srv := &http.Server{
		Addr:              defaultAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-shutdownCh
		logger.Info("received shutdown signal", zap.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(context.Background(), shutdownGracePeriod)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server shutdown failed", zap.Error(err))
		}
	}()

	logger.Info("starting oidc service", zap.String("addr", defaultAddr))

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fatal("http server", err)
	}
}

func fatal(msg string, err error) {
	log.Fatalf("%s: %v", msg, err)
}
