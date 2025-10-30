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

	"github.com/alkem-io/oidc-service/internal/challenge"
	"github.com/alkem-io/oidc-service/internal/config"
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

	challengeService := challenge.NewStubService()

	handler := server.NewRouter(server.Options{
		Logger:      logger,
		Maintenance: maint,
		Challenge:   challengeService,
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
