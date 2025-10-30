package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alkem-io/oidc-service/internal/challenge"
	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/maintenance"
	"github.com/alkem-io/oidc-service/internal/server"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestSecretsRedactedFromLogsAndReadiness(t *testing.T) {
	secret := "super-secret-token"
	redirect := "https://example.com/callback?code=" + secret + "&state=opaque"

	svc := &redactionChallengeService{
		redirect: redirect,
		readiness: challenge.ReadinessState{
			Status:  "ready",
			Hydra:   "ok",
			Kratos:  "ok",
			Version: "1.0.0",
		},
	}

	core, logs := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	router := server.NewRouter(server.Options{
		Logger:      logger,
		Maintenance: maintenance.NewState(config.MaintenanceState{}),
		Challenge:   svc,
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/login?login_challenge=test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect status, got %d", rec.Code)
	}

	sanitized := "https://example.com/callback"
	entries := logs.All()
	if len(entries) == 0 {
		t.Fatal("expected log entries for login resolution")
	}

	found := false
	for _, entry := range entries {
		if strings.Contains(entry.Message, secret) {
			t.Fatalf("log message leaked secret: %s", entry.Message)
		}

		ctxMap := entry.ContextMap()
		for key, value := range ctxMap {
			if str, ok := value.(string); ok && strings.Contains(str, secret) {
				t.Fatalf("log field %s leaked secret: %s", key, str)
			}
		}

		if entry.Message == "login challenge resolved" {
			found = true
			if value, ok := ctxMap["redirectTo"].(string); !ok || value != sanitized {
				t.Fatalf("expected sanitized redirect, got %v", ctxMap["redirectTo"])
			}
		}
	}

	if !found {
		t.Fatal("expected login challenge resolved log entry")
	}

	readyReq := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	readyRec := httptest.NewRecorder()
	router.ServeHTTP(readyRec, readyReq)

	if readyRec.Code != http.StatusOK {
		t.Fatalf("expected readiness status 200, got %d", readyRec.Code)
	}

	if strings.Contains(readyRec.Body.String(), secret) {
		t.Fatalf("readiness payload leaked secret: %s", readyRec.Body.String())
	}
}

type redactionChallengeService struct {
	redirect  string
	readiness challenge.ReadinessState
}

func (s *redactionChallengeService) ResolveLogin(_ context.Context, _ string) (*challenge.Resolution, error) {
	return &challenge.Resolution{RedirectURL: s.redirect}, nil
}

func (s *redactionChallengeService) ResolveConsent(_ context.Context, _ string) (*challenge.Resolution, error) {
	return &challenge.Resolution{RedirectURL: s.redirect}, nil
}

func (s *redactionChallengeService) Readiness(_ context.Context) challenge.ReadinessState {
	return s.readiness
}
