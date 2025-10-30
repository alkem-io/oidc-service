package kratos

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alkem-io/oidc-service/internal/challenge"
	"github.com/stretchr/testify/require"
)

func TestSessionResolverIdentityIDSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/sessions/whoami", r.URL.Path)
		require.Contains(t, r.Header.Get("Cookie"), "ory_kratos_session=session-token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"identity":{"id":"identity-id"}}`))
	}))
	defer server.Close()

	resolver, err := NewSessionResolver(SessionConfig{PublicURL: server.URL, Timeout: time.Second})
	require.NoError(t, err)

	id, err := resolver.IdentityID(context.Background(), "session-token")
	require.NoError(t, err)
	require.Equal(t, "identity-id", id)
}

func TestSessionResolverIdentityIDUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	resolver, err := NewSessionResolver(SessionConfig{PublicURL: server.URL})
	require.NoError(t, err)

	id, err := resolver.IdentityID(context.Background(), "session-token")
	require.Error(t, err)
	require.Equal(t, "", id)
	require.ErrorIs(t, err, challenge.ErrIdentitySessionInvalid)
}

func TestSessionResolverIdentityIDServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	resolver, err := NewSessionResolver(SessionConfig{PublicURL: server.URL})
	require.NoError(t, err)

	_, err = resolver.IdentityID(context.Background(), "session-token")
	require.Error(t, err)
	require.NotErrorIs(t, err, challenge.ErrIdentitySessionInvalid)
}
