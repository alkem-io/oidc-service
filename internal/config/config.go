package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// ServiceConfig captures runtime configuration sourced from environment variables.
type ServiceConfig struct {
	HydraAdminURL      string        `envconfig:"HYDRA_ADMIN_URL" required:"true"`
	HydraPublicURL     string        `envconfig:"HYDRA_PUBLIC_URL" required:"true"`
	KratosAdminURL     string        `envconfig:"KRATOS_ADMIN_URL" required:"true"`
	KratosPublicURL    string        `envconfig:"KRATOS_PUBLIC_URL" required:"true"`
	SynapseCallbackURL string        `envconfig:"SYNAPSE_CALLBACK_URL" required:"true"`
	AuthToken          string        `envconfig:"ADMIN_TOKEN"`
	CookieDomain       string        `envconfig:"COOKIE_DOMAIN" required:"true"`
	MaintenanceMode    bool          `envconfig:"MAINTENANCE_MODE" default:"false"`
	RetryAfterSeconds  int           `envconfig:"MAINTENANCE_RETRY" default:"0"`
	AllowInsecureHTTP  bool          `envconfig:"ALLOW_INSECURE_HTTP" default:"false"`
	ReadinessTimeout   time.Duration `envconfig:"READINESS_TIMEOUT" default:"5s"`
	LogLevel           string        `envconfig:"LOG_LEVEL" default:"info"`
}

// MaintenanceState exposes the cached maintenance toggle information.
type MaintenanceState struct {
	Enabled    bool
	Message    string
	RetryAfter *time.Duration
}

var (
	errInvalidURLScheme = errors.New("url must use https scheme")
	supportedLogLevels  = map[string]struct{}{
		"debug": {},
		"info":  {},
		"warn":  {},
		"error": {},
	}
)

// Load reads service configuration using the OIDC prefix (OIDC_* env vars).
func Load() (*ServiceConfig, error) {
	cfg := &ServiceConfig{}
	if err := envconfig.Process("OIDC", cfg); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (cfg *ServiceConfig) validate() error {
	if err := cfg.requireSecureURL(cfg.HydraAdminURL, "OIDC_HYDRA_ADMIN_URL"); err != nil {
		return err
	}
	if err := assertNotEmpty(cfg.HydraPublicURL, "OIDC_HYDRA_PUBLIC_URL"); err != nil {
		return err
	}
	if err := cfg.requireSecureURL(cfg.KratosAdminURL, "OIDC_KRATOS_ADMIN_URL"); err != nil {
		return err
	}
	if err := cfg.requireSecureURL(cfg.KratosPublicURL, "OIDC_KRATOS_PUBLIC_URL"); err != nil {
		return err
	}
	if err := assertNotEmpty(cfg.SynapseCallbackURL, "OIDC_SYNAPSE_CALLBACK_URL"); err != nil {
		return err
	}
	if err := assertNotEmpty(cfg.CookieDomain, "OIDC_COOKIE_DOMAIN"); err != nil {
		return err
	}
	if cfg.RetryAfterSeconds < 0 {
		return fmt.Errorf("OIDC_MAINTENANCE_RETRY must be >= 0")
	}

	if _, ok := supportedLogLevels[strings.ToLower(cfg.LogLevel)]; !ok {
		return fmt.Errorf("OIDC_LOG_LEVEL must be one of debug, info, warn, error")
	}

	return nil
}

func (cfg *ServiceConfig) requireSecureURL(raw, key string) error {
	if cfg.AllowInsecureHTTP {
		return assertNotEmpty(raw, key)
	}
	return assertHTTPS(raw, key)
}

func assertHTTPS(raw, key string) error {
	if err := assertNotEmpty(raw, key); err != nil {
		return err
	}

	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%s must be a valid url: %w", key, err)
	}

	if u.Scheme != "https" {
		return fmt.Errorf("%s: %w", key, errInvalidURLScheme)
	}

	return nil
}

func assertNotEmpty(value, key string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", key)
	}
	return nil
}

// Maintenance returns the current maintenance state derived from configuration.
func (cfg *ServiceConfig) Maintenance() MaintenanceState {
	if !cfg.MaintenanceMode {
		return MaintenanceState{Enabled: false}
	}

	var retry *time.Duration
	if cfg.RetryAfterSeconds > 0 {
		d := time.Duration(cfg.RetryAfterSeconds) * time.Second
		retry = &d
	}

	return MaintenanceState{
		Enabled:    true,
		Message:    "OIDC service temporarily unavailable",
		RetryAfter: retry,
	}
}
