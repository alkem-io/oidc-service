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

// NewHTTPReadinessProbe constructs a readiness probe backed by simple HTTP checks.
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

// Ready verifies that the upstream service reports a successful readiness response.
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
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("readiness endpoint returned status %d", resp.StatusCode)
}

// Version retrieves the reported version string from the upstream service.
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
	defer func() {
		_ = resp.Body.Close()
	}()

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
