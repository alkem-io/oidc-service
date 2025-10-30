package challenge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type httpReadinessProbe struct {
	client     *http.Client
	readyURL   string
	versionURL string
	token      string
}

// NewHTTPReadinessProbe constructs an HTTP-based ReadinessProbe using baseURL as the base for readyPath and versionPath.
// It resolves readyPath and versionPath against baseURL, trims the provided token for use as an optional Bearer token,
// and configures an http.Client with the provided timeout (defaults to 5s when timeout is <= 0).
// Returns an error if baseURL is empty or if parsing/resolving any URL fails.
func NewHTTPReadinessProbe(baseURL, readyPath, versionPath, token string, timeout time.Duration) (ReadinessProbe, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("base url is required")
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}

	readyURL, err := resolveURL(base, readyPath)
	if err != nil {
		return nil, fmt.Errorf("build ready url: %w", err)
	}

	versionURL, err := resolveURL(base, versionPath)
	if err != nil {
		return nil, fmt.Errorf("build version url: %w", err)
	}

	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	return &httpReadinessProbe{
		client:     &http.Client{Timeout: timeout},
		readyURL:   readyURL,
		versionURL: versionURL,
		token:      strings.TrimSpace(token),
	}, nil
}

func (p *httpReadinessProbe) Ready(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.readyURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("readiness endpoint returned status %d", resp.StatusCode)
}

func (p *httpReadinessProbe) Version(ctx context.Context) (string, error) {
	if p.versionURL == "" {
		return "", nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.versionURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("version endpoint returned status %d", resp.StatusCode)
	}

	var payload struct {
		Version string `json:"version"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}

	return strings.TrimSpace(payload.Version), nil
}

// resolveURL resolves the provided path against base and returns the resulting absolute URL string.
// If path is empty after trimming whitespace, it returns an empty string and no error.
// If the path cannot be parsed as a URL reference, it returns an error.
func resolveURL(base *url.URL, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}

	ref, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	return base.ResolveReference(ref).String(), nil
}