package server

import "testing"

func TestRedactRedirectURL(t *testing.T) {
	tests := map[string]string{ //nolint:gosec // test data, not real credentials
		"":                                    "",
		"https://example.com/callback":        "https://example.com/callback",
		"https://example.com/callback?code=1": "https://example.com/callback",
		"https://example.com/callback#frag":   "https://example.com/callback",
		"http://example.com/path?secret=1":    "http://example.com/path",
		"/relative/path":                      "/relative/path",
		":://bad":                             "redacted",
	}

	for input, expected := range tests {
		if got := redactRedirectURL(input); got != expected {
			t.Fatalf("redactRedirectURL(%q) = %q, want %q", input, got, expected)
		}
	}
}
