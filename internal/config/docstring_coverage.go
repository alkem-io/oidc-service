package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

// DocstringCoverageConfig exposes tunables for the docstring coverage CLI.
type DocstringCoverageConfig struct {
	Threshold float64 `envconfig:"DOCSTRING_COVERAGE_THRESHOLD" default:"80"`
}

// LoadDocstringCoverage reads coverage-specific configuration settings.
func LoadDocstringCoverage() (*DocstringCoverageConfig, error) {
	cfg := &DocstringCoverageConfig{}
	if err := envconfig.Process("OIDC", cfg); err != nil {
		return nil, fmt.Errorf("load docstring coverage config: %w", err)
	}
	if cfg.Threshold <= 0 || cfg.Threshold > 100 {
		return nil, fmt.Errorf("OIDC_DOCSTRING_COVERAGE_THRESHOLD must be within (0,100]")
	}
	return cfg, nil
}
