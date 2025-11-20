package support

import (
	"net/url"
	"testing"
)

// AssertStubRedirect validates that the provided Location header points to the stub callback
// and carries the expected flow and challenge identifiers. It returns the parsed query values
// for any additional custom assertions.
func AssertStubRedirect(t *testing.T, location string, expectedFlow, expectedChallenge string) url.Values {
	t.Helper()

	if location == "" {
		t.Fatalf("expected Location header to be set")
	}

	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("failed to parse redirect location %q: %v", location, err)
	}

	if parsed.Scheme != "https" || parsed.Host != "oidc.stub.local" || parsed.Path != "/callback" {
		t.Fatalf("unexpected stub redirect target %s", parsed.String())
	}

	query := parsed.Query()

	if expectedFlow != "" && query.Get("flow") != expectedFlow {
		t.Fatalf("expected flow=%s, got %s", expectedFlow, query.Get("flow"))
	}

	if expectedChallenge != "" && query.Get("challenge") != expectedChallenge {
		t.Fatalf("expected challenge=%s, got %s", expectedChallenge, query.Get("challenge"))
	}

	return query
}
