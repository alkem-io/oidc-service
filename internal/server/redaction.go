package server

import (
	"net/url"
	"strings"
)

// redactRedirectURL strips query parameters and fragments before logging redirects.
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
