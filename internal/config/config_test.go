package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadSuccess(t *testing.T) {
	setEnv(t, map[string]string{
		"OIDC_HYDRA_ADMIN_URL":      "https://hydra-admin.local",
		"OIDC_HYDRA_PUBLIC_URL":     "https://hydra.local",
		"OIDC_KRATOS_ADMIN_URL":     "https://kratos-admin.local",
		"OIDC_KRATOS_PUBLIC_URL":    "https://kratos-public.local",
		"OIDC_SYNAPSE_CALLBACK_URL": "https://synapse.local/callback",
		"OIDC_COOKIE_DOMAIN":        "local.alkem.io",
		"OIDC_MAINTENANCE_MODE":     "true",
		"OIDC_MAINTENANCE_RETRY":    "30",
		"OIDC_LOG_LEVEL":            "warn",
	})

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Equal(t, "https://hydra-admin.local", cfg.HydraAdminURL)
	require.Equal(t, "https://hydra.local", cfg.HydraPublicURL)
	require.Equal(t, "https://kratos-public.local", cfg.KratosPublicURL)
	require.Equal(t, time.Duration(5)*time.Second, cfg.ReadinessTimeout)

	state := cfg.Maintenance()
	require.True(t, state.Enabled)
	require.NotNil(t, state.RetryAfter)
	require.Equal(t, 30*time.Second, *state.RetryAfter)
}

func TestLoadMissingValue(t *testing.T) {
	setEnv(t, map[string]string{
		"OIDC_HYDRA_ADMIN_URL":   "https://hydra-admin.local",
		"OIDC_HYDRA_PUBLIC_URL":  "https://hydra.local",
		"OIDC_KRATOS_PUBLIC_URL": "https://kratos-public.local",
	})

	cfg, err := Load()
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestLoadRejectsNonHTTPS(t *testing.T) {
	setEnv(t, map[string]string{
		"OIDC_HYDRA_ADMIN_URL":      "http://hydra-admin.local",
		"OIDC_HYDRA_PUBLIC_URL":     "https://hydra.local",
		"OIDC_KRATOS_ADMIN_URL":     "https://kratos-admin.local",
		"OIDC_KRATOS_PUBLIC_URL":    "https://kratos-public.local",
		"OIDC_SYNAPSE_CALLBACK_URL": "https://synapse.local/callback",
		"OIDC_COOKIE_DOMAIN":        "local.alkem.io",
	})

	_, err := Load()
	require.Error(t, err)
}

func TestLoadAllowsHTTPWhenInsecureEnabled(t *testing.T) {
	setEnv(t, map[string]string{
		"OIDC_HYDRA_ADMIN_URL":      "http://hydra-admin.local",
		"OIDC_HYDRA_PUBLIC_URL":     "https://hydra.local",
		"OIDC_KRATOS_ADMIN_URL":     "http://kratos-admin.local",
		"OIDC_KRATOS_PUBLIC_URL":    "http://kratos-public.local",
		"OIDC_SYNAPSE_CALLBACK_URL": "https://synapse.local/callback",
		"OIDC_COOKIE_DOMAIN":        "local.alkem.io",
		"OIDC_ALLOW_INSECURE_HTTP":  "true",
	})

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.True(t, cfg.AllowInsecureHTTP)
}

func TestMaintenanceDisabled(t *testing.T) {
	setEnv(t, map[string]string{
		"OIDC_HYDRA_ADMIN_URL":      "https://hydra-admin.local",
		"OIDC_HYDRA_PUBLIC_URL":     "https://hydra.local",
		"OIDC_KRATOS_ADMIN_URL":     "https://kratos-admin.local",
		"OIDC_KRATOS_PUBLIC_URL":    "https://kratos-public.local",
		"OIDC_SYNAPSE_CALLBACK_URL": "https://synapse.local/callback",
		"OIDC_COOKIE_DOMAIN":        "local.alkem.io",
		"OIDC_MAINTENANCE_MODE":     "false",
	})

	cfg, err := Load()
	require.NoError(t, err)
	state := cfg.Maintenance()
	require.False(t, state.Enabled)
	require.Nil(t, state.RetryAfter)
}

func setEnv(t *testing.T, values map[string]string) {
	t.Helper()

	keys := []string{
		"OIDC_HYDRA_ADMIN_URL",
		"OIDC_HYDRA_PUBLIC_URL",
		"OIDC_KRATOS_ADMIN_URL",
		"OIDC_KRATOS_PUBLIC_URL",
		"OIDC_SYNAPSE_CALLBACK_URL",
		"OIDC_ADMIN_TOKEN",
		"OIDC_COOKIE_DOMAIN",
		"OIDC_MAINTENANCE_MODE",
		"OIDC_MAINTENANCE_RETRY",
		"OIDC_READINESS_TIMEOUT",
		"OIDC_LOG_LEVEL",
		"OIDC_ALLOW_INSECURE_HTTP",
	}

	for _, key := range keys {
		require.NoError(t, os.Unsetenv(key))
	}

	for key, value := range values {
		require.NoError(t, os.Setenv(key, value))
	}
}
