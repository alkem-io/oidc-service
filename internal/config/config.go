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
	HydraAdminURL       string        `envconfig:"HYDRA_ADMIN_URL" required:"true"`
	KratosAdminURL      string        `envconfig:"KRATOS_ADMIN_URL" required:"true"`
	KratosPublicURL     string        `envconfig:"KRATOS_PUBLIC_URL" required:"true"`
	AlkemioServerURL    string        `envconfig:"ALKEMIO_SERVER_URL" required:"true"`
	AlkemioResolvePath  string        `envconfig:"ALKEMIO_IDENTITY_RESOLVE_PATH" default:"/rest/internal/identity/resolve"`
	IdentityTimeout     time.Duration `envconfig:"ALKEMIO_IDENTITY_TIMEOUT" default:"30s"`
	IdentityMaxRetries  int           `envconfig:"ALKEMIO_IDENTITY_RETRIES" default:"5"`
	KratosBrowserURL    string        `envconfig:"KRATOS_BROWSER_URL"`
	LoginReturnBaseURL  string        `envconfig:"LOGIN_RETURN_BASE_URL"`
	WebBaseURL          string        `envconfig:"WEB_BASE_URL"`
	KratosSessionCookie string        `envconfig:"KRATOS_SESSION_COOKIE" default:"ory_kratos_session"`
	AuthToken           string        `envconfig:"ADMIN_TOKEN"`
	MaintenanceMode     bool          `envconfig:"MAINTENANCE_MODE" default:"false"`
	RetryAfterSeconds   int           `envconfig:"MAINTENANCE_RETRY" default:"0"`
	AllowInsecureHTTP   bool          `envconfig:"ALLOW_INSECURE_HTTP" default:"false"`
	ReadinessTimeout    time.Duration `envconfig:"READINESS_TIMEOUT" default:"5s"`
	LogLevel            string        `envconfig:"LOG_LEVEL" default:"info"`
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

const defaultKratosSessionCookie = "ory_kratos_session"

// Load reads service configuration using the OIDC prefix (OIDC_* env vars).
func Load() (*ServiceConfig, error) {
	cfg := &ServiceConfig{}
	if err := envconfig.Process("OIDC", cfg); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	cfg.KratosSessionCookie = strings.TrimSpace(cfg.KratosSessionCookie)
	if cfg.KratosSessionCookie == "" {
		cfg.KratosSessionCookie = defaultKratosSessionCookie
	}
	cfg.KratosBrowserURL = strings.TrimSpace(cfg.KratosBrowserURL)
	cfg.AlkemioServerURL = strings.TrimSpace(cfg.AlkemioServerURL)
	cfg.AlkemioResolvePath = normalizeResolvePath(cfg.AlkemioResolvePath)
	cfg.LoginReturnBaseURL = strings.TrimSpace(cfg.LoginReturnBaseURL)
	cfg.WebBaseURL = strings.TrimSpace(cfg.WebBaseURL)

	if cfg.LoginReturnBaseURL == "" {
		if computed := cfg.computeLoginReturnBaseURL(); computed != "" {
			cfg.LoginReturnBaseURL = computed
		}
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadDocstringCoverage reads coverage-specific configuration settings.

func (cfg *ServiceConfig) validate() error { //nolint:cyclop
	if err := cfg.requireSecureURL(cfg.HydraAdminURL, "OIDC_HYDRA_ADMIN_URL"); err != nil {
		return err
	}
	if err := cfg.requireSecureURL(cfg.KratosAdminURL, "OIDC_KRATOS_ADMIN_URL"); err != nil {
		return err
	}
	if err := cfg.requireSecureURL(cfg.KratosPublicURL, "OIDC_KRATOS_PUBLIC_URL"); err != nil {
		return err
	}
	if err := cfg.requireSecureURL(cfg.AlkemioServerURL, "OIDC_ALKEMIO_SERVER_URL"); err != nil {
		return err
	}
	if trimmed := strings.TrimSpace(cfg.KratosBrowserURL); trimmed != "" {
		if err := cfg.requireSecureURL(trimmed, "OIDC_KRATOS_BROWSER_URL"); err != nil {
			return err
		}
	}

	if trimmed := strings.TrimSpace(cfg.WebBaseURL); trimmed != "" {
		if err := cfg.requireSecureURL(trimmed, "OIDC_WEB_BASE_URL"); err != nil {
			return err
		}
	}

	if trimmed := strings.TrimSpace(cfg.LoginReturnBaseURL); trimmed != "" {
		if err := cfg.requireSecureURL(trimmed, "OIDC_LOGIN_RETURN_BASE_URL"); err != nil {
			return err
		}
	}
	if cfg.RetryAfterSeconds < 0 {
		return fmt.Errorf("OIDC_MAINTENANCE_RETRY must be >= 0")
	}

	if _, ok := supportedLogLevels[strings.ToLower(cfg.LogLevel)]; !ok {
		return fmt.Errorf("OIDC_LOG_LEVEL must be one of debug, info, warn, error")
	}

	if strings.TrimSpace(cfg.KratosSessionCookie) == "" {
		return fmt.Errorf("OIDC_KRATOS_SESSION_COOKIE is required")
	}
	if cfg.IdentityTimeout <= 0 {
		return fmt.Errorf("OIDC_ALKEMIO_IDENTITY_TIMEOUT must be > 0")
	}
	if cfg.IdentityMaxRetries <= 0 {
		return fmt.Errorf("OIDC_ALKEMIO_IDENTITY_RETRIES must be > 0")
	}

	return nil
}

func (cfg *ServiceConfig) computeLoginReturnBaseURL() string {
	if cfg.WebBaseURL == "" {
		return ""
	}

	base := strings.TrimRight(cfg.WebBaseURL, "/")
	if base == "" {
		return ""
	}

	return base + "/oidc/login"
}

func (cfg *ServiceConfig) requireSecureURL(raw, key string) error {
	if cfg.AllowInsecureHTTP {
		return assertHTTPOrHTTPS(raw, key)
	}
	return assertHTTPS(raw, key)
}

func assertHTTPOrHTTPS(raw, key string) error {
	if err := assertNotEmpty(raw, key); err != nil {
		return err
	}

	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%s must be a valid url: %w", key, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%s must use http or https scheme", key)
	}

	if strings.TrimSpace(u.Host) == "" {
		return fmt.Errorf("%s must include a host", key)
	}

	return nil
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

func normalizeResolvePath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimLeft(trimmed, "/")
	if trimmed == "" {
		return "/"
	}
	return "/" + trimmed
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
