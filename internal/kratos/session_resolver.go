package kratos

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/alkem-io/oidc-service/internal/challenge"
)

const (
	defaultSessionTimeout = 5 * time.Second
	defaultSessionCookie  = "ory_kratos_session"
)

// SessionConfig configures the Kratos session resolver.
type SessionConfig struct {
	PublicURL  string
	Timeout    time.Duration
	CookieName string
}

// SessionResolver resolves Kratos session cookies into identity identifiers.
type SessionResolver struct {
	baseURL    string
	client     *http.Client
	cookieName string
}

// NewSessionResolver constructs a session resolver targeting Kratos public endpoints.
func NewSessionResolver(cfg SessionConfig) (*SessionResolver, error) {
	trimmed := strings.TrimSpace(cfg.PublicURL)
	if trimmed == "" {
		return nil, fmt.Errorf("kratos public url is required")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid kratos public url: %w", err)
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("kratos public url must include scheme and host")
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultSessionTimeout
	}

	cookieName := strings.TrimSpace(cfg.CookieName)
	if cookieName == "" {
		cookieName = defaultSessionCookie
	}

	client := &http.Client{Timeout: timeout}
	return &SessionResolver{
		baseURL:    strings.TrimRight(parsed.String(), "/"),
		client:     client,
		cookieName: cookieName,
	}, nil
}

// IdentityID retrieves the Kratos identity ID associated with the provided session cookie.
func (r *SessionResolver) IdentityID(ctx context.Context, sessionCookie string) (string, error) {
	if r == nil {
		return "", challenge.ErrIdentitySessionInvalid
	}

	cookieValue := strings.TrimSpace(sessionCookie)
	if cookieValue == "" {
		return "", challenge.ErrIdentitySessionInvalid
	}

	endpoint := r.baseURL + "/sessions/whoami"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create kratos whoami request: %w", err)
	}

	req.AddCookie(&http.Cookie{Name: r.cookieName, Value: cookieValue})

	resp, err := r.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("kratos whoami request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusOK:
		var payload struct {
			Identity struct {
				ID string `json:"id"`
			} `json:"identity"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return "", fmt.Errorf("decode kratos whoami response: %w", err)
		}

		identityID := strings.TrimSpace(payload.Identity.ID)
		if identityID == "" {
			return "", challenge.ErrIdentitySessionInvalid
		}
		return identityID, nil

	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		return "", challenge.ErrIdentitySessionInvalid

	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("kratos whoami request failed (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}
