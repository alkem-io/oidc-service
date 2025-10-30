package hydra

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	hydraAdmin "github.com/ory/hydra-client-go/v2"
)

const defaultTimeout = 10 * time.Second

// Config captures the settings required to communicate with Hydra's Admin API.
type Config struct {
	AdminURL  string
	AuthToken string
	Timeout   time.Duration
}

// Client provides a thin wrapper around the generated Hydra Admin API client.
type Client struct {
	api *hydraAdmin.APIClient
}

// NewClient constructs an Admin API client configured for the Hydra Admin API.
// It validates that cfg.AdminURL is non-empty and includes a scheme and host, and
// returns an error if those checks fail. NewClient trims any trailing slash from
// the server URL, uses cfg.Timeout or a 10s default when timeout is not positive,
// and attaches an Authorization: Bearer <token> header when cfg.AuthToken is set.
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.AdminURL) == "" {
		return nil, fmt.Errorf("hydra admin url is required")
	}

	parsed, err := url.Parse(cfg.AdminURL)
	if err != nil {
		return nil, fmt.Errorf("invalid hydra admin url: %w", err)
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("hydra admin url must include scheme and host")
	}

	conf := hydraAdmin.NewConfiguration()
	conf.Servers = hydraAdmin.ServerConfigurations{
		{
			URL: strings.TrimRight(cfg.AdminURL, "/"),
		},
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	conf.HTTPClient = &http.Client{Timeout: timeout}

	if token := strings.TrimSpace(cfg.AuthToken); token != "" {
		conf.AddDefaultHeader("Authorization", "Bearer "+token)
	}

	return &Client{api: hydraAdmin.NewAPIClient(conf)}, nil
}

// Admin returns the underlying Hydra API client instance.
func (c *Client) Admin() *hydraAdmin.APIClient {
	return c.api
}