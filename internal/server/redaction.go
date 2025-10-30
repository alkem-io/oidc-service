package server

import (
	"net/url"
	"strings"
)

// redactRedirectURL produces a sanitized representation of a redirect URL suitable for logging.
// It trims surrounding whitespace and returns an empty string for empty input.
// On parse failure it returns "redacted".
// The returned value has user info, query parameters, and fragment removed.
// If the URL has no scheme and no host, it returns the path if present, otherwise "redacted".
// Otherwise it returns the scheme (followed by "://" when a host is present), the host, and the path.
func redactRedirectURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "redacted"
	}

	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""

	if parsed.Scheme == "" && parsed.Host == "" {
		if parsed.Path == "" {
			return "redacted"
		}
		return parsed.Path
	}

	var builder strings.Builder
	if parsed.Scheme != "" {
		builder.WriteString(parsed.Scheme)
		if parsed.Host != "" {
			builder.WriteString("://")
		}
	}
	if parsed.Host != "" {
		builder.WriteString(parsed.Host)
	}
	builder.WriteString(parsed.Path)

	if builder.Len() == 0 {
		return "redacted"
	}

	return builder.String()
}