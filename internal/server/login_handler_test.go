package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/challenge"
)

const (
	defaultLoginChallengePath = "/v1/oidc/login?login_challenge=test"
	forwardedProtoHeader      = "X-Forwarded-Proto"
	forwardedPrefixHeader     = "X-Forwarded-Prefix"
	appExampleHost            = "app.example"
	kratosBrowserPublicURL    = "https://kratos.example/ory/kratos/public"
	forwardedPrefixValue      = "/ory/kratos/public"
)

func TestLoginHandlerRedirectsOnSuccess(t *testing.T) {
	svc := &challengeServiceStub{
		resolveLogin: func(_ context.Context, challengeID string) (*challenge.Resolution, error) {
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
		resolveLogin: func(_ context.Context, challengeID string) (*challenge.Resolution, error) {
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

func TestLoginHandlerReturnsAlkemioErrors(t *testing.T) {
	testCases := []struct {
		name       string
		errorFn    func(challengeID string) challenge.Error
		statusCode int
		code       string
	}{
		{
			name:       "IdentityMissing",
			errorFn:    func(id string) challenge.Error { return challenge.NewAlkemioIdentityMissingError(id) },
			statusCode: http.StatusForbidden,
			code:       "alkemio_identity_missing",
		},
		{
			name:       "ResolutionFailed",
			errorFn:    func(id string) challenge.Error { return challenge.NewAlkemioResolutionError(id, "resolution failed") },
			statusCode: http.StatusBadGateway,
			code:       "alkemio_resolution_failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &challengeServiceStub{
				resolveLogin: func(_ context.Context, challengeID string) (*challenge.Resolution, error) {
					return nil, tc.errorFn(challengeID)
				},
			}

			handler := NewLoginHandler(LoginHandlerConfig{Logger: zap.NewNop(), Challenge: svc})

			req := httptest.NewRequest(http.MethodGet, "/v1/oidc/login?login_challenge=alkemio", nil)
			rec := httptest.NewRecorder()

			handler.Handle(rec, req)

			require.Equal(t, tc.statusCode, rec.Code)

			var payload map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
			require.Equal(t, tc.code, payload["error"])
		})
	}
}

func TestLoginHandlerProvidesSessionHint(t *testing.T) {
	var resolverCalled bool
	handler := NewLoginHandler(LoginHandlerConfig{
		Logger: zap.NewNop(),
		Challenge: &challengeServiceStub{
			resolveLogin: func(ctx context.Context, _ string) (*challenge.Resolution, error) {
				provider := challenge.IdentityHintProviderFromContext(ctx)
				require.NotNil(t, provider)
				id, err := provider.IdentityHint(ctx)
				require.NoError(t, err)
				require.Equal(t, "identity-id", id)
				return &challenge.Resolution{RedirectURL: "https://redirect"}, nil
			},
		},
		SessionResolver: sessionResolverStub{resolve: func(_ context.Context, session string) (string, error) {
			resolverCalled = true
			require.Equal(t, "session-token", session)
			return "identity-id", nil
		}},
	})

	req := httptest.NewRequest(http.MethodGet, defaultLoginChallengePath, nil)
	req.AddCookie(&http.Cookie{Name: "ory_kratos_session", Value: "session-token"}) //nolint:gosec // G124: outgoing request cookie in test; attributes are response-only
	rec := httptest.NewRecorder()

	handler.Handle(rec, req)

	require.True(t, resolverCalled)
	require.Equal(t, http.StatusFound, rec.Code)
}

func TestLoginHandlerMissingSessionCreatesHintProvider(t *testing.T) {
	handler := NewLoginHandler(LoginHandlerConfig{
		Logger: zap.NewNop(),
		Challenge: &challengeServiceStub{
			resolveLogin: func(ctx context.Context, cid string) (*challenge.Resolution, error) {
				provider := challenge.IdentityHintProviderFromContext(ctx)
				require.NotNil(t, provider)
				_, err := provider.IdentityHint(ctx)
				require.Error(t, err)
				require.ErrorIs(t, err, challenge.ErrIdentitySessionRequired)
				return nil, challenge.NewSessionRequiredError(cid)
			},
		},
		SessionResolver: sessionResolverStub{},
	})

	req := httptest.NewRequest(http.MethodGet, defaultLoginChallengePath, nil)
	rec := httptest.NewRecorder()

	handler.Handle(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, "session_required", payload["error"])
}

func TestLoginHandlerRedirectsToKratosLoginWhenSessionRequired(t *testing.T) {
	handler := NewLoginHandler(LoginHandlerConfig{
		Logger: zap.NewNop(),
		Challenge: &challengeServiceStub{
			resolveLogin: func(_ context.Context, challengeID string) (*challenge.Resolution, error) {
				return nil, challenge.NewSessionRequiredError(challengeID)
			},
		},
		KratosBrowserURL: "https://kratos.example",
	})

	req := httptest.NewRequest(http.MethodGet, defaultLoginChallengePath, nil)
	req.Host = appExampleHost
	req.Header.Set(forwardedProtoHeader, "https")
	rec := httptest.NewRecorder()

	handler.Handle(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)
	require.Equal(t, "https://kratos.example/self-service/login/browser?return_to=https%3A%2F%2Fapp.example%2Fv1%2Foidc%2Flogin%3Flogin_challenge%3Dtest", rec.Header().Get("Location"))
}

func TestLoginHandlerRedirectsToKratosLoginWhenSessionInvalid(t *testing.T) {
	handler := NewLoginHandler(LoginHandlerConfig{
		Logger: zap.NewNop(),
		Challenge: &challengeServiceStub{
			resolveLogin: func(_ context.Context, challengeID string) (*challenge.Resolution, error) {
				return nil, challenge.NewSessionInvalidError(challengeID)
			},
		},
		KratosBrowserURL: "https://kratos.example",
	})

	req := httptest.NewRequest(http.MethodGet, defaultLoginChallengePath, nil)
	req.Host = appExampleHost
	req.Header.Set(forwardedProtoHeader, "https")
	rec := httptest.NewRecorder()

	handler.Handle(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)
	require.Equal(t, "https://kratos.example/self-service/login/browser?return_to=https%3A%2F%2Fapp.example%2Fv1%2Foidc%2Flogin%3Flogin_challenge%3Dtest", rec.Header().Get("Location"))
}

func TestLoginHandlerRedirectsToKratosLoginWithForwardedPrefix(t *testing.T) {
	handler := NewLoginHandler(LoginHandlerConfig{
		Logger: zap.NewNop(),
		Challenge: &challengeServiceStub{
			resolveLogin: func(_ context.Context, challengeID string) (*challenge.Resolution, error) {
				return nil, challenge.NewSessionRequiredError(challengeID)
			},
		},
		KratosBrowserURL: kratosBrowserPublicURL,
	})

	req := httptest.NewRequest(http.MethodGet, defaultLoginChallengePath, nil)
	req.Host = appExampleHost
	req.Header.Set(forwardedProtoHeader, "https")
	req.Header.Set(forwardedPrefixHeader, forwardedPrefixValue)
	rec := httptest.NewRecorder()

	handler.Handle(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)
	expected := kratosBrowserPublicURL + "/self-service/login/browser?return_to=https%3A%2F%2Fapp.example%2Fory%2Fkratos%2Fpublic%2Fv1%2Foidc%2Flogin%3Flogin_challenge%3Dtest"
	require.Equal(t, expected, rec.Header().Get("Location"))
}

func TestLoginHandlerRedirectsWithReturnBaseURLOverride(t *testing.T) {
	handler := NewLoginHandler(LoginHandlerConfig{
		Logger: zap.NewNop(),
		Challenge: &challengeServiceStub{
			resolveLogin: func(_ context.Context, challengeID string) (*challenge.Resolution, error) {
				return nil, challenge.NewSessionRequiredError(challengeID)
			},
		},
		KratosBrowserURL: kratosBrowserPublicURL,
		ReturnBaseURL:    "https://proxy.example/oidc/login",
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/login?login_challenge=test&foo=bar", nil)
	req.Host = "internal.example"
	req.Header.Set(forwardedProtoHeader, "https")
	rec := httptest.NewRecorder()

	handler.Handle(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)
	expected := kratosBrowserPublicURL + "/self-service/login/browser?return_to=https%3A%2F%2Fproxy.example%2Foidc%2Flogin%3Ffoo%3Dbar%26login_challenge%3Dtest"
	require.Equal(t, expected, rec.Header().Get("Location"))
}

func TestLoginHandlerRedirectsToKratosLoginWithLoopbackHost(t *testing.T) {
	handler := NewLoginHandler(LoginHandlerConfig{
		Logger: zap.NewNop(),
		Challenge: &challengeServiceStub{
			resolveLogin: func(_ context.Context, challengeID string) (*challenge.Resolution, error) {
				return nil, challenge.NewSessionRequiredError(challengeID)
			},
		},
		KratosBrowserURL: kratosBrowserPublicURL,
	})

	req := httptest.NewRequest(http.MethodGet, "/oidc/login?login_challenge=test", nil)
	req.Host = "localhost:3000"
	req.Header.Set("X-Forwarded-Proto", "http")
	rec := httptest.NewRecorder()

	handler.Handle(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)
	expected := kratosBrowserPublicURL + "/self-service/login/browser?return_to=https%3A%2F%2Fkratos.example%2Foidc%2Flogin%3Flogin_challenge%3Dtest"
	require.Equal(t, expected, rec.Header().Get("Location"))
}

func TestLoginHandlerRedirectsToKratosLoginWithBrowserPathPrefix(t *testing.T) {
	handler := NewLoginHandler(LoginHandlerConfig{
		Logger: zap.NewNop(),
		Challenge: &challengeServiceStub{
			resolveLogin: func(_ context.Context, challengeID string) (*challenge.Resolution, error) {
				return nil, challenge.NewSessionRequiredError(challengeID)
			},
		},
		KratosBrowserURL: kratosBrowserPublicURL,
	})

	req := httptest.NewRequest(http.MethodGet, defaultLoginChallengePath, nil)
	req.Host = appExampleHost
	req.Header.Set(forwardedProtoHeader, "https")
	rec := httptest.NewRecorder()

	handler.Handle(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)
	expected := kratosBrowserPublicURL + "/self-service/login/browser?return_to=https%3A%2F%2Fapp.example%2Fv1%2Foidc%2Flogin%3Flogin_challenge%3Dtest"
	require.Equal(t, expected, rec.Header().Get("Location"))
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
