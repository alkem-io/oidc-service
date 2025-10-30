package kratos

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	kratosClient "github.com/ory/client-go"
)

const defaultTimeout = 10 * time.Second

// Config defines the parameters required to talk to Kratos' Admin API.
type Config struct {
	AdminURL  string
	AuthToken string
	Timeout   time.Duration
}

// Client wraps the generated Kratos API client to enforce deterministic configuration.
type Client struct {
	api *kratosClient.APIClient
}

// NewClient creates a Client configured to communicate with the Kratos Admin API at the address in cfg.AdminURL.
// It returns an error if cfg.AdminURL is empty, cannot be parsed, or does not include a scheme and host.
// The resulting client uses cfg.Timeout when greater than zero (otherwise a default timeout) and, if cfg.AuthToken is provided, sets a default "Authorization: Bearer <token>" header.
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.AdminURL) == "" {
		return nil, fmt.Errorf("kratos admin url is required")
	}

	parsed, err := url.Parse(cfg.AdminURL)
	if err != nil {
		return nil, fmt.Errorf("invalid kratos admin url: %w", err)
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("kratos admin url must include scheme and host")
	}

	conf := kratosClient.NewConfiguration()
	conf.Servers = kratosClient.ServerConfigurations{{
		URL: strings.TrimRight(cfg.AdminURL, "/"),
	}}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	conf.HTTPClient = &http.Client{Timeout: timeout}

	if token := strings.TrimSpace(cfg.AuthToken); token != "" {
		conf.AddDefaultHeader("Authorization", "Bearer "+token)
	}

	return &Client{api: kratosClient.NewAPIClient(conf)}, nil
}

// Admin exposes the underlying API client for advanced operations.
func (c *Client) Admin() *kratosClient.APIClient {
	return c.api
}