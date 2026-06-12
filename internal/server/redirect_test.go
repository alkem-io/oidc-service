package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedirectTargetAllowed(t *testing.T) {
	cases := []struct {
		name         string
		target       string
		allowedHosts []string
		want         bool
	}{
		{name: "empty target", target: "", want: false},
		{name: "relative path", target: "/oidc/login?login_challenge=abc", want: true},
		{name: "scheme-relative host", target: "//evil.example/phish", want: false},
		{name: "javascript scheme", target: "javascript:alert(1)", want: false},
		{name: "data scheme", target: "data:text/html,hi", want: false},
		{name: "https absolute, no pin", target: "https://hydra.example/oauth2/auth?login_verifier=abc", want: true},
		{name: "http absolute, no pin", target: "http://localhost:3000/logout", want: true},
		{name: "scheme without host", target: "https://", want: false},
		{name: "unparsable url", target: "https://%zz", want: false},
		{
			name:         "pinned host match",
			target:       "https://kratos.example/self-service/login/browser",
			allowedHosts: []string{"kratos.example"},
			want:         true,
		},
		{
			name:         "pinned host case-insensitive",
			target:       "https://KRATOS.example/self-service/login/browser",
			allowedHosts: []string{"kratos.EXAMPLE"},
			want:         true,
		},
		{
			name:         "pinned host mismatch",
			target:       "https://evil.example/self-service/login/browser",
			allowedHosts: []string{"kratos.example"},
			want:         false,
		},
		{
			name:         "pinned host port mismatch",
			target:       "https://kratos.example:8443/self-service/login/browser",
			allowedHosts: []string{"kratos.example"},
			want:         false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, redirectTargetAllowed(tc.target, tc.allowedHosts))
		})
	}
}

func TestSafeRedirectAllowsValidTarget(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oidc/login", nil)

	safeRedirect(rec, req, "https://hydra.example/oauth2/auth")

	require.Equal(t, http.StatusFound, rec.Code)
	require.Equal(t, "https://hydra.example/oauth2/auth", rec.Header().Get("Location"))
}

func TestSafeRedirectRejectsUnsafeTarget(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oidc/login", nil)

	safeRedirect(rec, req, "javascript:alert(1)")

	require.Equal(t, http.StatusBadGateway, rec.Code)
	require.Empty(t, rec.Header().Get("Location"))
}
