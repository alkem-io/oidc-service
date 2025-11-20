package alkemio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	defaultTimeout    = 30 * time.Second
	defaultMaxRetries = 1
	defaultBackoff    = 200 * time.Millisecond
)

// HTTPDoer abstracts the http.Client Do method for easier testing.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Config captures the settings required to call the Alkemio identity resolver.
type Config struct {
	BaseURL     string
	ResolvePath string
	Timeout     time.Duration
	MaxRetries  int
	Client      HTTPDoer
}

// IdentityResolver resolves Alkemio user IDs from Kratos authentication IDs.
type IdentityResolver struct {
	client     HTTPDoer
	resolveURL string
	timeout    time.Duration
	retries    int
}

// ErrNotFound signals that no Alkemio user mapping exists for the requested identity.
var ErrNotFound = errors.New("alkemio user mapping not found")

// NewIdentityResolver constructs a resolver with the supplied configuration.
func NewIdentityResolver(cfg Config) (*IdentityResolver, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("base url is required")
	}

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base url: %w", err)
	}

	path := strings.TrimSpace(cfg.ResolvePath)
	if path == "" {
		path = "/rest/internal/identity/resolve"
	}

	resolveURL, err := buildResolveURL(parsedBase, path)
	if err != nil {
		return nil, err
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	retries := cfg.MaxRetries
	if retries <= 0 {
		retries = defaultMaxRetries
	}

	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}

	return &IdentityResolver{
		client:     client,
		resolveURL: resolveURL,
		timeout:    timeout,
		retries:    retries,
	}, nil
}

// Resolve returns the Alkemio user ID for the provided Kratos authentication ID.
func (r *IdentityResolver) Resolve(ctx context.Context, authenticationID string) (string, error) {
	if r == nil {
		return "", errors.New("identity resolver is nil")
	}

	id := strings.TrimSpace(authenticationID)
	if id == "" {
		return "", errors.New("authentication id is required")
	}
	if !isUUID(id) {
		return "", fmt.Errorf("authentication id must be a valid uuid")
	}

	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	attempts := r.retries
	if attempts <= 0 {
		attempts = defaultMaxRetries
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		userID, err := r.resolveOnce(ctx, id)

		if err == nil {
			return userID, nil
		}

		if errors.Is(err, ErrNotFound) {
			return "", err
		}

		lastErr = err
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		if attempt == attempts || !isTemporary(err) {
			break
		}

		if err := r.wait(ctx, attempt); err != nil {
			return "", err
		}
	}

	if lastErr != nil {
		return "", lastErr
	}

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	return "", errors.New("identity resolution failed")
}

func (r *IdentityResolver) resolveOnce(ctx context.Context, authenticationID string) (string, error) {
	body, err := json.Marshal(map[string]string{"authenticationId": authenticationID})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.resolveURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", &temporaryError{err: err}
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return "", ErrNotFound
	case resp.StatusCode >= 500:
		return "", &temporaryError{err: fmt.Errorf("alkemio server returned status %d", resp.StatusCode)}
	case resp.StatusCode >= 400:
		return "", fmt.Errorf("alkemio server returned status %d", resp.StatusCode)
	}

	var payload struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", &temporaryError{err: fmt.Errorf("decode response: %w", err)}
	}

	userID := strings.TrimSpace(payload.UserID)
	if userID == "" {
		return "", fmt.Errorf("alkemio response missing userId")
	}
	if !isUUID(userID) {
		return "", fmt.Errorf("alkemio response userId must be a valid uuid")
	}

	return userID, nil
}

func (r *IdentityResolver) wait(ctx context.Context, attempt int) error {
	delay := backoffDuration(attempt)
	if delay <= 0 {
		delay = defaultBackoff
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (r *IdentityResolver) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}

	if r.timeout <= 0 {
		return ctx, func() {}
	}

	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= r.timeout {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, r.timeout)
}

func buildResolveURL(base *url.URL, path string) (string, error) {
	if base == nil {
		return "", fmt.Errorf("base url is required")
	}

	ref, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("invalid resolve path: %w", err)
	}

	return base.ResolveReference(ref).String(), nil
}

func backoffDuration(attempt int) time.Duration {
	if attempt <= 0 {
		return defaultBackoff
	}
	return time.Duration(attempt) * defaultBackoff
}

type temporaryError struct {
	err error
}

func (e *temporaryError) Error() string {
	if e == nil || e.err == nil {
		return "temporary error"
	}
	return e.err.Error()
}

func (e *temporaryError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func isTemporary(err error) bool {
	var tmp *temporaryError
	return errors.As(err, &tmp)
}

func isUUID(value string) bool {
	if value == "" {
		return false
	}
	_, err := uuid.Parse(value)
	return err == nil
}
