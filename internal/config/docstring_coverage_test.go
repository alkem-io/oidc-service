package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadDocstringCoverageDefaults(t *testing.T) {
	require.NoError(t, os.Unsetenv("OIDC_DOCSTRING_COVERAGE_THRESHOLD"))
	cfg, err := LoadDocstringCoverage()
	require.NoError(t, err)
	require.Equal(t, 80.0, cfg.Threshold)
}

func TestLoadDocstringCoverageCustom(t *testing.T) {
	t.Setenv("OIDC_DOCSTRING_COVERAGE_THRESHOLD", "72.5")
	cfg, err := LoadDocstringCoverage()
	require.NoError(t, err)
	require.Equal(t, 72.5, cfg.Threshold)
}

func TestLoadDocstringCoverageRejectsInvalid(t *testing.T) {
	t.Setenv("OIDC_DOCSTRING_COVERAGE_THRESHOLD", "0")
	_, err := LoadDocstringCoverage()
	require.Error(t, err)
}
