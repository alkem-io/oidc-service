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

func TestConsentHandlerRedirectsOnSuccess(t *testing.T) {
	svc := &challengeServiceStub{
		resolveConsent: func(_ context.Context, challengeID string) (*challenge.Resolution, error) {
			require.Equal(t, "consent-123", challengeID)
			return &challenge.Resolution{RedirectURL: "https://consent.redirect"}, nil
		},
	}

	handler := NewConsentHandler(zap.NewNop(), svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=consent-123", nil)
	rec := httptest.NewRecorder()

	handler.Handle(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)
	require.Equal(t, "https://consent.redirect", rec.Header().Get("Location"))
}

func TestConsentHandlerMissingChallengeReturnsBadRequest(t *testing.T) {
	handler := NewConsentHandler(zap.NewNop(), &challengeServiceStub{})

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent", nil)
	rec := httptest.NewRecorder()

	handler.Handle(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, "missing_challenge", payload["error"])
}

func TestConsentHandlerPropagatesServiceError(t *testing.T) {
	svc := &challengeServiceStub{
		resolveConsent: func(_ context.Context, challengeID string) (*challenge.Resolution, error) {
			return nil, challenge.NewInvalidChallengeError(challengeID)
		},
	}

	handler := NewConsentHandler(zap.NewNop(), svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=missing", nil)
	rec := httptest.NewRecorder()

	handler.Handle(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, "invalid_challenge", payload["error"])
}

func TestConsentHandlerReturnsAlkemioErrors(t *testing.T) {
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
				resolveConsent: func(_ context.Context, challengeID string) (*challenge.Resolution, error) {
					return nil, tc.errorFn(challengeID)
				},
			}

			handler := NewConsentHandler(zap.NewNop(), svc)

			req := httptest.NewRequest(http.MethodGet, "/v1/oidc/consent?consent_challenge=alkemio", nil)
			rec := httptest.NewRecorder()

			handler.Handle(rec, req)

			require.Equal(t, tc.statusCode, rec.Code)

			var payload map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
			require.Equal(t, tc.code, payload["error"])
		})
	}
}
