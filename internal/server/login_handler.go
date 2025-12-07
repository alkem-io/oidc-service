package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/alkem-io/oidc-service/internal/challenge"
	middlewarepkg "github.com/alkem-io/oidc-service/internal/middleware"
	"go.uber.org/zap"
)

// SessionIdentityResolver resolves a Kratos session cookie into an identity identifier.
type SessionIdentityResolver interface {
	// IdentityID returns the identity ID associated with the session cookie.
	IdentityID(ctx context.Context, sessionCookie string) (string, error)
}

// LoginHandler processes login challenges and delegates to the challenge service.
type LoginHandler struct {
	logger            *zap.Logger
	service           challenge.Service
	sessionResolver   SessionIdentityResolver
	paramKey          string
	sessionCookie     string
	kratosBrowserBase *url.URL
	returnBaseURL     *url.URL
}

// LoginHandlerConfig captures dependencies used by LoginHandler.
type LoginHandlerConfig struct {
	Logger           *zap.Logger
	Challenge        challenge.Service
	SessionResolver  SessionIdentityResolver
	SessionCookie    string
	KratosBrowserURL string
	ReturnBaseURL    string
}

// NewLoginHandler constructs a LoginHandler from the supplied configuration.
func NewLoginHandler(cfg LoginHandlerConfig) *LoginHandler {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	svc := cfg.Challenge
	if svc == nil {
		svc = challenge.NewStubService()
	}

	cookie := strings.TrimSpace(cfg.SessionCookie)
	if cookie == "" {
		cookie = "ory_kratos_session"
	}

	handler := &LoginHandler{
		logger:          logger,
		service:         svc,
		sessionResolver: cfg.SessionResolver,
		paramKey:        "login_challenge",
		sessionCookie:   cookie,
	}

	if trimmed := strings.TrimSpace(cfg.KratosBrowserURL); trimmed != "" {
		if parsed, err := url.Parse(trimmed); err != nil {
			logger.Warn("invalid kratos browser url, disabling login redirect", zap.Error(err))
		} else {
			handler.kratosBrowserBase = parsed
		}
	}

	if trimmed := strings.TrimSpace(cfg.ReturnBaseURL); trimmed != "" {
		if parsed, err := url.Parse(trimmed); err != nil {
			logger.Warn("invalid login return base url, falling back to request url", zap.Error(err))
		} else if parsed.Scheme == "" || parsed.Host == "" {
			logger.Warn("login return base url must include scheme and host", zap.String("value", trimmed))
		} else {
			handler.returnBaseURL = parsed
		}
	}

	return handler
}

// Handle parses the login challenge, injects session hints, and redirects on success.
func (h *LoginHandler) Handle(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	challengeID := strings.TrimSpace(r.URL.Query().Get(h.paramKey))
	if challengeID == "" {
		err := challenge.NewError(http.StatusBadRequest, "missing_challenge", "login challenge is required", "", nil)
		middlewarepkg.WriteChallengeError(w, r, err)
		return
	}

	ctx := h.attachIdentityHint(r.Context(), r)

	resolution, err := h.service.ResolveLogin(ctx, challengeID)
	if err != nil {
		if h.handleSessionRedirect(w, r, challengeID, started, err) {
			return
		}

		middlewarepkg.WriteChallengeError(w, r, err)
		return
	}

	redirectTo := resolution.RedirectURL
	logger := middlewarepkg.Logger(r.Context())
	if logger == nil {
		logger = h.logger
	}

	logger.Info(
		"login challenge resolved",
		zap.String("challengeId", challengeID),
		zap.String("redirectTo", redactRedirectURL(redirectTo)),
		zap.Duration("duration", time.Since(started)),
	)

	http.Redirect(w, r, redirectTo, http.StatusFound)
}

func (h *LoginHandler) attachIdentityHint(ctx context.Context, r *http.Request) context.Context {
	if h.sessionResolver == nil {
		return ctx
	}

	provider := h.buildHintProvider(r)
	if provider == nil {
		return ctx
	}

	return challenge.WithIdentityHintProvider(ctx, provider)
}

func (h *LoginHandler) buildHintProvider(r *http.Request) challenge.IdentityHintProvider {
	if h.sessionResolver == nil {
		return nil
	}

	cookie, err := r.Cookie(h.sessionCookie)
	if err != nil {
		return challenge.IdentityHintFunc(
			func(context.Context) (string, error) {
				return "", challenge.ErrIdentitySessionRequired
			},
		)
	}

	sessionToken := strings.TrimSpace(cookie.Value)
	if sessionToken == "" {
		return challenge.IdentityHintFunc(
			func(context.Context) (string, error) {
				return "", challenge.ErrIdentitySessionInvalid
			},
		)
	}

	return challenge.IdentityHintFunc(
		func(ctx context.Context) (string, error) {
			return h.sessionResolver.IdentityID(ctx, sessionToken)
		},
	)
}

func (h *LoginHandler) handleSessionRedirect(
	w http.ResponseWriter, r *http.Request, challengeID string, _ time.Time, err error,
) bool {
	if w == nil || r == nil {
		return false
	}

	var chErr challenge.Error
	if !errors.As(err, &chErr) {
		return false
	}

	switch chErr.Code() {
	case "session_required", "session_invalid":
		// eligible for redirect
	default:
		return false
	}

	redirectURL, buildErr := h.buildKratosLoginRedirect(r)
	if buildErr != nil {
		h.logger.Warn(
			"failed to build kratos login redirect", zap.String("challengeId", challengeID), zap.Error(buildErr),
		)
		return false
	}

	logger := middlewarepkg.Logger(r.Context())
	if logger == nil {
		logger = h.logger
	}

	logger.Info(
		"redirecting to kratos login", zap.String("challengeId", challengeID), zap.String("redirectTo", redirectURL),
	)

	http.Redirect(w, r, redirectURL, http.StatusFound)
	return true
}

func (h *LoginHandler) buildKratosLoginRedirect(r *http.Request) (string, error) {
	if h.kratosBrowserBase == nil {
		return "", errors.New("kratos login redirect disabled")
	}

	returnTo, err := h.buildReturnURL(r)
	if err != nil {
		return "", err
	}

	target := *h.kratosBrowserBase
	target.Path = joinURLPath(target.Path, "/self-service/login/browser")
	target.RawPath = ""
	query := target.Query()
	query.Set("return_to", returnTo)
	target.RawQuery = query.Encode()
	target.Fragment = ""

	return target.String(), nil
}

func (h *LoginHandler) buildReturnURL(r *http.Request) (string, error) {
	if h.returnBaseURL != nil {
		base := *h.returnBaseURL
		query := base.Query()
		for key, values := range r.URL.Query() {
			for _, value := range values {
				query.Set(key, value)
			}
		}
		base.RawQuery = query.Encode()
		return base.String(), nil
	}

	return h.requestReturnURL(r)
}

func (h *LoginHandler) requestReturnURL(r *http.Request) (string, error) {
	if r == nil {
		return "", errors.New("request not available")
	}

	scheme := forwardedProto(r)
	host := forwardedHost(r)

	if h.kratosBrowserBase != nil && shouldUseBrowserHostFallback(host) {
		if h.kratosBrowserBase.Scheme != "" {
			scheme = h.kratosBrowserBase.Scheme
		}
		host = h.kratosBrowserBase.Host
	}

	if strings.TrimSpace(host) == "" {
		return "", errors.New("request host not available")
	}

	path := requestPathWithPrefix(r)

	return scheme + "://" + host + path, nil
}

func joinURLPath(basePath, suffix string) string {
	suffix = strings.TrimSpace(suffix)
	if suffix == "" {
		return strings.TrimSpace(basePath)
	}
	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}
	trimmedBase := strings.TrimSpace(basePath)
	if trimmedBase == "" || trimmedBase == "/" {
		return suffix
	}
	trimmedBase = strings.TrimSuffix(trimmedBase, "/")
	return trimmedBase + suffix
}

func ensureLeadingSlash(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	if strings.HasPrefix(trimmed, "/") {
		return trimmed
	}
	return "/" + trimmed
}

func requestPathWithPrefix(r *http.Request) string {
	uri := firstForwarded(r.Header.Get("X-Forwarded-Uri"))
	prefix := firstForwarded(r.Header.Get("X-Forwarded-Prefix"))

	if uri == "" {
		uri = ensureLeadingSlash(r.URL.RequestURI())
		if prefix != "" {
			uri = joinURLPath(prefix, uri)
		}
	} else {
		uri = ensureLeadingSlash(uri)
		if prefix != "" && !strings.HasPrefix(uri, prefix) {
			uri = joinURLPath(prefix, uri)
		}
	}

	return uri
}

func forwardedProto(r *http.Request) string {
	proto := strings.ToLower(firstForwarded(r.Header.Get("X-Forwarded-Proto")))
	switch proto {
	case "http", "https":
		return proto
	}
	if r.TLS != nil {
		return "https"
	}
	if r.URL.Scheme != "" {
		switch strings.ToLower(r.URL.Scheme) {
		case "http", "https":
			return strings.ToLower(r.URL.Scheme)
		}
	}
	return "http"
}

func forwardedHost(r *http.Request) string {
	host := firstForwarded(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}

	port := strings.TrimSpace(firstForwarded(r.Header.Get("X-Forwarded-Port")))
	if port != "" && !strings.Contains(host, ":") {
		if (port == "80" && forwardedProto(r) == "http") || (port == "443" && forwardedProto(r) == "https") {
			return host
		}
		host = net.JoinHostPort(host, port)
	}

	return host
}

func shouldUseBrowserHostFallback(host string) bool {
	trimmed := strings.TrimSpace(host)
	if trimmed == "" {
		return true
	}

	name := trimmed
	if h, _, err := net.SplitHostPort(trimmed); err == nil {
		name = h
	}

	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "[")
	name = strings.TrimSuffix(name, "]")
	lower := strings.ToLower(name)

	if lower == "localhost" || lower == "::1" || lower == "0.0.0.0" {
		return true
	}
	if strings.HasPrefix(lower, "127.") {
		return true
	}

	return false
}

func firstForwarded(headerValue string) string {
	if headerValue == "" {
		return ""
	}
	parts := strings.Split(headerValue, ",")
	return strings.TrimSpace(parts[0])
}
