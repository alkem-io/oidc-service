package kratos

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/alkem-io/oidc-service/internal/challenge"
)

func TestSessionResolverResolve(t *testing.T) {
	testCases := []struct {
		name     string
		status   int
		body     string
		expected challenge.SessionAssurance
		err      error
	}{
		{
			name:     "aal2 session",
			status:   http.StatusOK,
			body:     `{"identity":{"id":"identity-id"},"authenticator_assurance_level":"aal2"}`,
			expected: challenge.SessionAssurance{IdentityID: "identity-id", AAL: "aal2"},
		},
		{
			name:     "session without assurance level",
			status:   http.StatusOK,
			body:     `{"identity":{"id":"identity-id"}}`,
			expected: challenge.SessionAssurance{IdentityID: "identity-id"},
		},
		{
			name:   "second factor owed",
			status: http.StatusForbidden,
			body:   `{"error":{"id":"session_aal2_required","code":403}}`,
			err:    challenge.ErrIdentitySessionAAL2Required,
		},
		{
			name:   "unauthorized",
			status: http.StatusUnauthorized,
			err:    challenge.ErrIdentitySessionInvalid,
		},
		{
			name:   "forbidden for another reason",
			status: http.StatusForbidden,
			body:   `{"error":{"id":"session_inactive","code":403}}`,
			err:    challenge.ErrIdentitySessionInvalid,
		},
		{
			name:   "forbidden with undecodable body",
			status: http.StatusForbidden,
			body:   `not json`,
			err:    challenge.ErrIdentitySessionInvalid,
		},
		{
			name:   "not found",
			status: http.StatusNotFound,
			err:    challenge.ErrIdentitySessionInvalid,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/sessions/whoami", r.URL.Path)
				cookie, err := r.Cookie("ory_kratos_session")
				require.NoError(t, err)
				require.Equal(t, "session-token", cookie.Value)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			resolver, err := NewSessionResolver(SessionConfig{PublicURL: server.URL, Timeout: time.Second})
			require.NoError(t, err)

			assurance, err := resolver.Resolve(context.Background(), "session-token")
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				require.Equal(t, challenge.SessionAssurance{}, assurance)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, assurance)
		})
	}
}

func TestSessionResolverResolveCustomCookieName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("kratos_custom_cookie")
		require.NoError(t, err)
		require.Equal(t, "session-token", cookie.Value)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"identity":{"id":"identity-id"},"authenticator_assurance_level":"aal1"}`))
	}))
	defer server.Close()

	resolver, err := NewSessionResolver(SessionConfig{PublicURL: server.URL, CookieName: "kratos_custom_cookie"})
	require.NoError(t, err)

	assurance, err := resolver.Resolve(context.Background(), "session-token")
	require.NoError(t, err)
	require.Equal(t, challenge.SessionAssurance{IdentityID: "identity-id", AAL: "aal1"}, assurance)
}

func TestSessionResolverResolveServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	resolver, err := NewSessionResolver(SessionConfig{PublicURL: server.URL})
	require.NoError(t, err)

	_, err = resolver.Resolve(context.Background(), "session-token")
	require.Error(t, err)
	require.NotErrorIs(t, err, challenge.ErrIdentitySessionInvalid)
}
