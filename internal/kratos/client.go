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

// NewClient instantiates a Kratos Admin API client with optional bearer authentication.
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
