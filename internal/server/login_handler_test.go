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

	handler := NewLoginHandler(LoginHandlerConfig{
		Logger:    zap.NewNop(),
		Challenge: svc,
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/login?login_challenge=test-challenge", nil)
	rec := httptest.NewRecorder()

	handler.Handle(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)
	require.Equal(t, "https://redirect.example", rec.Header().Get("Location"))
}

func TestLoginHandlerMissingChallengeReturnsBadRequest(t *testing.T) {
	handler := NewLoginHandler(LoginHandlerConfig{
		Logger:    zap.NewNop(),
		Challenge: &challengeServiceStub{},
	})

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

	handler := NewLoginHandler(LoginHandlerConfig{
		Logger:    zap.NewNop(),
		Challenge: svc,
	})

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
	handler := NewLoginHandler(LoginHandlerConfig{
		Logger: zap.NewNop(),
		Challenge: &challengeServiceStub{
			resolveLogin: func(ctx context.Context, challengeID string) (*challenge.Resolution, error) {
				return &challenge.Resolution{RedirectURL: "https://redirect"}, nil
			},
		},
		Metrics: metrics,
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/login?login_challenge=test", nil)
	rec := httptest.NewRecorder()

	handler.Handle(rec, req)

	require.Equal(t, "login", metrics.flow)
	require.Equal(t, "success", metrics.outcome)
	require.Equal(t, "none", metrics.errorCode)
}

func TestLoginHandlerProvidesSessionHint(t *testing.T) {
	var resolverCalled bool
	handler := NewLoginHandler(LoginHandlerConfig{
		Logger: zap.NewNop(),
		Challenge: &challengeServiceStub{
			resolveLogin: func(ctx context.Context, challengeID string) (*challenge.Resolution, error) {
				provider := challenge.IdentityHintProviderFromContext(ctx)
				require.NotNil(t, provider)
				id, err := provider.IdentityHint(ctx)
				require.NoError(t, err)
				require.Equal(t, "identity-id", id)
				return &challenge.Resolution{RedirectURL: "https://redirect"}, nil
			},
		},
		SessionResolver: sessionResolverStub{resolve: func(ctx context.Context, session string) (string, error) {
			resolverCalled = true
			require.Equal(t, "session-token", session)
			return "identity-id", nil
		}},
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/login?login_challenge=test", nil)
	req.AddCookie(&http.Cookie{Name: "ory_kratos_session", Value: "session-token"})
	rec := httptest.NewRecorder()

	handler.Handle(rec, req)

	require.True(t, resolverCalled)
	require.Equal(t, http.StatusFound, rec.Code)
}

func TestLoginHandlerMissingSessionCreatesHintProvider(t *testing.T) {
	handler := NewLoginHandler(LoginHandlerConfig{
		Logger: zap.NewNop(),
		Challenge: &challengeServiceStub{
			resolveLogin: func(ctx context.Context, challengeID string) (*challenge.Resolution, error) {
				provider := challenge.IdentityHintProviderFromContext(ctx)
				require.NotNil(t, provider)
				_, err := provider.IdentityHint(ctx)
				require.Error(t, err)
				require.ErrorIs(t, err, challenge.ErrIdentitySessionRequired)
				return nil, challenge.NewSessionRequiredError(challengeID)
			},
		},
		SessionResolver: sessionResolverStub{},
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/login?login_challenge=test", nil)
	rec := httptest.NewRecorder()

	handler.Handle(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, "session_required", payload["error"])
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

type sessionResolverStub struct {
	resolve func(ctx context.Context, session string) (string, error)
}

func (s sessionResolverStub) IdentityID(ctx context.Context, session string) (string, error) {
	if s.resolve == nil {
		return "", challenge.ErrIdentitySessionRequired
	}
	return s.resolve(ctx, session)
}
