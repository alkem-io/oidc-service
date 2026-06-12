package server

import (
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/zap"

	middlewarepkg "github.com/alkem-io/oidc-service/internal/middleware"
)

// safeRedirect is the single chokepoint through which every handler-issued
// HTTP redirect in this package flows. Before delegating to http.Redirect it
// re-validates the target:
//
//   - relative targets (no scheme, no host) are allowed — they stay on this
//     service's own origin (scheme-relative "//host" forms do not qualify);
//   - absolute targets must use http or https, which blocks javascript:,
//     data:, and similar scheme smuggling regardless of upstream behaviour;
//   - when allowedHosts is non-empty, the target's host must match one of
//     them (case-insensitive) — used where the expected host is known from
//     configuration, e.g. redirects built from the Kratos browser base URL.
//
// Hydra accept-response targets (the login/consent/logout redirect_to values)
// cannot be host-pinned here: they point at Hydra's *public* host, while the
// service is only configured with OIDC_HYDRA_ADMIN_URL, so the public host is
// not derivable from config. Those call sites rely on the scheme check plus
// the trust already placed in Hydra's admin API.
//
// A rejected target answers 502: every absolute target reaching this helper
// originates from an upstream (Hydra admin API, the configured Kratos base
// URL, or the operator-registered post-logout allow-list), so rejection
// signals a misconfigured or misbehaving upstream, not a bad client request.
//
// Accepted targets are answered with 302 Found, the status every challenge
// flow in this package uses.
func safeRedirect(w http.ResponseWriter, r *http.Request, target string, allowedHosts ...string) {
	if !redirectTargetAllowed(target, allowedHosts) {
		middlewarepkg.Logger(r.Context()).Warn(
			"rejected unsafe redirect target",
			zap.String("redirectTo", redactRedirectURL(target)),
		)
		http.Error(w, "invalid redirect target", http.StatusBadGateway)
		return
	}

	http.Redirect(w, r, target, http.StatusFound)
}

// redirectTargetAllowed reports whether target satisfies the redirect policy
// documented on safeRedirect.
func redirectTargetAllowed(target string, allowedHosts []string) bool {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" {
		return false
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return false
	}

	if parsed.Scheme == "" && parsed.Host == "" {
		// Relative path on this service's own origin.
		return true
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	if parsed.Host == "" {
		return false
	}
	if len(allowedHosts) == 0 {
		return true
	}

	host := strings.ToLower(parsed.Host)
	for _, allowed := range allowedHosts {
		if host == strings.ToLower(strings.TrimSpace(allowed)) {
			return true
		}
	}

	return false
}
