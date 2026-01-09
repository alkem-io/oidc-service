package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	kratosClient "github.com/ory/client-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/alkem-io/oidc-service/internal/alkemio"
)

const (
	testIdentityID = "8c0b7f5a-4dce-4d13-b6ad-6b2df2c1d10c"
	testActorID    = "1bcf5bd1-5f3e-4f01-9125-0edc93e5f5b1"
	testAgentID    = "6f4ed2d2-0ad0-4b83-8d43-8d9b9b4970b3"
)

type resolverStub struct {
	resolve func(ctx context.Context, authenticationID string) (*alkemio.IdentityMapping, error)
}

func (s resolverStub) Resolve(ctx context.Context, authenticationID string) (*alkemio.IdentityMapping, error) {
	if s.resolve != nil {
		return s.resolve(ctx, authenticationID)
	}
	return nil, errors.New("resolve not stubbed")
}

type kratosAdminStub struct {
	patchIdentity func(ctx context.Context, identityID string, patches []kratosClient.JsonPatch) error
}

func (s kratosAdminStub) PatchIdentity(ctx context.Context, identityID string, patches []kratosClient.JsonPatch) error {
	if s.patchIdentity != nil {
		return s.patchIdentity(ctx, identityID, patches)
	}
	return nil
}

func TestNewHandlerRequiresResolver(t *testing.T) {
	_, err := NewHandler(HandlerConfig{
		Kratos: kratosAdminStub{},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "resolver is required")
}

func TestNewHandlerRequiresKratos(t *testing.T) {
	_, err := NewHandler(HandlerConfig{
		Resolver: resolverStub{},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "kratos admin client is required")
}

// PostLogin tests - calls Kratos Admin API to patch identity

func TestPostLoginSuccess(t *testing.T) {
	var patchedIdentityID string
	var patchedValue map[string]interface{}

	resolverStub := resolverStub{
		resolve: func(_ context.Context, authenticationID string) (*alkemio.IdentityMapping, error) {
			require.Equal(t, testIdentityID, authenticationID)
			return &alkemio.IdentityMapping{
				UserID:  testActorID,
				AgentID: testAgentID,
			}, nil
		},
	}

	kratosStub := kratosAdminStub{
		patchIdentity: func(_ context.Context, identityID string, patches []kratosClient.JsonPatch) error {
			patchedIdentityID = identityID
			require.Len(t, patches, 1)
			require.Equal(t, "add", patches[0].Op)
			require.Equal(t, "/metadata_public", patches[0].Path)
			patchedValue = patches[0].Value.(map[string]interface{})
			return nil
		},
	}

	handler, err := NewHandler(HandlerConfig{
		Resolver: resolverStub,
		Kratos:   kratosStub,
		Logger:   zap.NewNop(),
	})
	require.NoError(t, err)

	body := `{"identity_id":"` + testIdentityID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/kratos/post-login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.PostLogin(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, testIdentityID, patchedIdentityID)
	require.Equal(t, testActorID, patchedValue["alkemio_actor_id"])
	require.Equal(t, testAgentID, patchedValue["alkemio_agent_id"])
}

func TestPostLoginPatchFailureReturns500(t *testing.T) {
	resolverStub := resolverStub{
		resolve: func(_ context.Context, _ string) (*alkemio.IdentityMapping, error) {
			return &alkemio.IdentityMapping{
				UserID:  testActorID,
				AgentID: testAgentID,
			}, nil
		},
	}

	kratosStub := kratosAdminStub{
		patchIdentity: func(_ context.Context, _ string, _ []kratosClient.JsonPatch) error {
			return errors.New("kratos api error")
		},
	}

	handler, err := NewHandler(HandlerConfig{
		Resolver: resolverStub,
		Kratos:   kratosStub,
		Logger:   zap.NewNop(),
	})
	require.NoError(t, err)

	body := `{"identity_id":"` + testIdentityID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/kratos/post-login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.PostLogin(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)

	var resp ErrorResponse
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	require.Equal(t, "patch_failed", resp.Error)
}

func TestPostLoginResolverErrorReturns500(t *testing.T) {
	resolverStub := resolverStub{
		resolve: func(_ context.Context, _ string) (*alkemio.IdentityMapping, error) {
			return nil, errors.New("database connection failed")
		},
	}

	kratosStub := kratosAdminStub{
		patchIdentity: func(_ context.Context, _ string, _ []kratosClient.JsonPatch) error {
			t.Fatal("patch should not be called when resolver fails")
			return nil
		},
	}

	handler, err := NewHandler(HandlerConfig{
		Resolver: resolverStub,
		Kratos:   kratosStub,
		Logger:   zap.NewNop(),
	})
	require.NoError(t, err)

	body := `{"identity_id":"` + testIdentityID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/kratos/post-login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.PostLogin(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)

	var resp ErrorResponse
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	require.Equal(t, "resolution_failed", resp.Error)
}

func TestPostLoginTimeoutReturns500(t *testing.T) {
	resolverStub := resolverStub{
		resolve: func(_ context.Context, _ string) (*alkemio.IdentityMapping, error) {
			return nil, context.DeadlineExceeded
		},
	}

	handler, err := NewHandler(HandlerConfig{
		Resolver: resolverStub,
		Kratos:   kratosAdminStub{},
		Logger:   zap.NewNop(),
	})
	require.NoError(t, err)

	body := `{"identity_id":"` + testIdentityID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/kratos/post-login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.PostLogin(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)

	var resp ErrorResponse
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	require.Equal(t, "resolution_failed", resp.Error)
}

func TestPostLoginMalformedJSONReturns400(t *testing.T) {
	handler, err := NewHandler(HandlerConfig{
		Resolver: resolverStub{},
		Kratos:   kratosAdminStub{},
		Logger:   zap.NewNop(),
	})
	require.NoError(t, err)

	body := `{not valid json}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/kratos/post-login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.PostLogin(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var resp ErrorResponse
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	require.Equal(t, "bad_request", resp.Error)
}
