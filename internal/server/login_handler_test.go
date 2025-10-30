package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alkem-io/oidc-service/internal/challenge"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLoginHandlerRedirectsOnSuccess(t *testing.T) {
	svc := &challengeServiceStub{
		resolveLogin: func(ctx context.Context, challengeID string) (*challenge.Resolution, error) {
			require.Equal(t, "test-challenge", challengeID)
			return &challenge.Resolution{RedirectURL: "https://redirect.example"}, nil
		},
	}

	handler := NewLoginHandler(zap.NewNop(), svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/login?login_challenge=test-challenge", nil)
	rec := httptest.NewRecorder()

	handler.Handle(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)
	require.Equal(t, "https://redirect.example", rec.Header().Get("Location"))
}

func TestLoginHandlerMissingChallengeReturnsBadRequest(t *testing.T) {
	handler := NewLoginHandler(zap.NewNop(), &challengeServiceStub{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/login", nil)
	rec := httptest.NewRecorder()

	handler.Handle(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, "missing_challenge", payload["error"])
}

func TestLoginHandlerPropagatesServiceError(t *testing.T) {
	svc := &challengeServiceStub{
		resolveLogin: func(ctx context.Context, challengeID string) (*challenge.Resolution, error) {
			return nil, challenge.NewInvalidChallengeError(challengeID)
		},
	}

	handler := NewLoginHandler(zap.NewNop(), svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/login?login_challenge=missing", nil)
	rec := httptest.NewRecorder()

	handler.Handle(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, "invalid_challenge", payload["error"])
}

func TestLoginHandlerRecordsMetrics(t *testing.T) {
	metrics := &challengeMetricsStub{}
	handler := NewLoginHandler(zap.NewNop(), &challengeServiceStub{
		resolveLogin: func(ctx context.Context, challengeID string) (*challenge.Resolution, error) {
			return &challenge.Resolution{RedirectURL: "https://redirect"}, nil
		},
	}, metrics)

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/login?login_challenge=test", nil)
	rec := httptest.NewRecorder()

	handler.Handle(rec, req)

	require.Equal(t, "login", metrics.flow)
	require.Equal(t, "success", metrics.outcome)
	require.Equal(t, "none", metrics.errorCode)
}

type challengeMetricsStub struct {
	flow      string
	outcome   string
	errorCode string
}

func (s *challengeMetricsStub) ObserveChallenge(flow, outcome, errorCode string, _ time.Duration) {
	s.flow = flow
	s.outcome = outcome
	s.errorCode = errorCode
}
