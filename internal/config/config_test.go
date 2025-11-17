package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadSuccess(t *testing.T) {
	setEnv(
		t, map[string]string{
			"OIDC_HYDRA_ADMIN_URL":               "https://hydra-admin.local",
			"OIDC_KRATOS_ADMIN_URL":              "https://kratos-admin.local",
			"OIDC_KRATOS_PUBLIC_URL":             "https://kratos-public.local",
			"OIDC_ALKEMIO_SERVER_URL":            "https://alkemio.local",
			"OIDC_ALKEMIO_IDENTITY_RESOLVE_PATH": " identity/resolve ",
			"OIDC_ALKEMIO_IDENTITY_TIMEOUT":      "10s",
			"OIDC_ALKEMIO_IDENTITY_RETRIES":      "3",
			"OIDC_KRATOS_BROWSER_URL":            "https://kratos-browser.local",
			"OIDC_LOGIN_RETURN_BASE_URL":         "https://oidc.example/oidc/login",
			"OIDC_MAINTENANCE_MODE":              "true",
			"OIDC_MAINTENANCE_RETRY":             "30",
			"OIDC_LOG_LEVEL":                     "warn",
		},
	)

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Equal(t, "https://hydra-admin.local", cfg.HydraAdminURL)
	require.Equal(t, "https://kratos-public.local", cfg.KratosPublicURL)
	require.Equal(t, "https://kratos-browser.local", cfg.KratosBrowserURL)
	require.Equal(t, "https://alkemio.local", cfg.AlkemioServerURL)
	require.Equal(t, "/identity/resolve", cfg.AlkemioResolvePath)
	require.Equal(t, 10*time.Second, cfg.IdentityTimeout)
	require.Equal(t, 3, cfg.IdentityMaxRetries)
	require.Equal(t, "https://oidc.example/oidc/login", cfg.LoginReturnBaseURL)
	require.Equal(t, time.Duration(5)*time.Second, cfg.ReadinessTimeout)
	require.Equal(t, "ory_kratos_session", cfg.KratosSessionCookie)

	state := cfg.Maintenance()
	require.True(t, state.Enabled)
	require.NotNil(t, state.RetryAfter)
	require.Equal(t, 30*time.Second, *state.RetryAfter)
}

func TestLoadComputesLoginReturnBaseURL(t *testing.T) {
	setEnv(
		t, map[string]string{
			"OIDC_HYDRA_ADMIN_URL":    "https://hydra-admin.local",
			"OIDC_KRATOS_ADMIN_URL":   "https://kratos-admin.local",
			"OIDC_KRATOS_PUBLIC_URL":  "https://kratos-public.local",
			"OIDC_ALKEMIO_SERVER_URL": "https://alkemio.local",
			"OIDC_KRATOS_BROWSER_URL": "https://kratos-browser.local",
			"OIDC_WEB_BASE_URL":       "https://platform.example",
		},
	)

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "https://platform.example/oidc/login", cfg.LoginReturnBaseURL)
}

func TestLoadMissingValue(t *testing.T) {
	setEnv(
		t, map[string]string{
			"OIDC_HYDRA_ADMIN_URL":    "https://hydra-admin.local",
			"OIDC_KRATOS_PUBLIC_URL":  "https://kratos-public.local",
			"OIDC_ALKEMIO_SERVER_URL": "https://alkemio.local",
		},
	)

	cfg, err := Load()
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestLoadRejectsNonHTTPS(t *testing.T) {
	setEnv(
		t, map[string]string{
			"OIDC_HYDRA_ADMIN_URL":    "http://hydra-admin.local",
			"OIDC_KRATOS_ADMIN_URL":   "https://kratos-admin.local",
			"OIDC_KRATOS_PUBLIC_URL":  "https://kratos-public.local",
			"OIDC_ALKEMIO_SERVER_URL": "https://alkemio.local",
		},
	)

	_, err := Load()
	require.Error(t, err)
}

func TestLoadAllowsHTTPWhenInsecureEnabled(t *testing.T) {
	setEnv(
		t, map[string]string{
			"OIDC_HYDRA_ADMIN_URL":     "http://hydra-admin.local",
			"OIDC_KRATOS_ADMIN_URL":    "http://kratos-admin.local",
			"OIDC_KRATOS_PUBLIC_URL":   "http://kratos-public.local",
			"OIDC_KRATOS_BROWSER_URL":  "http://kratos-browser.local",
			"OIDC_ALKEMIO_SERVER_URL":  "http://alkemio.local",
			"OIDC_ALLOW_INSECURE_HTTP": "true",
		},
	)

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.True(t, cfg.AllowInsecureHTTP)
}

func TestLoadRejectsInvalidSchemeWhenInsecureEnabled(t *testing.T) {
	setEnv(
		t, map[string]string{
			"OIDC_HYDRA_ADMIN_URL":     "ftp://hydra-admin.local",
			"OIDC_KRATOS_ADMIN_URL":    "https://kratos-admin.local",
			"OIDC_KRATOS_PUBLIC_URL":   "https://kratos-public.local",
			"OIDC_KRATOS_BROWSER_URL":  "https://kratos-browser.local",
			"OIDC_ALKEMIO_SERVER_URL":  "https://alkemio.local",
			"OIDC_ALLOW_INSECURE_HTTP": "true",
		},
	)

	_, err := Load()
	require.Error(t, err)
}

func TestLoadRejectsMissingHostWhenInsecureEnabled(t *testing.T) {
	setEnv(
		t, map[string]string{
			"OIDC_HYDRA_ADMIN_URL":     "http:///",
			"OIDC_KRATOS_ADMIN_URL":    "https://kratos-admin.local",
			"OIDC_KRATOS_PUBLIC_URL":   "https://kratos-public.local",
			"OIDC_KRATOS_BROWSER_URL":  "https://kratos-browser.local",
			"OIDC_ALKEMIO_SERVER_URL":  "https://alkemio.local",
			"OIDC_ALLOW_INSECURE_HTTP": "true",
		},
	)

	_, err := Load()
	require.Error(t, err)
}

func TestMaintenanceDisabled(t *testing.T) {
	setEnv(
		t, map[string]string{
			"OIDC_HYDRA_ADMIN_URL":    "https://hydra-admin.local",
			"OIDC_KRATOS_ADMIN_URL":   "https://kratos-admin.local",
			"OIDC_KRATOS_PUBLIC_URL":  "https://kratos-public.local",
			"OIDC_MAINTENANCE_MODE":   "false",
			"OIDC_ALKEMIO_SERVER_URL": "https://alkemio.local",
		},
	)

	cfg, err := Load()
	require.NoError(t, err)
	state := cfg.Maintenance()
	require.False(t, state.Enabled)
	require.Nil(t, state.RetryAfter)
}

func TestLoadCustomSessionCookie(t *testing.T) {
	setEnv(
		t, map[string]string{
			"OIDC_HYDRA_ADMIN_URL":       "https://hydra-admin.local",
			"OIDC_KRATOS_ADMIN_URL":      "https://kratos-admin.local",
			"OIDC_KRATOS_PUBLIC_URL":     "https://kratos-public.local",
			"OIDC_KRATOS_BROWSER_URL":    "https://kratos-browser.local",
			"OIDC_LOGIN_RETURN_BASE_URL": "https://oidc.example/oidc/login",
			"OIDC_KRATOS_SESSION_COOKIE": " kratos_custom ",
			"OIDC_ALKEMIO_SERVER_URL":    "https://alkemio.local",
		},
	)

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "kratos_custom", cfg.KratosSessionCookie)
}

func setEnv(t *testing.T, values map[string]string) {
	t.Helper()

	keys := []string{
		"OIDC_HYDRA_ADMIN_URL",
		"OIDC_KRATOS_ADMIN_URL",
		"OIDC_KRATOS_PUBLIC_URL",
		"OIDC_KRATOS_BROWSER_URL",
		"OIDC_LOGIN_RETURN_BASE_URL",
		"OIDC_WEB_BASE_URL",
		"OIDC_KRATOS_SESSION_COOKIE",
		"OIDC_ADMIN_TOKEN",
		"OIDC_MAINTENANCE_MODE",
		"OIDC_MAINTENANCE_RETRY",
		"OIDC_READINESS_TIMEOUT",
		"OIDC_LOG_LEVEL",
		"OIDC_ALLOW_INSECURE_HTTP",
		"OIDC_ALKEMIO_SERVER_URL",
		"OIDC_ALKEMIO_IDENTITY_RESOLVE_PATH",
		"OIDC_ALKEMIO_IDENTITY_TIMEOUT",
		"OIDC_ALKEMIO_IDENTITY_RETRIES",
	}

	for _, key := range keys {
		require.NoError(t, os.Unsetenv(key))
	}

	for key, value := range values {
		require.NoError(t, os.Setenv(key, value))
	}
}
